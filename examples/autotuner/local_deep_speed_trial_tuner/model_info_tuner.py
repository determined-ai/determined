import copy
import json
import logging
import pathlib
import shutil
import tempfile
import uuid
from typing import Any, List

import determined as det
from determined import searcher
from determined.common import util


class ModelInfoTuner(searcher.SearchMethod):
    def __init__(self, experiment_config: dict, max_length: int, model_dir: pathlib.Path) -> None:
        # since this is a single trial the hyperparameter space comprises a single point
        self.hyperparameters = experiment_config["hyperparameters"]
        self.max_length = max_length
        self.trial_closed = False
        self.model_info_config_path = self._create_model_info_ds_config(model_dir)
        self.model_info_profile_request_id = uuid.uuid4()

    def _create_model_info_ds_config(self, model_dir: pathlib.Path) -> str:
        # see https://github.com/microsoft/DeepSpeed/blob/c5f85858a87b9df811e7accb5f6e101a1c9f2f46/deepspeed/autotuning/autotuner.py#L695
        # to avoid stealing their implementation of replace_dict
        ds_config_file = self.hyperparameters["deepspeed_config"]
        model_dir.joinpath("profile_model_info").mkdir()
        with model_dir.joinpath(ds_config_file).open("r") as f1:
            ds_config = json.load(f1)
        ds_config["train_micro_batch_size_per_gpu"] = 1
        if "zero_optimization" not in ds_config:
            ds_config["zero_optimization"] = {"stage": 3}
        else:
            ds_config["zero_optimization"]["stage"] = 3
        ds_config["memory_break_down"] = False
        ds_config["autotuning"] = {
            "enabled": True,
            "model_info_path": "profile_model_info/model_info.json",
            "model_info": {"profile": True},
        }

        model_info_filename = "model_info_ds_config.json"
        model_info_config_path = model_dir.joinpath(model_info_filename)
        with model_info_config_path.open("w") as f2:
            json.dump(ds_config, f2)
        return model_info_filename

    def on_trial_created(
        self, _: searcher.SearcherState, __: uuid.UUID
    ) -> List[searcher.Operation]:
        return []

    def on_validation_completed(
        self,
        _: searcher.SearcherState,
        request_id: uuid.UUID,
        metric: Any,
        train_length: int,
    ) -> List[searcher.Operation]:
        logging.info(f"validation completed; metric={metric}, train_length={train_length}")
        return []

    # for passing HPs
    # allow the search method to do things to the experiment directory

    # liam's idea: ask the latest validation metrics of a trial using the trial id
    # maksim's idea: alter the searcher api to return trial results
    # - one step closer to multi-objective search

    # def on_trial_closed(
    #     self, _: searcher.SearcherState, request_id: uuid.UUID, trial_completion_info: Optional[dict[str, Any]] = None
    # ) -> List[searcher.Operation]:
    def on_trial_closed(self, _: searcher.SearcherState, request_id: uuid.UUID):
        if request_id == self.model_info_profile_request_id:
            logging.info("model info profile trial closed")

        logging.info("trial closed")
        self.trial_closed = True
        return [searcher.Shutdown()]

    def progress(self, searcher_state: searcher.SearcherState) -> float:
        if self.trial_closed:
            return 1.0
        (the_trial,) = searcher_state.trials_created
        return searcher_state.trial_progress[the_trial] / self.max_length

    def on_trial_exited_early(
        self, _: searcher.SearcherState, request_id: uuid.UUID, exited_reason: searcher.ExitedReason
    ) -> List[searcher.Operation]:
        logging.warning(f"Trial {request_id} exited early: {exited_reason}")
        return [searcher.Shutdown()]

    def initial_operations(self, _: searcher.SearcherState) -> List[searcher.Operation]:
        logging.info("initial_operations")

        model_info_hyperparameters = copy.deepcopy(self.hyperparameters)
        model_info_hyperparameters["deepspeed_config"] = self.model_info_config_path
        model_info_hyperparameters["deepspeed_mode"] = "model_info_profiling"
        create = searcher.Create(
            request_id=self.model_info_profile_request_id,
            hparams=model_info_hyperparameters,
            checkpoint=None,
        )
        validate_after = searcher.ValidateAfter(
            request_id=create.request_id, length=1  # self.max_length
        )
        close = searcher.Close(request_id=create.request_id)
        logging.debug(f"Create({create.request_id}, {create.hparams})")
        return [create, validate_after, close]


def main(exp_dir: str, exp_conf: str) -> None:
    exp_dir_path = pathlib.Path(exp_dir)

    with exp_dir_path.joinpath(exp_conf).open("r") as f:
        config = util.safe_load_yaml_with_exceptions(f)
    config["searcher"] = {
        "name": "custom",
        "metric": "accuracy",
        "smaller_is_better": False,
        "unit": "batches",
    }
    search_method = ModelInfoTuner(config, max_length=6, model_dir=exp_dir_path)

    searcher_dir = (
        pathlib.Path(exp_dir).joinpath("local_deep_speed_trial_tuner").joinpath("searcher_dir")
    )
    searcher_dir.mkdir(parents=True)
    search_runner = searcher.LocalSearchRunner(search_method, searcher_dir=searcher_dir)
    experiment_id = search_runner.run(config, model_dir=exp_dir_path)
    logging.info(f"Experiment {experiment_id} has been completed")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    cifar10_moe_path = (
        pathlib.Path(__file__)
        .absolute()
        .parent.parent.parent.joinpath("deepspeed")
        .joinpath("cifar10_moe")
    )
    print(cifar10_moe_path)

    with tempfile.TemporaryDirectory() as exp_dir:
        shutil.copytree(cifar10_moe_path, exp_dir, dirs_exist_ok=True)
        main(exp_dir, "zero_stages.yaml")
