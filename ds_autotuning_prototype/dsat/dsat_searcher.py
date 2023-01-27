import abc
import argparse
import copy
import logging
import uuid
from typing import Any, Dict, List, Union

import determined as det
from determined import searcher
from dsat import constants, utils


class DSATSearchMethod(searcher.SearchMethod):
    def __init__(
        self,
        submitted_config_dict: Dict[str, Any],
    ) -> None:
        self.submitted_hps = submitted_config_dict["hyperparameters"]
        self.autotuning_config = submitted_config_dict["hyperparameters"]["autotuning_config"]
        self.running_trials = 0

        self._model_profile_request_id = None
        self._all_search_methods = {"random": self._random_search, "basic": self._basic_search}
        self._search_method = self.autotuning_config["search_method"]
        assert (
            self._search_method in self._all_search_methods
        ), f"search_method must be one of {list(self._all_search_methods)}"

    def _basic_search(
        self, base_hps_with_profiling: Dict[str, Any], model_profile_metrics: Dict[str, Any]
    ) -> List[Dict[str, Any]]:
        return 2 * [base_hps_with_profiling]

    def _random_search(
        self, base_hps_with_profiling: Dict[str, Any], model_profile_metrics: Dict[str, Any]
    ) -> List[Dict[str, Any]]:
        pass

    def _generated_hparam_list(
        self, model_profile_metrics: Union[float, Dict[str, Any]]
    ) -> List[Dict[str, Any]]:
        """Generates a list of all hp dict combos which will be tested out."""
        # TODO: Add non-trivial logic.
        base_hps_with_profiling = copy.deepcopy(self.submitted_hps)
        utils.replace_dict_in_place(
            base_hps_with_profiling["ds_config"],
            {"flops_profiler": constants.FLOPS_PROFILER_CONFIG},
        )
        hparam_list = self._all_search_methods[self._search_method](
            base_hps_with_profiling, model_profile_metrics
        )
        return hparam_list

    def initial_operations(self, _: searcher.SearcherState) -> List[searcher.Operation]:
        model_profile_run_hps = copy.deepcopy(self.submitted_hps)
        utils.replace_dict_in_place(
            model_profile_run_hps["ds_config"],
            constants.MODEL_INFO_PROFILING_DS_CONFIG,
        )
        self._model_profile_request_id = uuid.uuid4()
        create = searcher.Create(
            request_id=self._model_profile_request_id,
            hparams=model_profile_run_hps,
            checkpoint=None,
        )
        # The model info profiling run only performs a single step.
        run = searcher.ValidateAfter(request_id=create.request_id, length=1)
        return [create, run]

    def on_trial_created(
        self, _: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        self.running_trials += 1
        print(f"Creating trial {request_id}, {self.running_trials} remaining")
        return []

    def on_validation_completed(
        self,
        _: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Union[float, Dict[str, Any]],
        train_length: int,
    ) -> List[searcher.Operation]:
        print(f"Completed trial {request_id}")
        operations = [searcher.Close(request_id=request_id)]
        # Could refactor and put the model profiling run here, if desireable.
        print("REPORTED METRICS", metric)
        if request_id == self._model_profile_request_id:
            model_profiling_info_results_dict = metric
            for hp_dict in self._generated_hparam_list(model_profiling_info_results_dict):
                print("GENERATED HPS", hp_dict)
                create = searcher.Create(
                    request_id=uuid.uuid4(),
                    hparams=hp_dict,
                    checkpoint=None,
                )
                run = searcher.ValidateAfter(
                    request_id=create.request_id, length=constants.DSAT_MAX_LENGTH_STEPS
                )
                operations.append(create)
                operations.append(run)

        return operations

    def on_trial_closed(
        self, _: searcher.SearcherState, request_id: uuid.UUID
    ) -> List[searcher.Operation]:
        self.running_trials -= 1
        print(f"Closing trial {request_id}, {self.running_trials} remaining")
        if not self.running_trials:
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


def get_parsed_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("-c", "--config_path", type=str, default="")
    parsed_args = parser.parse_args()

    return parsed_args


def main(core_context: det.core.Context) -> None:
    args = get_parsed_args()
    submitted_config_dict = utils.get_config_dict_from_yaml_path(args.config_path)
    # Save profiling results w/ wrapper; probably remove eventually, but useful for sanity checking.
    # Needs error handling if we keep this; currently reports success even if the Trial fails.
    submitted_config_dict["entrypoint"] += (
        "; python3 -m determined.launch.torch_distributed"
        " python3 -m dsat.checkpoint_profiling_results_wrapper"
    )
    search_method = DSATSearchMethod(submitted_config_dict)
    search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)

    search_runner.run(submitted_config_dict, model_dir=".")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    with det.core.init() as core_context:
        main(core_context)
