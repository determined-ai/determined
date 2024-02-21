import subprocess
import tempfile

import pytest

from determined.common import api, util
from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils
from tests import command as cmd
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests.cluster import test_checkpoints


@pytest.mark.e2e_cpu
def test_trial_error() -> None:
    sess = api_utils.user_session()
    exp.run_failure_test(
        sess,
        conf.fixtures_path("trial_error/const.yaml"),
        conf.fixtures_path("trial_error"),
        "NotImplementedError",
    )


@pytest.mark.e2e_cpu
def test_invalid_experiment() -> None:
    sess = api_utils.user_session()
    completed_process = exp.maybe_create_experiment(
        sess, conf.fixtures_path("invalid_experiment/const.yaml"), conf.cv_examples_path("mnist_tf")
    )
    assert completed_process.returncode != 0


@pytest.mark.e2e_cpu
def test_experiment_archive_unarchive() -> None:
    sess = api_utils.user_session()
    experiment_id = exp.create_experiment(
        sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), ["--paused"]
    )

    describe_args = [
        "det",
        "experiment",
        "describe",
        "--json",
        str(experiment_id),
    ]

    # Check that the experiment is initially unarchived.
    infos = detproc.check_json(sess, describe_args)
    assert len(infos) == 1
    assert not infos[0]["experiment"]["archived"]

    # Check that archiving a non-terminal experiment fails, then terminate it.
    with pytest.raises(subprocess.CalledProcessError):
        detproc.check_call(sess, ["det", "experiment", "archive", str(experiment_id)])
    detproc.check_call(sess, ["det", "experiment", "cancel", str(experiment_id)])

    # Check that we can archive and unarchive the experiment and see the expected effects.
    detproc.check_call(sess, ["det", "experiment", "archive", str(experiment_id)])
    infos = detproc.check_json(sess, describe_args)
    assert len(infos) == 1
    assert infos[0]["experiment"]["archived"]

    detproc.check_call(sess, ["det", "experiment", "unarchive", str(experiment_id)])
    infos = detproc.check_json(sess, describe_args)
    assert len(infos) == 1
    assert not infos[0]["experiment"]["archived"]


@pytest.mark.e2e_cpu
def test_create_test_mode() -> None:
    sess = api_utils.user_session()
    # test-mode should succeed with a valid experiment.
    command = [
        "det",
        "experiment",
        "create",
        "--test-mode",
        conf.fixtures_path("mnist_pytorch/adaptive_short.yaml"),
        conf.tutorials_path("mnist_pytorch"),
    ]
    output = detproc.check_output(sess, command)
    assert "Model definition test succeeded" in output, output

    # test-mode should fail when an error is introduced into the trial
    # implementation.
    command = [
        "det",
        "experiment",
        "create",
        "--test-mode",
        conf.fixtures_path("trial_error/const.yaml"),
        conf.fixtures_path("trial_error"),
    ]
    # We expect a failing exit code, but --test-mode doesn't actually emit to stderr.
    p = detproc.check_error(sess, command, "")
    assert p.stdout
    stdout = p.stdout.decode("utf8")
    assert "resources failed with non-zero exit code" in stdout, stdout


@pytest.mark.e2e_cpu
def test_trial_logs() -> None:
    sess = api_utils.user_session()
    experiment_id = exp.run_basic_test(
        sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    trial_id = exp.experiment_trials(sess, experiment_id)[0].trial.id
    detproc.check_call(sess, ["det", "trial", "logs", str(trial_id)])
    detproc.check_call(
        sess,
        ["det", "trial", "logs", "--head", "10", str(trial_id)],
    )
    detproc.check_call(
        sess,
        ["det", "trial", "logs", "--tail", "10", str(trial_id)],
    )


@pytest.mark.e2e_cpu
def test_labels() -> None:
    sess = api_utils.user_session()
    experiment_id = exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )

    label = "__det_test_dummy_label__"

    # Add a label and check that it shows up.
    detproc.check_call(sess, ["det", "e", "label", "add", str(experiment_id), label])
    output = detproc.check_output(sess, ["det", "e", "describe", str(experiment_id)])
    assert label in output

    # Remove the label and check that it doesn't show up.
    detproc.check_call(sess, ["det", "e", "label", "remove", str(experiment_id), label])
    output = detproc.check_output(sess, ["det", "e", "describe", str(experiment_id)])
    assert label not in output


