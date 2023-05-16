import copy
import json
import math
import pathlib
import shutil
import tempfile
from collections import deque
from typing import Any, Deque, Dict, List, Optional, Sequence, Union

import pytest

from determined import searcher
from determined.common.api import bindings
from determined.integrations.huggingface import get_hf_args_with_overwrites
from determined.pytorch.dsat import DSATTrial, DSATTrialTracker, _defaults, _utils
from determined.pytorch.dsat._dsat_search_method import ASHADSATSearchData
from determined.pytorch.dsat._run_dsat import get_custom_dsat_exp_conf_from_args
from determined.searcher import _search_method
from tests.custom_search_mocks import MockMasterSearchRunner

ERROR_METRIC_NAME = "error"

BASE_EXPERIMENT_FIXTURE_PATH = (
    pathlib.Path(__file__).resolve().parent.parent.joinpath("fixtures/deepspeed_autotune")
)
MODEL_DIR = BASE_EXPERIMENT_FIXTURE_PATH.joinpath("example_experiment")
DS_CONFIG_PATH = MODEL_DIR.joinpath("ds_config.json")
CONFIG_PATH = MODEL_DIR.joinpath("deepspeed.yaml")
DEFAULT_ARGS_DICT = {
    search_method_name: _utils.get_full_parser().parse_args(
        [search_method_name, str(CONFIG_PATH), str(MODEL_DIR)]
    )
    for search_method_name in _defaults.ALL_SEARCH_METHOD_CLASSES
}
for default_args in DEFAULT_ARGS_DICT.values():
    default_args.experiment_id = 0

DEFAULT_SEARCH_RUNNER_CONFIG_DICT = {
    search_method_name: _utils.get_search_runner_config_from_args(default_args)
    for search_method_name, default_args in DEFAULT_ARGS_DICT.items()
}
DEFAULT_CUSTOM_DSAT_EXP_CONFIG_DICT = {
    search_method_name: get_custom_dsat_exp_conf_from_args(default_args)
    for search_method_name, default_args in DEFAULT_ARGS_DICT.items()
}


MODEL_INFO_PROFILE_METRIC_FIXTURE = {
    "num_params": 60192808,
    "trainable_num_params": 60192808,
    "activation_mem_per_gpu": 1698283521,
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
    _defaults.OVERWRITE_KEY: {
        "train_batch_size": 1,
        "gradient_accumulation_steps": 1,
        "train_micro_batch_size_per_gpu": 1,
    },
}

HF_DS_CONFIG_PATH = BASE_EXPERIMENT_FIXTURE_PATH.joinpath("hf_integration_experiment").joinpath(
    "ds_config.json"
)
# HF args without any training batch size args and no deepspeed flag.
DEFAULT_HF_ARGS_WITHOUT_DEEPSPEED = """"
--model_name_or_path gpt2
--dataset_name wikitext
--dataset_config_name wikitext-2-raw-v1
--do_train
--do_eval
--max_steps 100
--logging_strategy steps
--logging_steps 10
--output_dir /tmp/test-clm
--eval_steps 10
--evaluation_strategy steps
--save_total_limit 3
--seed 1337
--save_strategy steps
--save_steps 20
--per_device_eval_batch_size 8
"""
DEFAULT_HF_ARGS_WITHOUT_DEEPSPEED = DEFAULT_HF_ARGS_WITHOUT_DEEPSPEED.split()


def _run_searcher(search_method_name: str, all_metrics):
    """
    Run a mocked version of the Determined master with a deterministic series of
    returned metrics for a given Deepspeed Autotune Custom Search Method
    """
    search_method = _defaults.ALL_SEARCH_METHOD_CLASSES[search_method_name]
    default_args = DEFAULT_ARGS_DICT[search_method_name]
    default_exp_config = DEFAULT_CUSTOM_DSAT_EXP_CONFIG_DICT[search_method_name]
    with tempfile.TemporaryDirectory() as searcher_dir:
        searcher_dir = pathlib.Path(searcher_dir)
        search_method = search_method(args=default_args, exp_config=default_exp_config)
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
    for search_method_name in _defaults.ALL_SEARCH_METHOD_CLASSES:
        # All of our search methods currently run all of the specified `max-trials` in the
        # happy path
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        model_info_profile_trial_metrics = [MODEL_INFO_PROFILE_METRIC_FIXTURE]
        successful_trial_metrics = [
            {_defaults.AUTOTUNING_ARG_DEFAULTS["metric"]: 0.0} for _ in range(exp_num_trials - 1)
        ]
        all_metrics = model_info_profile_trial_metrics + successful_trial_metrics
        search_runner = _run_searcher(search_method_name, all_metrics)
        assert len(search_runner.state.trials_created) == exp_num_trials
        assert len(search_runner.state.trials_closed) == exp_num_trials
        assert len(search_runner.state.trial_progress) == exp_num_trials
        for trial_uuid in search_runner.state.trial_progress:
            assert search_runner.state.trial_progress[trial_uuid] == 1.0
        assert not search_runner.state.experiment_failed
        assert search_runner.state.experiment_completed


@pytest.mark.timeout(5)
def test_continuous_failures() -> None:
    """
    Make sure that DSAT Search Methods can handle continuous failures. The experiment should be
    marked as failed.
    """
    for search_method_name in _defaults.ALL_SEARCH_METHOD_CLASSES:
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        model_info_profile_trial_metrics = [MODEL_INFO_PROFILE_METRIC_FIXTURE]
        failed_trial_metrics = [{ERROR_METRIC_NAME: True} for _ in range(exp_num_trials - 1)]
        all_metrics = model_info_profile_trial_metrics + failed_trial_metrics
        search_runner = _run_searcher(search_method_name, all_metrics)

        assert len(search_runner.state.trials_created) == exp_num_trials
        assert len(search_runner.state.failures) == exp_num_trials - 1
        assert len(search_runner.state.trials_closed) == exp_num_trials
        assert len(search_runner.state.trial_progress) == exp_num_trials
        assert search_runner.state.experiment_failed
        assert not search_runner.state.experiment_completed


