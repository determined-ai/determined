import copy
import logging
import pathlib
import pickle
import random
import uuid
from abc import abstractmethod
from typing import Any, Dict, Iterable, Iterator, List, Optional, Set, Tuple, Union

import numpy as np

from determined import searcher
from determined.pytorch.deepspeed import get_ds_config_from_hparams
from determined.pytorch.deepspeed.dsat import _defaults, _utils
from determined.util import merge_dicts

"""
TODOs:
    * Make sure we don't draw the same config twice in random search.
    * Give control over random seeds and checkpoint rng states.
    * Allow users to configure concurrent trials, somehow.
    * Move away from native DS AT autotuning config syntax, since we're not actually using it as
    they do and there's no reason to constrict ourselves in that way.
    * Clean up directory/import structure. Imports are onerously long now.
    * Don't use stage 3 if stages 1 or 2 work
    * Make it easy for users to subclass the base searcher and use it with dsat.autotune? Not sure
    how we'd do that.
"""


class DSATTrial:
    """
    Helper class for tracking the results and properties of individual Trials.
    """

    def __init__(
        self,
        hparams: Dict[str, Any],
        model_dir: str,
        slots: int,
        length: int,
        request_id: Optional[uuid.UUID] = None,
        parent: Optional["DSATTrial"] = None,
        search_data: Optional[Any] = None,
    ) -> None:
        self.hparams = hparams
        self.model_dir = model_dir
        self.slots = slots
        self.length = length
        self.request_id = request_id or uuid.uuid4()
        self.parent = parent
        # Arbitrary attribute for search-specific data tracking.
        self.search_data = search_data

        # Other attrs which are updated during training:
        # Boolean for tracking whether the Trial errored.
        self.metric = {}
        self.error = False
        self.children = set()

        # If a parent was specified, register the current Trial as the parent's child.
        if self.parent is not None:
            self.parent.children.add(self)

        self.lineage_root = self if self.parent is None else self.parent.lineage_root

        self.ds_config = get_ds_config_from_hparams(self.hparams, self.model_dir)
        # TODO: Leaving this as 1 right now. In general will need some custom logic here, especially
        # if we want to support both model and pipeline parallelism.
        self.mp_size = 1

        self._error_in_direct_history = False

    @property
    def lineage_set(self) -> Set["DSATTrial"]:
        """Computes set of trials in lineage tree."""
        root = self.lineage_root
        trials_set = {root}
        children = {c for c in root.children}
        while children:
            random_child = children.pop()
            trials_set.add(random_child)
            children |= random_child.children
        return trials_set

    @property
    def num_trials_in_lineage(self) -> int:
        """Computes total number of trials in lineage tree."""
        num_trials = len(self.lineage_set)
        return num_trials

    @property
    def error_in_direct_history(self) -> bool:
        if self._error_in_direct_history:
            return self._error_in_direct_history
        trial = self
        while trial is not None:
            if trial.error:
                return True
            trial = trial.parent
        return False

    @property
    def mbs_in_lineage(self) -> Set["DSATTrial"]:
        """
        Returns the set of all `train_micro_batch_size_per_gpu` (mbs) used in the Trial's lineage.
        """
        mbs_in_lineage = {t.mbs for t in self.lineage_set}
        return mbs_in_lineage

    @property
    def stage(self) -> int:
        return self.ds_config.get("zero_optimization", {}).get("stage", 0)

    @property
    def fp16(self) -> int:
        return self.ds_config.get("fp16", {}).get("enabled") or False

    @property
    def mbs(self) -> int:
        return self.ds_config["train_micro_batch_size_per_gpu"]

    # TODO: More important properties, like train_batch_size, gas, etc.