@pytest.mark.e2e_cpu
def test_end_to_end_adaptive() -> None:
    sess = api_utils.user_session()
    exp_id = exp.run_basic_test(
        sess,
        conf.fixtures_path("mnist_pytorch/adaptive_short.yaml"),
        conf.tutorials_path("mnist_pytorch"),
        None,
    )

    test_checkpoints.wait_for_gc_to_finish(sess, experiment_ids=[exp_id])

    # Check that validation accuracy look sane (more than 93% on MNIST).
    trials = exp.experiment_trials(sess, exp_id)
    best = None
    for trial in trials:
        assert len(trial.workloads) > 0
        last_validation = exp.workloads_with_validation(trial.workloads)[-1]
        accuracy = last_validation.metrics.avgMetrics["accuracy"]
        if not best or accuracy > best:
            best = accuracy

    assert best is not None
    assert best > 0.93

    # Check that the Experiment returns a sorted order of top checkpoints
    # without gaps. The top 2 checkpoints should be the first 2 of the top k
    # checkpoints if sorting is stable.
    d = client.Determined._from_session(sess)
    exp_ref = d.get_experiment(exp_id)

    top_2 = exp_ref.top_n_checkpoints(2)
    top_k = exp_ref.top_n_checkpoints(
        len(trials), sort_by="validation_loss", smaller_is_better=True
    )

    top_2_uuids = [c.uuid for c in top_2]
    top_k_uuids = [c.uuid for c in top_k]

    assert top_2_uuids == top_k_uuids[:2]

    # Check that metrics are truly in sorted order.
    assert all(c.training is not None for c in top_k)
    metrics = [
        c.training.validation_metrics["avgMetrics"]["validation_loss"]
        for c in top_k
        if c.training is not None
    ]

    assert metrics == sorted(metrics)

    # Check that changing smaller is better reverses the checkpoint ordering.
    top_k_reversed = exp_ref.top_n_checkpoints(
        len(trials), sort_by="validation_loss", smaller_is_better=False
    )
    top_k_reversed_uuids = [c.uuid for c in top_k_reversed]

    assert top_k_uuids == top_k_reversed_uuids[::-1]

    checkpoint = top_k[0]
    checkpoint.add_metadata({"testing": "metadata"})
    db_check = d.get_checkpoint(checkpoint.uuid)
    # Make sure the checkpoint metadata is correct and correctly saved to the db.
    # Beginning with 0.18 the TrialController contributes a few items to the dict.
    assert checkpoint.metadata
    assert checkpoint.metadata.get("testing") == "metadata"
    assert checkpoint.metadata.keys() == {
        "determined_version",
        "format",
        "framework",
        "steps_completed",
        "testing",
    }
    assert checkpoint.metadata == db_check.metadata

    checkpoint.add_metadata({"some_key": "some_value"})
    db_check = d.get_checkpoint(checkpoint.uuid)
    assert checkpoint.metadata.items() > {"testing": "metadata", "some_key": "some_value"}.items()
    assert checkpoint.metadata.keys() == {
        "determined_version",
        "format",
        "framework",
        "steps_completed",
        "testing",
        "some_key",
    }
    assert checkpoint.metadata == db_check.metadata

    checkpoint.add_metadata({"testing": "override"})
    db_check = d.get_checkpoint(checkpoint.uuid)
    assert checkpoint.metadata.items() > {"testing": "override", "some_key": "some_value"}.items()
    assert checkpoint.metadata == db_check.metadata

    checkpoint.remove_metadata(["some_key"])
    db_check = d.get_checkpoint(checkpoint.uuid)
    assert "some_key" not in checkpoint.metadata
    assert checkpoint.metadata["testing"] == "override"
    assert checkpoint.metadata == db_check.metadata


