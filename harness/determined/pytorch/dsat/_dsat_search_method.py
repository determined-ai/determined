import abc
import argparse
import collections
import copy
import dataclasses
import json
import logging
import pathlib
import pickle
import random
import uuid
from typing import Any, Deque, Dict, Iterable, Iterator, List, Optional, Set, Tuple, Union, cast

import numpy as np

from determined import searcher, util
from determined.experimental import client
from determined.pytorch import dsat
from determined.pytorch.dsat import defaults

logger = logging.getLogger("determined.pytorch")


class DSATTrial:
    """Encapsulation of DeepSpeed Autotune Trials.

    Simple objects for handling all pertinent information and results for every created Trial.
    Contains basic lineage tracking in which each `DSATTrial` instance holds direct references to
    its immediate parent and children, along with various helper properties.
    """

    def __init__(
        self,
        hparams: Dict[str, Any],
        model_dir: str,
        slots_per_trial: int,
        length: int,
        request_id: Optional[uuid.UUID] = None,
        parent: Optional["DSATTrial"] = None,
        search_data: Optional["DSATSearchData"] = None,
        searcher_metric_name: Optional[str] = None,
    ) -> None:
        self.hparams = hparams
        self.model_dir = model_dir
        self.slots_per_trial = slots_per_trial
        self.length = length
        self.request_id = request_id or uuid.uuid4()
        self.parent = parent
        # Arbitrary attribute for search-specific data tracking.
        self.search_data: Optional["DSATSearchData"] = search_data
        self.searcher_metric_name = searcher_metric_name

        # Other attrs which are updated during training:

        self.metric: Union[float, Dict[str, Any]] = {}
        self.error = False
        self.running = False
        self.children: Set["DSATTrial"] = set()

        # If a parent was specified, register the current Trial as the parent's child.
        if self.parent is not None:
            self.parent.children.add(self)

        self.lineage_root: DSATTrial = self if self.parent is None else self.parent.lineage_root

        # The DS config json file may either be in the specified model directory or in the base of
        # the workdir, if it was added as an `--include` arg.
        try:
            self.ds_config = dsat.get_ds_config_from_hparams(self.hparams, self.model_dir)
        except FileNotFoundError:
            self.ds_config = dsat.get_ds_config_from_hparams(self.hparams)

        self._error_in_direct_history = False

    @property
    def completed(self) -> bool:
        return bool(self.error or self.metric)

    @property
    def lineage_set(self) -> Set["DSATTrial"]:
        """Computes set of trials in lineage tree."""
        root = self.lineage_root
        trials_set = {root}
        children = set(root.children)
        while children:
            random_child = children.pop()
            trials_set.add(random_child)
            children |= random_child.children
        return trials_set

    @property
    def num_completed_trials_in_lineage(self) -> int:
        """Computes total number of trials in lineage tree."""
        num_trials = sum(trial.completed for trial in self.lineage_set)
        return num_trials

    @property
    def error_in_direct_history(self) -> bool:
        if self._error_in_direct_history:
            return self._error_in_direct_history
        trial: Optional["DSATTrial"] = self
        while trial is not None:
            if trial.error:
                return True
            trial = trial.parent
        return False

    @property
    def mbs_in_lineage(self) -> Set[int]:
        """
        Returns the set of all `train_micro_batch_size_per_gpu` (mbs) used in the Trial's lineage.
        """
        mbs_in_lineage = {t.mbs for t in self.lineage_set}
        return mbs_in_lineage

    @property
    def stage(self) -> int:
        return int(self.ds_config.get("zero_optimization", {}).get("stage", 0))

    @property
    def fp16(self) -> bool:
        return bool(self.ds_config.get("fp16", {}).get("enabled")) or False

    @property
    def mbs(self) -> int:
        assert "train_micro_batch_size_per_gpu" in self.ds_config, (
            "The DSATTrial must be provided with a `ds_config` that contains the"
            " key `train_micro_batch_size_per_gpu`"
        )
        assert isinstance(
            self.ds_config["train_micro_batch_size_per_gpu"], int
        ), "The DSATTrial must be provided an `int` value for `train_micro_batch_size_per_gpu`"
        return self.ds_config["train_micro_batch_size_per_gpu"]

    @property
    def create_and_val_ops(self) -> List[searcher.Operation]:
        """
        Returns a list with the searcher.Create and searcher.ValidateAfter operations
        needed to initiate and run the specified Trial.
        """
        create_op = searcher.Create(
            request_id=self.request_id,
            hparams=self.hparams,
            checkpoint=None,
        )
        validate_after_op = searcher.ValidateAfter(request_id=self.request_id, length=self.length)
        ops_list = [create_op, validate_after_op]

        return ops_list

    @property
    def searcher_metric_val(self) -> Optional[float]:
        if self.searcher_metric_name is None:
            return None
        if isinstance(self.metric, float):
            return self.metric
        val = self.metric.get(self.searcher_metric_name)
        if val is not None:
            return float(val)
        return val


class DSATModelProfileInfoTrial(DSATTrial):
    """
    Super class for differentiating the model profiling info run.
    """