@pytest.mark.timeout(5)
def test_one_off_failure() -> None:
    """Make sure that DSAT Search Methods can properly handle a single failure"""
    for search_method_name in _defaults.ALL_SEARCH_METHOD_CLASSES:
        exp_num_trials = _defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
        model_info_profile_trial_metrics = [MODEL_INFO_PROFILE_METRIC_FIXTURE]
        one_failed_trial_metrics = [{ERROR_METRIC_NAME: True}]
        successful_trial_metrics = [
            {_defaults.AUTOTUNING_ARG_DEFAULTS["metric"]: 0.0} for _ in range(exp_num_trials - 2)
        ]
        all_metrics = (
            model_info_profile_trial_metrics + one_failed_trial_metrics + successful_trial_metrics
        )
        search_runner = _run_searcher(search_method_name, all_metrics)

        assert len(search_runner.state.trials_created) == exp_num_trials
        assert len(search_runner.state.failures) == 1
        assert len(search_runner.state.trials_closed) == exp_num_trials
        assert len(search_runner.state.trial_progress) == exp_num_trials
        assert not search_runner.state.experiment_failed
        assert search_runner.state.experiment_completed


@pytest.mark.timeout(5)
def test_model_profile_info_run_failure() -> None:
    """Test DSAT with a failed model profile info run."""
    for search_method_name in _defaults.ALL_SEARCH_METHOD_CLASSES:
        failed_model_profile_info_trial_metrics = [
            {ERROR_METRIC_NAME: True},
        ]
        search_runner = _run_searcher(
            search_method_name,
            failed_model_profile_info_trial_metrics,
        )
        assert len(search_runner.state.trials_created) == 1
        assert len(search_runner.state.failures) == 1
        assert len(search_runner.state.trials_closed) == 1
        assert len(search_runner.state.trial_progress) == 1
        assert search_runner.state.experiment_failed
        assert not search_runner.state.experiment_completed


class TestDSATTrial:
    @pytest.mark.timeout(5)
    def setup_class(self):
        self.first_trial = DSATTrial(**DSATTRIAL_ARGS)

    @pytest.mark.timeout(5)
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
            trial.metric = {trial.searcher_metric_name: 0.0}
        assert trial.num_completed_trials_in_lineage == len(trials)

    @pytest.mark.timeout(5)
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


def queue_and_trial_tracker_builder(args):
    """Completes the model profile into trial and load up a queue of max_trials Trials."""
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
def basic_queue_and_trial_tracker():
    yield queue_and_trial_tracker_builder(DEFAULT_ARGS_DICT["_test"])


@pytest.fixture
def max_concurrent_trials_queue_and_tracker():
    args = copy.deepcopy(DEFAULT_ARGS_DICT["_test"])
    args.max_concurrent_trials = 2
    yield queue_and_trial_tracker_builder(args)


@pytest.fixture
def max_slots_queue_and_trial_tracker():
    args = copy.deepcopy(DEFAULT_ARGS_DICT["_test"])
    args.max_slots = 4
    yield queue_and_trial_tracker_builder(args)


@pytest.fixture
def failed_model_profile_info_queue_and_trial_tracker():
    exp_config = DEFAULT_CUSTOM_DSAT_EXP_CONFIG_DICT["_test"]
    trial_tracker = DSATTrialTracker(args=DEFAULT_ARGS_DICT["_test"], exp_config=exp_config)
    model_profile_info_trial = trial_tracker.create_model_profile_info_trial()
    trial_tracker.queue_and_register_trial(model_profile_info_trial)
    trial_tracker.report_trial_early_exit(trial_tracker.model_profile_info_trial)
    yield trial_tracker


@pytest.fixture
def early_stopping_queue_and_trial_tracker():
    """
    Returns a trial tracker whose early_stopping criteria should be triggered.
    """
    args = copy.deepcopy(DEFAULT_ARGS_DICT["_test"])
    args.early_stopping = 3
    _, trial_tracker = queue_and_trial_tracker_builder(args)
    # One successful initial trial.
    trial = trial_tracker.queue.popleft()
    trial_tracker.update_trial_metric(trial, {trial.searcher_metric_name: 0.0})
    for _ in range(args.early_stopping):
        trial = trial_tracker.queue.popleft()
        trial_tracker.report_trial_early_exit(trial)
    return trial_tracker


