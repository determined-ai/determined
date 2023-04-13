# type: ignore
import copy
import json
import os
import pathlib
import tempfile
from argparse import Namespace
from typing import Any, Dict, Iterator, List, Optional, Sequence, Union
from unittest import mock

from determined import searcher
import determined.pytorch.deepspeed as det_deepspeed
from deepspeed.runtime import config_utils
from determined.common.api import bindings

from determined import workload
from tests.experiment import utils  # noqa: I100
from tests.experiment.fixtures import deepspeed_linear_model
from tests.custom_search_mocks import SimulateMaster, MockMasterSearchRunner


######

from determined.pytorch.deepspeed.dsat import autotune, _utils, _defaults
from determined.pytorch.deepspeed.dsat import DSATTrialTracker, DSATTrial
from determined.pytorch.deepspeed.dsat._dsat_search_method import (
    SimpleBatchSearchMethod,
    DSATRandomSearchMethod,
)


BASE_EXPERIMENT_FIXTURE_PATH = (
    pathlib.Path(__file__).resolve().parent.parent.joinpath("fixtures/deepspeed_autotune")
)

MODEL_INFO_PROFILE_METRIC_FIXTURE = {
    "num_params": 60192808,
    "trainable_num_params": 60192808,
    "activation_mem_per_gpu": 89828352,
    "rank": 0,
    "gpu_mem": 15843721216,
}


@mock.patch("determined.experimental.client.create_experiment")
def test_autotuning_module_transforms(
    create_experiment_mock: mock.MagicMock,
) -> None:
    model_dir = BASE_EXPERIMENT_FIXTURE_PATH.joinpath("example_experiment")
    config_path = model_dir.joinpath("autotune_config.yaml")
    args = Namespace(
        config_path=config_path,
        model_dir=model_dir,
        tuner_type="random",
        search_runner_config=None,
    )
    autotune.run_autotuning(args)
    create_experiment_mock.assert_called_once()
    submitted_single_searcher_config_dict = create_experiment_mock.call_args_list[0].kwargs[
        "config"
    ]
    # The Search Runner should be defaulted to a "single" searcher
    assert submitted_single_searcher_config_dict.get("searcher", {}).get("name", "") == "single"


def test_simple_search_method_happy_path() -> None:
    model_dir = BASE_EXPERIMENT_FIXTURE_PATH.joinpath("example_experiment")
    config_path = model_dir.joinpath("autotune_config.yaml")
    submitted_config_dict = _utils.get_dict_from_yaml_or_json_path(config_path)

    with tempfile.TemporaryDirectory() as searcher_dir:
        searcher_dir = pathlib.Path(searcher_dir)
        search_method = SimpleBatchSearchMethod(
            submitted_config_dict=submitted_config_dict,
            model_dir=model_dir,
        )
        mock_master_obj = MockMaster(
            all_metrics=[MODEL_INFO_PROFILE_METRIC_FIXTURE, {"throughput": 1.0}]
        )
        search_runner = MockMasterSearchRunner(search_method, mock_master_obj, searcher_dir)
        search_runner.run(exp_config={}, context_dir="", includes=None)

    # TODO: Use a more dynamic value if/when we enable users to configure this
    exp_num_trials = _defaults.AUTOTUNING_DICT["tuner_num_trials"]
    assert len(search_runner.state.trials_created) == exp_num_trials
    assert len(search_runner.state.trials_closed) == exp_num_trials
    assert len(search_runner.state.trial_progress) == exp_num_trials
    # TODO: Handle the progress being 6 every time...
    # for trial_uuid in search_runner.state.trial_progress:
    #     assert(search_runner.state.trial_progress[trial_uuid] == 1.0)


# TODO: Check that the trial tracker stops early when we trigger the early stopping criteria


def test_random_search_method() -> None:
    pass


def test_trial_tracker() -> None:
    pass


def test_mbs_binary_search() -> None:
    pass


class MockMaster:
    def __init__(self, all_metrics: List[Union[float, Dict[str, Any]]]) -> None:
        self.events_queue: List[bindings.v1SearcherEvent] = []
        self.events_count = 0
        self.all_metrics = all_metrics
        self.metric_index = 0
        self.overall_progress = 0.0

    def handle_post_operations(
        self, event: bindings.v1SearcherEvent, operations: List[searcher.Operation]
    ) -> None:
        self._remove_upto(event)
        self._process_operations(operations)

    def _remove_upto(self, event: bindings.v1SearcherEvent) -> None:
        for i, e in enumerate(self.events_queue):
            if e.id == event.id:
                self.events_queue = self.events_queue[i + 1 :]
                return

        raise RuntimeError(f"event not found in events queue: {event}")

    def _process_operations(self, operations: List[searcher.Operation]) -> None:
        for op in operations:
            self._append_events_for_op(op)  # validate_after returns two events.

    def handle_get_events(self) -> Optional[Sequence[bindings.v1SearcherEvent]]:
        return self.events_queue

    def _append_events_for_op(self, op: searcher.Operation) -> None:
        if type(op) == searcher.ValidateAfter:
            index = min(self.metric_index, len(self.all_metrics) - 1)
            metric = self.all_metrics[index]
            validation_completed = bindings.v1ValidationCompleted(
                requestId=str(op.request_id),
                metric=metric,
                validateAfterLength=str(op.length),
            )
            self.metric_index += 1
            self.events_count += 1
            event = bindings.v1SearcherEvent(
                id=self.events_count, validationCompleted=validation_completed
            )
            self.events_queue.append(event)

            trial_progress = bindings.v1TrialProgress(
                requestId=str(op.request_id), partialUnits=float(op.length)
            )
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialProgress=trial_progress)
            self.events_queue.append(event)

        if type(op) == searcher.Create:
            trial_created = bindings.v1TrialCreated(requestId=str(op.request_id))
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialCreated=trial_created)
            self.events_queue.append(event)

        if type(op) == searcher.Progress:  # no events
            self.overall_progress

        if type(op) == searcher.Close:
            trial_closed = bindings.v1TrialClosed(requestId=str(op.request_id))
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialClosed=trial_closed)
            self.events_queue.append(event)

        if type(op) == searcher.Shutdown:
            exp_state = bindings.experimentv1State.STATE_COMPLETED
            exp_inactive = bindings.v1ExperimentInactive(experimentState=exp_state)
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, experimentInactive=exp_inactive)
            self.events_queue.append(event)

        # Exited Early??


# def test_overwrite_deepspeed_config() -> None:
#     base_ds_config = deepspeed_config
#     source_ds_config = {
#         "train_micro_batch_size_per_gpu": 2,
#         "optimizer": {"params": {"lr": 0.001}},
#     }
#     expected_config = copy.deepcopy(deepspeed_config)
#     expected_config["train_micro_batch_size_per_gpu"] = 2
#     expected_config["optimizer"]["params"]["lr"] = 0.001
#     result = det_deepspeed.overwrite_deepspeed_config(base_ds_config, source_ds_config)
#     assert result == expected_config

#     # Test load base deepspeed config from json file.
#     base_ds_config = str(
#         pathlib.Path(__file__).resolve().parent.parent.joinpath("fixtures/ds_config.json")
#     )
#     result = det_deepspeed.overwrite_deepspeed_config(base_ds_config, source_ds_config)
#     assert result == expected_config

#     # Test fail invalid base_ds_config argument.
#     with pytest.raises(TypeError, match="Expected string or dict for base_ds_config argument."):
#         _ = det_deepspeed.overwrite_deepspeed_config([1, 2], source_ds_config)