class DSATTrialTracker:
    """Primary stateful object for tracking DeepSpeed Autotune Experiments.

    Holds references to all genereated `DSATTrial` instances, as well as the
    `DSATModelProfileInfoTrial` and handles queueing through its `queue` attribute.
    Class for organizing DSATTrial instances and retrieving pertinent info. Provides helper
    functions for generating the appropriate `DSATModelProfileInfoTrial` and `DSATTrial` instances
    with consistent batch sizes and configurations in line with CLI arguments.
    """

    def __init__(
        self,
        args: argparse.Namespace,
        exp_config: Dict[str, Any],
    ) -> None:
        self.exp_config = exp_config
        self.max_trials: int = args.max_trials
        self.max_concurrent_trials = args.max_concurrent_trials
        self.max_slots: int = args.max_slots
        self.model_dir = args.model_dir
        self.searcher_metric = args.metric
        self.start_profile_step = args.start_profile_step
        self.end_profile_step = args.end_profile_step
        self.zero_stages = set(args.zero_stages)

        # Derived attributes
        self.slots_per_trial: int = self.exp_config["resources"]["slots_per_trial"]
        self.hparams: Dict[str, Any] = self.exp_config["hyperparameters"]

        self.smaller_is_better = dsat.smaller_is_better(self.searcher_metric)

        self.model_profile_info_trial: Optional["DSATTrial"] = None
        self.num_trials_since_best_result: int = 0
        self.successful_stages: Set[int] = set()
        self._all_trials_dict: Dict[uuid.UUID, "DSATTrial"] = {}
        self.queue: Deque["DSATTrial"] = collections.deque()

        self._mem_per_gpu_per_stage: Optional[Dict[int, int]] = None
        self._approx_max_mbs_per_stage: Optional[Dict[int, int]] = None

    def __len__(self) -> int:
        return len(self._all_trials_dict)

    def __getitem__(self, request_id: uuid.UUID) -> DSATTrial:
        return self._all_trials_dict[request_id]

    def __iter__(self) -> Iterator[Tuple[uuid.UUID, "DSATTrial"]]:
        return iter(self._all_trials_dict.items())

    def __contains__(self, item: Union[uuid.UUID, DSATTrial]) -> bool:
        if isinstance(item, uuid.UUID):
            return item in self._all_trials_dict
        elif isinstance(item, DSATTrial):
            return item in self._all_trials_dict.values()
        else:
            raise ValueError(
                f"Expected a `uuid.UUID` or `DSATTrial` instance, instead received an object of"
                f" type {type(item)}"
            )

    def create_trial(
        self,
        hparams: Dict[str, Any],
        search_data: Optional[Any] = None,
        parent_trial: Optional[DSATTrial] = None,
    ) -> DSATTrial:
        """
        Helper function which creates a new `DSATTrial` object of the appropriate length, given the
        config, while also enforcing a consistent DS batch size configuration.
        """
        # Create a consistent batch size configuration which obeys the DS constraints.
        self.enforce_consistent_batch_config(hparams)

        # For some reason, DS (0.8.3) exits in the DeepSpeedEngine.step call when
        # DeepSpeedEngine.global_step (initiated at zero) equals end_profile_step + 1,
        # with global_step updated *before* this check happens. So, we need to run for
        # a length of end_profile_step + 1 to trigger the exit. Presumably an off-by-one error
        # on their end.
        trial = DSATTrial(
            hparams=hparams,
            model_dir=self.model_dir,
            slots_per_trial=self.slots_per_trial,
            length=self.end_profile_step + 1,
            parent=parent_trial,
            search_data=search_data,
            searcher_metric_name=self.searcher_metric,
        )
        return trial

    def create_model_profile_info_trial(
        self,
    ) -> DSATModelProfileInfoTrial:
        # Create the special hp dictionary used for the model profile info run.
        model_profile_info_hps = copy.deepcopy(self.hparams)
        model_profile_info_hps[defaults.OVERWRITE_KEY] = util.merge_dicts(
            model_profile_info_hps.get(defaults.OVERWRITE_KEY, {}),
            defaults.MODEL_INFO_PROFILE_DS_CONFIG,
        )
        self.enforce_consistent_batch_config(model_profile_info_hps)

        model_profile_info_trial = DSATModelProfileInfoTrial(
            hparams=model_profile_info_hps,
            model_dir=self.model_dir,
            slots_per_trial=self.slots_per_trial,
            length=1,  # Only need a single step.
        )
        self.model_profile_info_trial = model_profile_info_trial
        return model_profile_info_trial

    def queue_and_register_trial(self, trial: DSATTrial) -> None:
        """
        Helper function which both adds the `trial` to the queue and the internal dictionary
        tracking all trials.
        """
        # Verify that the given trial was not previously completed.
        for other_trial in self.completed_trials:
            if trial.hparams == other_trial.hparams:
                logger.warning(
                    f"Skipping attempt to queue Trial identical to {other_trial.request_id}"
                )
        self._all_trials_dict[trial.request_id] = trial
        self.queue.append(trial)

    def enforce_consistent_batch_config(self, hparams: Dict[str, Any]) -> None:
        """Enforces a consistent batch size configuration by altering `hparams` in-place."""
        try:
            ds_config = dsat.get_ds_config_from_hparams(hparams, self.model_dir)
        except FileNotFoundError:
            # In case the DS json config was added as an `--include` arg.
            ds_config = dsat.get_ds_config_from_hparams(hparams)
        batch_size_config = dsat.get_batch_config_from_mbs_gas_and_slots(
            ds_config, slots=self.slots_per_trial
        )
        hparams[defaults.OVERWRITE_KEY] = util.merge_dicts(
            hparams[defaults.OVERWRITE_KEY], batch_size_config
        )

    def update_trial_metric(
        self,
        trial: DSATTrial,
        metric: Union[float, Dict[str, Any]],
    ) -> None:
        """
        Updates the Trial Tracker after metrics have been reported, attaching the reported metrics
        to the `DSATTrial` instnace and updating early-stopping bookkeeping.
        """
        trial.metric = metric

        # The model info profiling run's metric will not contain the searcher metric key and should
        # not be counted against the early stopping criteria.
        if not isinstance(trial, DSATModelProfileInfoTrial):
            self.successful_stages.add(trial.stage)
            trial_is_best = self.best_trial == trial
            if trial_is_best:
                self.num_trials_since_best_result = 0
            else:
                self.num_trials_since_best_result += 1
        trial.running = False

    def report_trial_early_exit(self, trial: DSATTrial) -> None:
        # `self.num_trials_since_best_result` is only incremented after a best trial has been
        # established.
        if self.best_trial is not None:
            self.num_trials_since_best_result += 1

        trial.error = True
        trial.running = False

    def _fetch_model_profile_info_data(self, param_name: str) -> int:
        assert (
            self.model_profile_info_trial is not None
        ), f"The `DSATModelProfileInfoTrial` must be run before requesting its `{param_name}`"
        assert isinstance(
            self.model_profile_info_trial.metric, dict
        ), "The `DSATModelProfileInfoTrial` must be provided with a metric dictionary"
        assert param_name in self.model_profile_info_trial.metric, (
            "The `DSATModelProfileInfoTrial` must be provided with a metric dict that contains the"
            f" key `{param_name}`"
        )
        assert isinstance(
            self.model_profile_info_trial.metric[param_name], int
        ), f"The `DSATModelProfileInfoTrial` must be provided an `int` value for `{param_name}`"
        return int(self.model_profile_info_trial.metric[param_name])

    @property
    def gpu_mem(self) -> int:
        """
        Returns the available GPU memory in bytes according to the `DSATModelProfileInfoTrial`
        """
        return self._fetch_model_profile_info_data("gpu_mem")

    @property
    def num_params(self) -> int:
        """
        Returns the number of params according to the `DSATModelProfileInfoTrial`
        """
        return self._fetch_model_profile_info_data("num_params")

    @property
    def trainable_num_params(self) -> int:
        """
        Returns the number of trainable params according to the `DSATModelProfileInfoTrial`
        """
        return self._fetch_model_profile_info_data("trainable_num_params")

    @property
    def activation_mem_per_gpu(self) -> int:
        """
        Returns the amount of activation memory per gpu in bytes according to
        the `DSATModelProfileInfoTrial`
        """
        return self._fetch_model_profile_info_data("activation_mem_per_gpu")

    @property
    def mem_per_gpu_per_stage(self) -> Dict[int, int]:
        """
        Returns the required gpu memory in bytes, per stage, according to whether fp16 training was
        used (other low-precision cases not handled).
        """
        assert (
            self.model_profile_info_trial is not None
        ), "The model profile info Trial must be run before calling this method."
        fp16 = self.model_profile_info_trial.fp16
        if self._mem_per_gpu_per_stage is None:
            params_mem = self.num_params * (2 if fp16 else 4)
            # Gradients must be converted to fp32 to update master weights, so they eventually
            # require the same memory regardless of whether mixed-precision is used.
            gradients_mem = self.trainable_num_params * 4
            # optimizer_mem assumes Adam, following DS. TODO: don't assume this (MLG-584).
            master_params_mem = 4 if fp16 else 0
            momentum_mem = variance_mem = 4
            optimizer_mem = self.trainable_num_params * (
                master_params_mem + momentum_mem + variance_mem
            )

            non_activation_mem_per_gpu_per_stage = {
                0: params_mem + gradients_mem + optimizer_mem,
                1: params_mem + gradients_mem + optimizer_mem // self.slots_per_trial,
                2: params_mem + (gradients_mem + optimizer_mem) // self.slots_per_trial,
                3: (params_mem + gradients_mem + optimizer_mem) // self.slots_per_trial,
            }
            # In DS there is an mp_size int which can be used for model parallelism and also enters
            # the memory computation, but we will not support that feature at the moment.

            mem_per_gpu_per_stage = {
                stage: mem + self.activation_mem_per_gpu
                for stage, mem in non_activation_mem_per_gpu_per_stage.items()
            }
            self._mem_per_gpu_per_stage = mem_per_gpu_per_stage
        return self._mem_per_gpu_per_stage

    @property
    def approx_max_mbs_per_stage(self) -> Dict[int, int]:
        """
        Returns the approximate max train_micro_batch_size_per_gpu (mbs) per stage.
        """
        if self._approx_max_mbs_per_stage is None:
            self._approx_max_mbs_per_stage = {
                stage: max((self.gpu_mem - mem) // self.activation_mem_per_gpu, 1)
                for stage, mem in self.mem_per_gpu_per_stage.items()
            }
        return self._approx_max_mbs_per_stage

    def _best_trial_fn(self, trials: Iterable["DSATTrial"]) -> Optional["DSATTrial"]:
        trials_with_searcher_metric = [
            trial
            for trial in trials
            if not isinstance(trial, DSATModelProfileInfoTrial)
            and isinstance(trial.metric, dict)
            and self.searcher_metric in trial.metric
        ]
        if not trials_with_searcher_metric:
            return None

        min_or_max = min if self.smaller_is_better else max
        best_trial = min_or_max(
            trials_with_searcher_metric,
            key=lambda trial: trial.metric
            if isinstance(trial.metric, float)
            else float(trial.metric[self.searcher_metric]),
        )
        return best_trial

    @property
    def best_trials_by_stage(self) -> Dict[int, Optional["DSATTrial"]]:
        _best_trials_by_stage: Dict[int, Optional["DSATTrial"]] = {}
        for stage in range(4):
            trials_to_check = [trial for _, trial in self if trial.stage == stage]
            best_trial = self._best_trial_fn(trials_to_check)
            _best_trials_by_stage[stage] = best_trial
        return _best_trials_by_stage

    @property
    def best_trial(self) -> Optional["DSATTrial"]:
        best_trial = self._best_trial_fn(
            trial for trial in self.best_trials_by_stage.values() if trial is not None
        )
        return best_trial

    @property
    def running_trials(self) -> List[DSATTrial]:
        return [trial for _, trial in self if trial.running]

    @property
    def completed_trials(self) -> List[DSATTrial]:
        return [trial for _, trial in self if trial.completed]

    @property
    def num_running_trials(self) -> int:
        return len(self.running_trials)

    @property
    def num_completed_trials(self) -> int:
        return len(self.completed_trials)

    @property
    def max_trials_queued(self) -> bool:
        return len(self.queue) >= self.max_trials

    @property
    def max_trials_are_running_or_closed(self) -> bool:
        return self.num_running_trials + self.num_completed_trials >= self.max_trials

    @property
    def should_be_failure(self) -> bool:
        model_profile_info_trial_failed = (
            self.model_profile_info_trial is not None and self.model_profile_info_trial.error
        )
        every_autotuning_trial_failed = all(
            trial.error
            for _, trial in self
            if trial.completed and not isinstance(trial, DSATModelProfileInfoTrial)
        )
        return model_profile_info_trial_failed or every_autotuning_trial_failed

    @property
    def can_run_more_trials(self) -> int:
        if not self.queue:
            return False
        if self.max_trials_are_running_or_closed:
            return False
        if self.num_running_trials >= self.max_concurrent_trials:
            return False
        if self.max_slots is not None:
            occupied_slots = self.num_running_trials * self.slots_per_trial
            remaining_slots = self.max_slots - occupied_slots
            trials_available_with_remaining_slots = remaining_slots // self.slots_per_trial
            return trials_available_with_remaining_slots > 0
        return True


class BaseDSATSearchMethod(searcher.SearchMethod):
    """Base class for all Determined AI DeepSpeed Autotune searchers.

    Contains two abstract methods: `get_trials_after_validation_completed` and
    `get_trials_after_early_exit` which return iterables of `DSATTrial` after their respective
    events occur. The `early_stopping_triggered` and `choose_next_trial_from_queue` methods are also
    provided with the intention of overwriting for further fine-grained control. The base class
    ensures that global constraints such as `max_trials`, `max_concurrent_trials`, and `max_slots`
    are respected by all subclasses.  The `trial_tracker` attribute (a `DSATTrialTracker` instance)
    is the stateful object which tracks results and the queued Trials.
    """

    def __init__(self, args: argparse.Namespace, exp_config: Dict[str, Any]) -> None:
        # Storing args so that additional args can be inherited by child classes
        self.args = args
        self.exp_config = exp_config
        self.trial_tracker = DSATTrialTracker(args=args, exp_config=exp_config)
        self.rng = np.random.default_rng(seed=args.random_seed)
        random.seed(args.random_seed)

        self._tracker_ckpt_path = "trial_tracker.pkl"
        self._py_rand_ckpt_path = "py_random_state.pkl"
        self._np_rand_ckpt_path = "np_rng.pkl"

    @abc.abstractmethod
    def get_trials_after_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        metric: Union[float, Dict[str, Any]],
    ) -> Iterable[DSATTrial]:
        """
        All returned `DSATTrial`s will be `append`-ed to `self.trial_tracker.queue` in the order
        they are provided.
        """
        pass

    @abc.abstractmethod
    def get_trials_after_early_exit(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        exited_reason: searcher.ExitedReason,
    ) -> Iterable[DSATTrial]:
        """
        All returned `DSATTrial`s will be `append`-ed to `self.trial_tracker.queue` in the order
        they are provided.
        """
        pass

    def choose_next_trial_from_queue(self) -> DSATTrial:
        """
        Called whenever resources exist to run an additional Trial. Overwrite if more complex
        logic is needed.
        """

        next_trial = self.trial_tracker.queue.popleft()
        return next_trial

    def initial_operations(
        self, searcher_state: searcher.SearcherState
    ) -> List[searcher.Operation]:
        """
        Submits the model info profiling run in order to collect model and resources info to
        inform the search.
        """

        model_profile_info_trial = self.trial_tracker.create_model_profile_info_trial()
        self.trial_tracker.queue_and_register_trial(model_profile_info_trial)
        self.trial_tracker.queue.popleft()  # Needed for bookkeeping.
        ops = model_profile_info_trial.create_and_val_ops
        return ops

    def on_trial_created(
        self, searcher_state: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        return []

    def on_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
        train_length: int,
    ) -> List[searcher.Operation]:
        last_trial = self.trial_tracker[request_id]
        self.trial_tracker.update_trial_metric(trial=last_trial, metric=metric)

        if isinstance(last_trial, DSATModelProfileInfoTrial):
            logger.info(f"Approx. max mbs per stage: {self.trial_tracker.approx_max_mbs_per_stage}")
            logger.info(f"Approx. GPU memory per stage: {self.trial_tracker.mem_per_gpu_per_stage}")
            logger.info(f"Total GPU memory: {self.trial_tracker.gpu_mem}")

        if not self.trial_tracker.max_trials_queued and not self.should_shutdown():
            new_trials = self.get_trials_after_validation_completed(
                searcher_state=searcher_state,
                last_trial=last_trial,
                metric=metric,
            )
            for trial in new_trials:
                self.trial_tracker.queue_and_register_trial(trial)

        # All DS AT Trials should be closed after validation.
        return [searcher.Close(request_id)]

    def on_trial_exited_early(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        exited_reason: searcher.ExitedReason,
    ) -> List["searcher.Operation"]:
        last_trial = self.trial_tracker[request_id]
        self.trial_tracker.report_trial_early_exit(last_trial)

        new_ops_list: List["searcher.Operation"] = []
        if exited_reason != searcher.ExitedReason.ERRORED:
            # In case of INVALID_HP or USER_CANCELED, shut down the searcher.
            logger.info(
                f"Shutting down: unexpected early exit due to {exited_reason}"
                f"\nLast trial: {last_trial}, request_id: {request_id}"
            )
            new_ops_list.append(searcher.Shutdown(failure=self.trial_tracker.should_be_failure))
        if not self.trial_tracker.max_trials_queued and not self.should_shutdown():
            # ERRORED Trials generally corresponds to OOMs, after which we may want to submit
            # follow-on Trials.
            new_trials = self.get_trials_after_early_exit(
                searcher_state=searcher_state,
                last_trial=last_trial,
                exited_reason=exited_reason,
            )
            for trial in new_trials:
                self.trial_tracker.queue_and_register_trial(trial)
            self.trial_tracker.queue

        return new_ops_list

    def on_trial_closed(
        self, searcher_state: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        new_ops_list: List[searcher.Operation] = []
        if self.should_shutdown():
            if self.trial_tracker.best_trial is not None and self.args.run_full_experiment:
                submitted_config = dsat.get_dict_from_yaml_or_json_path(self.args.config_path)
                optimal_config = util.merge_dicts(
                    submitted_config, {"hyperparameters": self.trial_tracker.best_trial.hparams}
                )
                # Delete the keys which enforce autotuning code paths
                del optimal_config["hyperparameters"][defaults.OVERWRITE_KEY]["autotuning"]
                del optimal_config["hyperparameters"][defaults.USE_DSAT_MODE_KEY]
                client.create_experiment(optimal_config, self.args.model_dir, self.args.include)

            new_ops_list.append(searcher.Shutdown(failure=self.trial_tracker.should_be_failure))
        else:
            while self.trial_tracker.can_run_more_trials:
                next_trial = self.choose_next_trial_from_queue()
                next_trial.running = True
                new_ops_list.extend(next_trial.create_and_val_ops)

        return new_ops_list

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        progress = len(searcher_state.trials_closed) / self.trial_tracker.max_trials
        return progress

    def save_method_state(self, path: pathlib.Path) -> None:
        with path.joinpath(self._tracker_ckpt_path).open("wb") as f:
            pickle.dump(self.trial_tracker, f)
        with path.joinpath(self._py_rand_ckpt_path).open("wb") as f:
            pickle.dump(random.getstate(), f)
        with path.joinpath(self._np_rand_ckpt_path).open("wb") as f:
            pickle.dump(self.rng, f)
        if self.trial_tracker.best_trial is not None:
            with path.joinpath("best_ds_config.json").open("w") as ds_config_f:
                best_ds_metrics = copy.deepcopy(self.trial_tracker.best_trial.ds_config)
                del best_ds_metrics["autotuning"]
                json.dump(best_ds_metrics, ds_config_f)
            with path.joinpath("best_ds_metrics.json").open("w") as ds_metrics_f:
                json.dump(self.trial_tracker.best_trial.metric, ds_metrics_f)

    def load_method_state(self, path: pathlib.Path) -> None:
        logger.info("Restoring searcher state from checkpoint.")
        with path.joinpath(self._tracker_ckpt_path).open("rb") as f:
            self.trial_tracker = cast(DSATTrialTracker, pickle.load(f))
        with path.joinpath(self._py_rand_ckpt_path).open("rb") as f:
            py_random_state = pickle.load(f)
            random.setstate(py_random_state)
        with path.joinpath(self._np_rand_ckpt_path).open("rb") as f:
            self.rng = pickle.load(f)

    def should_shutdown(self) -> bool:
        """
        Conditions on which to shutdown the search.
        """
        if (
            self.trial_tracker.model_profile_info_trial is not None
            and self.trial_tracker.model_profile_info_trial.error
        ):
            logger.info(
                "Shutting down: error in model profile info Trial."
                " You may need to specify a configuration which can successfully run with"
                " `train_micro_batch_size_per_gpu = 1`."
            )
            return True
        if self.early_stopping_triggered():
            logger.info("Shutting down: early stopping criteria met.")
            return True
        if self.trial_tracker.num_completed_trials >= self.trial_tracker.max_trials:
            logger.info("Shutting down: all Trials completed.")
            return True
        return False

    def early_stopping_triggered(self) -> bool:
        """
        Overwrite to implement search-method-specific early-stopping logic.
        """
        return False


@dataclasses.dataclass
class DSATSearchData:
    """Basic binary-search type data used to guide DS AT."""

    lo: int
    hi: int


class RandomDSATSearchMethod(BaseDSATSearchMethod):
    """
    Implements a random search through DeepSpeed configuration space with an approximate binary
    search on batch sizes.  Utilizes aggressive early stopping based on the results of other Trials
    and heuristics based on domain knowledge of DeepSpeed. Uses two search-specific arguments:

    Args:
        trials_per_random_config:
            the maximum number of Trials which will be used to optimize each randomly-generated
            configuration
        early_stopping:
            the maximum number of Trials to run without improving results after a best-found
            configuration has been established
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self.trials_per_random_config = self.args.trials_per_random_config
        self.early_stopping: int = self.args.early_stopping

    def get_trials_after_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[DSATTrial]:
        new_trials = []
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            new_trials = self.get_trial_list_after_model_profile_info_run()
        else:
            new_trials = self.get_trial_list_after_successful_run(last_trial)

        return new_trials

    def get_trials_after_early_exit(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        exited_reason: searcher.ExitedReason,
    ) -> List[DSATTrial]:
        new_trials = []

        if self.should_stop_lineage(last_trial):
            logger.info(f"Killing trial {last_trial.request_id}")
            new_trials.append(self.get_random_trial())
        else:
            if last_trial.search_data is None:
                return new_trials
            new_search_data = copy.deepcopy(last_trial.search_data)
            new_search_data.hi = last_trial.mbs - 1

            mbs = self.get_random_mbs_from_search_data(new_search_data)
            new_hparams = copy.deepcopy(last_trial.hparams)
            new_hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs

            trial = self.trial_tracker.create_trial(
                hparams=new_hparams,
                search_data=new_search_data,
                parent_trial=last_trial,
            )
            new_trials.append(trial)
        return new_trials

    def choose_next_trial_from_queue(self) -> DSATTrial:
        """
        Continually removes Trials whose lineages should be stopped from the front of the queue
        while adding their corresponding replacements, finally returning the next Trial which should
        be run.
        """

        next_trial = self.trial_tracker.queue.popleft()
        while self.should_stop_lineage(next_trial):
            self.trial_tracker.queue_and_register_trial(self.get_random_trial())
            next_trial = self.trial_tracker.queue.popleft()

        return next_trial

    def get_trial_list_after_model_profile_info_run(self) -> List[DSATTrial]:
        new_trials = []
        concurrent_trials = self.args.max_concurrent_trials
        if self.args.max_slots is not None:
            concurrent_trials_from_slots = self.args.max_slots // self.trial_tracker.slots_per_trial
            concurrent_trials = min(concurrent_trials, concurrent_trials_from_slots)
        for _ in range(concurrent_trials):
            trial = self.get_random_trial()
            new_trials.append(trial)
        return new_trials

    def get_trial_list_after_successful_run(
        self,
        last_trial: DSATTrial,
    ) -> List[DSATTrial]:
        if self.should_stop_lineage(trial=last_trial) or last_trial.search_data is None:
            return [self.get_random_trial()]

        new_search_data = copy.deepcopy(last_trial.search_data)
        new_search_data.lo = last_trial.mbs + 1
        # It is possible lo > hi in the case where initial soft ceiling computation was innaccurate
        # in which case we double hi.
        if new_search_data.lo > new_search_data.hi:
            new_search_data.hi *= 2

        mbs = self.get_random_mbs_from_search_data(new_search_data)

        new_hparams = copy.deepcopy(last_trial.hparams)
        new_hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs
        trial = self.trial_tracker.create_trial(
            hparams=new_hparams,
            search_data=new_search_data,
            parent_trial=last_trial,
        )
        return [trial]

    def should_stop_lineage(self, trial: DSATTrial) -> bool:
        # General conditions
        assert trial.search_data is not None
        failed_on_min_mbs = trial.error and trial.search_data and trial.mbs <= trial.search_data.lo

        exceeded_trials_per_random_config_limit = (
            trial.num_completed_trials_in_lineage >= self.trials_per_random_config
        )

        # DS domain knowledge: if stages 1 or 2 run successfully, there is no need to use stage 3.
        stage_one_or_two_successful = {1, 2} & self.trial_tracker.successful_stages
        should_stop_this_stage_3_trial = trial.stage == 3 and stage_one_or_two_successful

        # Check if other same-stage trials have successfully run with larger batch sizes than this
        # lineage can possibly run.

        other_configs_run_larger_batch_sizes = (
            trial.error_in_direct_history
            and trial.search_data
            and any(
                other_trial.mbs >= trial.search_data.hi
                for _, other_trial in self.trial_tracker
                if other_trial.stage == trial.stage and other_trial.searcher_metric_val is not None
            )
        )

        if (
            failed_on_min_mbs
            or exceeded_trials_per_random_config_limit
            or should_stop_this_stage_3_trial
            or other_configs_run_larger_batch_sizes
        ):
            return True

        return False

    def get_random_mbs_from_search_data(self, search_data: DSATSearchData) -> int:
        """
        Randomly choose a mbs given the `search_data` bounds. Random choice covers a larger search
        volume than simply choosing the midpoint. Draws from a binomial distribution, to keep the
        results still somewhat focused near the midpoint.
        """
        mbs: int = search_data.lo + self.rng.binomial(search_data.hi - search_data.lo, 0.5)
        return mbs

    def get_random_hparams_and_search_data(
        self, zero_stage: int
    ) -> Tuple[Dict[str, Any], DSATSearchData]:
        zero_optim_config = dsat.get_random_zero_optim_config(zero_stage)
        new_hparams = copy.deepcopy(self.trial_tracker.hparams)
        new_hparams[defaults.OVERWRITE_KEY] = util.merge_dicts(
            new_hparams.get(defaults.OVERWRITE_KEY, {}),
            {"zero_optimization": zero_optim_config},
        )

        # If a best trial has been established for the given stage, use its search data bounds to
        # choose a better starting point.
        best_trial_for_stage = self.trial_tracker.best_trials_by_stage[zero_stage]

        if best_trial_for_stage is not None and best_trial_for_stage.search_data is not None:
            new_search_data = copy.deepcopy(best_trial_for_stage.search_data)
            # Update the floor to one greater than the mbs used and raise the ceiling to be
            # the maximum between the largest mbs trial of this stage which was successful, the
            # best trial's ceiling, and twice as large as the floor.
            new_search_data.lo = best_trial_for_stage.mbs + 1
            largest_successful_batch_size_for_stage = max(
                t.mbs
                for t in self.trial_tracker.completed_trials
                if t.stage == best_trial_for_stage.stage
                and isinstance(t.metric, dict)
                and t.metric.get(self.trial_tracker.searcher_metric) is not None
            )
            new_search_data.hi = max(
                largest_successful_batch_size_for_stage, new_search_data.hi, 2 * new_search_data.lo
            )
        # Otherwise choose the corresponding search data based on approximate computations
        else:
            random_zero_stage_max_mbs = self.trial_tracker.approx_max_mbs_per_stage[zero_stage]
            new_search_data = DSATSearchData(lo=1, hi=random_zero_stage_max_mbs)

        # Randomly choose the actual batch size.
        mbs = self.get_random_mbs_from_search_data(new_search_data)
        new_hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs
        return new_hparams, new_search_data

    def get_random_trial(self) -> DSATTrial:
        # Choose the stage randomly from user provided stages, after some performance filtering.
        # If stage one or two was successful, don't continue with stage 3.
        stage_one_and_two = {1, 2}
        successful_one_or_two_stages = stage_one_and_two & self.trial_tracker.successful_stages
        filtered_zero_stages = successful_one_or_two_stages & self.trial_tracker.zero_stages
        if filtered_zero_stages:
            zero_stage = random.choice(list(filtered_zero_stages))
        else:
            zero_stage = random.choice(list(self.trial_tracker.zero_stages))

        hparams, search_data = self.get_random_hparams_and_search_data(zero_stage)
        random_trial = self.trial_tracker.create_trial(hparams=hparams, search_data=search_data)
        return random_trial

    def early_stopping_triggered(self) -> bool:
        if self.early_stopping is None:
            return False
        return self.trial_tracker.num_trials_since_best_result >= self.early_stopping


class BinarySearchDSATSearchMethod(BaseDSATSearchMethod):
    """Basic binary search over randomly generated configurations.

    Randomly generates as many DeepSpeed configurations as can be concurrently tested, per the
    CLI arguments, and performs a binary search over batch size. Each such lineage runs to
    completion or until the `max_trials` limit is hit. Lineages whose binary search ends before
    `max_trials` is hit are replaced with newly generated random configurations. One search-specific
    argument:

    Args:
        search_range_factor:
            adjusts the initial binary search range by raising the ceiling by a factor of
            `search_range_factor`
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self.search_range_factor = self.args.search_range_factor

    def get_trials_after_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[DSATTrial]:
        new_trials = []
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            new_trials = self.get_trial_list_after_model_profile_info_run()
        else:
            new_trials = self.get_trial_list_after_successful_run(last_trial)

        return new_trials

    def get_trials_after_early_exit(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        exited_reason: searcher.ExitedReason,
    ) -> List[DSATTrial]:
        new_trials = []
        if last_trial.search_data is None:
            return [self.get_random_trial()]
        new_search_data = copy.deepcopy(last_trial.search_data)
        new_search_data.hi = last_trial.mbs - 1
        if new_search_data.lo > new_search_data.hi:
            return [self.get_random_trial()]

        mbs = (new_search_data.hi + new_search_data.lo) // 2
        new_hparams = copy.deepcopy(last_trial.hparams)
        new_hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs

        trial = self.trial_tracker.create_trial(
            hparams=new_hparams,
            search_data=new_search_data,
            parent_trial=last_trial,
        )
        new_trials.append(trial)
        return new_trials

    def get_trial_list_after_model_profile_info_run(self) -> List[DSATTrial]:
        new_trials = []
        concurrent_trials = self.args.max_concurrent_trials
        if self.args.max_slots is not None:
            concurrent_trials_from_slots = self.args.max_slots // self.trial_tracker.slots_per_trial
            concurrent_trials = min(concurrent_trials, concurrent_trials_from_slots)
        for _ in range(concurrent_trials):
            trial = self.get_random_trial()
            new_trials.append(trial)
        return new_trials

    def get_trial_list_after_successful_run(
        self,
        last_trial: DSATTrial,
    ) -> List[DSATTrial]:
        if last_trial.search_data is None:
            return [self.get_random_trial()]
        new_search_data = copy.deepcopy(last_trial.search_data)
        new_search_data.lo = last_trial.mbs + 1
        if new_search_data.lo > new_search_data.hi:
            return [self.get_random_trial()]

        mbs = (new_search_data.hi + new_search_data.lo) // 2
        new_hparams = copy.deepcopy(last_trial.hparams)
        new_hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs
        trial = self.trial_tracker.create_trial(
            hparams=new_hparams,
            search_data=new_search_data,
            parent_trial=last_trial,
        )
        return [trial]

    def get_random_hparams_and_search_data(
        self, zero_stage: int
    ) -> Tuple[Dict[str, Any], DSATSearchData]:
        zero_optim_config = dsat.get_random_zero_optim_config(zero_stage)
        new_hparams = copy.deepcopy(self.trial_tracker.hparams)
        new_hparams[defaults.OVERWRITE_KEY] = util.merge_dicts(
            new_hparams.get(defaults.OVERWRITE_KEY, {}),
            {"zero_optimization": zero_optim_config},
        )

        random_zero_stage_max_mbs = self.trial_tracker.approx_max_mbs_per_stage[zero_stage]

        # The default `search_range_factor = 1.` value makes the ceiling coincide with
        # the predicted max mbs, but we give the user a handle to alter this range as needed.
        lo = 1
        hi = int(self.search_range_factor * random_zero_stage_max_mbs)
        hi = max(hi, lo)
        new_search_data = DSATSearchData(lo=1, hi=hi)

        mbs = (new_search_data.hi + new_search_data.lo) // 2
        new_hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs
        return new_hparams, new_search_data

    def get_random_trial(self) -> DSATTrial:
        # Choose the stage randomly from user provided stages, after some performance filtering.
        # If stage one or two was successful, don't continue with stage 3.
        zero_stage = random.choice(list(self.trial_tracker.zero_stages))
        hparams, search_data = self.get_random_hparams_and_search_data(zero_stage)
        random_trial = self.trial_tracker.create_trial(hparams=hparams, search_data=search_data)
        return random_trial


class ASHADSATSearchData(DSATSearchData):
    def __init__(self, curr_rung: int, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self.curr_rung = curr_rung


class ASHADSATSearchMethod(BaseDSATSearchMethod):
    """Asynchronous Successive Halving Algorithm (ASHA)

    Adaptive search through randomly-generated DeepSpeed configurations which tunes the batch size
    through a binary search and uses the number of Trials in this search as the finite-resource of
    ASHA. Search-specific arguments:

    Args:
        asha_early_stopping:
            ASHA early stopping parameter (`s` in arxiv:1810.05934)
        max_rungs:
            Maximum number of rungs
        min_binary_search_trials:
            Minimum number of binary search Trials to run per random configuration
        divisor:
            ASHA divisor parameter (`eta` in arxiv:1810.05934), controlling the growth in
            resources and population thinning across rungs
        search_range_factor:
            adjusts the initial binary search range by raising the ceiling by a factor of
            `search_range_factor`
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self.divisor: int = self.args.divisor
        self.max_rungs: int = self.args.max_rungs
        self.min_binary_search_trials: int = self.args.min_binary_search_trials
        self.asha_early_stopping: int = self.args.asha_early_stopping
        self.search_range_factor: float = self.args.search_range_factor

    def get_trials_after_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[DSATTrial]:
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            new_trials = self.get_trial_list_after_model_profile_info_run()
        else:
            new_trials = [self.get_next_trial(last_trial)]
        return new_trials

    def get_trials_after_early_exit(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        exited_reason: searcher.ExitedReason,
    ) -> List[DSATTrial]:
        new_trial = [self.get_next_trial(last_trial)]
        return new_trial

    def choose_next_trial_from_queue(self) -> DSATTrial:
        """
        Schedule the trial with the largest `search_data.curr_rung` value.
        """

        def curr_rung_key(trial: DSATTrial) -> int:
            assert trial.search_data
            assert isinstance(trial.search_data, ASHADSATSearchData)
            return trial.search_data.curr_rung

        highest_rung_trial = max(self.trial_tracker.queue, key=curr_rung_key)
        # If there are multiple such trials, choose the one with the longest lineage so that
        # trials are promoted more quickly.
        assert highest_rung_trial.search_data
        assert isinstance(highest_rung_trial.search_data, ASHADSATSearchData)
        highest_curr_rung = highest_rung_trial.search_data.curr_rung
        all_highest_curr_rung_trials_in_queue = [
            t
            for t in self.trial_tracker.queue
            if t.search_data
            and isinstance(t.search_data, ASHADSATSearchData)
            and t.search_data.curr_rung == highest_curr_rung
        ]

        next_trial = max(
            all_highest_curr_rung_trials_in_queue, key=lambda t: t.num_completed_trials_in_lineage
        )
        self.trial_tracker.queue.remove(next_trial)

        return next_trial

    def get_next_trial(self, last_trial: DSATTrial) -> DSATTrial:
        next_trial = None
        assert last_trial.search_data is not None and isinstance(
            last_trial.search_data, ASHADSATSearchData
        )
        if not self.lineage_completed_rung(last_trial, last_trial.search_data.curr_rung):
            next_trial = self.get_next_trial_in_lineage(last_trial)
        if next_trial is None:
            next_lineage = self.get_next_promotable_lineage()
            if next_lineage is not None:
                next_trial = self.get_next_trial_in_lineage(next_lineage)
                if next_trial is not None:
                    assert next_trial.search_data
                    assert isinstance(next_trial.search_data, ASHADSATSearchData)
                    # Promote to next rung
                    next_trial.search_data.curr_rung += 1
        if next_trial is None:
            next_trial = self.get_random_trial()
        return next_trial

    def get_trial_list_after_model_profile_info_run(
        self,
    ) -> List[DSATTrial]:
        new_trials = []
        max_num_trials = min(
            self.trial_tracker.max_concurrent_trials, self.trial_tracker.max_trials
        )
        if self.trial_tracker.max_slots:
            max_trials_by_slot = self.trial_tracker.max_slots // self.trial_tracker.slots_per_trial
            max_num_trials = min(max_num_trials, max_trials_by_slot)
        for _ in range(max_num_trials):
            new_trials.append(self.get_random_trial())
        return new_trials

    @property
    def rungs(self) -> Dict[int, List[DSATTrial]]:
        """
        A dictionary of lists of the latest trials in each lineage which have completed the
        specified rung.
        """
        rungs = collections.defaultdict(list)
        for root in self.get_all_latest_trials_in_lineages():
            assert isinstance(root.search_data, ASHADSATSearchData)
            rung_idx = 0
            while self.lineage_completed_rung(root, rung_idx):
                rungs[rung_idx].append(root)
                rung_idx += 1
        return rungs

    def get_all_latest_trials_in_lineages(self) -> List[DSATTrial]:
        """
        Returns a list of the latest trials in each lineage.
        """
        lineage_root_set = [
            trial
            for _, trial in self.trial_tracker
            if not isinstance(trial, DSATModelProfileInfoTrial)
            and self.get_latest_trial_in_lineage(trial) == trial
            and trial.search_data is not None
            and isinstance(trial.search_data, ASHADSATSearchData)
        ]
        return lineage_root_set

    def lineage_completed_rung(self, trial: DSATTrial, rung_idx: int) -> bool:
        assert trial.search_data
        assert isinstance(trial.search_data, ASHADSATSearchData)
        latest_trial = self.get_latest_trial_in_lineage(trial)
        assert latest_trial.search_data
        assert isinstance(latest_trial.search_data, ASHADSATSearchData)
        if latest_trial.search_data.curr_rung > rung_idx:
            return True
        if trial.num_completed_trials_in_lineage >= self.max_trials_for_rung_idx(rung_idx):
            return True
        # Also need to cover the cases where a binary search stopped before using all available
        # resources (trials) in its current rung. Only need to check for curr_rung = rung_idx.
        if latest_trial.search_data.curr_rung == rung_idx:
            failed_on_min_mbs = (
                latest_trial.error and latest_trial.mbs == latest_trial.search_data.lo
            )
            trivial_search_data = latest_trial.search_data.hi == latest_trial.search_data.lo
            if trivial_search_data or failed_on_min_mbs:
                return True
        return False

    def get_next_promotable_lineage(self) -> Optional[DSATTrial]:
        # Cannot promote from the top rung (rung_idx == self.max_rung - 1)
        for rung_idx in reversed(range(self.max_rungs - 1)):
            next_promotable_trial = self.get_next_promotable_lineage_in_rung(rung_idx)
            if next_promotable_trial is not None:
                return next_promotable_trial
        return None

    def get_next_promotable_lineage_in_rung(self, rung_idx: int) -> Optional[DSATTrial]:
        """
        Returns the latest trial in the next promotable lineage in the given rung.
        """
        top_trials = self.get_top_lineages_in_rung(rung_idx)
        for trial in top_trials:
            latest_trial = self.get_latest_trial_in_lineage(trial)
            assert latest_trial.search_data
            assert isinstance(latest_trial.search_data, ASHADSATSearchData)
            already_promoted = latest_trial.search_data.curr_rung > rung_idx
            if not already_promoted:
                return self.get_latest_trial_in_lineage(trial)
        return None

    def get_top_lineages_in_rung(self, rung_idx: int) -> List[DSATTrial]:
        """
        Returns the best trial in each of the top 1 / divisor fraction of lineages from the given
        rung, per the ASHA paper.
        """
        completed_lineages_in_rung = self.rungs[rung_idx]
        k = len(completed_lineages_in_rung) // self.divisor
        if not k:
            return []
        best_trials: List[DSATTrial] = []
        for lin in completed_lineages_in_rung:
            best_trial = self.get_best_trial_in_lineage(lin, max_rung_idx=rung_idx)
            if best_trial is not None:
                best_trials.append(best_trial)
        reverse = not self.trial_tracker.smaller_is_better
        best_trials.sort(
            key=lambda t: t.searcher_metric_val is not None and t.searcher_metric_val,
            reverse=reverse,
        )
        return best_trials[:k]

    def get_best_trial_in_lineage(
        self, trial: DSATTrial, max_rung_idx: Optional[int] = None
    ) -> Optional[DSATTrial]:
        trials_with_metrics = [t for t in trial.lineage_set if t.searcher_metric_val is not None]
        if max_rung_idx is not None:
            filtered_trials_with_metrics: List[DSATTrial] = []
            for t in trials_with_metrics:
                assert t.search_data
                assert isinstance(t.search_data, ASHADSATSearchData)
                if t.search_data.curr_rung <= max_rung_idx:
                    filtered_trials_with_metrics.append(t)
            trials_with_metrics = filtered_trials_with_metrics
        if not trials_with_metrics:
            return None
        min_or_max = min if self.trial_tracker.smaller_is_better else max
        return min_or_max(
            trials_with_metrics,
            key=lambda t: t.searcher_metric_val is not None and t.searcher_metric_val,
        )

    def get_latest_trial_in_lineage(self, trial: DSATTrial) -> DSATTrial:
        while trial.children:
            assert len(trial.children) <= 1  # Sanity check
            trial = next(iter(trial.children))
        return trial

    def get_next_trial_in_lineage(self, trial: DSATTrial) -> Optional[DSATTrial]:
        latest_trial = self.get_latest_trial_in_lineage(trial)
        assert latest_trial.search_data is not None
        new_search_data = copy.deepcopy(latest_trial.search_data)
        if latest_trial.searcher_metric_val is not None:
            new_search_data.lo = latest_trial.mbs + 1
        else:
            new_search_data.hi = latest_trial.mbs - 1

        if new_search_data.hi < new_search_data.lo:
            return None

        mbs = (new_search_data.hi + new_search_data.lo) // 2

        new_hparams = copy.deepcopy(latest_trial.hparams)
        new_hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs
        next_trial = self.trial_tracker.create_trial(
            hparams=new_hparams,
            search_data=new_search_data,
            parent_trial=latest_trial,
        )
        return next_trial

    def max_trials_for_rung_idx(self, rung_idx: int) -> int:
        max_trials: int = self.min_binary_search_trials * self.divisor ** (
            self.asha_early_stopping + rung_idx
        )
        return max_trials

    def get_random_hparams_and_search_data(
        self, zero_stage: int
    ) -> Tuple[Dict[str, Any], ASHADSATSearchData]:
        zero_optim_config = dsat.get_random_zero_optim_config(zero_stage)
        new_hparams = copy.deepcopy(self.trial_tracker.hparams)
        new_hparams[defaults.OVERWRITE_KEY] = util.merge_dicts(
            new_hparams.get(defaults.OVERWRITE_KEY, {}),
            {"zero_optimization": zero_optim_config},
        )

        random_zero_stage_max_mbs = self.trial_tracker.approx_max_mbs_per_stage[zero_stage]
        lo = 1
        hi = int(random_zero_stage_max_mbs * self.search_range_factor)
        hi = max(hi, lo)
        new_search_data = ASHADSATSearchData(lo=1, hi=hi, curr_rung=0)

        # Randomly choose the actual batch size.
        mbs = (new_search_data.hi + new_search_data.lo) // 2
        new_hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs
        return new_hparams, new_search_data

    def get_random_trial(self) -> DSATTrial:
        zero_stage = random.choice(list(self.trial_tracker.zero_stages))
        hparams, search_data = self.get_random_hparams_and_search_data(zero_stage)
        random_trial = self.trial_tracker.create_trial(hparams=hparams, search_data=search_data)
        return random_trial


class TestDSATSearchMethod(BaseDSATSearchMethod):
    """Searcher for basic testing purposes.

    Submits Trials with linearly increasing batch sizes, from 2 up to max_trials
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

    def get_trials_after_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[DSATTrial]:
        new_trials = []
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            # Delete special DS keys which force a model profiling info run.
            hparams_without_profile_info_keys = last_trial.hparams
            del hparams_without_profile_info_keys[defaults.OVERWRITE_KEY]["autotuning"][
                "model_info"
            ]
            del hparams_without_profile_info_keys[defaults.OVERWRITE_KEY]["autotuning"][
                "model_info_path"
            ]
            for tmbs in range(2, self.trial_tracker.max_trials + 1):
                hparams = copy.deepcopy(hparams_without_profile_info_keys)
                hparams[defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = tmbs
                # Choose a random zero stage:
                hparams[defaults.OVERWRITE_KEY]["zero_optimization"] = {
                    "stage": random.choice(list(self.args.zero_stages))
                }
                trial = self.trial_tracker.create_trial(
                    hparams=hparams,
                    search_data=None,
                    parent_trial=None,
                )
                new_trials.append(trial)
        return new_trials

    def get_trials_after_early_exit(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        exited_reason: searcher.ExitedReason,
    ) -> List[DSATTrial]:
        return []