class DSATModelProfileInfoTrial(DSATTrial):
    """
    Super class for processing the model profiling info run.

    # TODO: avoid various recomputations.
    """

    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)
        self._mem_per_gpu_per_stage = None
        self._viable_zero_stages = None
        self._max_mbs_per_stage = None

    @property
    def gpu_mem(self) -> int:
        """
        Returns the available GPU memory in bytes.
        """
        return self.metric["gpu_mem"]

    @property
    def num_params(self) -> int:
        return self.metric["num_params"]

    @property
    def trainable_num_params(self) -> int:
        return self.metric["trainable_num_params"]

    @property
    def activation_mem_per_gpu(self) -> int:
        return self.metric["activation_mem_per_gpu"]

    @property
    def mem_per_gpu_per_stage(self) -> Dict[int, int]:
        """
        Returns the required gpu memory in bytes, per stage.
        """
        if self._mem_per_gpu_per_stage is None:
            params_mem = self.num_params * (2 if self.fp16 else 4)
            gradients_mem = self.trainable_num_params * (2 if self.fp16 else 4)
            # optimizer_mem assumes Adam, following DS. TODO: don't assume this.
            optimizer_mem = self.trainable_num_params * (16 if self.fp16 else 8)

            non_activation_mem_per_gpu_per_stage = {
                0: params_mem + gradients_mem + optimizer_mem,
                1: params_mem + gradients_mem + optimizer_mem // self.slots,
                2: params_mem + (gradients_mem + optimizer_mem) // self.slots,
                3: (params_mem + gradients_mem + optimizer_mem) // self.slots,
            }
            if self.mp_size > 1:
                non_activation_mem_per_gpu_per_stage = {
                    stage: mem // self.mp_size
                    for stage, mem in non_activation_mem_per_gpu_per_stage.items()
                }
            # No need to divide by mp_size below because self.activation_mem_per_gpu already has the
            # model parallelism accounted for (at least approximately).
            mem_per_gpu_per_stage = {
                stage: mem + self.activation_mem_per_gpu
                for stage, mem in non_activation_mem_per_gpu_per_stage.items()
            }
            self._mem_per_gpu_per_stage = mem_per_gpu_per_stage
        return self._mem_per_gpu_per_stage

    @property
    def viable_zero_stages(self) -> Set[int]:
        """
        Returns the set of viable zero stages based on a rough computation.
        """
        # TODO: Add a configurable fudge factor for a little leeway?
        if self._viable_zero_stages is None:
            self._viable_zero_stages = {
                stage for stage, mem in self.mem_per_gpu_per_stage.items() if mem < self.gpu_mem
            }
        return self._viable_zero_stages

    @property
    def max_mbs_per_stage(self) -> Dict[int, int]:
        """
        Returns the approximate max train_micro_batch_size_per_gpu (mbs) per stage.
        """
        if self._max_mbs_per_stage is None:
            self._max_mbs_per_stage = {
                stage: (self.gpu_mem - mem) // self.activation_mem_per_gpu
                for stage, mem in self.mem_per_gpu_per_stage.items()
                if stage in self.viable_zero_stages
            }
        return self._max_mbs_per_stage