class TestDSATTrialTracker:
    @pytest.mark.timeout(5)
    def test_trial_registration(self, basic_queue_and_trial_tracker):
        queued_trials, trial_tracker = basic_queue_and_trial_tracker
        for trial in queued_trials:
            assert trial.request_id in trial_tracker

    @pytest.mark.timeout(5)
    def test_trial_queue_and_state_all_successes(self, basic_queue_and_trial_tracker):
        """
        Verify the expected trial tracker states are accurate when all trials succeed.
        """
        queued_trials, trial_tracker = basic_queue_and_trial_tracker
        for idx, trial in enumerate(queued_trials):
            num_trials_in_queue = len(queued_trials) - idx
            assert len(trial_tracker.queue) == num_trials_in_queue
            assert trial_tracker.num_completed_trials == 1 + idx
            assert not trial.running
            assert trial_tracker.can_run_more_trials

            popped_trial = trial_tracker.queue.popleft()
            popped_trial.running = True

            assert popped_trial == trial
            assert len(trial_tracker.queue) == num_trials_in_queue - 1
            assert trial_tracker.num_completed_trials == 1 + idx
            assert trial_tracker.num_running_trials == 1

            trial_tracker.update_trial_metric(
                popped_trial, {popped_trial.searcher_metric_name: 0.0}
            )
            assert trial_tracker.num_completed_trials == 2 + idx
            assert trial_tracker.num_running_trials == 0

        assert not trial_tracker.can_run_more_trials
        assert len(trial_tracker.queue) == 0
        assert trial_tracker.max_trials_are_running_or_closed
        assert not trial_tracker.should_be_failure

    @pytest.mark.timeout(5)
    def test_trial_queue_and_state_all_errors(self, basic_queue_and_trial_tracker):
        """
        Verify the expected trial tracker states are accurate when all trials fail.
        """
        queued_trials, trial_tracker = basic_queue_and_trial_tracker
        for idx, trial in enumerate(queued_trials):
            num_trials_in_queue = len(queued_trials) - idx
            assert len(trial_tracker.queue) == num_trials_in_queue
            assert trial_tracker.num_completed_trials == 1 + idx
            assert not trial.running
            assert trial_tracker.can_run_more_trials

            popped_trial = trial_tracker.queue.popleft()
            popped_trial.running = True

            assert popped_trial == trial
            assert len(trial_tracker.queue) == num_trials_in_queue - 1
            assert trial_tracker.num_completed_trials == 1 + idx
            assert trial_tracker.num_running_trials == 1

            trial_tracker.report_trial_early_exit(popped_trial)
            assert trial_tracker.num_completed_trials == 2 + idx
            assert trial_tracker.num_running_trials == 0

        assert not trial_tracker.can_run_more_trials
        assert len(trial_tracker.queue) == 0
        assert trial_tracker.max_trials_are_running_or_closed
        assert trial_tracker.should_be_failure

    @pytest.mark.timeout(5)
    def test_max_concurrent_trials(self, max_concurrent_trials_queue_and_tracker):
        """
        Verify that `max_concurrent_trials` is respected.
        """
        _, trial_tracker = max_concurrent_trials_queue_and_tracker
        while trial_tracker.can_run_more_trials:
            popped_trial = trial_tracker.queue.popleft()
            trial_tracker.update_trial_metric(
                popped_trial, {popped_trial.searcher_metric_name: 0.0}
            )
            assert trial_tracker.num_running_trials <= trial_tracker.max_concurrent_trials

    @pytest.mark.timeout(5)
    def test_max_slots(self, max_slots_queue_and_trial_tracker):
        """
        Verify that `max_slots` is respected.
        """
        _, trial_tracker = max_slots_queue_and_trial_tracker
        while trial_tracker.can_run_more_trials:
            popped_trial = trial_tracker.queue.popleft()
            trial_tracker.update_trial_metric(
                popped_trial, {popped_trial.searcher_metric_name: 0.0}
            )
            assert (
                trial_tracker.num_running_trials * popped_trial.slots_per_trial
                <= trial_tracker.max_slots
            )

    @pytest.mark.timeout(5)
    def test_best_metric_tracking(self, basic_queue_and_trial_tracker):
        """
        Uses a series of successful trials where each trial is better than the previous one.
        """
        _, trial_tracker = basic_queue_and_trial_tracker
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


def search_state_and_method_builder(args):
    """
    Creates the appropriate `BaseDSATSearchMethod` superclass instance with a completed model
    profile info run and a populated queue.
    """
    exp_config = get_custom_dsat_exp_conf_from_args(args)
    search_method = _defaults.ALL_SEARCH_METHOD_CLASSES[args.search_method](
        args=args, exp_config=exp_config
    )
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
def default_random_state_and_search_method():
    searcher_state, search_method = search_state_and_method_builder(DEFAULT_ARGS_DICT["random"])
    yield searcher_state, search_method


class TestRandomDSATSearchMethodTrialCreation:
    """
    Testing the various `RandomDSATSearchMethod` methods related to trial creation.
    """

    @pytest.mark.timeout(5)
    def test_random_hparams_and_search_data(self, default_random_state_and_search_method):
        _, search_method = default_random_state_and_search_method
        for _ in range(100):
            for stage in range(4):
                hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
                mbs = hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"]
                assert hparams[_defaults.OVERWRITE_KEY]["zero_optimization"]["stage"] == stage
                assert search_data.lo <= mbs <= search_data.hi

    @pytest.mark.timeout(5)
    def test_random_hparams_and_search_data_after_best(
        self, default_random_state_and_search_method
    ):
        for _ in range(100):
            _, search_method = default_random_state_and_search_method
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

    @pytest.mark.timeout(5)
    def test_lineage_continuation_after_failures(self, default_random_state_and_search_method):
        """
        Verifying that a lineage will be attempted for `trials_per_random_config` total attempts
        even when each trial fails.
        """
        searcher_state, search_method = default_random_state_and_search_method
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

    @pytest.mark.timeout(5)
    def test_lineage_continuation_after_successes(self, default_random_state_and_search_method):
        """
        Verifying that a lineage will be attempted for `trials_per_random_config` total attempts
        even when each trial succeeds, each improving on the last.
        """
        searcher_state, search_method = default_random_state_and_search_method
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


