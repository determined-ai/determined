import copy
import pathlib
import tempfile
from collections import deque
from typing import Any, Deque, Dict, List, Optional, Sequence, Union

import pytest

from determined import searcher
from determined.common.api import bindings
from determined.pytorch.dsat import (
    BaseDSATSearchMethod,
    DSATTrial,
    DSATTrialTracker,
    _defaults,
    _dsat_search_method,
    _utils,
)
from determined.pytorch.dsat._run_dsat import get_custom_dsat_exp_conf_from_args
from determined.searcher import _search_method
from tests.custom_search_mocks import MockMasterSearchRunner

ERROR_METRIC_NAME = "error"

BASE_EXPERIMENT_FIXTURE_PATH = (
    pathlib.Path(__file__).resolve().parent.parent.joinpath("fixtures/deepspeed_autotune")
)
MODEL_DIR = BASE_EXPERIMENT_FIXTURE_PATH.joinpath("example_experiment")
CONFIG_PATH = MODEL_DIR.joinpath("deepspeed.yaml")
DEFAULT_ARGS = _utils.get_parser().parse_args([str(CONFIG_PATH), str(MODEL_DIR)])
DEFAULT_ARGS.experiment_id = 0
DEFAULT_SEARCH_RUNNER_CONFIG = _utils.get_search_runner_config_from_args(DEFAULT_ARGS)
DEFAULT_CUSTOM_DSAT_EXP_CONFIG = get_custom_dsat_exp_conf_from_args(DEFAULT_ARGS)

MODEL_INFO_PROFILE_METRIC_FIXTURE = {
    "num_params": 60192808,
    "trainable_num_params": 60192808,
    "activation_mem_per_gpu": 89828352,
    "rank": 0,
    "gpu_mem": 15843721216,
}


DSATTRIAL_ARGS = {
    "hparams": {"deepspeed_config": "ds_config.json"},
    "model_dir": BASE_EXPERIMENT_FIXTURE_PATH.joinpath("example_experiment"),
    "slots_per_trial": 2,
    "length": 5,
}

HPARAMS_FIXTURE = {
    "deepspeed_config": "ds_config.json",
    _defaults.OVERWRITE_KEY: {"train_micro_batch_size_per_gpu": 1},
}


