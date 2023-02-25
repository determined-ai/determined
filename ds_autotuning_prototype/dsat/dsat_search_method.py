import copy
import logging
import pathlib
import pickle
import random
import uuid
from abc import abstractmethod
from typing import Any, Dict, List, Optional, Set, Tuple, Union

import determined as det
from determined import searcher
from dsat import constants, utils


class DSATTrial:
    """
    Serializable helper class for tracking the results and properties of individual Trials.
    """

    def __init__(
        self,
        hparams: Dict[str, Any],
        is_model_profiling_info_run: bool = False,
        request_id: Optional[uuid.UUID] = None,
        metric: Optional[Any] = None,
        parent: Optional["DSATTrial"] = None,
        children: Optional[Set["DSATTrial"]] = None,
        search_data: Optional[Any] = None,
    ) -> None:
        self.hparams = hparams
        self.ds_config = self.hparams["ds_config"]
        self.is_model_profiling_info_run = is_model_profiling_info_run
        self.request_id = request_id or uuid.uuid4()
        self.metric = metric

        # Properties for lineage tracking.
        self.parent = parent
        self.children = children or set()

        # Arbitrary attribute for search-specific data tracking.
        self.search_data = search_data

    @property
    def zero_stage(self):
        try:
            zero_stage = int(self.hparams["ds_config"]["zero_optimization"]["stage"])
        except KeyError:
            zero_stage = 0  # The DS Default. TODO: add to constants.py
        return zero_stage

    def record_metric(self, metric: Dict[str, Any]) -> None:
        self.metric = metric

    def set_search_data(self, search_data: Any) -> None:
        self.search_data = search_data

    def add_child(self, trial: "DSATTrial") -> None:
        """Register child-parent relationship in lineage tree."""
        self.children.add(trial.request_id)
        trial.parent = self

    def get_state_dict(self) -> Dict[str, Any]:
        state_dict = {
            "hparams": self.hparams,
            "request_id": self.request_id,
            "metric": self.metric,
            "is_model_profiling_info_run": self.is_model_profiling_info_run,
            "parent": self.parent,
            "children": self.children,
            "search_data": self.search_data,
        }
        return state_dict

    @classmethod
    def from_state_dict(cls, state_dict: Dict[str, Any]) -> "DSATTrial":
        return cls(**state_dict)