@pytest.fixture
def long_random_state_and_search_method():
    """For long-running tests which need a longer max_trials."""
    args = copy.deepcopy(DEFAULT_ARGS_DICT["random"])
    args.max_trials = 10**4
    args.trials_per_random_config = 10**4
    searcher_state, search_method = search_state_and_method_builder(args)
    yield searcher_state, search_method


class TestRandomDSATSearchMethodSearch:
    @pytest.mark.timeout(5)
    def test_search_happy_path(self, long_random_state_and_search_method):
        """
        Ensure that when the actual `train_micro_batch_size_per_gpu` lies between the
        search bounds, this optimal value will be found.
        """
        searcher_state, search_method = long_random_state_and_search_method
        search_method.trial_tracker.queue.clear()
        # Test for that all stages successfully find all possible values in their search range.
        # Reverse the stage range so that early stopping of stage-3 trials is not triggered.
        for stage in reversed(range(4)):
            _, search_data = search_method.get_random_hparams_and_search_data(stage)
            num_possible_mbs = search_data.hi - search_data.lo + 1
            for target_mbs in range(search_data.lo, search_data.hi + 1):
                search_method.trial_tracker.queue.clear()
                hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
                first_trial = search_method.trial_tracker.create_trial(hparams, search_data)
                search_method.trial_tracker.queue_and_register_trial(first_trial)
                curr_trial = search_method.trial_tracker.queue.popleft()
                for _ in range(num_possible_mbs):
                    assert curr_trial.search_data.lo <= curr_trial.mbs <= curr_trial.search_data.hi
                    if curr_trial.mbs >= target_mbs:
                        search_method.on_trial_exited_early(
                            searcher_state, curr_trial.request_id, searcher.ExitedReason.ERRORED
                        )
                        assert search_method.trial_tracker.queue
                    else:
                        search_method.on_validation_completed(
                            searcher_state,
                            curr_trial.request_id,
                            {curr_trial.searcher_metric_name: 0.0},
                            curr_trial.length,
                        )
                        assert search_method.trial_tracker.queue
                    if curr_trial.mbs == target_mbs:
                        break
                    curr_trial = search_method.trial_tracker.queue.popleft()
                    # queue should now be empty
                    assert not search_method.trial_tracker.queue
                    # Every trial should belong to the same lineage.
                    assert curr_trial.lineage_root == first_trial
                assert curr_trial.mbs == target_mbs


class TestRandomDSATSearchMethodShouldStopLineage:
    """
    Testing the various conditions which should trigger RandomDSATSearchMethod.should_stop_lineage
    """

    @pytest.mark.timeout(5)
    def test_trials_per_random_config_stopping(self, default_random_state_and_search_method):
        """
        Test that we respect the trials_per_random_config bound.
        """
        assert True
        searcher_state, search_method = default_random_state_and_search_method
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

    @pytest.mark.timeout(5)
    def test_stop_stage_3(self, default_random_state_and_search_method):
        """
        Verify that we stop a stage 3 lineage when a successful stage-1 or 2 trial has been found.
        """
        searcher_state, search_method = default_random_state_and_search_method
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

    @pytest.mark.timeout(5)
    def test_stop_after_fail_on_min_mbs(self, default_random_state_and_search_method):
        """
        Verify that we stop a lineage after a trial erors out when attempting its minimum batch
        size.
        """
        searcher_state, search_method = default_random_state_and_search_method
        for stage in range(4):
            hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
            hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = search_data.lo
            trial = search_method.trial_tracker.create_trial(hparams, search_data)
            search_method.trial_tracker.queue_and_register_trial(trial)
            search_method.trial_tracker.queue.popleft()
            search_method.trial_tracker.report_trial_early_exit(trial)
            assert search_method.should_stop_lineage(trial)

    @pytest.mark.timeout(5)
    def test_stop_after_max_possible_mbs_run(self, default_random_state_and_search_method):
        """
        Verify that we stop a lineage after a trial has attempted its largest possible batch size
        once a hard ceiling has been established.
        """
        searcher_state, search_method = default_random_state_and_search_method
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

    @pytest.mark.timeout(5)
    def test_stop_when_other_configs_run_larger_batches(
        self, default_random_state_and_search_method
    ):
        """
        Verify that we stop a lineage which cannot possibly run batches as large as other same-stage
        configs can run.
        """
        searcher_state, search_method = default_random_state_and_search_method
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

    @pytest.mark.timeout(5)
    def test_pruning_stage_3_trials(self, default_random_state_and_search_method):
        """
        Test the pruning of stage 3 trials.
        """
        searcher_state, search_method = default_random_state_and_search_method
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

    @pytest.mark.timeout(5)
    def test_queue_pruning_small_mbs_trials(self, default_random_state_and_search_method):
        """
        Test the pruning of trials with smaller `train_micro_batch_size_per_gpu` than
        already-successfully-run trials of the same stage.
        """
        searcher_state, search_method = default_random_state_and_search_method
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


@pytest.fixture
def long_binary_state_and_search_method():
    """For long-running tests which need a longer max_trials."""
    args = copy.deepcopy(DEFAULT_ARGS_DICT["binary"])
    args.max_trials = 10**4
    searcher_state, search_method = search_state_and_method_builder(args)
    yield searcher_state, search_method


