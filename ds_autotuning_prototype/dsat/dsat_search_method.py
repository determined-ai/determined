import argparse
import copy
import logging
import uuid
from typing import Any, Dict, List, Optional, Set, Union

from dsat import constants, utils

import determined as det
from determined import searcher


class DSATTrial:
    def __init__(self, hparams: Dict[str, Any], checkpoint: Optional[str] = None) -> None:
        self.hparams = hparams
        self.checkpoint = checkpoint
        self.request_id = uuid.uuid4()
        self.metric = None

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


class DSATSearchMethod(searcher.SearchMethod):
    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
    ) -> None:
        self._submitted_hps = submitted_config_dict["hyperparameters"]
        self._slots = submitted_config_dict["resources"]["slots_per_trial"]
        self._autotuning_config = self._submitted_hps["autotuning_config"]
        self._ds_config = self._submitted_hps["ds_config"]
        self._all_trials_dict = dict()
        self._running_trial_ids = set()
        self._base_hps_with_profiling = copy.deepcopy(self._submitted_hps)
        utils.replace_dict_in_place(
            self._base_hps_with_profiling["ds_config"],
            {"flops_profiler": constants.FLOPS_PROFILER_CONFIG},
        )

        self._viable_zero_stages = {}
        self._model_profiling_info_results_dict = dict()

        self._all_search_methods = {"random": self._random_search, "basic": self._basic_search}
        self._search_method = self._autotuning_config["search_method"]
        assert (
            self._search_method in self._all_search_methods
        ), f"search_method must be one of {list(self._all_search_methods)}"

    # def _get_memory_required_per_gpu_per_stage(self):
    #     # Modified from DS.
    #     num_params = self.get_model_num_params()
    #     fp16_enabled = self.fp16_enabled()
    #
    #     if not num_params:
    #         return 0
    #     # assume the model uses Adam optimizer
    #     # ZeroStageEnum.disabled:
    #     params_mem = num_params * (2 if fp16_enabled else 4)
    #     gradients_mem = num_params * (2 if fp16_enabled else 4)
    #     optimizer_mem = num_params * (16 if fp16_enabled else 8)
    #
    #     if zero_stage >= ZeroStageEnum.optimizer_states:
    #         optimizer_mem = optimizer_mem / self._slots
    #
    #     if zero_stage >= ZeroStageEnum.gradients:
    #         gradients_mem = gradients_mem / self._slots
    #
    #     if zero_stage >= ZeroStageEnum.weights:
    #         params_mem = params_mem / self._slots
    #
    #     mem_per_gpu = (params_mem + gradients_mem + optimizer_mem) / self.mp_size()
    #
    #     return mem_per_gpu
    #
    # def _get_viable_zero_stages(self) -> Set[int]:
    #     gpu_mem_in_bytes = self._model_profiling_info_results_dict["gpu_mem_in_bytes"]
    #     activation_mem_per_gpu_in_bytes = self._model_profiling_info_results_dict[
    #         "activation_mem_per_gpu"
    #     ]

    def _basic_search(self) -> List[Dict[str, Any]]:
        return 2 * [self._base_hps_with_profiling]

    def _random_search(self) -> List[Dict[str, Any]]:
        pass

    def _generated_hparam_list(self) -> List[Dict[str, Any]]:
        """Generates a list of all hp dict combos which will be tested out."""
        # TODO: Add non-trivial logic.
        hparam_list = self._all_search_methods[self._search_method]()
        return hparam_list

    def _get_and_register_trial(self, hparams: Dict[str, Any]) -> DSATTrial:
        trial = DSATTrial(hparams)
        self._all_trials_dict[trial.request_id] = trial
        return trial

    def _get_and_register_trial_ops_list(
        self, id: uuid.UUID, length: int
    ) -> List[searcher.Operation]:
        trial = self._all_trials_dict[id]
        ops_list = trial.get_ops_list(length=length)
        self._running_trial_ids.add(id)
        return ops_list

    def initial_operations(self, _: searcher.SearcherState) -> List[searcher.Operation]:
        model_profile_info_hps = copy.deepcopy(self._submitted_hps)
        utils.replace_dict_in_place(
            model_profile_info_hps["ds_config"],
            constants.MODEL_INFO_PROFILING_DS_CONFIG,
        )
        model_profile_info_trial = self._get_and_register_trial(hparams=model_profile_info_hps)
        self._model_profile_info_trial_id = model_profile_info_trial.request_id
        # Only a single step is required for the model profiling run.
        return self._get_and_register_trial_ops_list(self._model_profile_info_trial_id, length=1)

    def on_trial_created(
        self, _: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        return []

    def on_validation_completed(
        self,
        _: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
        train_length: int,
    ) -> List[searcher.Operation]:
        print(f"Completed trial {request_id}")
        self._running_trial_ids.remove(request_id)
        ops = [searcher.Close(request_id=request_id)]
        if request_id == self._model_profile_info_trial_id:
            self._model_profiling_info_results_dict = metric
            for hparams in self._generated_hparam_list():
                new_trial = self._get_and_register_trial(hparams=hparams)
                ops.extend(new_trial.get_ops_list(length=constants.DSAT_MAX_LENGTH_STEPS))
        return ops

    def on_trial_closed(
        self, _: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        self._running_trial_ids.remove(request_id)
        print(f"Closing trial {request_id}, {len(self._running_trial_ids)} remaining")
        if not self._running_trial_ids:
            return [searcher.Shutdown()]
        return []

    def on_trial_exited_early(
        self,
        _: searcher.SearcherState,
        request_id: uuid.UUID,
        exited_reason: searcher.ExitedReason,
    ) -> List[searcher.Operation]:
        print("EXITED REASON", exited_reason)
        return []

    def progress(self, _: searcher.SearcherState) -> float:
        return 0