@pytest.mark.e2e_cpu
def test_log_null_bytes() -> None:
    sess = api_utils.user_session()
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["write_null"] = True
    config_obj["max_restarts"] = 0
    config_obj["searcher"]["max_length"] = {"batches": 1}
    experiment_id = exp.run_basic_test_with_temp_config(
        sess, config_obj, conf.fixtures_path("no_op"), 1
    )

    trials = exp.experiment_trials(sess, experiment_id)
    assert len(trials) == 1
    logs = exp.trial_logs(sess, trials[0].trial.id)
    assert len(logs) > 0


@pytest.mark.e2e_cpu
def test_graceful_trial_termination() -> None:
    sess = api_utils.user_session()
    config_obj = conf.load_config(conf.fixtures_path("no_op/grid-graceful-trial-termination.yaml"))
    exp.run_basic_test_with_temp_config(sess, config_obj, conf.fixtures_path("no_op"), 2)


@pytest.mark.e2e_cpu
def test_kill_experiment_ignoring_preemption() -> None:
    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("core_api/sleep.yaml"),
        conf.fixtures_path("core_api"),
        None,
    )
    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.RUNNING)

    bindings.post_CancelExperiment(sess, id=exp_id)
    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.STOPPING_CANCELED)

    bindings.post_KillExperiment(sess, id=exp_id)
    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.CANCELED)


@pytest.mark.e2e_cpu
def test_fail_on_first_validation() -> None:
    sess = api_utils.user_session()
    error_log = "failed on first validation"
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["fail_on_first_validation"] = error_log
    exp.run_failure_test_with_temp_config(
        sess,
        config_obj,
        conf.fixtures_path("no_op"),
        error_log,
    )


@pytest.mark.e2e_cpu
def test_perform_initial_validation() -> None:
    sess = api_utils.user_session()
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config = conf.set_max_length(config, {"batches": 1})
    config = conf.set_perform_initial_validation(config, True)
    exp_id = exp.run_basic_test_with_temp_config(sess, config, conf.fixtures_path("no_op"), 1)
    exp.assert_performed_initial_validation(sess, exp_id)


@pytest.mark.e2e_cpu_2a
@pytest.mark.parametrize(
    "name,searcher_cfg",
    [
        (
            "random",
            {
                "metric": "validation_error",
                "name": "random",
                "max_length": {
                    "batches": 3000,
                },
                "max_trials": 8,
                "max_concurrent_trials": 1,
            },
        ),
        (
            "grid",
            {
                "metric": "validation_error",
                "name": "random",
                "max_length": {
                    "batches": 3000,
                },
                "max_trials": 8,
                "max_concurrent_trials": 1,
            },
        ),
    ],
)
def test_max_concurrent_trials(name: str, searcher_cfg: str) -> None:
    sess = api_utils.user_session()
    config_obj = conf.load_config(conf.fixtures_path("no_op/single-very-many-long-steps.yaml"))
    config_obj["name"] = f"{name} searcher max concurrent trials test"
    config_obj["searcher"] = searcher_cfg
    config_obj["hyperparameters"]["x"] = {
        "type": "categorical",
        # Intentionally give the searcher more to do, in case a bug involves exceeding max
        # concurrent trials.
        "vals": list(range(16)),
    }

    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            util.yaml_safe_dump(config_obj, f)
        experiment_id = exp.create_experiment(sess, tf.name, conf.fixtures_path("no_op"), [])

    try:
        exp.wait_for_experiment_active_workload(sess, experiment_id)
        trials = exp.wait_for_at_least_n_trials(sess, experiment_id, 1)
        assert len(trials) == 1, trials

        for t in trials:
            exp.kill_trial(sess, t.trial.id)

        # Give the experiment time to refill max_concurrent_trials.
        trials = exp.wait_for_at_least_n_trials(sess, experiment_id, 2)

        # The experiment handling the cancel message and waiting for it to be cancelled slyly
        # (hackishly) allows us to synchronize with the experiment state after after canceling
        # the first two trials.
        exp.cancel_single(sess, experiment_id)

        # Make sure that there were never more than 2 total trials created.
        trials = exp.wait_for_at_least_n_trials(sess, experiment_id, 2)
        assert len(trials) == 2, trials

    finally:
        exp.kill_single(sess, experiment_id)