class DSATTrialTracker:
    """
    Class for organizing DSATTrial instances and retrieving pertinent info.
    """

    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
        model_dir: str,
    ) -> None:
        self.submitted_config_dict = submitted_config_dict
        self.model_dir = model_dir

        # Various derived attributes
        self.slots = self.submitted_config_dict["resources"]["slots_per_trial"]
        self.smaller_is_better = self.submitted_config_dict["searcher"].get(
            "smaller_is_better", _defaults.SMALLER_IS_BETTER
        )
        self.submitted_hps = self.submitted_config_dict["hyperparameters"]
        self.ds_config = get_ds_config_from_hparams(self.submitted_hps, self.model_dir)
        self.fp16 = self.ds_config.get("fp16", {}).get("enabled") or False

        self.autotuning_config = _defaults.AUTOTUNING_DICT  # TODO: let the user configure more.
        self.searcher_metric_name = self.autotuning_config["metric"] = self.submitted_config_dict[
            "searcher"
        ]["metric"]
        self.tuner_num_trials = self.autotuning_config["tuner_num_trials"]
        self.tuner_early_stopping = self.autotuning_config["tuner_early_stopping"]
        self.num_tuning_micro_batch_sizes = self.autotuning_config["num_tuning_micro_batch_sizes"]

        self.submitted_hps_with_autotuning = merge_dicts(
            self.submitted_hps, {_defaults.OVERWRITE_KEY: {"autotuning": self.autotuning_config}}
        )

        # Also add an internal key to the HP dict which enable the DSAT code path for Trial classes.
        self.submitted_hps_with_autotuning[_defaults.USE_DSAT_MODE_KEY] = True

        self.model_profile_info_trial = None
        self.best_trial = None
        self.num_trials_since_best_result = 0
        self.successful_stages = set()
        self._all_trials_dict = {}

    def __len__(self) -> int:
        return len(self._all_trials_dict)

    def __getitem__(self, request_id: uuid.UUID) -> DSATTrial:
        return self._all_trials_dict[request_id]

    def __iter__(self) -> Iterator[DSATTrial]:
        return iter(self._all_trials_dict.values())

    def create_trial(
        self,
        hparams: Dict[str, Any],
        length: Optional[int] = None,
        search_data: Optional[Any] = None,
        parent_trial: Optional[DSATTrial] = None,
    ) -> DSATTrial:
        """
        Creates a new `DSATTrial` object, updates lineages as appropriate, and updates the
        searcher's Trial tracking dictionary.
        """
        # Create a consistent batch size configuration which obeys the DS constraints.
        self.enforce_consistent_batch_config(hparams)
        if length is None:
            # Get the default length from the autotuning config.
            # DS has a fixed notion of what a step is while Determined does not. Make sure
            # there are no issues in reconciling this fact.
            # The +1 is required to align DS step/DET max_length conventions.
            # TODO: Clean all of this up.
            length = self.autotuning_config["end_profile_step"] + 1

        trial = DSATTrial(
            hparams=hparams,
            model_dir=self.model_dir,
            slots=self.slots,
            length=length,
            parent=parent_trial,
            search_data=search_data,
        )
        self._all_trials_dict[trial.request_id] = trial
        # TODO: Delete print test.
        logging.info(f"=============Total Trials Created: {len(self)}=============")
        return trial

    def create_model_profile_info_trial(
        self,
        length: int = 1,
    ) -> DSATModelProfileInfoTrial:
        # Create the special hp dictionary used for the model profile info run.
        model_profile_info_hps = copy.deepcopy(self.submitted_hps_with_autotuning)
        model_profile_info_hps[_defaults.OVERWRITE_KEY] = merge_dicts(
            model_profile_info_hps.get(_defaults.OVERWRITE_KEY, {}),
            _defaults.MODEL_INFO_PROFILE_DS_CONFIG,
        )
        self.enforce_consistent_batch_config(model_profile_info_hps)

        model_profile_info_trial = DSATModelProfileInfoTrial(
            hparams=model_profile_info_hps,
            model_dir=self.model_dir,
            slots=self.slots,
            length=length,
        )
        self._all_trials_dict[model_profile_info_trial.request_id] = model_profile_info_trial
        self.model_profile_info_trial = model_profile_info_trial
        return model_profile_info_trial

    def enforce_consistent_batch_config(self, hparams: Dict[str, Any]) -> None:
        """Enforces a consistent batch size configuration by altering `hparams` in-place."""
        # TODO: Talk to Liam about this, because this function adjusts `train_batch_size`, whereas
        # he probably wants this to be the only constant, in order to hold training dynamics fixed.
        # We are optimizing different things.
        ds_config = get_ds_config_from_hparams(hparams, self.model_dir)
        batch_size_config = _utils.get_batch_config_from_mbs_gas_and_slots(
            ds_config, slots=self.slots
        )
        hparams[_defaults.OVERWRITE_KEY] = merge_dicts(
            hparams[_defaults.OVERWRITE_KEY], batch_size_config
        )

    def get_root_trial_set(self, include_model_profile_info_trial: bool = False) -> Set[DSATTrial]:
        """
        Returns the set DSATTrials which are the root element in their lineage.
        """
        # TODO: Delete if not used anywhere.
        root_trial_set = set()
        for trial in self:
            if trial.parent is None:
                if (
                    isinstance(trial, DSATModelProfileInfoTrial)
                    and not include_model_profile_info_trial
                ):
                    continue
                root_trial_set.add(trial)
        return root_trial_set

    @property
    def best_autotuning_metric_val(self) -> bool:
        autotuning_metric_vals = [
            t.metric[self.searcher_metric_name]
            for t in self
            if self.searcher_metric_name in t.metric
        ]
        return max(autotuning_metric_vals) if autotuning_metric_vals else None

    @property
    def all_trials_created(self) -> bool:
        return len(self) >= self.tuner_num_trials

    @property
    def early_stopping_triggered(self) -> bool:
        return self.num_trials_since_best_result >= self.tuner_early_stopping

    @property
    def all_trials_closed_or_errored(self) -> bool:
        return all(t.error or t.metric for t in self)

    @property
    def should_shutdown(self) -> bool:
        if self.model_profile_info_trial is not None and self.model_profile_info_trial.error:
            logging.info("Shutting down: error in model profile info Trial.")
            return True
        elif self.early_stopping_triggered:
            logging.info("Shutting down: early stopping criteria met.")
            return True
        elif self.all_trials_created and self.all_trials_closed_or_errored:
            logging.info("Shutting down: all Trials completed.")
            return True
        else:
            return False

    def update_trial_metric(
        self,
        trial: DSATTrial,
        metric: Dict[str, Any],
    ) -> None:
        """
        Updates the Trial Tracker after metrics have been reported, attaching the reported metrics
        to the `DSATTrial` instnace and updating early-stopping bookkeeping.
        """
        trial.metric = metric

        # The model info profiling run's metric will not contain the searcher metric key and should
        # not be counted against the early stopping criteria.
        if not isinstance(trial, DSATModelProfileInfoTrial):
            searcher_metric_value = metric.get(self.searcher_metric_name)
            trial_is_best = self.best_autotuning_metric_val is None or (
                searcher_metric_value < self.best_autotuning_metric_val
                if self.smaller_is_better
                else searcher_metric_value > self.best_autotuning_metric_val
            )
            self.successful_stages.add(trial.stage)
            if trial_is_best:
                self.best_trial = trial
                self.num_trials_since_best_result = 0
            else:
                self.num_trials_since_best_result += 1