class TestBinaryDSATSearchMethod:
    @pytest.mark.timeout(5)
    def test_binary_happy_path(self, long_binary_state_and_search_method):
        """
        Ensure that when the actual `train_micro_batch_size_per_gpu` lies between the
        search bounds, this optimal value will be found.
        """
        searcher_state, search_method = long_binary_state_and_search_method
        search_method.trial_tracker.queue.clear()
        # Test for that all stages successfully find all possible values in their search range:
        for stage in range(4):
            _, search_data = search_method.get_random_hparams_and_search_data(stage)
            num_possible_mbs = search_data.hi - search_data.lo + 1
            for target_mbs in range(search_data.lo, search_data.hi + 1):
                search_method.trial_tracker.queue.clear()
                hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
                first_trial = search_method.trial_tracker.create_trial(hparams, search_data)
                search_method.trial_tracker.queue_and_register_trial(first_trial)
                curr_trial = search_method.trial_tracker.queue.popleft()
                for num_halvings in range(1, num_possible_mbs + 1):
                    assert curr_trial.search_data.lo <= curr_trial.mbs <= curr_trial.search_data.hi
                    if curr_trial.mbs >= target_mbs:
                        search_method.on_trial_exited_early(
                            searcher_state, curr_trial.request_id, searcher.ExitedReason.ERRORED
                        )
                        assert search_method.trial_tracker.queue
                    else:
                        search_method.on_validation_completed(
                            searcher_state,
                            curr_trial.request_id,
                            {curr_trial.searcher_metric_name: 0.0},
                            curr_trial.length,
                        )
                        assert search_method.trial_tracker.queue
                    if curr_trial.mbs == target_mbs:
                        # Affirm that the solution was found as quickly as expected.
                        assert num_halvings <= int(math.log(num_possible_mbs, 2)) + 1
                        break
                    curr_trial = search_method.trial_tracker.queue.popleft()
                    # queue should now be empty
                    assert not search_method.trial_tracker.queue
                    # Every trial should belong to the same lineage.
                    assert curr_trial.lineage_root == first_trial
                assert curr_trial.mbs == target_mbs

    @pytest.mark.timeout(5)
    def test_binary_no_trials_can_run(self, long_binary_state_and_search_method):
        """
        Verify expected behavior if every trial fails to even run batch size one.
        """
        searcher_state, search_method = long_binary_state_and_search_method
        search_method.trial_tracker.queue.clear()
        # Test for that all stages successfully find all possible values in their search range:
        for stage in range(4):
            _, search_data = search_method.get_random_hparams_and_search_data(stage)
            num_possible_mbs = search_data.hi - search_data.lo + 1
            target_mbs = 0
            search_method.trial_tracker.queue.clear()
            hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
            first_trial = search_method.trial_tracker.create_trial(hparams, search_data)
            search_method.trial_tracker.queue_and_register_trial(first_trial)
            curr_trial = search_method.trial_tracker.queue.popleft()
            for num_halvings in range(1, num_possible_mbs + 1):
                assert curr_trial.search_data.lo <= curr_trial.mbs <= curr_trial.search_data.hi
                assert curr_trial.mbs > target_mbs
                search_method.on_trial_exited_early(
                    searcher_state,
                    curr_trial.request_id,
                    searcher.ExitedReason.ERRORED,
                )
                assert search_method.trial_tracker.queue
                if curr_trial.mbs == curr_trial.search_data.lo:
                    # Next trial should start a new lineage in this case.
                    next_lineage_trial = search_method.trial_tracker.queue.popleft()
                    assert not search_method.trial_tracker.queue
                    assert next_lineage_trial.lineage_root != first_trial
                    assert num_halvings <= int(math.log(num_possible_mbs, 2)) + 1
                    break
                else:
                    curr_trial = search_method.trial_tracker.queue.popleft()
                    assert not search_method.trial_tracker.queue
                    assert curr_trial.lineage_root == first_trial

    @pytest.mark.timeout(5)
    def test_binary_range_too_small(self, long_binary_state_and_search_method):
        """
        Ensure that if the actual optimal batch size is larger than the initial range (which
        hopefully never happens, but is possible), then the largest batch size in the range is
        returned.
        """
        searcher_state, search_method = long_binary_state_and_search_method
        search_method.trial_tracker.queue.clear()
        # test for that all stages successfully find all possible values in their search range:
        for stage in range(4):
            _, search_data = search_method.get_random_hparams_and_search_data(stage)
            num_possible_mbs = search_data.hi - search_data.lo + 1
            target_mbs = search_data.hi + 1
            search_method.trial_tracker.queue.clear()
            hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
            first_trial = search_method.trial_tracker.create_trial(hparams, search_data)
            search_method.trial_tracker.queue_and_register_trial(first_trial)
            curr_trial = search_method.trial_tracker.queue.popleft()
            for num_halvings in range(1, num_possible_mbs + 1):
                assert curr_trial.search_data.lo <= curr_trial.mbs <= curr_trial.search_data.hi
                assert curr_trial.mbs < target_mbs
                search_method.on_validation_completed(
                    searcher_state,
                    curr_trial.request_id,
                    {curr_trial.searcher_metric_name: 0.0},
                    curr_trial.length,
                )
                assert search_method.trial_tracker.queue
                if curr_trial.mbs == search_data.hi:
                    # Next trial should start a new lineage in this case.
                    next_lineage_trial = search_method.trial_tracker.queue.popleft()
                    assert not search_method.trial_tracker.queue
                    assert next_lineage_trial.lineage_root != first_trial
                    assert num_halvings <= int(math.log(num_possible_mbs, 2)) + 1
                    break
                curr_trial = search_method.trial_tracker.queue.popleft()
                assert not search_method.trial_tracker.queue
                assert curr_trial.lineage_root == first_trial


@pytest.fixture
def default_asha_state_and_search_method():
    searcher_state, search_method = search_state_and_method_builder(DEFAULT_ARGS_DICT["asha"])
    yield searcher_state, search_method


