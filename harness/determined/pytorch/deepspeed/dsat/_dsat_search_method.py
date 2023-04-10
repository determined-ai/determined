import copy
import inspect
import logging
import pathlib
import pickle
import random
import uuid
from abc import abstractmethod
from os import stat
from typing import Any, Dict, List, Optional, Set, Tuple, Union

from determined import searcher
from determined.pytorch.deepspeed import get_ds_config_from_hparams
from determined.pytorch.deepspeed.dsat import _defaults, _utils
from determined.util import merge_dicts


class DSATTrial:
    """
    Helper class for tracking the results and properties of individual Trials.
    """

    def __init__(
        self,
        hparams: Dict[str, Any],
        model_dir: str,
        slots: int,
        request_id: Optional[uuid.UUID] = None,
        metric: Optional[Dict[str, Any]] = None,
        parent: Optional["DSATTrial"] = None,
        children: Optional[Set["DSATTrial"]] = None,
        search_data: Optional[Any] = None,
        error: bool = False,
    ) -> None:
        self.hparams = hparams
        self.model_dir = model_dir
        self.slots = slots
        self.request_id = request_id or uuid.uuid4()
        self.metric = metric if metric is not None else {}

        # Properties for lineage tracking.
        self.parent = parent
        self.children = children or set()
        if self.parent is not None:
            self.parent.children.add(self)
        self.lineage_root = self if self.parent is None else self.parent.lineage_root

        # Arbitrary attribute for search-specific data tracking.
        self.search_data = search_data

        # Booleans for tracking whether the Trial errored.
        self.error = error

        self.ds_config = get_ds_config_from_hparams(self.hparams, self.model_dir)
        self.fp16 = self.ds_config.get("fp16", {}).get("enabled") or False
        # TODO: Leaving this as 1 right now. In general will need some custom logic here, especially
        # if we want to support both model and pipeline parallelism.
        self.mp_size = 1

        self._error_in_direct_history = False

    @property
    def lineage_set(self) -> Set["DSATTrial"]:
        """Computes set of trials in lineage tree."""
        root = self.lineage_root
        trials_set = {root}
        children = root.children
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
        mbs_in_lineage = {t.ds_config["train_micro_batch_size_per_gpu"] for t in self.lineage_set}
        return mbs_in_lineage


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
        zero_search_config: Optional[str] = None,
        all_trials_dict: Optional[Dict[uuid.UUID, DSATTrial]] = None,
        should_stop: bool = False,
        best_autotuning_metric_val: Optional[Any] = None,
        num_trials_since_best_result: int = 0,
    ) -> None:
        self.submitted_config_dict = submitted_config_dict
        self.model_dir = model_dir
        self.zero_search_config = zero_search_config
        self.all_trials_dict = all_trials_dict if all_trials_dict is not None else {}
        self.should_stop = should_stop
        self.best_autotuning_metric_val = best_autotuning_metric_val
        self.num_trials_since_best_result = num_trials_since_best_result

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

        # Non-trivial values instantiated after model profiling run
        self.model_profile_info_trial = None

    def __len__(self) -> int:
        return len(self.all_trials_dict)

    def __getitem__(self, request_id: uuid.UUID) -> DSATTrial:
        return self.all_trials_dict[request_id]

    def create_trial(
        self,
        hparams: Dict[str, Any],
        search_data: Optional[Any] = None,
        parent_trial: Optional[DSATTrial] = None,
    ) -> DSATTrial:
        """
        Creates a new `DSATTrial` object, updates lineages as appropriate, and updates the
        searcher's Trial tracking dictionary.
        """
        # Create a consistent batch size configuration which obeys the DS constraints.
        self.enforce_consistent_batch_config(hparams)

        trial = DSATTrial(
            hparams=hparams,
            model_dir=self.model_dir,
            slots=self.slots,
            parent=parent_trial,
            search_data=search_data,
        )
        self.all_trials_dict[trial.request_id] = trial
        self.update_should_stop()
        # TODO: Delete print test.
        logging.info(f"=============Total Trials Created: {len(self)}=============")
        return trial

    def create_model_profile_info_trial(
        self,
    ) -> DSATModelProfileInfoTrial:
        # Create the special hp dictionary used for the model profile info run.
        model_profile_info_hps = copy.deepcopy(self.submitted_hps)
        model_profile_info_hps[_defaults.OVERWRITE_KEY] = merge_dicts(
            model_profile_info_hps.get(_defaults.OVERWRITE_KEY, {}),
            _defaults.MODEL_INFO_PROFILE_DS_CONFIG,
        )
        self.enforce_consistent_batch_config(model_profile_info_hps)

        model_profile_info_trial = DSATModelProfileInfoTrial(
            hparams=model_profile_info_hps, model_dir=self.model_dir, slots=self.slots
        )
        self.all_trials_dict[model_profile_info_trial.request_id] = model_profile_info_trial
        self.model_profile_info_trial = model_profile_info_trial
        return model_profile_info_trial

    def enforce_consistent_batch_config(self, hparams: Dict[str, Any]) -> None:
        """Enforces a consistent batch size configuration by altering `hparams` in-place."""
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
        root_trial_set = set()
        for trial in self.all_trials_dict.values():
            if trial.parent is None:
                if (
                    isinstance(trial, DSATModelProfileInfoTrial)
                    and not include_model_profile_info_trial
                ):
                    continue
                root_trial_set.add(trial)
        return root_trial_set

    def get_create_val_ops_list_from_trial(
        self, trial: DSATTrial, length: Optional[int] = None
    ) -> List[searcher.Operation]:
        if length is None:
            # Get the default length from the autotuning config.
            # DS has a fixed notion of what a step is while Determined does not. Make sure
            # there are no issues in reconciling this fact.
            # The +1 is required to align DS step/DET max_length conventions.
            # TODO: Clean all of this up.
            length = self.autotuning_config["end_profile_step"] + 1
        create_op = searcher.Create(
            request_id=trial.request_id,
            hparams=trial.hparams,
            checkpoint=None,
        )
        validate_after_op = searcher.ValidateAfter(request_id=trial.request_id, length=length)
        ops_list = [create_op, validate_after_op]
        return ops_list

    def update_metrics(
        self,
        request_id: uuid.UUID,
        metric: Optional[Dict[str, Any]] = None,
    ) -> None:
        """
        Updates the Trial Tracker after metrics have been reported or an error has occurred, in
        in which case `metrics = None`.
        """
        last_trial = self[request_id]
        last_trial.error = metric is None
        metric = {} if metric is None else metric
        last_trial.metric = metric

        # TODO: Curently not counting explicit OOMs or other errors (which are sometimes also
        # opaque OOMs) against num_trials_since_best_result
        # because otherwise early Trials can just all OOM, early stopping is triggered, and no
        # non-trivial results are returned. Should discuss, though.
        searcher_metric_value = metric.get(self.searcher_metric_name)
        if searcher_metric_value is not None:
            last_trial_is_best = self.best_autotuning_metric_val is None or (
                searcher_metric_value < self.best_autotuning_metric_val
                if self.smaller_is_better
                else searcher_metric_value > self.best_autotuning_metric_val
            )
            if last_trial_is_best:
                self.best_trial = last_trial
                self.best_autotuning_metric_val = searcher_metric_value
                self.num_trials_since_best_result = 0
            else:
                self.num_trials_since_best_result += 1
            self.update_should_stop()

    def update_should_stop(self) -> None:
        if not self.should_stop:
            if self.num_trials_since_best_result == self.tuner_early_stopping:
                logging.info("Early stopping criteria met, searcher will shut down.")
                self.should_stop = True
            if len(self) == self.tuner_num_trials:
                logging.info("All Trials completed, searcher will shut down.")
                self.should_stop = True

    def get_state_dict(self) -> Dict[str, Any]:
        entire_dict = self.__dict__
        init_args = inspect.signature(self.__init__).parameters
        state_dict = {k: v for k, v in entire_dict.items() if k in init_args}
        return state_dict

    @classmethod
    def from_state_dict(cls, state_dict: Dict[str, Any]) -> "DSATTrialTracker":
        trial_tracker = cls(**state_dict)
        return trial_tracker