@pytest.mark.e2e_cpu
def test_experiment_list_columns() -> None:
    sess = api_utils.user_session()
    exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/single-nested-hps.yaml"),
        conf.fixtures_path("no_op"),
        ["--project", "1"],
    )
    exp_hyperparameters = [
        "global_batch_size",
        "metrics_progression",
        "metrics_base.min_val",
        "metrics_base.max_val",
    ]
    exp_metrics = [
        "validation.validation_error.min",
        "validation.validation_error.max",
        "validation.validation_error.last",
        "validation.validation_error.mean",
    ]
    columns = bindings.get_GetProjectColumns(sess, id=1)

    column_values = {c.column for c in columns.columns}
    for hp in exp_hyperparameters:
        assert "hp." + hp in column_values
    for mc in exp_metrics:
        assert mc in column_values


@pytest.mark.e2e_cpu
def test_metrics_range_by_project() -> None:
    sess = api_utils.user_session()
    exp.run_basic_test(
        sess,
        conf.fixtures_path("core_api/arbitrary_workload_order.yaml"),
        conf.fixtures_path("core_api"),
        1,
        expect_workloads=True,
        expect_checkpoints=True,
    )
    ranges = bindings.get_GetProjectNumericMetricsRange(sess, id=1)

    assert ranges.ranges is not None
    for r in ranges.ranges:
        assert r.min <= r.max


@pytest.mark.e2e_cpu
def test_core_api_arbitrary_workload_order() -> None:
    sess = api_utils.user_session()
    experiment_id = exp.run_basic_test(
        sess,
        conf.fixtures_path("core_api/arbitrary_workload_order.yaml"),
        conf.fixtures_path("core_api"),
        1,
        expect_workloads=True,
        expect_checkpoints=True,
    )

    trials = exp.experiment_trials(sess, experiment_id)
    assert len(trials) == 1
    trial = trials[0]

    steps = exp.workloads_with_training(trial.workloads)
    assert len(steps) == 11
    validations = exp.workloads_with_validation(trial.workloads)
    assert len(validations) == 11
    checkpoints = exp.workloads_with_checkpoint(trial.workloads)
    assert len(checkpoints) == 11


@pytest.mark.e2e_cpu
@pytest.mark.parametrize(
    "stage,ntrials,expect_workloads,expect_checkpoints",
    [
        ("0_start", 1, False, False),
        ("1_metrics", 1, True, False),
        ("2_checkpoints", 1, True, True),
        ("3_hpsearch", 10, True, True),
    ],
)
def test_core_api_tutorials(
    stage: str, ntrials: int, expect_workloads: bool, expect_checkpoints: bool
) -> None:
    sess = api_utils.user_session()
    exp.run_basic_test(
        sess,
        conf.tutorials_path(f"core_api/{stage}.yaml"),
        conf.tutorials_path("core_api"),
        ntrials,
        expect_workloads=expect_workloads,
        expect_checkpoints=expect_checkpoints,
    )


@pytest.mark.parallel
def test_core_api_distributed_tutorial() -> None:
    sess = api_utils.user_session()
    exp.run_basic_test(
        sess, conf.tutorials_path("core_api/4_distributed.yaml"), conf.tutorials_path("core_api"), 1
    )


@pytest.mark.e2e_cpu
def test_core_api_pytorch_profiler_tensorboard() -> None:
    # Ensure tensorboard will load for an experiment which runs pytorch profiler,
    # and doesn't report metrics or checkpoints.
    # If the profiler trace file is not synced, the tensorboard will not load.
    sess = api_utils.user_session()
    exp_id = exp.run_basic_test(
        sess,
        conf.fixtures_path("core_api/pytorch_profiler_sync.yaml"),
        conf.fixtures_path("core_api"),
        1,
        expect_workloads=False,
        expect_checkpoints=False,
    )

    command = [
        "tensorboard",
        "start",
        str(exp_id),
        "--no-browser",
    ]

    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err