@pytest.fixture
def long_large_max_resource_asha_state_and_search_method():
    args = copy.deepcopy(DEFAULT_ARGS_DICT["asha"])
    args.max_trials = 10**3
    args.max_binary_search_trials = 10**3
    searcher_state, search_method = search_state_and_method_builder(args)
    yield searcher_state, search_method


@pytest.fixture
def long_large_min_resource_asha_state_and_search_method():
    """
    For long-running tests which need a longer max_trials and resources.
    """
    args = copy.deepcopy(DEFAULT_ARGS_DICT["asha"])
    args.max_trials = 10**3
    args.min_binary_search_trials = 10**3
    searcher_state, search_method = search_state_and_method_builder(args)
    yield searcher_state, search_method


class TestASHADSATSearchMethod:
    @pytest.mark.timeout(5)
    def test_binary_happy_path(self, long_large_min_resource_asha_state_and_search_method):
        searcher_state, search_method = long_large_min_resource_asha_state_and_search_method
        search_method.trial_tracker.queue.clear()
        # Test for that all stages successfully find all possible values in their search range:
        stage = 1
        _, search_data = search_method.get_random_hparams_and_search_data(stage)
        num_possible_mbs = search_data.hi - search_data.lo + 1
        for target_mbs in range(search_data.lo, search_data.hi + 1):
            search_method.trial_tracker.queue.clear()
            hparams, search_data = search_method.get_random_hparams_and_search_data(stage)
            first_trial = search_method.trial_tracker.create_trial(hparams, search_data)
            search_method.trial_tracker.queue_and_register_trial(first_trial)
            curr_trial = search_method.trial_tracker.queue.popleft()
            for num_halvings in range(1, num_possible_mbs + 1):
                assert curr_trial.search_data.lo <= curr_trial.mbs <= curr_trial.search_data.hi
                if curr_trial.mbs >= target_mbs:
                    search_method.on_trial_exited_early(
                        searcher_state, curr_trial.request_id, searcher.ExitedReason.ERRORED
                    )
                    assert search_method.trial_tracker.queue
                else:
                    search_method.on_validation_completed(
                        searcher_state,
                        curr_trial.request_id,
                        {curr_trial.searcher_metric_name: 0.0},
                        curr_trial.length,
                    )
                    assert search_method.trial_tracker.queue
                if curr_trial.mbs == target_mbs:
                    # Affirm that the solution was found as quickly as expected.
                    assert num_halvings <= int(math.log(num_possible_mbs, 2)) + 1
                    break
                curr_trial = search_method.trial_tracker.queue.popleft()
                # queue should now be empty
                assert not search_method.trial_tracker.queue
                # Every trial should belong to the same lineage.
                assert curr_trial.lineage_root == first_trial
            assert curr_trial.mbs == target_mbs

    @pytest.mark.timeout(5)
    def test_get_top_lineages_in_rung(self, long_large_max_resource_asha_state_and_search_method):
        searcher_state, search_method = long_large_max_resource_asha_state_and_search_method
        search_method.trial_tracker.queue.clear()
        metrics = list(range(10 * search_method.divisor))
        for metric in metrics:
            hparams, search_data = search_method.get_random_hparams_and_search_data(1)
            trial = None
            for idx in range(search_method.min_binary_search_trials):
                trial = search_method.trial_tracker.create_trial(
                    hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=trial
                )
                assert trial.search_data.curr_rung == 0
                search_method.trial_tracker.queue_and_register_trial(trial)
                search_method.trial_tracker.update_trial_metric(
                    trial, {trial.searcher_metric_name: metric}
                )
                assert len(trial.lineage_set) == idx + 1
            assert search_method.lineage_completed_rung(trial, 0)
            assert not search_method.lineage_completed_rung(trial, 1)

        top_trials = search_method.get_top_lineages_in_rung(0)
        assert len(top_trials) == len(search_method.rungs[0]) // search_method.divisor
        if search_method.trial_tracker.smaller_is_better:
            expected_metrics = metrics[: len(top_trials)]
        else:
            expected_metrics = list(reversed(metrics[len(top_trials) :]))
        actual_metrics = [
            search_method.get_best_trial_in_lineage(t).metric[trial.searcher_metric_name]
            for t in top_trials
        ]
        assert expected_metrics == actual_metrics

    @pytest.mark.timeout(5)
    def test_basic_promotion(self, long_large_max_resource_asha_state_and_search_method):
        searcher_state, search_method = long_large_max_resource_asha_state_and_search_method
        search_method.trial_tracker.queue.clear()
        # Complete enough trials so that some can be promoted.
        for _ in range(search_method.divisor):
            hparams, search_data = search_method.get_random_hparams_and_search_data(1)
            trial = None
            for trial_num in range(search_method.min_binary_search_trials):
                trial = search_method.trial_tracker.create_trial(
                    hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=trial
                )
                assert trial.search_data.curr_rung == 0
                search_method.trial_tracker.queue_and_register_trial(trial)
                search_method.trial_tracker.update_trial_metric(
                    trial, {trial.searcher_metric_name: 0.0}
                )
                assert len(trial.lineage_set) == trial_num + 1
            assert search_method.lineage_completed_rung(trial, 0)
            assert not search_method.lineage_completed_rung(trial, 1)

        next_promotable_lineage = search_method.get_next_promotable_lineage()
        assert next_promotable_lineage is not None
        search_method.promote_all_trials_in_lineage(next_promotable_lineage)
        next_trial = search_method.get_next_trial_in_lineage(next_promotable_lineage)
        assert next_trial.search_data.curr_rung == 1
        assert len(next_trial.lineage_set) == search_method.min_binary_search_trials + 1

    @pytest.mark.timeout(5)
    def test_lineage_continutation(self, long_large_max_resource_asha_state_and_search_method):
        """
        Verify that we continue trials which have not yet completed their rung.
        """
        searcher_state, search_method = long_large_max_resource_asha_state_and_search_method
        search_method.trial_tracker.queue.clear()
        hparams, search_data = search_method.get_random_hparams_and_search_data(1)
        first_trial = curr_trial = search_method.trial_tracker.create_trial(
            hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=None
        )
        search_method.trial_tracker.queue_and_register_trial(first_trial)
        _ = search_method.trial_tracker.queue.popleft()
        for trial_num in range(search_method.min_binary_search_trials):
            assert curr_trial.search_data.curr_rung == 0
            assert curr_trial.lineage_root == first_trial
            assert curr_trial.num_completed_trials_in_lineage == trial_num
            assert not search_method.lineage_completed_rung(curr_trial, 0)
            search_method.on_validation_completed(
                searcher_state=searcher_state,
                request_id=curr_trial.request_id,
                metric={curr_trial.searcher_metric_name: 0.0},
                train_length=curr_trial.length,
            )
            assert curr_trial.completed
            curr_trial = search_method.trial_tracker.queue.popleft()

        assert search_method.lineage_completed_rung(first_trial, 0)
        assert curr_trial.lineage_root != first_trial

    @pytest.mark.timeout(5)
    def test_top_promotion(self, long_large_max_resource_asha_state_and_search_method):
        """
        Verify that if multiple lineages can be promoted, we promote from the higest-rung lineage
        available.
        """
        searcher_state, search_method = long_large_max_resource_asha_state_and_search_method
        good_metric, bad_metric = (
            (0.0, 1.0) if search_method.trial_tracker.smaller_is_better else (1.0, 0.0)
        )
        # Fill two rungs with trials search_method.divisor trials, so that there are enough to
        # promote from the top rung.
        for _ in range(search_method.divisor):
            hparams, search_data = search_method.get_random_hparams_and_search_data(1)
            trial = None
            for trial_num in range(search_method.min_binary_search_trials * search_method.divisor):
                trial = search_method.trial_tracker.create_trial(
                    hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=trial
                )
                search_method.trial_tracker.queue_and_register_trial(trial)
                _ = search_method.trial_tracker.queue.popleft()
                search_method.trial_tracker.update_trial_metric(
                    trial=trial,
                    metric={trial.searcher_metric_name: bad_metric},
                )
            # Remember to promote the lineage up to the second rung.
            search_method.promote_all_trials_in_lineage(trial)

        # Submit another lineage which completes the lowest rung with better metrics than the
        # lineage above, so that it is promotable.
        hparams, search_data = search_method.get_random_hparams_and_search_data(1)
        trial = None
        for trial_num in range(search_method.min_binary_search_trials):
            trial = search_method.trial_tracker.create_trial(
                hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=trial
            )
            search_method.trial_tracker.queue_and_register_trial(trial)
            _ = search_method.trial_tracker.queue.popleft()
            search_method.trial_tracker.update_trial_metric(
                trial=trial,
                metric={trial.searcher_metric_name: good_metric},
            )

        # Verify the counting above and that the next promoted trial will come from the topmost
        # possible rung.
        assert len(search_method.rungs[0]) == search_method.divisor + 1
        assert len(search_method.rungs[1]) == search_method.divisor
        assert search_method.get_next_promotable_lineage_in_rung(0) is not None
        assert search_method.get_next_promotable_lineage_in_rung(1) is not None
        assert search_method.get_next_promotable_lineage_in_rung(0).search_data.curr_rung == 0
        assert search_method.get_next_promotable_lineage_in_rung(1).search_data.curr_rung == 1
        assert search_method.get_next_promotable_lineage().search_data.curr_rung == 1

    @pytest.mark.timeout(5)
    def test_max_resource_respected(self, long_large_max_resource_asha_state_and_search_method):
        """
        Verify that we respect the maximum resource per lineage.
        """
        # Create a lineage with the maximum resource per lineage
        searcher_state, search_method = long_large_max_resource_asha_state_and_search_method
        search_method.trial_tracker.queue.clear()
        hparams, search_data = search_method.get_random_hparams_and_search_data(1)
        trial = search_method.trial_tracker.create_trial(
            hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=None
        )
        search_method.trial_tracker.queue_and_register_trial(trial)
        _ = search_method.trial_tracker.queue.popleft()
        for _ in range(search_method.max_binary_search_trials):
            search_method.trial_tracker.update_trial_metric(
                trial, {trial.searcher_metric_name: 0.0}
            )
            trial = search_method.trial_tracker.create_trial(
                hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=trial
            )
            search_method.trial_tracker.queue_and_register_trial(trial)
            _ = search_method.trial_tracker.queue.popleft()
        assert search_method.lineage_completed_rung(trial, search_method.max_rung_ceiling - 1)
        assert search_method.get_next_promotable_lineage() is None

    @pytest.mark.timeout(5)
    def test_no_continuation_for_completed_lineages(
        self, long_large_max_resource_asha_state_and_search_method
    ):
        """
        Verify that lineages which have completed their binary search are not continued.
        """
        searcher_state, search_method = long_large_max_resource_asha_state_and_search_method
        search_method.trial_tracker.queue.clear()
        hparams, _ = search_method.get_random_hparams_and_search_data(1)
        search_data = ASHADSATSearchData(lo=1, hi=1, curr_rung=0)
        hparams = copy.deepcopy(hparams)
        hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = (
            search_data.hi + search_data.lo
        ) // 2
        trial = search_method.trial_tracker.create_trial(
            hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=None
        )
        search_method.trial_tracker.queue_and_register_trial(trial)
        _ = search_method.trial_tracker.queue.popleft()
        search_method.trial_tracker.update_trial_metric(trial, {trial.searcher_metric_name: 0.0})
        assert search_method.get_next_trial_in_lineage(trial) is None

    @pytest.mark.timeout(5)
    def test_completed_binary_search_lineages_are_counted_complete(
        self, long_large_max_resource_asha_state_and_search_method
    ):
        """
        Verify that if a lineage successfully completes its binary search mid-rung, that lineage
        is counted as having completed the rung.
        """
        searcher_state, search_method = long_large_max_resource_asha_state_and_search_method
        search_method.trial_tracker.queue.clear()
        hparams, _ = search_method.get_random_hparams_and_search_data(1)
        search_data = ASHADSATSearchData(lo=1, hi=1, curr_rung=0)
        hparams = copy.deepcopy(hparams)
        hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = (
            search_data.hi + search_data.lo
        ) // 2
        successful_trial = search_method.trial_tracker.create_trial(
            hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=None
        )
        search_method.trial_tracker.queue_and_register_trial(successful_trial)
        _ = search_method.trial_tracker.queue.popleft()
        search_method.trial_tracker.update_trial_metric(
            successful_trial, {successful_trial.searcher_metric_name: 0.0}
        )
        assert search_method.lineage_completed_rung(
            successful_trial, successful_trial.search_data.curr_rung
        )

    @pytest.mark.timeout(5)
    def test_failed_binary_search_lineages_are_counted_complete(
        self, long_large_max_resource_asha_state_and_search_method
    ):
        """
        Verify that if a lineage fails its binary search mid-rung by failing on the minimum
        possible batch size, that lineage is counted as having completed the rung.
        """
        searcher_state, search_method = long_large_max_resource_asha_state_and_search_method
        search_method.trial_tracker.queue.clear()
        hparams, _ = search_method.get_random_hparams_and_search_data(1)
        search_data = ASHADSATSearchData(lo=1, hi=2, curr_rung=0)
        hparams = copy.deepcopy(hparams)
        hparams[_defaults.OVERWRITE_KEY]["train_micro_batch_size_per_gpu"] = (
            search_data.hi + search_data.lo
        ) // 2
        failed_trial = search_method.trial_tracker.create_trial(
            hparams=hparams, search_data=copy.deepcopy(search_data), parent_trial=None
        )
        search_method.trial_tracker.queue_and_register_trial(failed_trial)
        _ = search_method.trial_tracker.queue.popleft()
        search_method.trial_tracker.report_trial_early_exit(failed_trial)
        assert search_method.lineage_completed_rung(
            failed_trial, failed_trial.search_data.curr_rung
        )


