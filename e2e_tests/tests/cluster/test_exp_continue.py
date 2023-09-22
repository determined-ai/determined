import tempfile
from typing import List, Tuple

import pytest

from determined.common import yaml
from determined.common.api import bindings
from determined.common.api.bindings import experimentv1State
from tests import api_utils
from tests import config as conf
from tests import experiment as exp

from .test_groups import det_cmd


@pytest.mark.e2e_cpu
def test_continue_config_file_cli() -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump({"hyperparameters": {"metrics_sigma": 1.0}}, f)
        det_cmd(["e", "continue", str(exp_id), "--config-file", tf.name], check=True)

    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)


@pytest.mark.e2e_cpu
def test_continue_config_file_and_args_cli() -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    expected_name = "checkThis"
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            yaml.dump({"name": expected_name, "hyperparameters": {"metrics_sigma": -1.0}}, f)

        stdout = det_cmd(
            [
                "e",
                "continue",
                str(exp_id),
                "--config-file",
                tf.name,
                "--config",
                "hyperparameters.metrics_sigma=1.0",
                "-f",
            ],
            check=True,
        ).stdout
        # Follow works till end of trial.
        assert "resources exited successfully with a zero exit code" in stdout.decode("utf-8")

    # Name is also still applied.
    sess = api_utils.determined_test_session()
    resp = bindings.get_GetExperiment(sess, experimentId=exp_id)
    assert resp.experiment.config["name"] == expected_name
    assert resp.experiment.state == experimentv1State.COMPLETED  # Follow goes till completion.

    # Experiment original config is not updated. This might be a slight abuse of the webui's
    # meaning original config as "pre merged". I imagine most user's continues won't change config
    # besides maybe length.
    assert expected_name not in resp.experiment.originalConfig


@pytest.mark.e2e_cpu
def test_continue_fixing_broken_config() -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    det_cmd(
        ["e", "continue", str(exp_id), "--config", "hyperparameters.metrics_sigma=1.0"], check=True
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)

    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1

    # Trial logs show both tasks logs with the failure message in it.
    trial_logs = "\n".join(exp.trial_logs(trials[0].trial.id))
    assert "assert 0 <= self.metrics_sigma" in trial_logs
    assert "resources exited successfully with a zero exit code" in trial_logs


@pytest.mark.e2e_cpu
def test_continue_max_restart() -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0", "--config", "max_restarts=2"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1

    def count_times_ran() -> int:
        return "\n".join(exp.trial_logs(trials[0].trial.id)).count("assert 0 <= self.metrics_sigma")

    def get_trial_restarts() -> int:
        experiment_trials = exp.experiment_trials(exp_id)
        assert len(experiment_trials) == 1
        return experiment_trials[0].trial.restarts

    assert count_times_ran() == 3
    assert get_trial_restarts() == 2

    det_cmd(["e", "continue", str(exp_id)], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)
    assert count_times_ran() == 6
    assert get_trial_restarts() == 2

    det_cmd(["e", "continue", str(exp_id), "--config", "max_restarts=1"], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)
    assert count_times_ran() == 8
    assert get_trial_restarts() == 1