class DSATTrialTracker:
    """
    Class for organizing DSATTrial instances and retrieving their info.
    """

    def __init__(self) -> None:
        self._all_trials_dict = {}
        # Altered after running autotuning trials.
        self.best_autotuning_metric_val = None
        self.num_trials_since_best_result = None
        self.should_early_stop = False

    def get_trial_by_id(self, request_id: uuid.UUID) -> DSATTrial:
        return self._all_trials_dict[request_id]

    def create_trial(
        self,
        hparams: Dict[str, Any],
        is_model_profiling_info_run: bool = False,
        search_data: Optional[Any] = None,
        parent_trial: Optional[DSATTrial] = None,
    ) -> DSATTrial:
        """
        Creates a new `DSATTrial` object, updates lineages as appropriate, and updates the
        searcher's Trial tracking dictionary.
        """
        trial = DSATTrial(hparams=hparams, is_model_profiling_info_run=is_model_profiling_info_run)
        if search_data is not None:
            trial.set_search_data(search_data)
        if parent_trial is not None:
            parent_trial.add_child(trial)
        self._all_trials_dict[trial.request_id] = trial
        return trial

    def get_closed_trials_dict(
        self, searcher_state: searcher.SearcherState, zero_stage: Optional[int] = None
    ) -> Dict[str, DSATTrial]:
        closed_request_ids = searcher_state.trials_closed
        closed_trials_dict = self._get_trials_dict_from_request_id_set(
            closed_request_ids, zero_stage
        )
        return closed_trials_dict

    def get_failed_trials_dict(
        self, searcher_state: searcher.SearcherState, zero_stage: Optional[int]
    ) -> Dict[str, DSATTrial]:
        failed_request_ids = searcher_state.failures
        failed_trials_dict = self._get_trials_dict_from_request_id_set(
            failed_request_ids, zero_stage
        )
        return failed_trials_dict

    def get_running_trials_dict(
        self, searcher_state: searcher.SearcherState, zero_stage: Optional[int]
    ) -> Dict[str, DSATTrial]:
        running_request_ids = searcher_state.trials_created - (
            searcher_state.trials_closed | searcher_state.failures
        )
        running_trials_dict = self._get_trials_dict_from_request_id_set(
            running_request_ids, zero_stage
        )
        return running_trials_dict

    def _get_trials_dict_from_request_id_set(
        self, request_id_set: Set[int], zero_stage: Optional[int]
    ):
        for r_id in request_id_set:
            trial = self._all_trials_dict[r_id]
            if zero_stage is None or trial.zero_stage == zero_stage:
                request_id_set[r_id] = trial

    def get_ops_list_from_trial(self, trial: DSATTrial, length: int) -> List[searcher.Operation]:
        create_op = searcher.Create(
            request_id=trial.request_id,
            hparams=trial.hparams,
            checkpoint=None,
        )
        validate_after_op = searcher.ValidateAfter(request_id=trial.request_id, length=length)
        ops = [create_op, validate_after_op]
        return ops

    def get_trial_children(self, trial: DSATTrial) -> Set[DSATTrial]:
        trial_children = {self._all_trials_dict[r_id] for r_id in trial.children}
        return trial_children

    def get_trial_parent(self, trial: DSATTrial) -> DSATTrial:
        trial_parent = self._all_trials_dict[trial.parent]
        return trial_parent

    def is_trial_lineage_root(self, trial: DSATTrial) -> bool:
        return trial.parent is None

    def get_trial_lineage_root(self, trial: DSATTrial) -> DSATTrial:
        """Returns the root Trial object in the present lineage."""
        if self.is_trial_lineage_root(trial):
            return trial
        else:
            return self.get_trial_lineage_root(self.get_trial_parent(trial))

    def get_trial_lineage_set(self, trial: DSATTrial) -> Set[DSATTrial]:
        """Computes set of trials in lineage tree."""
        root = self.get_trial_lineage_root(trial)
        trials_set = {root}
        children = self.get_trial_children(root)
        while children:
            random_child = children.pop()
            trials_set.add(random_child)
            children |= self.get_trial_children(random_child)
        return trials_set

    def get_num_trials_in_lineage(self, trial: DSATTrial) -> int:
        """Computes total number of trials in lineage tree."""
        num_trials = len(self.get_trial_lineage_set(trial))
        return num_trials

    def update_best_trial_info(
        self,
        last_trial: DSATTrial,
        metric: Dict[str, Any],
        searcher_metric_name: str,
        smaller_is_better: bool,
    ):
        if constants.OOM_KEY in metric:
            last_trial_is_best = False
        else:
            searcher_metric_value = metric[searcher_metric_name]
            last_trial_is_best = self.best_autotuning_metric_val is None or (
                searcher_metric_value < self.best_autotuning_metric_val
                if smaller_is_better
                else searcher_metric_value > self.best_autotuning_metric_val
            )
        if last_trial_is_best:
            self.best_autotuning_metric_val = searcher_metric_value
            self.num_trials_since_best_result = 0
        else:
            self.num_trials_since_best_result += 1

    def get_state_dict(self) -> Dict[uuid.UUID, Any]:
        state_dict = {
            "all_trials_dict": self._all_trials_dict,
            "best_autotuning_metric_val": self.best_autotuning_metric_val,
            "num_trials_since_best_result": self.num_trials_since_best_result,
            "should_early_stop": self.should_early_stop,
        }
        return state_dict

    @classmethod
    def from_state_dict(cls, state_dict: Dict[uuid.UUID, Any]) -> "DSATTrialTracker":
        trial_tracker = cls()
        trial_tracker._all_trials_dict = state_dict["all_trials_dict"]
        trial_tracker.best_autotuning_metric_val = state_dict["best_autotuning_metric_val"]
        trial_tracker.num_trials_since_best_result = state_dict["num_trials_since_best_result"]
        trial_tracker.should_early_stop = state_dict["should_early_stop"]
        return trial_tracker


