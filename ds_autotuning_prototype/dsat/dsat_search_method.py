import copy
import logging
import uuid
from abc import abstractmethod
from typing import Any, Dict, Generator, List, Optional, Set, Tuple, Union

from dsat import constants, utils
from torch import optim

import determined as det
from determined import searcher


class DSATTrial:
    def __init__(self, hparams: Dict[str, Any], checkpoint: Optional[str] = None) -> None:
        self.hparams = hparams
        self.ds_config = self.hparams["ds_config"]
        self.checkpoint = checkpoint
        self.request_id = uuid.uuid4()
        self.metric = None

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


class DSATSearchMethodBase(searcher.SearchMethod):
    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
    ) -> None:
        self._submitted_config_dict = submitted_config_dict
        self._submitted_hps = self._submitted_config_dict["hyperparameters"]
        self._ds_config = self._submitted_hps["ds_config"]
        self._autotuning_config = self._submitted_hps["autotuning_config"]

        self._all_trials_dict = dict()
        self._model_profile_info_trial_id = None

        self._base_hparams_with_profiling = copy.deepcopy(self._submitted_hps)
        utils.replace_dict_in_place(
            self._base_hparams_with_profiling["ds_config"],
            {"flops_profiler": constants.FLOPS_PROFILER_CONFIG},  # TODO: don't hardcode profiler.
        )

        # Non-trivial values instantiated after model profiling run
        self._viable_zero_stages = set()
        self._model_profiling_info_results_dict = dict()

    def _get_viable_zero_stages(self) -> Set[int]:
        """
        Returns the set of viable zero stages based on a rough computation.
        """
        gpu_mem_in_bytes = self._model_profiling_info_results_dict["gpu_mem_in_bytes"]
        activation_mem_per_gpu_in_bytes = self._model_profiling_info_results_dict[
            "activation_mem_per_gpu"
        ]
        num_params = self._model_profiling_info_results_dict["num_params"]
        trainable_num_params = self._model_profiling_info_results_dict["trainable_num_params"]
        try:
            fp16 = self._ds_config["fp16"]["enabled"]
        except KeyError:
            fp16 = False
        params_mem = num_params * (2 if fp16 else 4)
        gradients_mem = trainable_num_params * (2 if fp16 else 4)
        # optimizer_mem assumes Adam like DS. TODO: don't assume this.
        optimizer_mem = trainable_num_params * (16 if fp16 else 8)

        slots = self._submitted_config_dict["resources"]["slots_per_trial"]
        non_activation_mem_per_gpu_per_stage = {
            0: params_mem + gradients_mem + optimizer_mem,
            1: params_mem + gradients_mem + optimizer_mem // slots,
            2: params_mem + (gradients_mem + optimizer_mem) // slots,
            3: (params_mem + gradients_mem + optimizer_mem) // slots,
        }
        mem_per_gpu_per_stage = {
            stage: mem + activation_mem_per_gpu_in_bytes
            for stage, mem in non_activation_mem_per_gpu_per_stage.items()
        }
        # TODO: account for model parallelism.
        viable_stages = {
            stage for stage, mem in mem_per_gpu_per_stage.items() if mem < gpu_mem_in_bytes
        }
        return viable_stages

    @abstractmethod
    def hparam_generator(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
    ) -> Generator[Dict[str, Any], None, None]:
        """Generates a all hp dicts to be run in this search in resonse to the request_id of the
        last completed trial, its reported metric, the searcher state."""
        pass

    def _get_and_register_trial(self, hparams: Dict[str, Any]) -> DSATTrial:
        trial = DSATTrial(hparams)
        self._all_trials_dict[trial.request_id] = trial
        return trial

    def _get_and_register_trial_ops_list(
        self, id: uuid.UUID, length: int
    ) -> List[searcher.Operation]:
        trial = self._all_trials_dict[id]
        ops_list = trial.get_ops_list(length=length)
        return ops_list

    def _process_model_profiling_info_run(self, metric: Dict[str, Any]):
        self._model_profiling_info_results_dict = metric
        self._viable_zero_stages = self._get_viable_zero_stages()
        logging.info(f"Viable zero stages: {self._viable_zero_stages}")

    def initial_operations(
        self, searcher_state: searcher.SearcherState
    ) -> List[searcher.Operation]:
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
        if request_id == self._model_profile_info_trial_id:
            self._process_model_profiling_info_run(metric)
        # All DS AT Trials should be closed upon completion.
        ops = [searcher.Close(request_id=request_id)]
        for hparams in self.hparam_generator(
            searcher_state=searcher_state, request_id=request_id, metric=metric
        ):
            new_trial = self._get_and_register_trial(hparams=hparams)
            # TODO: get the length from the config
            new_ops = self._get_and_register_trial_ops_list(
                id=new_trial.request_id, length=constants.DSAT_MAX_LENGTH_STEPS
            )
            ops.extend(new_ops)
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
        if request_id == self._model_profile_info_trial_id:
            logging.info(f"model profiling run failed due to {exited_reason}, shutting down")
            return [searcher.Shutdown()]
        logging.info("EXITED REASON", exited_reason)
        return []

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        return 0


class DSATBasicSearchMethod(DSATSearchMethodBase):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        # Probably move this to the main class eventually.
        zero_stages = self._autotuning_config["zero_stages"]
        # Make zero_stages iterable. Should probably have some asserts.
        if zero_stages == "all":
            zero_stages == range(4)
        if isinstance(zero_stages, int):
            zero_stages == (zero_stages,)

        num_trials = self._autotuning_config["num_trials"]
        if isinstance(num_trials, int):
            num_trials == (num_trials,)
        assert len(zero_stages) == len(
            num_trials
        ), "num_trials incompatible with zero_stages config"
        self._remaining_trials_per_stage = {
            stage: num for stage, num in zip(zero_stages, num_trials)
        }

    def hparam_generator(
        self,
        searcher_state: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
    ) -> List[Dict[str, Any]]:
        if request_id == self._model_profile_info_trial_id:
            if not metric:
                raise ValueError("No metric reported for info profiling run; failed?")
            for stage, num in self._remaining_trials_per_stage.items():
                for _ in range(num):
                    hparams = copy.deepcopy(self._base_hparams_with_profiling)
                    zero_optim_config = utils.get_random_zero_optim_dict_for_zero_stage(stage)
                    utils.replace_dict_in_place(
                        hparams["ds_config"], {"zero_optimization": zero_optim_config}
                    )
                    del hparams["autotuning_config"]  # Ugly cleanup for better Web UI visuals.
                    logging.info(hparams)
                    yield hparams
