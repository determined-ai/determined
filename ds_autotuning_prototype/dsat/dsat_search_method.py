import copy
import logging
import random
import uuid
from abc import abstractmethod
from typing import Any, Dict, List, Optional, Set, Tuple, Union

import determined as det
from determined import searcher
from dsat import constants, utils


class DSATTrial:
    def __init__(
        self,
        hparams: Dict[str, Any],
        checkpoint: Optional[str] = None,
        is_model_profiling_info_run: bool = False,
    ) -> None:
        self.hparams = hparams
        self.ds_config = self.hparams["ds_config"]
        self.checkpoint = checkpoint
        self.request_id = uuid.uuid4()
        self.metric = None
        self.is_model_profiling_info_run = is_model_profiling_info_run

        # Properties for lineage tracking.
        self._parent = None
        self._children = set()

        # Arbitrary attribute for search-specific data tracking.
        self._search_data = None

    @property
    def zero_stage(self):
        try:
            zero_stage = int(self.hparams["ds_config"]["zero_optimization"]["stage"])
        except KeyError:
            zero_stage = 0  # The DS Default. TODO: add to constants.py
        return zero_stage

    def record_metric(self, metric: float) -> None:
        self.metric = metric

    def get_state_dict(self) -> Dict[str, Any]:
        pass

    def load_state_dict(self) -> None:
        pass

    def get_create_operation(self) -> searcher.Operation:
        create_op = searcher.Create(
            request_id=self.request_id,
            hparams=self.hparams,
            checkpoint=self.checkpoint,
        )
        return create_op

    def get_validate_after_operation(self, length: int) -> searcher.Operation:
        validate_after_op = searcher.ValidateAfter(request_id=self.request_id, length=length)
        return validate_after_op

    def get_ops_list(self, length: int) -> List[searcher.Operation]:
        return [self.get_create_operation(), self.get_validate_after_operation(length)]

    def add_child(self, trial: "DSATTrial") -> None:
        """Register child-parent relationship in lineage tree."""
        self._children.add(trial)
        trial._parent = self

    @property
    def is_lineage_root(self) -> bool:
        return self._parent is None

    def get_lineage_root(self) -> "DSATTrial":
        """Returns the root Trial object in the present lineage."""
        if self.is_lineage_root:
            return self
        else:
            return self._parent.get_lineage_root()

    def get_num_trials_in_lineage(self) -> int:
        """Computes total number of trials in lineage tree, starting from root."""
        root = self.get_lineage_root()
        children = root._children
        num_trials = 1 + len(children)
        while children:
            random_child = children.pop()
            new_children = random_child._children
            num_trials += len(new_children)
            children |= new_children
        return num_trials

    def get_trial_lineage_set(self) -> Set["DSATTrial"]:
        """Computes total number of trials in lineage tree, starting from root."""
        root = self.get_lineage_root()
        trials_set = {root}
        children = root._children
        while children:
            random_child = children.pop()
            trials_set.add(random_child)
            children |= random_child._children
        return trials_set

    def set_search_data(self, search_data: Any) -> None:
        self._search_data = search_data


