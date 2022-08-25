import json
import subprocess
import tempfile

import numpy as np
import pytest

from determined.common import yaml
from determined.experimental import Determined
from tests import config as conf
from tests import experiment as exp
from tests.cluster.test_checkpoints import wait_for_gc_to_finish
from tests.fixtures.metric_maker.metric_maker import structure_equal, structure_to_metrics


@pytest.mark.e2e_cpu
def test_trial_error() -> None:
    exp.run_failure_test(
        conf.fixtures_path("trial_error/const.yaml"),
        conf.fixtures_path("trial_error"),
        "NotImplementedError",
    )


@pytest.mark.e2e_cpu
def test_invalid_experiment() -> None:
    completed_process = exp.maybe_create_experiment(
        conf.fixtures_path("invalid_experiment/const.yaml"), conf.cv_examples_path("mnist_tf")
    )
    assert completed_process.returncode != 0


@pytest.mark.e2e_cpu
def test_metric_gathering() -> None:
    """
    Confirm that metrics are gathered from the trial the way that we expect.
    """
    experiment_id = exp.run_basic_test(
        conf.fixtures_path("metric_maker/const.yaml"), conf.fixtures_path("metric_maker"), 1
    )

    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1

    # Read the structure of the metrics directly from the config file
    config = conf.load_config(conf.fixtures_path("metric_maker/const.yaml"))

    base_value = config["hyperparameters"]["starting_base_value"]
    gain_per_batch = config["hyperparameters"]["gain_per_batch"]
    training_structure = config["hyperparameters"]["training_structure"]["val"]
    validation_structure = config["hyperparameters"]["validation_structure"]["val"]

    scheduling_unit = 100

    # Check training metrics.
    full_trial_metrics = exp.trial_metrics(trials[0].trial.id)
    batches_trained = 0
    for step in full_trial_metrics["steps"]:
        metrics = step["metrics"]

        actual = metrics["batch_metrics"]
        assert len(actual) == scheduling_unit

        first_base_value = base_value + batches_trained
        batch_values = first_base_value + gain_per_batch * np.arange(scheduling_unit)
        expected = [structure_to_metrics(value, training_structure) for value in batch_values]
        assert structure_equal(expected, actual)
        batches_trained = step["total_batches"]

    # Check validation metrics.
    validation_workloads = exp.workloads_with_validation(trials[0].workloads)
    for validation in validation_workloads:
        actual = validation.metrics.avgMetrics
        batches_trained = validation.totalBatches

        value = base_value + batches_trained
        expected = structure_to_metrics(value, validation_structure)
        assert structure_equal(expected, actual)


@pytest.mark.e2e_cpu
def test_nan_metrics() -> None:
    """
    Confirm that NaN and Infinity metrics are gathered from the trial.
    """
    exp_id = exp.run_basic_test(
        conf.fixtures_path("metric_maker/nans.yaml"), conf.fixtures_path("metric_maker"), 1
    )
    trials = exp.experiment_trials(exp_id)
    config = conf.load_config(conf.fixtures_path("metric_maker/nans.yaml"))
    base_value = config["hyperparameters"]["starting_base_value"]
    gain_per_batch = config["hyperparameters"]["gain_per_batch"]

    # Infinity and NaN cannot be processed in the YAML->JSON deserializer
    # Add them to expected values here
    training_structure = config["hyperparameters"]["training_structure"]["val"]
    training_structure["inf"] = "Infinity"
    training_structure["nan"] = "NaN"
    training_structure["nanarray"] = ["NaN", "NaN"]
    validation_structure = config["hyperparameters"]["validation_structure"]["val"]
    validation_structure["neg_inf"] = "-Infinity"

    # Check training metrics.
    full_trial_metrics = exp.trial_metrics(trials[0].trial.id)
    batches_trained = 0
    for step in full_trial_metrics["steps"]:
        metrics = step["metrics"]
        actual = metrics["batch_metrics"]
        first_base_value = base_value + batches_trained
        batch_values = first_base_value + gain_per_batch * np.arange(5)
        expected = [structure_to_metrics(value, training_structure) for value in batch_values]
        assert structure_equal(expected, actual)
        batches_trained = step["total_batches"]

    # Check validation metrics.
    validation_workloads = exp.workloads_with_validation(trials[0].workloads)
    for validation in validation_workloads:
        actual = validation.metrics.avgMetrics
        batches_trained = validation.totalBatches
        expected = structure_to_metrics(base_value, validation_structure)
        assert structure_equal(expected, actual)


@pytest.mark.e2e_cpu
def test_experiment_archive_unarchive() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), ["--paused"]
    )

    describe_args = [
        "det",
        "-m",
        conf.make_master_url(),
        "experiment",
        "describe",
        "--json",
        str(experiment_id),
    ]

    # Check that the experiment is initially unarchived.
    infos = json.loads(subprocess.check_output(describe_args))
    assert len(infos) == 1
    assert not infos[0]["experiment"]["archived"]

    # Check that archiving a non-terminal experiment fails, then terminate it.
    with pytest.raises(subprocess.CalledProcessError):
        subprocess.check_call(
            ["det", "-m", conf.make_master_url(), "experiment", "archive", str(experiment_id)]
        )
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "experiment", "cancel", str(experiment_id)]
    )

    # Check that we can archive and unarchive the experiment and see the expected effects.
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "experiment", "archive", str(experiment_id)]
    )
    infos = json.loads(subprocess.check_output(describe_args))
    assert len(infos) == 1
    assert infos[0]["experiment"]["archived"]

    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "experiment", "unarchive", str(experiment_id)]
    )
    infos = json.loads(subprocess.check_output(describe_args))
    assert len(infos) == 1
    assert not infos[0]["experiment"]["archived"]