class TestHFConfigOverwriting:
    @pytest.mark.timeout(5)
    def test_overwritten_args(self):
        """
        Verify that `get_hf_args_with_overwrites` returns the expected args.
        """
        optional_arg_possibilities = [
            [],
            ["--per_device_train_batch_size", "8"],
            ["--gradient_accumulation_steps", "4"],
            ["--per_device_train_batch_size", "8", "--gradient_accumulation_steps", "4"],
        ]
        for optional_args in optional_arg_possibilities:
            with tempfile.TemporaryDirectory() as d:
                ds_config_path = pathlib.Path(d).joinpath("ds_config.json")
                shutil.copyfile(HF_DS_CONFIG_PATH, ds_config_path)
                args = (
                    DEFAULT_HF_ARGS_WITHOUT_DEEPSPEED
                    + ["--deepspeed", str(ds_config_path)]
                    + optional_args
                )
                args = get_hf_args_with_overwrites(args=args, hparams=HPARAMS_FIXTURE)
                hf_flag_to_ds_key = {
                    "--per_device_train_batch_size": "train_micro_batch_size_per_gpu",
                    "--gradient_accumulation_steps": "gradient_accumulation_steps",
                }
                for idx in range(len(args)):
                    if args[idx] in hf_flag_to_ds_key:
                        hf_flag = args[idx]
                        ds_key = hf_flag_to_ds_key[hf_flag]
                        expected_hf_value = HPARAMS_FIXTURE[_defaults.OVERWRITE_KEY][ds_key]
                        actual_hf_value = HPARAMS_FIXTURE[_defaults.OVERWRITE_KEY][ds_key]
                        assert actual_hf_value == expected_hf_value

    @pytest.mark.timeout(5)
    def test_overwritten_config_file(self):
        """
        Verify that `get_hf_args_with_overwrites` overwrite the ds config file.
        """
        with tempfile.TemporaryDirectory() as d:
            overwrite_dict = HPARAMS_FIXTURE[_defaults.OVERWRITE_KEY]
            ds_config_path = pathlib.Path(d).joinpath("ds_config.json")
            shutil.copyfile(HF_DS_CONFIG_PATH, ds_config_path)

            # Verify that the original config values are different from those we are overwriting.
            with open(ds_config_path, "r") as f:
                original_ds_config = json.load(f)
                for k, v in overwrite_dict.items():
                    assert original_ds_config.get(k) != v
            args = DEFAULT_HF_ARGS_WITHOUT_DEEPSPEED + ["--deepspeed", str(ds_config_path)]
            _ = get_hf_args_with_overwrites(args=args, hparams=HPARAMS_FIXTURE)
            with open(ds_config_path, "r") as f:
                overwritten_ds_config = json.load(f)
                for k, v in overwrite_dict.items():
                    assert overwritten_ds_config.get(k) == v


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