class DSATModelProfilingInfo:
    """
    Helper class for processing the model profiling info run.
    """

    def __init__(
        self,
        request_id: uuid.UUID,
        model_profiling_info_results: Dict[str, Any],
        slots: int,
        fp16: bool,
    ) -> None:
        self.request_id = request_id
        self.model_profiling_info_results = model_profiling_info_results
        self.slots = slots
        self.fp16 = fp16

        self.gpu_mem_in_bytes = self.model_profiling_info_results["gpu_mem_in_bytes"]
        self.activation_mem_per_gpu_in_bytes = self.model_profiling_info_results[
            "activation_mem_per_gpu"
        ]
        self.num_params = self.model_profiling_info_results["num_params"]
        self.trainable_num_params = self.model_profiling_info_results["trainable_num_params"]

        self.mem_per_gpu_per_stage = self.get_mem_per_gpu_per_stage()
        self.viable_zero_stages = self.get_viable_zero_stages()
        self.max_mbs_per_stage = self.get_max_mbs_per_stage()

    def get_mem_per_gpu_per_stage(self) -> Dict[int, int]:
        """
        Returns the required gpu memory in bytes, per stage.
        """
        params_mem = self.num_params * (2 if self.fp16 else 4)
        gradients_mem = self.trainable_num_params * (2 if self.fp16 else 4)
        # optimizer_mem assumes Adam like DS. TODO: don't assume this.
        optimizer_mem = self.trainable_num_params * (16 if self.fp16 else 8)

        # TODO: account for model parallel degree.
        non_activation_mem_per_gpu_per_stage = {
            0: params_mem + gradients_mem + optimizer_mem,
            1: params_mem + gradients_mem + optimizer_mem // self.slots,
            2: params_mem + (gradients_mem + optimizer_mem) // self.slots,
            3: (params_mem + gradients_mem + optimizer_mem) // self.slots,
        }
        mem_per_gpu_per_stage = {
            stage: mem + self.activation_mem_per_gpu_in_bytes
            for stage, mem in non_activation_mem_per_gpu_per_stage.items()
        }
        return mem_per_gpu_per_stage

    def get_viable_zero_stages(self) -> Set[int]:
        """
        Returns the set of viable zero stages based on a rough computation.
        """
        # TODO: account for model parallelism. Add a fudge factor for a little leeway?
        viable_stages = {
            stage
            for stage, mem in self.mem_per_gpu_per_stage.items()
            if mem < self.gpu_mem_in_bytes
        }
        logging.info(f"Viable zero stages: {viable_stages}")
        return viable_stages

    def get_max_mbs_per_stage(self) -> Dict[int, int]:
        """
        Returns the approximate max train_micro_batch_size_per_gpu (mbs) per stage.
        """
        max_mbs_per_stage = {
            stage: (self.gpu_mem_in_bytes - mem) // self.activation_mem_per_gpu_in_bytes
            for stage, mem in self.mem_per_gpu_per_stage.items()
            if stage in self.viable_zero_stages
        }
        return max_mbs_per_stage

    def get_state_dict(self) -> Dict[str, Any]:
        state_dict = {
            "request_id": self.request_id,
            "model_profiling_info_results": self.model_profiling_info_results,
            "slots": self.slots,
            "fp16": self.fp16,
        }
        return state_dict

    @classmethod
    def from_state_dict(cls, state_dict: Dict[str, Any]) -> "DSATModelProfilingInfo":
        return cls(**state_dict)


