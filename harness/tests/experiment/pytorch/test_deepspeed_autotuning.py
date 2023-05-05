import pathlib
import tempfile
from argparse import Namespace
from collections import deque
from typing import Any, Deque, Dict, List, Optional, Sequence, Union
from unittest import mock

import pytest

from determined import searcher
from determined.common.api import bindings
from determined.pytorch.dsat import __main__, _defaults, _utils
from determined.pytorch.dsat._dsat_search_method import BaseDSATSearchMethod
from determined.pytorch.dsat._run_dsat import build_exp_conf_from_args
from tests.custom_search_mocks import MockMasterSearchRunner

ERROR_METRIC_NAME = "error"

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


def _run_searcher(search_method: BaseDSATSearchMethod, all_metrics):
    """
    Run a mocked version of the Determined master with a deterministic series of
    returned metrics for a given Deepspeed Autotune Custom Search Method
    """
    model_dir = BASE_EXPERIMENT_FIXTURE_PATH.joinpath("example_experiment")
    config_path = model_dir.joinpath("autotune_config.yaml")
    with tempfile.TemporaryDirectory() as searcher_dir:
        args = Namespace(
            model_dir=model_dir,
            config_path=config_path,
            # DEFAULTS
            tuner_type=_defaults.AUTOTUNING_ARG_DEFAULTS["tuner-type"],
            max_trials=_defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"],
            max_concurrent_trials=_defaults.AUTOTUNING_ARG_DEFAULTS["max-concurrent-trials"],
            zero_stages=_defaults.AUTOTUNING_ARG_DEFAULTS["zero-stages"],
            trials_per_random_config=_defaults.AUTOTUNING_ARG_DEFAULTS["trials-per-random-config"],
            start_profile_step=_defaults.AUTOTUNING_ARG_DEFAULTS["start-profile-step"],
            end_profile_step=_defaults.AUTOTUNING_ARG_DEFAULTS["end-profile-step"],
            metric=_defaults.AUTOTUNING_ARG_DEFAULTS["metric"],
            random_seed=_defaults.AUTOTUNING_ARG_DEFAULTS["random-seed"],
            # NONE TYPES
            max_slots=None,
            early_stopping=None,
            experiment_id=None,
        )

        config_dict = build_exp_conf_from_args(args)
        searcher_dir = pathlib.Path(searcher_dir)
        search_method = search_method(args=args, exp_config=config_dict)
        mock_master_obj = MockMaster(all_metrics=all_metrics)
        search_runner = MockMasterSearchRunner(search_method, mock_master_obj, searcher_dir)
        search_runner.run(exp_config={}, context_dir="", includes=None)
    return search_runner


@pytest.mark.timeout(5)
def test_deepspeed_autotune_happy_path() -> None:
    """
    Simulate the Deepspeed Autotune Search Methods end to end and make sure
    nothing falls over
    """
    for search_method in _defaults.ALL_SEARCH_METHOD_CLASSES.values():
        search_runner = _run_searcher(
            search_method, [MODEL_INFO_PROFILE_METRIC_FIXTURE, {"throughput": 1.0}]
        )
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        assert len(search_runner.state.trials_created) == exp_num_trials
        assert len(search_runner.state.trials_closed) == exp_num_trials
        assert len(search_runner.state.trial_progress) == exp_num_trials
        for trial_uuid in search_runner.state.trial_progress:
            assert search_runner.state.trial_progress[trial_uuid] == 1.0
        assert search_runner.state.experiment_failed == False
        assert search_runner.state.experiment_completed == True