@pytest.mark.e2e_cpu
def test_create_test_mode() -> None:
    # test-mode should succeed with a valid experiment.
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "experiment",
        "create",
        "--test-mode",
        conf.fixtures_path("mnist_pytorch/adaptive_short.yaml"),
        conf.tutorials_path("mnist_pytorch"),
    ]
    output = subprocess.check_output(command, universal_newlines=True)
    assert "Model definition test succeeded" in output

    # test-mode should fail when an error is introduced into the trial
    # implementation.
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "experiment",
        "create",
        "--test-mode",
        conf.fixtures_path("trial_error/const.yaml"),
        conf.fixtures_path("trial_error"),
    ]
    with pytest.raises(subprocess.CalledProcessError):
        subprocess.check_call(command)


@pytest.mark.e2e_cpu
def test_trial_logs() -> None:
    experiment_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0].trial.id
    subprocess.check_call(["det", "-m", conf.make_master_url(), "trial", "logs", str(trial_id)])
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "trial", "logs", "--head", "10", str(trial_id)],
    )
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "trial", "logs", "--tail", "10", str(trial_id)],
    )


@pytest.mark.e2e_cpu
def test_labels() -> None:
    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-one-short-step.yaml"), conf.fixtures_path("no_op"), None
    )

    label = "__det_test_dummy_label__"

    # Add a label and check that it shows up.
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "e", "label", "add", str(experiment_id), label]
    )
    output = subprocess.check_output(
        ["det", "-m", conf.make_master_url(), "e", "describe", str(experiment_id)]
    ).decode()
    assert label in output

    # Remove the label and check that it doesn't show up.
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "e", "label", "remove", str(experiment_id), label]
    )
    output = subprocess.check_output(
        ["det", "-m", conf.make_master_url(), "e", "describe", str(experiment_id)]
    ).decode()
    assert label not in output


@pytest.mark.e2e_cpu
def test_end_to_end_adaptive() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("mnist_pytorch/adaptive_short.yaml"),
        conf.tutorials_path("mnist_pytorch"),
        None,
    )

    wait_for_gc_to_finish(experiment_id=exp_id)

    # Check that validation accuracy look sane (more than 93% on MNIST).
    trials = exp.experiment_trials(exp_id)
    best = None
    for trial in trials:
        assert len(trial.workloads) > 0
        last_validation = exp.workloads_with_validation(trial.workloads)[-1]
        accuracy = last_validation.metrics.avgMetrics["accuracy"]
        if not best or accuracy > best:
            best = accuracy

    assert best is not None
    assert best > 0.93

    # Check that ExperimentReference returns a sorted order of top checkpoints
    # without gaps. The top 2 checkpoints should be the first 2 of the top k
    # checkpoints if sorting is stable.
    d = Determined(conf.make_master_url())
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
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["write_null"] = True
    config_obj["max_restarts"] = 0
    config_obj["searcher"]["max_length"] = {"batches": 1}
    experiment_id = exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 1)

    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1
    logs = exp.trial_logs(trials[0].trial.id)
    assert len(logs) > 0


@pytest.mark.e2e_cpu
def test_graceful_trial_termination() -> None:
    config_obj = conf.load_config(conf.fixtures_path("no_op/grid-graceful-trial-termination.yaml"))
    exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 2)


@pytest.mark.e2e_cpu
def test_fail_on_first_validation() -> None:
    error_log = "failed on first validation"
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["fail_on_first_validation"] = error_log
    exp.run_failure_test_with_temp_config(
        config_obj,
        conf.fixtures_path("no_op"),
        error_log,
    )


@pytest.mark.e2e_cpu
def test_perform_initial_validation() -> None:
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config = conf.set_max_length(config, {"batches": 1})
    config = conf.set_perform_initial_validation(config, True)
    exp_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("no_op"), 1)
    exp.assert_performed_initial_validation(exp_id)


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
    exp.run_basic_test(
        conf.tutorials_path(f"core_api/{stage}.yaml"),
        conf.tutorials_path("core_api"),
        ntrials,
        expect_workloads=expect_workloads,
        expect_checkpoints=expect_checkpoints,
    )


@pytest.mark.parallel
def test_core_api_distributed_tutorial() -> None:
    exp.run_basic_test(
        conf.tutorials_path("core_api/4_distributed.yaml"), conf.tutorials_path("core_api"), 1
    )


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
            yaml.dump(config_obj, f)
        experiment_id = exp.create_experiment(tf.name, conf.fixtures_path("no_op"), [])

    try:
        exp.wait_for_experiment_active_workload(experiment_id)
        trials = exp.wait_for_at_least_n_trials(experiment_id, 1)
        assert len(trials) == 1, trials

        for t in trials:
            exp.cancel_trial(t.trial.id)

        # Give the experiment time to refill max_concurrent_trials.
        trials = exp.wait_for_at_least_n_trials(experiment_id, 2)

        # The experiment handling the cancel message and waiting for it to be cancelled slyly
        # (hackishly) allows us to synchronize with the experiment state after after canceling
        # the first two trials.
        exp.cancel_single(experiment_id)

        # Make sure that there were never more than 2 total trials created.
        trials = exp.wait_for_at_least_n_trials(experiment_id, 2)
        assert len(trials) == 2, trials

    finally:
        exp.cancel_single(experiment_id)