class BaseDSATSearchMethod(searcher.SearchMethod):
    """
    Base class for all DS AT searchers. Written so that only the `get_new_searcher_ops_list` method
    needs to be written overwritten when subclassing (at a minimum).
    """

    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
        model_dir: str,
    ) -> None:
        self.submitted_config_dict = submitted_config_dict
        self.model_dir = model_dir

        self.trial_tracker = DSATTrialTracker(
            submitted_config_dict=submitted_config_dict,
            model_dir=model_dir,
        )

    @abstractmethod
    def get_trials_after_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        metric: Union[float, Dict[str, Any]],
    ) -> Iterable[DSATTrial]:
        """
        To be defined in all subclasses.

        Generates a list of new operations to run based on the results of the last successful trial.
        """
        pass

    @abstractmethod
    def get_trials_after_early_exit(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        exited_reason: searcher.ExitedReason,
    ) -> Iterable[DSATTrial]:
        """
        To be defined in all subclasses.

        Generates a list of new operations to run after the last trial exited early.
        """
        pass

    def get_ops_list_for_trials(self, trials: Iterable[DSATTrial]) -> List[searcher.Operation]:
        """
        Returns a list with the Create and ValidateAfter operations needed to initiate and run
        the specified Trial.
        """
        ops_list = []
        for t in trials:
            create_op = searcher.Create(
                request_id=t.request_id,
                hparams=t.hparams,
                checkpoint=None,
            )
            ops_list.append(create_op)
            validate_after_op = searcher.ValidateAfter(request_id=t.request_id, length=t.length)
            ops_list.append(validate_after_op)

        return ops_list

    def initial_operations(
        self, searcher_state: searcher.SearcherState
    ) -> List[searcher.Operation]:
        """
        Submits the model info profiling run in order to collect model and resources info to
        inform the search.
        """
        model_profile_info_trial = self.trial_tracker.create_model_profile_info_trial()
        # Only a single step is required for the model profiling run.
        ops = self.get_ops_list_for_trials([model_profile_info_trial])
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
        # TODO: Remove print tests.
        logging.info(f"Calling on_validation_completed for {request_id}")

        last_trial = self.trial_tracker[request_id]
        self.trial_tracker.update_trial_metric(trial=last_trial, metric=metric)

        # TODO: remove some of these info logs. Some are just for testing.
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            logging.info(f"Approx. max mbs per stage: {last_trial.max_mbs_per_stage}")
            logging.info(f"Approx. GPU memory per stage: {last_trial.mem_per_gpu_per_stage}")
            logging.info(f"Total GPU memory: {last_trial.gpu_mem}")
            logging.info(f"Viable zero stages: {last_trial.viable_zero_stages}")

        # All DS AT Trials should be closed after validation.
        new_ops_list = [searcher.Close(request_id)]
        if not self.trial_tracker.all_trials_created and not self.trial_tracker.should_shutdown:
            new_trials = self.get_trials_after_validation_completed(
                searcher_state=searcher_state,
                last_trial=last_trial,
                metric=metric,
            )
            additional_ops_list = self.get_ops_list_for_trials(new_trials)
            new_ops_list.extend(additional_ops_list)
        return new_ops_list

    def on_trial_closed(
        self, searcher_state: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        # TODO: Remove print tests.
        logging.info(f"Calling on_trial_closed for {request_id}")
        last_trial = self.trial_tracker[request_id]
        logging.info(f"metrics for closed trial {last_trial.metric}")

        new_ops_list = []
        if self.trial_tracker.should_shutdown:
            new_ops_list.append(searcher.Shutdown())
        return new_ops_list

    def on_trial_exited_early(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        exited_reason: searcher.ExitedReason,
    ) -> List[searcher.Operation]:
        # TODO: Remove print tests.
        logging.info(f"Calling on_trial_exited_early for {request_id}")

        last_trial = self.trial_tracker[request_id]
        last_trial.error = True

        new_ops_list = []
        if exited_reason != searcher.ExitedReason.ERRORED:
            # In case of INVALID_HP or USER_CANCELED, shut down the searcher.
            logging.info(f"Shutting down: unexpected early exit due to {exited_reason}")
            new_ops_list.append(searcher.Shutdown())
        elif not self.trial_tracker.all_trials_created and not self.trial_tracker.should_shutdown:
            # ERRORED Trials generally corresponds to OOMs, after which we may want to submit
            # follow-on Trials.
            new_trials = self.get_trials_after_early_exit(
                searcher_state=searcher_state,
                last_trial=last_trial,
                exited_reason=exited_reason,
            )
            additional_ops_list = self.get_ops_list_for_trials(new_trials)
            new_ops_list.extend(additional_ops_list)

        return new_ops_list

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        progress = len(searcher_state.trials_closed) / len(searcher_state.trials_created)
        return progress

    def save_method_state(self, path: pathlib.Path) -> None:
        checkpoint_path = path.joinpath("trial_tracker.pkl")
        with checkpoint_path.open("wb") as f:
            pickle.dump(self.trial_tracker, f)

    def load_method_state(self, path: pathlib.Path) -> None:
        logging.info(f"Restoring searcher state from checkpoint.")
        checkpoint_path = path.joinpath("trial_tracker.pkl")
        with checkpoint_path.open("rb") as f:
            self.trial_tracker = pickle.load(f)


class RandomDSATSearchMethod(BaseDSATSearchMethod):
    """
    Semi-random search through parameters space. Attaches search_data of the form
    {"lo": lo,  "hi": hi} which defines the inclusive bouneds on the train_micro_batch_size_per_gpu
    that can be selected for the trial.
    """

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        # TODO: Save/restore rng state in checkpoints.
        self.rng = np.random.default_rng(42)

    def get_trials_after_validation_completed(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[DSATTrial]:
        new_trials = []
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            new_trials = self.get_trial_list_after_model_profile_info_run()
        elif last_trial.num_trials_in_lineage < self.trial_tracker.num_tuning_micro_batch_sizes:
            trial = self.get_trial_after_autotuning_run(last_trial)
            # trial will be None if the current lineage should be terminated.
            new_trials = [trial] if trial is not None else [self.get_random_trial()]
        else:
            new_trials = [self.get_random_trial()]
        return new_trials

    def get_trials_after_early_exit(
        self,
        searcher_state: searcher.SearcherState,
        last_trial: DSATTrial,
        exited_reason: searcher.ExitedReason,
    ) -> List[DSATTrial]:
        # TODO: delete print test
        logging.info("Calling get_trials_after_early_exit")

        if last_trial.num_trials_in_lineage < self.trial_tracker.num_tuning_micro_batch_sizes:
            new_search_data = copy.deepcopy(last_trial.search_data)
            "Lower the ceiling after the failure."
            new_search_data["hi"] = last_trial.mbs - 1

            mbs = self.get_random_mbs_from_search_data(new_search_data)
            new_hparams = copy.deepcopy(last_trial.hparams)
            new_hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs

            trial = self.trial_tracker.create_trial(
                hparams=new_hparams,
                search_data=new_search_data,
                parent_trial=last_trial,
            )
            new_trials = [trial]
        else:
            new_trials = [self.get_random_trial()]
        return new_trials

    def get_trial_list_after_model_profile_info_run(
        self,
    ) -> List[DSATTrial]:
        new_trials = []
        # One trial for each stage. This also sets the number of concurrent trials, which should
        # really be configurable.
        for zero_stage in range(1, 4):
            trial = self.get_random_trial(zero_stage)
            new_trials.append(trial)
        return new_trials

    def get_trial_after_autotuning_run(
        self,
        last_trial: DSATTrial,
    ) -> Optional[DSATTrial]:
        # TODO: remove below print tests.
        logging.info("**************** BSZ History ****************")
        bsz_history = []
        print_trial = last_trial
        while print_trial is not None:
            bsz = print_trial.ds_config["train_micro_batch_size_per_gpu"]
            bsz_history.append(bsz)
            print_trial = print_trial.parent
        logging.info(f"History (to-be-submitted last): {str(list(reversed(bsz_history)))}")
        logging.info("**************** BSZ History End ****************")

        # TODO: verify we are always quitting when no more non-trivial trials are possible.

        # Let the best trial inform the next one, if it exists.
        if self.trial_tracker.best_trial is not None:
            new_search_data = copy.deepcopy(self.trial_tracker.best_trial.search_data)
            # Update the floor to one greater than the mbs used.
            new_search_data["lo"] = self.trial_tracker.best_trial.mbs + 1
        else:
            # Otherwise, start from the data from the last trial (which was successful).
            new_search_data = copy.deepcopy(last_trial.search_data)
            new_search_data["lo"] = last_trial.mbs + 1
            if not last_trial.error_in_direct_history:
                # If no error has occurred in this trial's direct history, the ceiling is still
                # soft and should be increased.
                new_search_data["hi"] *= 2

        # Catch all cases where we should end this lineage by returning None
        if self.should_stop_lineage(last_trial=last_trial, new_search_data=new_search_data):
            logging.info(f"Killing lineage for trial {last_trial.request_id}")
            return None
        else:
            mbs = self.get_random_mbs_from_search_data(new_search_data)
            while mbs in last_trial.mbs_in_lineage:
                mbs = self.get_random_mbs_from_search_data(new_search_data)

            new_hparams = copy.deepcopy(last_trial.hparams)
            new_hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mbs

            trial = self.trial_tracker.create_trial(
                hparams=new_hparams,
                search_data=new_search_data,
                parent_trial=last_trial,
            )
            return trial

    def should_stop_lineage(self, last_trial: DSATTrial, new_search_data: Dict[str, int]) -> bool:
        """
        Helper function for determining if we should stop the lineage, given the last trial and the
        search data that would be used if continuing the lineage.
        """
        # Various stopping conditions
        trivial_search_range = new_search_data["hi"] < new_search_data["lo"]

        # DS domain knowledge: if stages 1 or 2 run successfully, there is no need to use stage 3.
        should_stop_this_stage_3_trial = last_trial.stage == 3 and any(
            n in self.trial_tracker.successful_stages for n in (1, 2)
        )

        num_mbs_in_search_range = new_search_data["hi"] - new_search_data["lo"] + 1
        num_tested_mbs_in_search_range = sum(
            new_search_data["lo"] <= mbs <= new_search_data["hi"]
            for mbs in last_trial.mbs_in_lineage
        )
        all_mbs_already_tested = num_tested_mbs_in_search_range >= num_mbs_in_search_range

        should_stop_lineage = (
            trivial_search_range or all_mbs_already_tested or should_stop_this_stage_3_trial
        )
        return should_stop_lineage

    def get_random_mbs_from_search_data(self, search_data: Dict[str, int]) -> int:
        num_possible_mbs = search_data["hi"] - search_data["lo"] + 1
        mbs = search_data["lo"] + self.rng.binomial(n=num_possible_mbs, p=0.5)
        return mbs

    def get_random_hparams_and_search_data(
        self, zero_stage: Optional[int] = None
    ) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        # Select a random viable zero stage and randomly choose from the relevant parameters for
        # that stage.

        # DS domain knowledge: if stages 1 or 2 run successfully, there is no need to use stage 3,
        # due to the increased communication costs. We also don't need to choose stage 0, since that
        # would leverage DS at all.
        if zero_stage is None:
            relevant_stages = {
                s for s in self.trial_tracker.model_profile_info_trial.viable_zero_stages if s != 0
            }
            stage_1_and_2 = {1, 2}
            if stage_1_and_2 & self.trial_tracker.successful_stages:
                relevant_stages &= stage_1_and_2
            zero_stage = random.choice(list(relevant_stages))

        zero_optim_config = _utils.get_random_zero_optim_config(zero_stage)
        new_hparams = copy.deepcopy(self.trial_tracker.submitted_hps_with_autotuning)
        new_hparams[_defaults.OVERWRITE_KEY] = merge_dicts(
            new_hparams.get(_defaults.OVERWRITE_KEY, {}),
            {"zero_optimization": zero_optim_config},
        )

        # If a best trial has been established, use its search data bounds.
        if self.trial_tracker.best_trial is not None:
            new_search_data = copy.deepcopy(self.trial_tracker.best_trial.new_search_data)
            # Update the floor to one greater than the mbs used.
            new_search_data["lo"] = self.trial_tracker.best_trial.mbs + 1
        # Otherwise choose the corrsponding search data based on approximate computations
        else:
            random_zero_stage_max_mbs = (
                self.trial_tracker.model_profile_info_trial.max_mbs_per_stage[zero_stage]
            )
            new_search_data = {
                "lo": 1,
                "hi": 2 * random_zero_stage_max_mbs - 1,
            }

        # Randomly choose the actual batch size by drawing from a binomial distribution
        new_hparams[_defaults.OVERWRITE_KEY][
            "train_micro_batch_size_per_gpu"
        ] = self.get_random_mbs_from_search_data(new_search_data)
        return (new_hparams, new_search_data)

    def get_random_trial(self, zero_stage: Optional[int] = None) -> DSATTrial:
        hparams, search_data = self.get_random_hparams_and_search_data(zero_stage)
        random_trial = self.trial_tracker.create_trial(hparams=hparams, search_data=search_data)
        return random_trial


class SimpleDSATSearchMethod(BaseDSATSearchMethod):
    """
    Dumb searcher which just submits Trials with linearly increasing batch sizes, from 2 up to
    self.trial_tracker.tuner_num_trials.
    """

    def __init__(self, *args, **kwargs):
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
            del hparams_without_profile_info_keys[_defaults.OVERWRITE_KEY]["autotuning"][
                "model_info"
            ]
            del hparams_without_profile_info_keys[_defaults.OVERWRITE_KEY]["autotuning"][
                "model_info_path"
            ]
            for tmbs in range(2, self.trial_tracker.tuner_num_trials + 1):
                hparams_without_profile_info_keys["train_micro_batch_size_per_gpu"] = tmbs
                trial = self.trial_tracker.create_trial(
                    hparams=hparams_without_profile_info_keys,
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