@pytest.mark.timeout(5)
def test_continuous_failures() -> None:
    """
    Make sure that DSAT Search Methods can handle continuous failures.
    Note that the `ERROR_METRIC_NAME` triggered `v1TrialExitedEarly` event
    will happen for all trials after the first model profile info and single
    successful run
    """
    for search_method in _defaults.ALL_SEARCH_METHOD_CLASSES.values():
        all_mock_metrics = [
            MODEL_INFO_PROFILE_METRIC_FIXTURE,
            {"throughput": 1.0},
            {ERROR_METRIC_NAME: 1.0},
        ]
        search_runner = _run_searcher(search_method, all_mock_metrics)
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        assert len(search_runner.state.trials_created) == exp_num_trials
        assert len(search_runner.state.failures) == exp_num_trials - 2
        assert len(search_runner.state.trials_closed) == exp_num_trials
        assert len(search_runner.state.trial_progress) == exp_num_trials
        assert search_runner.state.experiment_failed == False
        assert search_runner.state.experiment_completed == True


@pytest.mark.timeout(5)
def test_one_off_failure() -> None:
    """Make sure that DSAT Search Methods can properly handle a single failure"""
    for search_method in _defaults.ALL_SEARCH_METHOD_CLASSES.values():
        all_mock_metrics = [
            MODEL_INFO_PROFILE_METRIC_FIXTURE,
            {ERROR_METRIC_NAME: 1.0},
            {"throughput": 1.0},
        ]
        search_runner = _run_searcher(search_method, all_mock_metrics)
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        assert len(search_runner.state.trials_created) == exp_num_trials
        assert len(search_runner.state.failures) == 1
        assert len(search_runner.state.trials_closed) == exp_num_trials
        assert len(search_runner.state.trial_progress) == exp_num_trials
        assert search_runner.state.experiment_failed == False
        assert search_runner.state.experiment_completed == True


@pytest.mark.timeout(5)
def test_simple_model_profile_info_run_fails() -> None:
    """Run the random search method where the model profile info run fails"""
    for search_method in _defaults.ALL_SEARCH_METHOD_CLASSES.values():
        all_mock_metrics = [
            {ERROR_METRIC_NAME: 1.0},
        ]
        search_runner = _run_searcher(
            search_method,
            all_mock_metrics,
        )
        assert len(search_runner.state.trials_created) == 1
        assert len(search_runner.state.failures) == 1
        assert len(search_runner.state.trials_closed) == 1
        assert len(search_runner.state.trial_progress) == 1
        assert search_runner.state.experiment_failed == True
        assert search_runner.state.experiment_completed == False


