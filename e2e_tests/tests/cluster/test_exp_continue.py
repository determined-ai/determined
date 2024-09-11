from typing import Any, Dict, List, Tuple

import pytest

from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils, detproc
from tests import experiment as exp
from tests.experiment import noop


@pytest.mark.e2e_cpu
def test_continue_max_restart() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, [noop.Exit(7)], config={"max_restarts": 2})
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    trials = exp.experiment_trials(sess, exp_ref.id)
    assert len(trials) == 1

    def count_times_ran() -> int:
        return sum("New trial runner" in log for log in exp.trial_logs(sess, trials[0].trial.id))

    def get_trial_restarts() -> int:
        experiment_trials = exp.experiment_trials(sess, exp_ref.id)
        assert len(experiment_trials) == 1
        return experiment_trials[0].trial.restarts

    assert count_times_ran() == 3
    assert get_trial_restarts() == 2

    detproc.check_call(
        sess, ["det", "e", "continue", str(exp_ref.id), "--config", "max_restarts=1"]
    )
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR
    assert count_times_ran() == 5
    assert get_trial_restarts() == 1


@pytest.mark.e2e_cpu
def test_continue_trial_time() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, [noop.Exit(7)])
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    sess = api_utils.user_session()

    def exp_start_end_time() -> Tuple[str, str]:
        e = bindings.get_GetExperiment(sess, experimentId=exp_ref.id).experiment
        assert e.endTime is not None
        return e.startTime, e.endTime

    def trial_start_end_time() -> Tuple[str, str]:
        experiment_trials = exp.experiment_trials(sess, exp_ref.id)
        assert len(experiment_trials) == 1
        assert experiment_trials[0].trial.endTime is not None
        return experiment_trials[0].trial.startTime, experiment_trials[0].trial.endTime

    exp_orig_start, exp_orig_end = exp_start_end_time()
    trial_orig_start, trial_orig_end = trial_start_end_time()

    detproc.check_call(sess, ["det", "e", "continue", str(exp_ref.id)])
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    exp_new_start, exp_new_end = exp_start_end_time()
    trial_new_start, trial_new_end = trial_start_end_time()

    assert exp_orig_start == exp_new_start
    assert trial_orig_start == trial_new_start

    assert exp_new_end > exp_orig_end
    assert trial_new_end > trial_orig_end

    # Task times are updated.
    experiment_trials = exp.experiment_trials(sess, exp_ref.id)
    assert len(experiment_trials) == 1
    task_ids = experiment_trials[0].trial.taskIds
    assert task_ids is not None
    assert len(task_ids) == 2

    assert task_ids[1] == task_ids[0] + "-1"  # Task IDs are formatted prevTaskID-N

    task = bindings.get_GetTask(sess, taskId=task_ids[1]).task
    assert task.startTime > exp_orig_end
    assert task.endTime is not None
    assert task.endTime > task.startTime


@pytest.mark.e2e_cpu
def test_continue_batches() -> None:
    sess = api_utils.user_session()
    # Experiment fails before first checkpoint.
    exp_ref = noop.create_experiment(sess, [noop.Report({"x": 1}), noop.Exit(7)])
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR

    trials = exp.experiment_trials(sess, exp_ref.id)
    assert len(trials) == 1
    trial_id = trials[0].trial.id

    def get_metric_list() -> List[int]:
        resp_list = bindings.get_GetTrainingMetrics(sess, trialIds=[trial_id])
        return [metric.metrics["avg_metrics"]["x"] for resp in resp_list for metric in resp.metrics]

    metrics = get_metric_list()
    assert metrics == [1]

    # Experiment has to start over since we didn't checkpoint.
    # Note that the uncheckpointed `1` metric will be lost.
    detproc.check_call(
        sess,
        [
            "det",
            "e",
            "continue",
            str(exp_ref.id),
            *noop.cli_config_overrides(
                [
                    noop.Report({"x": 2}),
                    noop.Report({"x": 3}),
                    noop.Checkpoint(),
                    noop.Report({"x": 4}),
                    noop.Exit(8),
                ],
            ),
        ],
    )
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.ERROR
    metrics = get_metric_list()
    assert metrics == [2, 3, 4]

    # Now when we restart, we only lose the uncheckpointed `4` metric.
    detproc.check_call(
        sess,
        [
            "det",
            "e",
            "continue",
            str(exp_ref.id),
            *noop.cli_config_overrides(
                [
                    noop.Report({"x": 2}),
                    noop.Report({"x": 3}),
                    noop.Checkpoint(),
                    noop.Report({"x": 5}),
                    noop.Exit(0),
                ],
            ),
        ],
    )
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    metrics = get_metric_list()
    assert metrics == [2, 3, 5]


GRID_CONFIG: Dict[str, Any] = {
    "searcher": {
        "name": "grid",
    },
    "hyperparameters": {
        "val": {
            "type": "categorical",
            "vals": [1, 2],
        },
    },
}

RANDOM_CONFIG: Dict[str, Any] = {
    "searcher": {
        "name": "random",
        "max_trials": 2,
    },
}


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("searcher_type", ["random", "grid"])
def test_continue_hp_search_cli(searcher_type: str) -> None:
    sess = api_utils.user_session()
    config = {"random": RANDOM_CONFIG, "grid": GRID_CONFIG}[searcher_type]
    exp_ref = noop.create_experiment(sess, config=config)
    trials = exp.experiment_trials(sess, exp_ref.id)
    for t in trials:
        if t.trial.id % 2 == 0:
            exp.kill_trial(sess, t.trial.id)

    assert len(trials) == 2

    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    detproc.check_call(sess, ["det", "e", "continue", str(exp_ref.id)])

    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    trials = exp.experiment_trials(sess, exp_ref.id)
    for t in trials:
        assert t.trial.state == bindings.trialv1State.COMPLETED


@pytest.mark.e2e_cpu
def test_continue_hp_search_single_cli() -> None:
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess)
    trials = exp.experiment_trials(sess, exp_ref.id)
    assert len(trials) == 1
    exp.kill_trial(sess, trials[0].trial.id)

    assert exp_ref.wait(interval=0.01) == client.ExperimentState.CANCELED

    detproc.check_call(sess, ["det", "e", "continue", str(exp_ref.id)])

    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    trials = exp.experiment_trials(sess, exp_ref.id)
    assert trials[0].trial.state == bindings.trialv1State.COMPLETED