class DSATSearchMethodBase(searcher.SearchMethod):
    """
    Base searcher class implementing common methods.
    """

    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
    ) -> None:
        self._submitted_config_dict = submitted_config_dict
        self._searcher_metric_name = self._submitted_config_dict["searcher"]["metric"]
        self._smaller_is_better = self._submitted_config_dict["searcher"].get(
            "smaller_is_better", True
        )
        self._submitted_hps = self._submitted_config_dict["hyperparameters"]
        self._ds_config = self._submitted_hps["ds_config"]
        self._tbs, self._mbs, self._gas = utils.get_tbs_mps_gas(self._ds_config)
        self._autotuning_config = self._ds_config["autotuning"]
        self._tuner_num_trials = self._autotuning_config["tuner_num_trials"]
        self._num_tuning_micro_batch_sizes = self._autotuning_config["num_tuning_micro_batch_sizes"]
        self._tuner_early_stopping = self._autotuning_config["num_tuning_micro_batch_sizes"]

        self.trial_tracker = DSATTrialTracker()

        # Non-trivial values instantiated after model profiling run
        self.model_profile_info = None

    @abstractmethod
    def get_new_searcher_ops_list(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
        last_trial: DSATTrial,
    ) -> List[searcher.Operation]:
        """Generates a list of new operations to run."""
        pass

    def initial_operations(
        self, searcher_state: searcher.SearcherState
    ) -> List[searcher.Operation]:
        """
        Submits the model info profiling run in order to collect model and resources info to
        inform the search.
        """
        model_profile_info_hps = copy.deepcopy(self._submitted_hps)
        utils.replace_dict_in_place(
            model_profile_info_hps["ds_config"],
            constants.MODEL_INFO_PROFILING_DS_CONFIG,
        )
        model_profile_info_trial = self.trial_tracker.create_trial(
            hparams=model_profile_info_hps, is_model_profiling_info_run=True
        )
        # Only a single step is required for the model profiling run.
        ops = self.trial_tracker.get_ops_list_from_trial(trial=model_profile_info_trial, length=1)
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
        last_trial = self.trial_tracker.get_trial_by_id(request_id)
        if last_trial.is_model_profiling_info_run:
            slots = self._submitted_config_dict["resources"]["slots_per_trial"]
            if "fp16" in self._ds_config:
                fp16 = self._ds_config["fp16"]["enabled"]
            else:
                fp16 = False
            self.model_profile_info = DSATModelProfilingInfo(
                request_id=request_id,
                model_profiling_info_results=metric,
                slots=slots,
                fp16=fp16,
            )
        else:
            last_trial.record_metric(metric)
            self.trial_tracker.update_best_trial_info(
                last_trial=last_trial,
                metric=metric,
                searcher_metric_name=self._searcher_metric_name,
                smaller_is_better=self._smaller_is_better,
            )

        # All DS AT Trials should be closed upon completion.
        ops = [searcher.Close(request_id=request_id)]

        # Abandon the search if the early stopping criteria is met, othewise continues
        self.trial_tracker.should_early_stop = (
            self.trial_tracker.should_early_stop
            or self.trial_tracker.num_trials_since_best_result == self._tuner_early_stopping
        )
        if self.trial_tracker.should_early_stop:
            new_ops_list = []
            logging.info("Early stopping criteria met, no new Trials will be submitted.")
        else:
            new_ops_list = self.get_new_searcher_ops_list(
                searcher_state=searcher_state,
                request_id=request_id,
                metric=metric,
                last_trial=last_trial,
            )
        ops.extend(new_ops_list)
        return ops

    def on_trial_closed(
        self, searcher_state: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        running_trial_ids = (
            searcher_state.trials_created - searcher_state.trials_closed - searcher_state.failures
        )
        if not running_trial_ids:
            return [searcher.Shutdown()]
        return []

    def on_trial_exited_early(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        exited_reason: searcher.ExitedReason,
    ) -> List[searcher.Operation]:
        last_trial = self.trial_tracker.get_trial_by_id(request_id)
        if last_trial.is_model_profiling_info_run:
            return [searcher.Shutdown()]
        return []

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        return 0

    def save_method_state(self, path: pathlib.Path) -> None:
        state_dict = {
            "trial_tracker": self.trial_tracker.get_state_dict(),
            "model_profile_info": self.model_profile_info.get_state_dict()
            if self.model_profile_info is not None
            else None,
        }
        checkpoint_path = path.joinpath("state_dict.pkl")
        with checkpoint_path.open("wb") as f:
            pickle.dump(state_dict, f)

    def load_method_state(self, path: pathlib.Path) -> None:
        checkpoint_path = path.joinpath("state_dict.pkl")
        with checkpoint_path.open("rb") as f:
            state_dict = pickle.load(f)
            self.trial_tracker = DSATTrialTracker.from_state_dict(state_dict["trial_tracker"])
            self.model_profile_info = DSATModelProfilingInfo.from_state_dict(
                state_dict["model_profile_info"]
                if state_dict["model_profile_info"] is not None
                else None
            )


class DSATRandomSearchMethod(DSATSearchMethodBase):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        # TODO: get desired zero stages from config. Currently just running all viable.

    def get_new_searcher_ops_list(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
        last_trial: DSATTrial,
    ) -> List[searcher.Operation]:
        last_trial = self.trial_tracker.get_trial_by_id(request_id)
        if last_trial.is_model_profiling_info_run:
            new_ops_list = self._get_ops_list_after_model_profiling_info_run(metric)
        elif len(searcher_state.trials_created) < self._tuner_num_trials:
            new_ops_list = self._get_ops_list_after_autotuning_run(metric, last_trial)
        else:
            new_ops_list = []
        return new_ops_list

    def _get_ops_list_after_model_profiling_info_run(
        self,
        metric: Union[float, Dict[str, Any]],
    ) -> List[searcher.Operation]:
        approx_num_lineages = self._tuner_num_trials // self._num_tuning_micro_batch_sizes
        new_ops_list = []
        for _ in range(approx_num_lineages):
            hparams, search_data = self._get_random_hparams_and_search_data()
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams, search_data=search_data, parent_trial=None
            )
            # A +1 is required to align DS step/DET max_length conventions.
            length = self._autotuning_config.get("end_profile_step", constants.END_PROFILE_STEP) + 1
            new_ops = self.trial_tracker.get_ops_list_from_trial(trial=new_trial, length=length)
            new_ops_list.extend(new_ops)
        return new_ops_list

    def _get_ops_list_after_autotuning_run(
        self,
        metric: Union[float, Dict[str, Any]],
        last_trial: DSATTrial,
    ) -> List[searcher.Operation]:
        if (
            self.trial_tracker.get_num_trials_in_lineage(last_trial)
            < self._num_tuning_micro_batch_sizes
        ):
            hparams, search_data = self._get_hparams_and_search_data_from_results(
                last_trial=last_trial, metric=metric
            )
            parent_trial = last_trial
        else:
            hparams, search_data = self._get_random_hparams_and_search_data()
            parent_trial = None
        if hparams is None:
            new_ops_list = []
        else:
            new_trial = self.trial_tracker.create_trial(
                hparams=hparams, search_data=search_data, parent_trial=parent_trial
            )
            new_ops_list = self.trial_tracker.get_ops_list_from_trial(
                trial=new_trial, length=constants.END_PROFILE_STEP
            )
        return new_ops_list

    def _get_hparams_and_search_data_from_results(
        self,
        last_trial: DSATTrial,
        metric: Union[float, Dict[str, Any]],
    ) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        """
        Perform a slightly modified binary search on the train_micro_batch_size_per_gpu.
        """
        lo, hi = last_trial.search_data["lo"], last_trial.search_data["hi"]
        mid = (lo + hi) // 2
        last_trial_oom = constants.OOM_KEY in metric
        oom_in_lineage = last_trial_oom or last_trial.search_data["oom_in_lineage"]
        # TODO: edge cases and +- 1 error checks.
        if last_trial_oom:
            hi = mid - 1
        else:
            lo = mid + 1
            hi = (
                hi if oom_in_lineage else int(1.05 * hi)
            )  # TODO: let user configure ceiling factor. Current number is just a guess, and maybe
            # what native DS AT does.
        new_mid = (lo + hi) // 2
        if new_mid == lo:
            new_hparams = None
        else:
            new_hparams = copy.deepcopy(last_trial.hparams)
            new_hparams["ds_config"]["train_micro_batch_size_per_gpu"] = new_mid
        return new_hparams, {"lo": lo, "hi": hi, "oom_in_lineage": oom_in_lineage}

    def _get_random_hparams_and_search_data(self) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        random_zero_stage = random.choice(tuple(self.model_profile_info.viable_zero_stages))
        new_hparams = copy.deepcopy(self._submitted_hps)
        zero_optim_config = utils.get_random_zero_optim_dict_for_zero_stage(random_zero_stage)
        utils.replace_dict_in_place(
            new_hparams["ds_config"], {"zero_optimization": zero_optim_config}
        )
        initialsearch_data = {
            "lo": 1,
            "hi": 2 * self.model_profile_info.max_mbs_per_stage[random_zero_stage] - 1,
            "oom_in_lineage": False,
        }
        mid = (initialsearch_data["lo"] + initialsearch_data["hi"]) // 2
        new_hparams["ds_config"]["train_micro_batch_size_per_gpu"] = mid
        return (new_hparams, initialsearch_data)