class DSATSearchMethodBase(searcher.SearchMethod):
    """
    Base class for all DS AT searchers. Written so that only the `get_new_searcher_ops_list` method
    needs to be written overwritten when subclassing (at a minimum).
    """

    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
        model_dir: str,
        zero_search_config: Optional[str] = None,
    ) -> None:
        self.submitted_config_dict = submitted_config_dict
        self.model_dir = model_dir
        self.zero_search_config = (
            None
            if zero_search_config is None
            else _utils.get_dict_from_yaml_or_json_path(zero_search_config)
        )
        self.zero_optim_search_space = _utils.get_zero_optim_search_space(
            zero_search_config=self.zero_search_config
        )
        # TODO: Delete print test
        logging.info(f"SEARCH SPACE {self.zero_optim_search_space}")

        self.trial_tracker = DSATTrialTracker(
            submitted_config_dict=submitted_config_dict,
            model_dir=model_dir,
            zero_search_config=zero_search_config,
        )

    @abstractmethod
    def get_new_searcher_ops_list(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[searcher.Operation]:
        """
        Generates a list of new operations to run based on the results of the last trial.
        Errored trials return `metric = None`.
        """
        pass

    def initial_operations(
        self, searcher_state: searcher.SearcherState
    ) -> List[searcher.Operation]:
        """
        Submits the model info profiling run in order to collect model and resources info to
        inform the search.
        """
        model_profile_info_trial = self.trial_tracker.create_model_profile_info_trial()
        # Only a single step is required for the model profiling run.
        ops = self.trial_tracker.get_create_val_ops_list_from_trial(
            trial=model_profile_info_trial, length=1
        )
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
        self.trial_tracker.update_metrics(request_id=request_id, metric=metric)

        # TODO: remove some of these info logs. Some are just for testing.
        last_trial = self.trial_tracker[request_id]
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            logging.info(f"Approx. max mbs per stage: {last_trial.max_mbs_per_stage}")
            logging.info(f"Approx. GPU memory per stage: {last_trial.mem_per_gpu_per_stage}")
            logging.info(f"Total GPU memory: {last_trial.gpu_mem}")
            logging.info(f"Viable zero stages: {last_trial.viable_zero_stages}")

        # All DS AT Trials should be closed after validation.
        new_ops_list = [searcher.Close(request_id)]
        if not self.trial_tracker.should_stop:
            additional_ops_list = self.get_new_searcher_ops_list(
                searcher_state=searcher_state,
                request_id=request_id,
                metric=metric,
            )
            new_ops_list.extend(additional_ops_list)
        return new_ops_list

    def on_trial_closed(
        self, searcher_state: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        # TODO: Remove print tests.
        logging.info(f"Calling on_trial_closed for {request_id}")
        last_trial = self.trial_tracker[request_id]
        logging.info(f"metrics for closed trial {last_trial.metric}")
        if self.trial_tracker.should_stop:
            # Shutdown if `should_stop` is True, and this was the last running trial.
            running_trials = searcher_state.trials_created - searcher_state.trials_closed
            new_ops_list = [searcher.Shutdown()] if not running_trials else []
        else:
            new_ops_list = []
        return new_ops_list

    def on_trial_exited_early(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        exited_reason: searcher.ExitedReason,
    ) -> List[searcher.Operation]:
        # TODO: Remove print tests.
        logging.info(f"Calling on_trial_exited_early for {request_id}")
        self.trial_tracker.update_metrics(request_id=request_id, metric=None)

        last_trial = self.trial_tracker[request_id]
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            logging.info(
                "**** Shutting down DeepSpeed Autotune: Error in Model Profiling Info Trial ****"
            )
            new_ops_list = [searcher.Shutdown()]
        elif exited_reason == searcher.ExitedReason.ERRORED:
            if self.trial_tracker.should_stop:
                # Shutdown if `should_stop` is True, once all currently-running trials have completed.
                running_trials = searcher_state.trials_created - searcher_state.trials_closed
                new_ops_list = [searcher.Shutdown()] if not running_trials else []
            else:
                new_ops_list = self.get_new_searcher_ops_list(
                    searcher_state=searcher_state,
                    request_id=request_id,
                    metric=None,
                )
        else:
            # TODO: this code should never be reached, except for user error or explicit Experiment
            # cancellation. Here as a test due to previous intermittent errors which reached this
            # block. To be deleted.
            logging.info(f"############### SHOULD NOT HAVE BEEN REACHED ##############")
            logging.info(f"**** Shutting down DeepSpeed Autotune due to {exited_reason} ****")
            logging.info(f"############### SHOULD NOT HAVE BEEN REACHED ##############")
            raise RuntimeError(f"Something went wrong: Trial existed with {exited_reason}")
            # After deleting the above tests, we will just shut down in these cases:
            # new_ops_list = [searcher.Shutdown()]

        return new_ops_list

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        progress = len(searcher_state.trials_closed | searcher_state.failures) / len(
            searcher_state.trials_created
        )
        return progress

    def save_method_state(self, path: pathlib.Path) -> None:
        checkpoint_path = path.joinpath("state_dict.pkl")
        with checkpoint_path.open("wb") as f:
            pickle.dump(self.trial_tracker.get_state_dict(), f)

    def load_method_state(self, path: pathlib.Path) -> None:
        logging.info(f"Restoring searcher state from checkpoint.")
        checkpoint_path = path.joinpath("state_dict.pkl")
        with checkpoint_path.open("rb") as f:
            state_dict = pickle.load(f)
            self.trial_tracker = DSATTrialTracker.from_state_dict(state_dict)

    def _state_print_checks(self, searcher_state) -> None:
        # TODO: Delete when done testing
        running_trials_from_searcher_state = searcher_state.trials_created - (
            searcher_state.trials_closed | searcher_state.failures
        )
        logging.info(
            f"SearcherState: Created Trials ({len(searcher_state.trials_created)}) {searcher_state.trials_created}"
        )
        logging.info(
            f"SearcherState: Closed Trials ({len(searcher_state.trials_closed)}) {searcher_state.trials_closed}"
        )
        logging.info(
            f"SearcherState: Failed Trials ({len(searcher_state.failures)}) {searcher_state.failures}"
        )
        searcher_state_failed_and_closed = searcher_state.failures & searcher_state.trials_closed
        logging.info(
            f"SearcherState: Failed and Closed Trials ({len(searcher_state_failed_and_closed)}) {searcher_state_failed_and_closed}"
        )
        logging.info(
            f"SearcherState: Running Trials ({len(running_trials_from_searcher_state)}) {running_trials_from_searcher_state}"
        )


class DSATRandomSearchMethod(DSATSearchMethodBase):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

    def get_new_searcher_ops_list(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[searcher.Operation]:
        last_trial = self.trial_tracker[request_id]
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            new_ops_list = self.get_ops_list_after_model_profile_info_run()
        else:
            new_ops_list = self.get_ops_list_after_autotuning_run(last_trial)
        return new_ops_list

    def get_ops_list_after_model_profile_info_run(
        self,
    ) -> List[searcher.Operation]:
        # This isn't actually how native DS AT uses num_tuning_micro_batch_sizes, but it's a good
        # enough placeholder usage until we get other aspects of custom searcher DS AT to work.
        approx_num_lineages = (
            self.trial_tracker.tuner_num_trials // self.trial_tracker.num_tuning_micro_batch_sizes
        )
        new_ops_list = []
        for _ in range(approx_num_lineages):
            hparams, search_data = self.get_random_hparams_and_search_data()
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams,
                search_data=search_data,
                parent_trial=None,
            )
            new_ops = self.trial_tracker.get_create_val_ops_list_from_trial(trial=new_trial)
            new_ops_list.extend(new_ops)
        return new_ops_list

    def get_ops_list_after_autotuning_run(
        self,
        last_trial: DSATTrial,
    ) -> List[searcher.Operation]:
        if last_trial.num_trials_in_lineage < self.trial_tracker.num_tuning_micro_batch_sizes:
            hparams, search_data = self.get_hparams_and_search_data_after_trial(
                last_trial=last_trial,
            )
            parent_trial = last_trial

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

        else:
            hparams, search_data = self.get_random_hparams_and_search_data()
            parent_trial = None

        if hparams is None:
            # hparams will be None if a previously used configuration was suggested by the searcher.
            new_ops_list = []
        else:
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams,
                search_data=search_data,
                parent_trial=parent_trial,
            )
            new_ops_list = self.trial_tracker.get_create_val_ops_list_from_trial(trial=new_trial)
        return new_ops_list

    def get_hparams_and_search_data_after_trial(
        self,
        last_trial: DSATTrial,
    ) -> Tuple[Optional[Dict[str, Any]], Optional[Dict[str, int]]]:
        """
        Get new hparams and search data according to the results of a parent trial.
        Performs a slightly modified binary search on the train_micro_batch_size_per_gpu.
        """
        # TODO: verify we are always quitting when no more non-trivial trials are possible.

        lo, hi = last_trial.search_data["lo"], last_trial.search_data["hi"]
        mid = (lo + hi) // 2
        # TODO: edge cases and +- 1 error checks.
        if last_trial.error:
            hi = mid - 1
        else:
            lo = mid + 1
            hi = (
                hi if last_trial.error_in_direct_history else int(1.05 * hi)
            )  # TODO: let user configure ceiling factor. Current number is just a guess, and maybe
            # what native DS AT does?
        new_mid = (lo + hi) // 2
        if new_mid in last_trial.mbs_in_lineage:
            # Already tried this configuration.
            new_hparams = new_search_data = None
        else:
            new_hparams = copy.deepcopy(last_trial.hparams)
            new_hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = new_mid
            new_search_data = {"lo": lo, "hi": hi}
        return new_hparams, new_search_data

    def get_random_hparams_and_search_data(self) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        # TODO: verify that we are not repeating a previously attempted config.
        relevant_zero_stages = self.trial_tracker.model_profile_info_trial.viable_zero_stages & set(
            self.zero_optim_search_space
        )
        random_zero_stage = random.choice(tuple(relevant_zero_stages))
        zero_optim_config = _utils.get_random_zero_optim_dict_from_search_space(
            random_zero_stage, self.zero_optim_search_space
        )
        new_hparams = copy.deepcopy(self.trial_tracker.submitted_hps_with_autotuning)
        new_hparams[_defaults.OVERWRITE_KEY] = merge_dicts(
            new_hparams.get(_defaults.OVERWRITE_KEY, {}),
            {"zero_optimization": zero_optim_config},
        )
        random_zero_stage_max_mbs = self.trial_tracker.model_profile_info_trial.max_mbs_per_stage[
            random_zero_stage
        ]
        lo, hi = 1, 2 * random_zero_stage_max_mbs - 1
        mid = (lo + hi) // 2
        search_data = {
            "lo": lo,
            "hi": hi,
        }
        new_hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = mid
        return (new_hparams, search_data)


class SimpleBatchSearch(DSATSearchMethodBase):
    """
    Dumb searcher which just submits Trials with linearly increasing batch sizes, from 2 up to
    self.trial_tracker.tuner_num_trials.
    """

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

    def get_new_searcher_ops_list(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Optional[Union[float, Dict[str, Any]]] = None,
    ) -> List[searcher.Operation]:
        last_trial = self.trial_tracker[request_id]
        if isinstance(last_trial, DSATModelProfileInfoTrial):
            # Delete special DS keys which force a model profiling info run.
            hparams_without_profile_info_keys = last_trial.hparams
            del hparams_without_profile_info_keys[_defaults.OVERWRITE_KEY]["autotuning"][
                "model_info"
            ]
            del hparams_without_profile_info_keys[_defaults.OVERWRITE_KEY]["autotuning"][
                "model_info_path"
            ]
            new_ops_list = self.get_ops_list_after_model_profile_info_run(
                hparams_without_profile_info_keys
            )
        else:
            new_ops_list = []
        return new_ops_list

    def get_ops_list_after_model_profile_info_run(
        self, hparams: Dict[str, Any]
    ) -> List[searcher.Operation]:
        # This isn't actually how native DS AT uses num_tuning_micro_batch_sizes, but it's a good
        # enough placeholder usage until we get other aspects of custom searcher DS AT to work.
        new_ops_list = []
        for tmbs in range(2, self.trial_tracker.tuner_num_trials + 1):
            hparams["train_micro_batch_size_per_gpu"] = tmbs
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams,
                search_data=None,
                parent_trial=None,
            )
            new_ops = self.trial_tracker.get_create_val_ops_list_from_trial(trial=new_trial)
            new_ops_list.extend(new_ops)
        return new_ops_list