class MockMaster:
    """
    Sends v1 metrics back to the Search Runner in the manner defined with the
    `all_metrics` list of dictionaries.

    The metrics are sent as a `v1ValidationCompleted` metric event. When the key for
    the metric is instead `ERROR_METRIC_NAME`, this signals to the `MockMaster` to
    instead send a `v1TrialExitedEarly` event to the Search Runner.

    The last element of the `all_metrics` list will be repeated until the Search Runner
    quits.
    """

    def __init__(self, all_metrics: List[Union[float, Dict[str, Any]]]) -> None:
        self.events_queue: Deque[bindings.v1SearcherEvent] = deque([])
        self.events_count = 0
        self.all_metrics = all_metrics
        self.metric_index = 0

    def handle_post_operations(
        self, event: bindings.v1SearcherEvent, operations: List[searcher.Operation]
    ) -> None:
        self._remove_upto(event)
        self._process_operations(operations)

    def _remove_upto(self, event: bindings.v1SearcherEvent) -> None:
        while len(self.events_queue) > 0:
            e = self.events_queue.popleft()
            if e.id == event.id:
                return

        raise RuntimeError(f"event not found in events queue: {event}")

    def _process_operations(self, operations: List[searcher.Operation]) -> None:
        for op in operations:
            self._append_events_for_op(op)  # validate_after returns two events.

    def handle_get_events(self) -> Optional[Sequence[bindings.v1SearcherEvent]]:
        return list(self.events_queue)

    def _append_events_for_op(self, op: searcher.Operation) -> None:
        if isinstance(op, searcher.ValidateAfter):
            index = min(self.metric_index, len(self.all_metrics) - 1)
            metric = self.all_metrics[index]
            self.metric_index += 1
            if ERROR_METRIC_NAME in metric:
                trial_exited_early = bindings.v1TrialExitedEarly(
                    requestId=str(op.request_id),
                    exitedReason=bindings.v1TrialExitedEarlyExitedReason.UNSPECIFIED,
                )
                self.events_count += 1
                event = bindings.v1SearcherEvent(
                    id=self.events_count, trialExitedEarly=trial_exited_early
                )
                self.events_queue.append(event)

                trial_closed = bindings.v1TrialClosed(requestId=str(op.request_id))
                self.events_count += 1
                event = bindings.v1SearcherEvent(id=self.events_count, trialClosed=trial_closed)
                self.events_queue.append(event)
            else:
                validation_completed = bindings.v1ValidationCompleted(
                    requestId=str(op.request_id),
                    metric=metric,
                    validateAfterLength=str(op.length),
                )

                self.events_count += 1
                event = bindings.v1SearcherEvent(
                    id=self.events_count, validationCompleted=validation_completed
                )
                self.events_queue.append(event)

                # Send 1.0 to signal it was completed
                trial_progress = bindings.v1TrialProgress(
                    requestId=str(op.request_id), partialUnits=1.0
                )
                self.events_count += 1
                event = bindings.v1SearcherEvent(id=self.events_count, trialProgress=trial_progress)
                self.events_queue.append(event)

        elif isinstance(op, searcher.Create):
            trial_created = bindings.v1TrialCreated(requestId=str(op.request_id))
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialCreated=trial_created)
            self.events_queue.append(event)

        elif isinstance(op, searcher.Progress):  # no events
            pass

        elif isinstance(op, searcher.Close):
            trial_closed = bindings.v1TrialClosed(requestId=str(op.request_id))
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, trialClosed=trial_closed)
            self.events_queue.append(event)

        elif isinstance(op, searcher.Shutdown):
            exp_state = (
                bindings.experimentv1State.ERROR
                if op.failure
                else bindings.experimentv1State.COMPLETED
            )
            exp_inactive = bindings.v1ExperimentInactive(experimentState=exp_state)
            self.events_count += 1
            event = bindings.v1SearcherEvent(id=self.events_count, experimentInactive=exp_inactive)
            self.events_queue.append(event)


class TestDSATTrial:
    def setup_class(self):
        self.first_trial = DSATTrial(**DSATTRIAL_ARGS)

    def test_lineage_methods(self):
        # Create a lineage of ten trials.
        trials = [self.first_trial]
        for _ in range(10):
            trials.append(DSATTrial(parent=trials[-1], **DSATTRIAL_ARGS))

        for idx, trial in enumerate(trials):
            if idx == 0:
                assert trial.parent is None
            else:
                assert trial.parent == trials[idx - 1]
            if idx != len(trials) - 1:
                assert trial.children == set((trials[idx + 1],))
            else:
                assert trial.children == set()
            assert trial.lineage_root == self.first_trial
            assert trial.lineage_set == set(trials)
            assert trial.num_trials_in_lineage == len(trials)

    def test_error_checking(self):
        initial_successful_trials = [self.first_trial]
        for _ in range(10):
            initial_successful_trials.append(
                DSATTrial(parent=initial_successful_trials[-1], **DSATTRIAL_ARGS)
            )

        errored_trial = DSATTrial(parent=initial_successful_trials[-1], **DSATTRIAL_ARGS)
        errored_trial.error = True
        alternating_errored_trials = [errored_trial]
        for _ in range(10):
            last_trial = alternating_errored_trials[-1]
            next_trial = DSATTrial(parent=last_trial, **DSATTRIAL_ARGS)
            if not last_trial.error:
                next_trial.error
            alternating_errored_trials.append(next_trial)

        all_trials = initial_successful_trials + alternating_errored_trials

        seen_errored = False
        for trial in all_trials:
            if trial.error:
                seen_errored = True
            if not seen_errored:
                assert not trial.error_in_direct_history
            else:
                assert trial.error_in_direct_history