def _run_searcher(search_method: BaseDSATSearchMethod, all_metrics):
    """
    Run a mocked version of the Determined master with a deterministic series of
    returned metrics for a given Deepspeed Autotune Custom Search Method
    """
    with tempfile.TemporaryDirectory() as searcher_dir:
        searcher_dir = pathlib.Path(searcher_dir)
        search_method = search_method(args=DEFAULT_ARGS, exp_config=DEFAULT_CUSTOM_DSAT_EXP_CONFIG)
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
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        model_info_profile_trial_metrics = [MODEL_INFO_PROFILE_METRIC_FIXTURE]
        successful_trial_metrics = [
            {_defaults.AUTOTUNING_ARG_DEFAULTS["metric"]: 0.0} for _ in range(exp_num_trials - 1)
        ]
        all_metrics = model_info_profile_trial_metrics + successful_trial_metrics
        search_runner = _run_searcher(search_method, all_metrics)
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
    Make sure that DSAT Search Methods can handle continuous failures. The experiment should be
    marked as failed.
    """
    for search_method in _defaults.ALL_SEARCH_METHOD_CLASSES.values():
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        model_info_profile_trial_metrics = [MODEL_INFO_PROFILE_METRIC_FIXTURE]
        failed_trial_metrics = [{ERROR_METRIC_NAME: True} for _ in range(exp_num_trials - 1)]
        all_metrics = model_info_profile_trial_metrics + failed_trial_metrics
        search_runner = _run_searcher(search_method, all_metrics)

        assert len(search_runner.state.trials_created) == exp_num_trials
        assert len(search_runner.state.failures) == exp_num_trials - 1
        assert len(search_runner.state.trials_closed) == exp_num_trials
        assert len(search_runner.state.trial_progress) == exp_num_trials
        assert search_runner.state.experiment_failed == True
        assert search_runner.state.experiment_completed == False


@pytest.mark.timeout(5)
def test_one_off_failure() -> None:
    """Make sure that DSAT Search Methods can properly handle a single failure"""
    for search_method in _defaults.ALL_SEARCH_METHOD_CLASSES.values():
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        model_info_profile_trial_metrics = [MODEL_INFO_PROFILE_METRIC_FIXTURE]
        one_failed_trial_metrics = [{ERROR_METRIC_NAME: True}]
        successful_trial_metrics = [
            {_defaults.AUTOTUNING_ARG_DEFAULTS["metric"]: 0.0} for _ in range(exp_num_trials - 2)
        ]
        all_metrics = (
            model_info_profile_trial_metrics + one_failed_trial_metrics + successful_trial_metrics
        )
        search_runner = _run_searcher(search_method, all_metrics)

        assert len(search_runner.state.trials_created) == exp_num_trials
        assert len(search_runner.state.failures) == 1
        assert len(search_runner.state.trials_closed) == exp_num_trials
        assert len(search_runner.state.trial_progress) == exp_num_trials
        assert search_runner.state.experiment_failed == False
        assert search_runner.state.experiment_completed == True


@pytest.mark.timeout(5)
def test_model_profile_info_run_failure() -> None:
    """Test DSAT with a failed model profile info run."""
    for search_method in _defaults.ALL_SEARCH_METHOD_CLASSES.values():
        failed_model_profile_info_trial_metrics = [
            {ERROR_METRIC_NAME: True},
        ]
        search_runner = _run_searcher(
            search_method,
            failed_model_profile_info_trial_metrics,
        )
        assert len(search_runner.state.trials_created) == 1
        assert len(search_runner.state.failures) == 1
        assert len(search_runner.state.trials_closed) == 1
        assert len(search_runner.state.trial_progress) == 1
        assert search_runner.state.experiment_failed == True
        assert search_runner.state.experiment_completed == False


@pytest.mark.timeout(5)
class TestDSATTrial:
    def setup_class(self):
        self.first_trial = DSATTrial(**DSATTRIAL_ARGS)

    def test_lineage_methods(self):
        """
        Testing expected behavior of lineage properties.
        """
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
            assert trial.num_completed_trials_in_lineage == idx
            trial.closed = True
        assert trial.num_completed_trials_in_lineage == len(trials)

    def test_error_history(self):
        """
        Testing error history.
        """
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


def trial_tracker_builder(args):
    """
    Completes the model profile into trial and load up a queue of max_trials Trials.
    """
    exp_config = get_custom_dsat_exp_conf_from_args(args)
    trial_tracker = DSATTrialTracker(args=args, exp_config=exp_config)
    model_profile_info_trial = trial_tracker.create_model_profile_info_trial()
    trial_tracker.queue_and_register_trial(model_profile_info_trial)
    trial_tracker.update_trial_metric(
        trial_tracker.queue.popleft(), MODEL_INFO_PROFILE_METRIC_FIXTURE
    )

    queued_trials = []
    for idx in range(trial_tracker.max_trials - 1):
        overwrites = {_defaults.OVERWRITE_KEY: {"zero_optimization": {"stage": 1 + (idx % 3)}}}
        hparams = {**HPARAMS_FIXTURE, **overwrites}
        trial = trial_tracker.create_trial(hparams)
        queued_trials.append(trial)
        trial_tracker.queue_and_register_trial(trial)
    return queued_trials, trial_tracker


@pytest.fixture
def basic_trial_tracker():
    yield trial_tracker_builder(DEFAULT_ARGS)


@pytest.fixture
def max_concurrent_trials_tracker():
    args = copy.deepcopy(DEFAULT_ARGS)
    args.max_concurrent_trials = 2
    yield trial_tracker_builder(args)


@pytest.fixture
def max_slots_tracker():
    args = copy.deepcopy(DEFAULT_ARGS)
    args.max_slots = 4
    yield trial_tracker_builder(args)


@pytest.fixture
def failed_model_profile_info_trial_tracker():
    exp_config = DEFAULT_CUSTOM_DSAT_EXP_CONFIG
    trial_tracker = DSATTrialTracker(args=DEFAULT_ARGS, exp_config=exp_config)
    model_profile_info_trial = trial_tracker.create_model_profile_info_trial()
    trial_tracker.queue_and_register_trial(model_profile_info_trial)
    trial_tracker.report_trial_early_exit(trial_tracker.model_profile_info_trial)
    yield trial_tracker


@pytest.fixture
def early_stopping_trial_tracker():
    """
    Returns a trial tracker whose early_stopping criteria should be triggered.
    """
    args = copy.deepcopy(DEFAULT_ARGS)
    args.early_stopping = 3
    _, trial_tracker = trial_tracker_builder(args)
    # One successful initial trial.
    trial = trial_tracker.queue.popleft()
    trial_tracker.update_trial_metric(trial, {trial.searcher_metric_name: 0.0})
    for _ in range(args.early_stopping):
        trial = trial_tracker.queue.popleft()
        trial_tracker.report_trial_early_exit(trial)
    return trial_tracker


@pytest.mark.timeout(5)
class TestDSATTrialTracker:
    def test_trial_registration(self, basic_trial_tracker):
        queued_trials, trial_tracker = basic_trial_tracker
        for trial in queued_trials:
            assert trial.request_id in trial_tracker

    def test_should_shutdown_after_model_profile_info_failure(
        self, failed_model_profile_info_trial_tracker
    ):
        trial_tracker = failed_model_profile_info_trial_tracker
        assert trial_tracker.should_shutdown

    def test_should_shutdown_after_early_stopping(self, early_stopping_trial_tracker):
        trial_tracker = early_stopping_trial_tracker
        assert trial_tracker.should_shutdown

    def test_trial_queue_and_state_all_successes(self, basic_trial_tracker):
        """
        Verify the expected trial tracker states are accurate when all trials succeed.
        """
        queued_trials, trial_tracker = basic_trial_tracker
        for idx, trial in enumerate(queued_trials):
            num_trials_in_queue = len(queued_trials) - idx
            assert len(trial_tracker.queue) == num_trials_in_queue
            assert trial_tracker.num_closed_trials == 1 + idx
            assert not trial.running
            assert trial_tracker.can_run_more_trials

            popped_trial = trial_tracker.queue.popleft()
            popped_trial.running = True

            assert popped_trial == trial
            assert len(trial_tracker.queue) == num_trials_in_queue - 1
            assert trial_tracker.num_closed_trials == 1 + idx
            assert trial_tracker.num_running_trials == 1

            trial_tracker.update_trial_metric(
                popped_trial, {popped_trial.searcher_metric_name: 0.0}
            )
            assert trial_tracker.num_closed_trials == 2 + idx
            assert trial_tracker.num_running_trials == 0

        assert not trial_tracker.can_run_more_trials
        assert len(trial_tracker.queue) == 0
        assert trial_tracker.max_trials_are_running_or_closed
        assert trial_tracker.should_shutdown
        assert not trial_tracker.should_be_failure

    def test_trial_queue_and_state_all_errors(self, basic_trial_tracker):
        """
        Verify the expected trial tracker states are accurate when all trials fail.
        """
        queued_trials, trial_tracker = basic_trial_tracker
        for idx, trial in enumerate(queued_trials):
            num_trials_in_queue = len(queued_trials) - idx
            assert len(trial_tracker.queue) == num_trials_in_queue
            assert trial_tracker.num_closed_trials == 1 + idx
            assert not trial.running
            assert trial_tracker.can_run_more_trials

            popped_trial = trial_tracker.queue.popleft()
            popped_trial.running = True

            assert popped_trial == trial
            assert len(trial_tracker.queue) == num_trials_in_queue - 1
            assert trial_tracker.num_closed_trials == 1 + idx
            assert trial_tracker.num_running_trials == 1

            trial_tracker.report_trial_early_exit(popped_trial)
            assert trial_tracker.num_closed_trials == 2 + idx
            assert trial_tracker.num_running_trials == 0

        assert not trial_tracker.can_run_more_trials
        assert len(trial_tracker.queue) == 0
        assert trial_tracker.max_trials_are_running_or_closed
        assert trial_tracker.should_shutdown
        assert trial_tracker.should_be_failure

    def test_max_concurrent_trials(self, max_concurrent_trials_tracker):
        """
        Verify that `max_concurrent_trials` is respected.
        """
        _, trial_tracker = max_concurrent_trials_tracker
        while trial_tracker.can_run_more_trials:
            popped_trial = trial_tracker.queue.popleft()
            trial_tracker.update_trial_metric(
                popped_trial, {popped_trial.searcher_metric_name: 0.0}
            )
            assert trial_tracker.num_running_trials <= trial_tracker.max_concurrent_trials

    def test_max_slots(self, max_slots_tracker):
        """
        Verify that `max_slots` is respected.
        """
        _, trial_tracker = max_slots_tracker
        while trial_tracker.can_run_more_trials:
            popped_trial = trial_tracker.queue.popleft()
            trial_tracker.update_trial_metric(
                popped_trial, {popped_trial.searcher_metric_name: 0.0}
            )
            assert (
                trial_tracker.num_running_trials * popped_trial.slots_per_trial
                <= trial_tracker.max_slots
            )

    def test_best_metric_tracking(self, basic_trial_tracker):
        """
        Uses a series of successful trials where each trial is better than the previous one.
        """
        _, trial_tracker = basic_trial_tracker
        metrics = [n for n in range(len(trial_tracker) - 1)]
        if not trial_tracker.smaller_is_better:
            metrics = list(reversed(metrics))
        while trial_tracker.can_run_more_trials:
            popped_trial = trial_tracker.queue.popleft()
            trial_tracker.update_trial_metric(
                popped_trial, {popped_trial.searcher_metric_name: metrics.pop()}
            )
            assert trial_tracker.best_trial == popped_trial
            assert trial_tracker.best_trials_by_stage[popped_trial.stage] == popped_trial


def random_search_builder(args):
    """
    Creates a `RandomDSATSearchMethod` instance with a completed model profile info run.
    """
    exp_config = get_custom_dsat_exp_conf_from_args(args)
    search_method = _dsat_search_method.RandomDSATSearchMethod(args=args, exp_config=exp_config)
    searcher_state = _search_method.SearcherState()
    search_method.initial_operations(searcher_state)
    search_method.on_validation_completed(
        searcher_state,
        search_method.trial_tracker.model_profile_info_trial.request_id,
        MODEL_INFO_PROFILE_METRIC_FIXTURE,
        search_method.trial_tracker.model_profile_info_trial.length,
    )
    return searcher_state, search_method


@pytest.fixture
def default_random_search_method():
    searcher_state, search_method = random_search_builder(DEFAULT_ARGS)
    yield searcher_state, search_method


class TestRandomDSATSearchMethodTrialCreation:
    """
    Testing the various `RandomDSATSearchMethod` methods related to trial creation.
    """

    def test_random_hparams_and_search_data(self, default_random_search_method):
        _, search_method = default_random_search_method
        for _ in range(100):
            for stage in range(4):
                hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
                mbs = hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"]
                assert hparams[_defaults.OVERWRITE_KEY]["zero_optimization"]["stage"] == stage
                assert search_data.lo <= mbs <= search_data.hi

    def test_random_hparams_and_search_data_after_best(self, default_random_search_method):
        for _ in range(100):
            _, search_method = default_random_search_method
            for stage in range(4):
                hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
                trial = search_method.trial_tracker.create_trial(hparams, search_data)
                search_method.trial_tracker.queue_and_register_trial(trial)
                search_method.trial_tracker.queue.popleft()
                search_method.trial_tracker.update_trial_metric(
                    trial, {trial.searcher_metric_name: 0.0}
                )
                _, new_search_data = search_method.get_random_hparams_and_search_data(stage)
                assert new_search_data.lo <= new_search_data.hi

    def test_lineage_continuation_after_failures(self, default_random_search_method):
        """
        Verifying that a lineage will be attempted for `trials_per_random_config` total attempts
        even when each trial fails.
        """
        searcher_state, search_method = default_random_search_method
        # Take and fail the next trial
        first_trial = next_trial = search_method.choose_next_trial_from_queue()
        # Remove everything else, so that we only have this lineage to handle.
        search_method.trial_tracker.queue.clear()
        # The next search_method.trials_per_random_config - 1 trials should have the
        # first trial as their parent.
        for _ in range(search_method.trials_per_random_config - 1):
            search_method.on_trial_exited_early(
                searcher_state, next_trial.request_id, searcher.ExitedReason.ERRORED
            )
            next_trial = search_method.choose_next_trial_from_queue()
            assert next_trial.lineage_root == first_trial
        # And the next trial should be from a new lineage.
        search_method.on_trial_exited_early(
            searcher_state, next_trial.request_id, searcher.ExitedReason.ERRORED
        )
        next_trial = search_method.choose_next_trial_from_queue()
        assert next_trial.lineage_root != first_trial

    def test_lineage_continuation_after_successes(self, default_random_search_method):
        """
        Verifying that a lineage will be attempted for `trials_per_random_config` total attempts
        even when each trial succeeds, each improving on the last.
        """
        searcher_state, search_method = default_random_search_method
        # Take and fail the next trial
        first_trial = next_trial = search_method.choose_next_trial_from_queue()
        metrics = list(range(search_method.trials_per_random_config))
        if search_method.trial_tracker.smaller_is_better:
            metrics = metrics[::-1]
        # Remove everything else, so that we only have this lineage to handle.
        search_method.trial_tracker.queue.clear()
        # The next search_method.trials_per_random_config - 1 trials should have the
        # first trial as their parent.
        for idx in range(search_method.trials_per_random_config - 1):
            search_method.on_validation_completed(
                searcher_state,
                next_trial.request_id,
                {next_trial.searcher_metric_name: metrics[idx]},
                searcher.ExitedReason.ERRORED,
            )
            next_trial = search_method.choose_next_trial_from_queue()
            assert next_trial.lineage_root == first_trial
        # And the next trial should be from a new lineage.
        search_method.on_validation_completed(
            searcher_state,
            next_trial.request_id,
            {next_trial.searcher_metric_name: metrics[idx]},
            searcher.ExitedReason.ERRORED,
        )
        next_trial = search_method.choose_next_trial_from_queue()
        assert next_trial.lineage_root != first_trial


class TestRandomDSATSearchMethodShouldStopLineage:
    """
    Testing the various conditions which should trigger RandomDSATSearchMethod.should_stop_lineage
    """

    def test_trials_per_random_config_stopping(self, default_random_search_method):
        """
        Test that we respect the trials_per_random_config bound.
        """
        searcher_state, search_method = default_random_search_method
        trial = None
        for stage in range(4):
            for _ in range(search_method.trials_per_random_config):
                hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
                trial = search_method.trial_tracker.create_trial(
                    HPARAMS_FIXTURE, search_data, parent_trial=trial
                )
                search_method.trial_tracker.queue_and_register_trial(trial)
                search_method.trial_tracker.report_trial_early_exit(trial)

            assert search_method.should_stop_lineage(trial)

    def test_stop_stage_3(self, default_random_search_method):
        """
        Verify that we stop a stage 3 lineage when a successful stage-1 or 2 trial has been found.
        """
        searcher_state, search_method = default_random_search_method
        trial_dict_by_stage = {}
        for stage in (1, 2, 3):
            overwrites = {_defaults.OVERWRITE_KEY: {"zero_optimization": {"stage": stage}}}
            hparams = {**HPARAMS_FIXTURE, **overwrites}
            trial_dict_by_stage[stage] = search_method.trial_tracker.create_trial(hparams)
        search_method.trial_tracker.update_trial_metric(
            trial_dict_by_stage[3], {trial_dict_by_stage[3].searcher_metric_name: 0}
        )
        assert not search_method.should_stop_lineage(trial_dict_by_stage[3])

        search_method.trial_tracker.report_trial_early_exit(trial_dict_by_stage[1])
        assert not search_method.should_stop_lineage(trial_dict_by_stage[3])

        search_method.trial_tracker.update_trial_metric(
            trial_dict_by_stage[2], {trial_dict_by_stage[3].searcher_metric_name: 0}
        )
        assert search_method.should_stop_lineage(trial_dict_by_stage[3])

    def test_stop_after_fail_on_min_mbs(self, default_random_search_method):
        """
        Verify that we stop a lineage after a trial erors out when attempting its minimum batch
        size.
        """
        searcher_state, search_method = default_random_search_method
        for stage in range(4):
            hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
            hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = search_data.lo
            trial = search_method.trial_tracker.create_trial(hparams, search_data)
            search_method.trial_tracker.queue_and_register_trial(trial)
            search_method.trial_tracker.queue.popleft()
            search_method.trial_tracker.report_trial_early_exit(trial)
            assert search_method.should_stop_lineage(trial)

    def test_stop_after_max_possible_mbs_run(self, default_random_search_method):
        """
        Verify that we stop a lineage after a trial has attempted its largest possible batch size
        once a hard ceiling has been established.
        """
        searcher_state, search_method = default_random_search_method
        # Go through stages in reversed order, in order to avoid early stage-3 exiting triggers.
        for stage in reversed(range(4)):
            # Lineage should be abandoned regardless of whether the follow-on Trial errors.
            for should_error_next_trial in (True, False):
                # First fail on batch size of two, establishing a hard ceiling.
                hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
                hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = 2
                errored_trial = search_method.trial_tracker.create_trial(hparams, search_data)
                search_method.trial_tracker.queue_and_register_trial(errored_trial)
                search_method.trial_tracker.queue.popleft()
                search_method.trial_tracker.report_trial_early_exit(errored_trial)

                # Then update the ceiling and run a follow-on trial which attempts to run at the
                # established hard ceilng (which should be `train_micro_batch_size_per_gpu = 1`)
                next_trial = search_method.get_trials_after_early_exit(
                    searcher_state, errored_trial, searcher.ExitedReason.ERRORED
                )[0]
                assert next_trial.mbs == 1
                search_method.trial_tracker.queue_and_register_trial(next_trial)
                search_method.trial_tracker.queue.popleft()
                if should_error_next_trial:
                    search_method.trial_tracker.report_trial_early_exit(next_trial)
                else:
                    search_method.trial_tracker.update_trial_metric(
                        next_trial, {next_trial.searcher_metric_name: 0.0}
                    )

                assert search_method.should_stop_lineage(next_trial)

    def test_stop_when_other_configs_run_larger_batches(self, default_random_search_method):
        """
        Verify that we stop a lineage which cannot possibly run batches as large as other same-stage
        configs can run.
        """
        searcher_state, search_method = default_random_search_method
        for stage in range(4):
            hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
            good_hparams = copy.deepcopy(hparams)
            good_hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = 2
            good_trial = search_method.trial_tracker.create_trial(good_hparams, search_data)
            search_method.trial_tracker.queue_and_register_trial(good_trial)
            search_method.trial_tracker.queue.popleft()
            search_method.trial_tracker.update_trial_metric(
                good_trial, {good_trial.searcher_metric_name: 0.0}
            )

            bad_hparams = copy.deepcopy(hparams)
            bad_hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = 1
            bad_trial = search_method.trial_tracker.create_trial(bad_hparams, search_data)
            search_method.trial_tracker.queue_and_register_trial(bad_trial)
            search_method.trial_tracker.queue.popleft()
            search_method.trial_tracker.report_trial_early_exit(bad_trial)
            assert search_method.should_stop_lineage(bad_trial)


class TestRandomDSATSearchMethodChooseNextTrial:
    """
    Testing the various conditions which should non-trivially trigger
    RandomDSATSearchMethod.choose_next_trial_from_queue
    """

    def test_pruning_stage_3_trials(self, default_random_search_method):
        """
        Test the pruning of stage 3 trials.
        """
        searcher_state, search_method = default_random_search_method
        # Run a successful stage-1 trial.
        hparams, search_data = search_method.get_random_hparams_and_search_data(1)
        successful_trial = search_method.trial_tracker.create_trial(hparams, search_data)
        search_method.trial_tracker.queue_and_register_trial(successful_trial)
        search_method.trial_tracker.queue.popleft()
        search_method.trial_tracker.update_trial_metric(
            successful_trial, {successful_trial.searcher_metric_name: 0.0}
        )

        # Queue up a number of stage-3 trials and verify that choose_next_trial_from_queue
        # returns a non-stage-3 trial and that no other stage-3 trials remain in the queue.
        stage_three_trials = []
        for _ in range(10):
            hparams, search_data = search_method.get_random_hparams_and_search_data(3)
            trial = search_method.trial_tracker.create_trial(hparams, search_data)
            stage_three_trials.append(trial)
            search_method.trial_tracker.queue_and_register_trial(trial)

        # Then empty the queue and verify that all the trials which actually run are not
        # stage 3, but rather their replacements.
        while search_method.trial_tracker.queue:
            next_trial = search_method.choose_next_trial_from_queue()
            assert next_trial.stage != 3

    def test_queue_pruning_small_mbs_trials(self, default_random_search_method):
        """
        Test the pruning of trials with smaller `train_micro_batch_size_per_gpu` than
        already-successfully-run trials of the same stage.
        """
        searcher_state, search_method = default_random_search_method
        # Run successful train_micro_batch_size_per_gpu = 2 trials.
        for stage in reversed(range(4)):
            hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
            hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = 2
            successful_trial = search_method.trial_tracker.create_trial(hparams, search_data)
            search_method.trial_tracker.queue_and_register_trial(successful_trial)
            search_method.trial_tracker.queue.popleft()
            search_method.trial_tracker.update_trial_metric(
                successful_trial, {successful_trial.searcher_metric_name: 0.0}
            )

            # Queue up a number of smaller batch, same-stage trials and verify that they get pruned
            smaller_batch_trials = []
            for _ in range(10):
                hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
                hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = 1
                trial = search_method.trial_tracker.create_trial(hparams, search_data)
                smaller_batch_trials.append(trial)
                search_method.trial_tracker.queue_and_register_trial(trial)

            # All of the above mbs=1 trials should get pruned and replaced by larger trials.
            while search_method.trial_tracker.queue:
                next_trial = search_method.choose_next_trial_from_queue()
                assert next_trial.mbs >= 2


class MockMaster:
    """
    Sends v1 metrics back to the Search Runner in the manner defined with the
    `all_metrics` list of dictionaries.

    The metrics are sent as a `v1ValidationCompleted` metric event. When the key for
    the metric is instead `ERROR_METRIC_NAME`, this signals to the `MockMaster` to
    instead send a `v1TrialExitedEarly` event to the Search Runner.
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
            metric = self.all_metrics[self.metric_index]
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