@pytest.mark.e2e_cpu
def test_continue_trial_time() -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "hyperparameters.metrics_sigma=-1.0"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    sess = api_utils.determined_test_session()

    def exp_start_end_time() -> Tuple[str, str]:
        e = bindings.get_GetExperiment(sess, experimentId=exp_id).experiment
        assert e.endTime is not None
        return e.startTime, e.endTime

    def trial_start_end_time() -> Tuple[str, str]:
        experiment_trials = exp.experiment_trials(exp_id)
        assert len(experiment_trials) == 1
        assert experiment_trials[0].trial.endTime is not None
        return experiment_trials[0].trial.startTime, experiment_trials[0].trial.endTime

    exp_orig_start, exp_orig_end = exp_start_end_time()
    trial_orig_start, trial_orig_end = trial_start_end_time()

    det_cmd(["e", "continue", str(exp_id)], check=True)
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    exp_new_start, exp_new_end = exp_start_end_time()
    trial_new_start, trial_new_end = trial_start_end_time()

    assert exp_orig_start == exp_new_start
    assert trial_orig_start == trial_new_start

    assert exp_new_end > exp_orig_end
    assert trial_new_end > trial_orig_end

    # Task times are updated.
    experiment_trials = exp.experiment_trials(exp_id)
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
    # Experiment fails before first checkpoint.
    exp_id = exp.create_experiment(
        conf.fixtures_path("mnist_pytorch/failable.yaml"),
        conf.fixtures_path("mnist_pytorch"),
        ["--config", "environment.environment_variables=['FAIL_AT_BATCH=2']"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)

    sess = api_utils.determined_test_session()
    trials = exp.experiment_trials(exp_id)
    assert len(trials) == 1
    trial_id = trials[0].trial.id

    def assert_exited_at(batch_idx: int) -> None:
        assert f"failed at this batch {batch_idx}" in "\n".join(exp.trial_logs(trial_id))

    assert_exited_at(2)

    def get_metric_list() -> List[bindings.v1MetricsReport]:
        resp_list = bindings.get_GetValidationMetrics(sess, trialIds=[trial_id])
        return [metric for resp in resp_list for metric in resp.metrics]

    metrics = get_metric_list()
    assert len(metrics) == 2

    first_metric_ids = []
    i = 1
    for m in metrics:
        first_metric_ids.append(m.id)
        assert m.totalBatches == i
        i += 1

    # Experiment has to start over since we didn't checkpoint.
    # We must invalidate all previous reported metrics.
    # This time experiment makes it a validation after the first checkpoint.
    det_cmd(
        [
            "e",
            "continue",
            str(exp_id),
            "--config",
            "environment.environment_variables=['FAIL_AT_BATCH=5']",
        ],
        check=True,
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.ERROR)
    assert_exited_at(5)

    second_metric_ids = []
    metrics = get_metric_list()
    assert len(metrics) == 5
    i = 1
    for m in metrics:
        assert m.id not in first_metric_ids  # Invalidated first metrics.
        second_metric_ids.append(m.id)
        assert m.totalBatches == i
        i += 1

    # We lose one metric since we are continuing from first checkpoint.
    # We correctly stop at total_batches.
    det_cmd(
        [
            "e",
            "continue",
            str(exp_id),
            "--config",
            "environment.environment_variables=['FAIL_AT_BATCH=-1']",
        ],
        check=True,
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)

    metrics = get_metric_list()
    assert len(metrics) == 8
    i = 1
    for m in metrics:
        if m.totalBatches <= 3:
            assert m.id in second_metric_ids
        else:
            assert m.id not in second_metric_ids
        assert m.totalBatches == i
        i += 1


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("continue_max_length", [405, 500])
def test_continue_workloads_searcher(continue_max_length: int) -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        [],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)

    det_cmd(
        [
            "e",
            "continue",
            str(exp_id),
            "--config",
            "searcher.max_length.batches={continue_max_length}",
            "--config",
            "searcher.name=single",
        ],
        check=True,
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("continue_max_length", [2, 3])
def test_continue_pytorch_completed_searcher(continue_max_length: int) -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("mnist_pytorch/failable.yaml"),
        conf.fixtures_path("mnist_pytorch"),
        ["--config", "searcher.max_length.batches=3"],
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)

    # Train for less or the same time has no error.
    det_cmd(
        [
            "e",
            "continue",
            str(exp_id),
            "--config",
            f"searcher.max_length.batches={continue_max_length}",
            "--config",
            "searcher.name=single",
        ],
        check=True,
    )
    exp.wait_for_experiment_state(exp_id, experimentv1State.COMPLETED)