class DSATSearchMethodBase(searcher.SearchMethod):
    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
    ) -> None:
        self._submitted_config_dict = submitted_config_dict
        self._submitted_hps = self._submitted_config_dict["hyperparameters"]
        self._ds_config = self._submitted_hps["ds_config"]
        self._autotuning_config = self._submitted_hps["autotuning_config"]
        # Track all trial objects in a dict indexed by request_id.
        self._all_trials_dict = dict()
        self._model_profile_info_trial_id = None

        self._base_hparams_with_profiling = copy.deepcopy(self._submitted_hps)
        utils.replace_dict_in_place(
            self._base_hparams_with_profiling["ds_config"],
            {"flops_profiler": constants.FLOPS_PROFILER_CONFIG},  # TODO: don't hardcode profiler.
        )

        # Non-trivial values instantiated after model profiling run
        self._model_profiling_info_results_dict = dict()
        self._gpu_mem_in_bytes = None
        self._activation_mem_per_gpu_in_bytes = None
        self._num_params = None
        self._trainable_num_params = None
        self._mem_per_gpu_per_stage = dict()
        self._viable_zero_stages = set()
        self._max_mbs_per_stage = dict()

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
        model_profile_info_hps = copy.deepcopy(self._submitted_hps)
        utils.replace_dict_in_place(
            model_profile_info_hps["ds_config"],
            constants.MODEL_INFO_PROFILING_DS_CONFIG,
        )
        model_profile_info_trial = self._get_and_register_trial(
            hparams=model_profile_info_hps, is_model_profiling_info_run=True
        )
        self._model_profile_info_trial_id = model_profile_info_trial.request_id
        # Only a single step is required for the model profiling run.
        return self._get_trial_ops_list_from_id(self._model_profile_info_trial_id, length=1)

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
        last_trial = self._all_trials_dict[request_id]
        if last_trial.is_model_profiling_info_run:
            self._process_model_profiling_info_run(metric)
        # All DS AT Trials should be closed upon completion.
        ops = [searcher.Close(request_id=request_id)]
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
        last_trial = self._all_trials_dict[request_id]
        if last_trial.is_model_profiling_info_run:
            logging.info(f"model profiling run failed due to {exited_reason}, shutting down")
            return [searcher.Shutdown()]
        logging.info("EXITED REASON", exited_reason)
        return []

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        return 0

    def _get_closed_trials_dict(
        self, searcher_state: searcher.SearcherState, zero_stage: Optional[int] = None
    ) -> Dict[str, DSATTrial]:
        closed_request_ids = searcher_state.trials_closed
        closed_trials_dict = self._get_trials_dict_from_request_id_set(
            closed_request_ids, zero_stage
        )
        return closed_trials_dict

    def _get_failed_trials_dict(
        self, searcher_state: searcher.SearcherState, zero_stage: Optional[int]
    ) -> Dict[str, DSATTrial]:
        failed_request_ids = searcher_state.failures
        failed_trials_dict = self._get_trials_dict_from_request_id_set(
            failed_request_ids, zero_stage
        )
        return failed_trials_dict

    def _get_running_trials_dict(
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
        for rid in request_id_set:
            trial = self._all_trials_dict[rid]
            if zero_stage is None or trial.zero_stage == zero_stage:
                request_id_set[rid] = trial

    def _get_mem_per_gpu_per_stage(self) -> Dict[int, int]:
        """
        Returns the required gpu memory per stage.
        """
        try:
            fp16 = self._ds_config["fp16"]["enabled"]
        except KeyError:
            fp16 = False
        params_mem = self._num_params * (2 if fp16 else 4)
        gradients_mem = self._trainable_num_params * (2 if fp16 else 4)
        # optimizer_mem assumes Adam like DS. TODO: don't assume this.
        optimizer_mem = self._trainable_num_params * (16 if fp16 else 8)

        slots = self._submitted_config_dict["resources"]["slots_per_trial"]
        # TODO: account for model parallel degree.
        non_activation_mem_per_gpu_per_stage = {
            0: params_mem + gradients_mem + optimizer_mem,
            1: params_mem + gradients_mem + optimizer_mem // slots,
            2: params_mem + (gradients_mem + optimizer_mem) // slots,
            3: (params_mem + gradients_mem + optimizer_mem) // slots,
        }
        mem_per_gpu_per_stage = {
            stage: mem + self._activation_mem_per_gpu_in_bytes
            for stage, mem in non_activation_mem_per_gpu_per_stage.items()
        }
        return mem_per_gpu_per_stage

    def _get_viable_zero_stages(self) -> Set[int]:
        """
        Returns the set of viable zero stages based on a rough computation.
        """
        mem_per_gpu_per_stage = self._get_mem_per_gpu_per_stage()
        # TODO: account for model parallelism.
        viable_stages = {
            stage for stage, mem in mem_per_gpu_per_stage.items() if mem < self._gpu_mem_in_bytes
        }
        return viable_stages

    def _get_max_mbs_per_stage(self) -> Dict[int, int]:
        """
        Returns the approximate max train_micro_batch_size_per_gpu per stage.
        """
        max_mbs_per_stage = {
            stage: (self._gpu_mem_in_bytes - mem) // self._activation_mem_per_gpu_in_bytes
            for stage, mem in self._mem_per_gpu_per_stage.items()
            if stage in self._viable_zero_stages
        }
        return max_mbs_per_stage

    def _process_model_profiling_info_run(self, metric: Dict[str, Any]):
        self._model_profiling_info_results_dict = metric
        self._gpu_mem_in_bytes = self._model_profiling_info_results_dict["gpu_mem_in_bytes"]
        self._activation_mem_per_gpu_in_bytes = self._model_profiling_info_results_dict[
            "activation_mem_per_gpu"
        ]
        self._num_params = self._model_profiling_info_results_dict["num_params"]
        self._trainable_num_params = self._model_profiling_info_results_dict["trainable_num_params"]

        self._mem_per_gpu_per_stage = self._get_mem_per_gpu_per_stage()
        self._viable_zero_stages = self._get_viable_zero_stages()
        self._max_mbs_per_stage = self._get_max_mbs_per_stage()
        logging.info(f"Viable zero stages: {self._viable_zero_stages}")
        logging.info(f"approx max mbs: {self._max_mbs_per_stage}")

    def _get_and_register_trial(
        self,
        hparams: Dict[str, Any],
        is_model_profiling_info_run: bool = False,
        search_data: Optional[Any] = None,
        parent_trial: Optional[DSATTrial] = None,
    ) -> DSATTrial:
        trial = DSATTrial(hparams=hparams, is_model_profiling_info_run=is_model_profiling_info_run)
        if search_data is not None:
            trial.set_search_data(search_data)
        if parent_trial is not None:
            parent_trial.add_child(trial)
        self._all_trials_dict[trial.request_id] = trial
        return trial

    def _get_trial_ops_list_from_id(
        self, request_id: uuid.UUID, length: int
    ) -> List[searcher.Operation]:
        trial = self._all_trials_dict[request_id]
        ops_list = trial.get_ops_list(length=length)
        return ops_list


class DSATRandomSearchMethod(DSATSearchMethodBase):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        # TODO: get desired zero stages from config. Currently just running all viable.
        self._tuner_num_trials = self._autotuning_config["tuner_num_trials"]
        self._num_tuning_micro_batch_sizes = self._autotuning_config["num_tuning_micro_batch_sizes"]

    def get_new_searcher_ops_list(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
        last_trial: DSATTrial,
    ) -> List[searcher.Operation]:
        last_trial = self._all_trials_dict[request_id]
        # TODO: clean up.
        if last_trial.is_model_profiling_info_run:
            if not metric:
                raise ValueError("No metric reported for info profiling run; failed?")
            approx_num_lineages = self._tuner_num_trials // self._num_tuning_micro_batch_sizes
            new_ops_list = []
            for _ in range(approx_num_lineages):
                hparams, search_data = self._get_random_hparams_and_search_data()
                new_trial = self._get_and_register_trial(
                    hparams=hparams, search_data=search_data, parent_trial=None
                )
                new_ops = self._get_trial_ops_list_from_id(
                    request_id=new_trial.request_id, length=constants.DSAT_MAX_LENGTH_STEPS
                )
                new_ops_list.extend(new_ops)
        elif len(searcher_state.trials_created) < self._tuner_num_trials:
            if last_trial.get_num_trials_in_lineage() < self._num_tuning_micro_batch_sizes:
                hparams, search_data = self._get_hparams_and_search_data_from_results(
                    last_trial=last_trial, metric=metric
                )
                parent_trial = last_trial
            else:
                hparams, search_data = self._get_random_hparams_and_search_data()
                parent_trial = None
            new_trial = self._get_and_register_trial(
                hparams=hparams, search_data=search_data, parent_trial=parent_trial
            )
            new_ops_list = self._get_trial_ops_list_from_id(
                request_id=new_trial.request_id, length=constants.DSAT_MAX_LENGTH_STEPS
            )
        else:
            new_ops_list = []
        return new_ops_list

    def _get_hparams_and_search_data_from_results(
        self,
        last_trial: DSATTrial,
        metric: Union[float, Dict[str, Any]],
    ) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        lo, hi = last_trial._search_data["lo"], last_trial._search_data["hi"]
        mid = (lo + hi) // 2
        last_trial_oom = metric.get(constants.OOM_KEY, False)
        oom_in_lineage = last_trial_oom or last_trial._search_data["oom_in_lineage"]
        # TODO: edge cases and +- 1 error checks.
        if last_trial_oom:
            hi = mid - 1
        else:
            lo = mid + 1
            hi = (
                hi if oom_in_lineage else int(1.25 * hi)
            )  # TODO: let user configure ceiling factor. Current number is just a guess.
        new_mid = (lo + hi) // 2
        new_hparams = copy.deepcopy(last_trial.hparams)
        new_hparams["ds_config"]["train_micro_batch_size_per_gpu"] = new_mid
        return new_hparams, {"lo": lo, "hi": hi, "oom_in_lineage": oom_in_lineage}

    def _get_random_hparams_and_search_data(self) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        random_zero_stage = random.choice(tuple(self._viable_zero_stages))
        new_hparams = copy.deepcopy(self._base_hparams_with_profiling)
        zero_optim_config = utils.get_random_zero_optim_dict_for_zero_stage(random_zero_stage)
        utils.replace_dict_in_place(
            new_hparams["ds_config"], {"zero_optimization": zero_optim_config}
        )
        initial_search_data = {
            "lo": 1,
            "hi": self._max_mbs_per_stage[random_zero_stage],
            "oom_in_lineage": False,
        }
        mid = (initial_search_data["lo"] + initial_search_data["hi"]) // 2
        new_hparams["ds_config"]["train_micro_batch_size_per_gpu"] = mid
        del new_hparams["autotuning_config"]  # Cleanup for better Web UI visuals.
        return (new_hparams, initial_search_data)
