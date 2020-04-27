import json
import operator
import os
import subprocess
import tempfile
import time
from typing import Dict, Set

import numpy as np
import pytest
import yaml

from determined.experimental import Determined
from tests.integrations import config as conf
from tests.integrations import experiment as exp
from tests.integrations.fixtures.metric_maker.metric_maker import (
    structure_equal,
    structure_to_metrics,
)


@pytest.mark.e2e_cpu  # type: ignore
def test_trial_error() -> None:
    exp.run_failure_test(
        conf.fixtures_path("trial_error/const.yaml"),
        conf.fixtures_path("trial_error"),
        "NotImplementedError",
    )


@pytest.mark.e2e_cpu  # type: ignore
def test_invalid_experiment() -> None:
    completed_process = exp.maybe_create_experiment(
        conf.fixtures_path("invalid_experiment/const.yaml"), conf.official_examples_path("mnist_tf")
    )
    assert completed_process.returncode != 0


@pytest.mark.e2e_cpu  # type: ignore
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

    batches_per_step = 100

    # Check training metrics.
    full_trial_metrics = exp.trial_metrics(trials[0].id)
    for step in full_trial_metrics.steps:
        metrics = step.metrics
        assert metrics["num_inputs"] == batches_per_step

        actual = metrics["batch_metrics"]
        assert len(actual) == batches_per_step

        first_base_value = base_value + (step.id - 1) * batches_per_step
        batch_values = first_base_value + gain_per_batch * np.arange(batches_per_step)
        expected = [structure_to_metrics(value, training_structure) for value in batch_values]
        assert structure_equal(expected, actual)

    # Check validation metrics.
    for step in trials[0].steps:
        validation = step.validation
        metrics = validation.metrics
        actual = metrics["validation_metrics"]

        value = base_value + step.id * batches_per_step
        expected = structure_to_metrics(value, validation_structure)
        assert structure_equal(expected, actual)


@pytest.mark.e2e_gpu  # type: ignore
def test_gc_checkpoints_s3(secrets: Dict[str, str]) -> None:
    config = exp.s3_checkpoint_config(secrets)
    run_gc_checkpoints_test(config)


@pytest.mark.e2e_cpu  # type: ignore
def test_gc_checkpoints_lfs() -> None:
    run_gc_checkpoints_test(exp.shared_fs_checkpoint_config())


def run_gc_checkpoints_test(checkpoint_storage: Dict[str, str]) -> None:
    fixtures = [
        (
            conf.fixtures_path("no_op/gc_checkpoints_decreasing.yaml"),
            {"COMPLETED": {8, 9, 10}, "DELETED": {1, 2, 3, 4, 5, 6, 7}},
        ),
        (
            conf.fixtures_path("no_op/gc_checkpoints_increasing.yaml"),
            {"COMPLETED": {1, 2, 3, 9, 10}, "DELETED": {4, 5, 6, 7, 8}},
        ),
    ]

    all_checkpoints = []
    for base_conf_path, result in fixtures:
        config = conf.load_config(str(base_conf_path))
        config["checkpoint_storage"].update(checkpoint_storage)

        with tempfile.NamedTemporaryFile() as tf:
            with open(tf.name, "w") as f:
                yaml.dump(config, f)

            experiment_id = exp.create_experiment(tf.name, conf.fixtures_path("no_op"))

        exp.wait_for_experiment_state(experiment_id, "COMPLETED")

        # Checkpoints are not marked as deleted until gc_checkpoint task starts.
        retries = 5
        for retry in range(retries):
            trials = exp.experiment_trials(experiment_id)
            assert len(trials) == 1

            checkpoints = sorted(
                (step.checkpoint for step in trials[0].steps), key=operator.itemgetter("step_id"),
            )
            assert len(checkpoints) == 10
            by_state = {}  # type: Dict[str, Set[int]]
            for checkpoint in checkpoints:
                by_state.setdefault(checkpoint.state, set()).add(checkpoint.step_id)

            if by_state == result:
                all_checkpoints.append((config, checkpoints))
                break

            if retry + 1 == retries:
                assert by_state == result

            time.sleep(1)

    # Check that the actual checkpoint storage (for shared_fs) reflects the
    # deletions. We want to wait for the GC containers to exit, so check
    # repeatedly with a timeout.
    max_checks = 30
    for check in range(max_checks):
        time.sleep(1)
        try:
            for config, checkpoints in all_checkpoints:
                checkpoint_config = config["checkpoint_storage"]

                if checkpoint_config["type"] == "shared_fs" and (
                    "storage_path" not in checkpoint_config
                ):
                    if "tensorboard_path" in checkpoint_config:
                        checkpoint_config["storage_path"] = checkpoint_config.get(
                            "tensorboard_path", None
                        )
                    else:
                        checkpoint_config["storage_path"] = checkpoint_config.get(
                            "checkpoint_path", None
                        )

                    root = os.path.join(
                        checkpoint_config["host_path"], checkpoint_config["storage_path"]
                    )

                    for checkpoint in checkpoints:
                        dirname = os.path.join(root, checkpoint.uuid)
                        if checkpoint.state == "COMPLETED":
                            assert os.path.isdir(dirname)
                        elif checkpoint.state == "DELETED":
                            assert not os.path.exists(dirname)
        except AssertionError:
            if check == max_checks - 1:
                raise
        else:
            break


@pytest.mark.e2e_cpu  # type: ignore
def test_experiment_delete() -> None:
    subprocess.check_call(["det", "-m", conf.make_master_url(), "user", "whoami"])

    experiment_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )

    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "experiment", "delete", str(experiment_id), "--yes"],
        env={**os.environ, "DET_ADMIN": "1"},
    )

    # "det experiment describe" call should fail, because the
    # experiment is no longer in the database.
    with pytest.raises(subprocess.CalledProcessError):
        subprocess.check_call(
            ["det", "-m", conf.make_master_url(), "experiment", "describe", str(experiment_id)]
        )


@pytest.mark.e2e_cpu  # type: ignore
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
    assert not infos[0]["archived"]

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
    assert infos[0]["archived"]

    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "experiment", "unarchive", str(experiment_id)]
    )
    infos = json.loads(subprocess.check_output(describe_args))
    assert len(infos) == 1
    assert not infos[0]["archived"]


@pytest.mark.e2e_cpu  # type: ignore
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
        conf.official_examples_path("mnist_pytorch"),
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


@pytest.mark.e2e_cpu  # type: ignore
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


@pytest.mark.e2e_cpu  # type: ignore
def test_end_to_end_adaptive() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("mnist_pytorch/adaptive_short.yaml"),
        conf.official_examples_path("mnist_pytorch"),
        None,
    )

    # Check that validation accuracy look sane (more than 93% on MNIST).
    trials = exp.experiment_trials(exp_id)
    best = None
    for trial in trials:
        assert len(trial.steps)
        last_step = trial.steps[-1]
        accuracy = last_step.validation.metrics["validation_metrics"]["accuracy"]
        if not best or accuracy > best:
            best = accuracy

    assert best is not None
    assert best > 0.93

    # Check that ExperimentReference returns a sorted order of top checkpoints
    # without gaps. The top 2 checkpoints should be the first 2 of the top k
    # checkpoints if sorting is stable.
    exp_ref = Determined(conf.make_master_url()).get_experiment(exp_id)

    top_2 = exp_ref.top_n_checkpoints(2)
    top_k = exp_ref.top_n_checkpoints(len(trials))

    top_2_uuids = [c.uuid for c in top_2]
    top_k_uuids = [c.uuid for c in top_k]

    assert top_2_uuids == top_k_uuids[:2]

    # Check that metrics are truly in sorted order.
    metrics = [c.validation.metrics["validation_metrics"]["validation_loss"] for c in top_k]

    assert metrics == sorted(metrics)

    # Check that changing smaller is better reverses the checkpoint ordering.
    top_k_reversed = exp_ref.top_n_checkpoints(
        len(trials), sort_by="validation_loss", smaller_is_better=False
    )
    top_k_reversed_uuids = [c.uuid for c in top_k_reversed]

    assert top_k_uuids == top_k_reversed_uuids[::-1]


@pytest.mark.e2e_cpu  # type: ignore
def test_log_null_bytes() -> None:
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["write_null"] = True
    config_obj["max_restarts"] = 0
    config_obj["searcher"]["max_steps"] = 1
    experiment_id = exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 1)

    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1
    logs = exp.trial_logs(trials[0].id)
    assert len(logs) > 0


@pytest.mark.e2e_gpu  # type: ignore
def test_s3_no_creds(secrets: Dict[str, str]) -> None:
    config = conf.load_config(conf.official_examples_path("mnist_pytorch/const.yaml"))
    config["checkpoint_storage"] = exp.s3_checkpoint_config_no_creds()
    config.setdefault("environment", {})
    config["environment"].setdefault("environment_variables", [])
    config["environment"]["environment_variables"] += [
        f"AWS_ACCESS_KEY_ID={secrets['INTEGRATIONS_S3_ACCESS_KEY']}",
        f"AWS_SECRET_ACCESS_KEY={secrets['INTEGRATIONS_S3_SECRET_KEY']}",
    ]
    exp.run_basic_test_with_temp_config(config, conf.official_examples_path("mnist_pytorch"), 1)


@pytest.mark.parallel  # type: ignore
def test_pytorch_parallel() -> None:
    config = conf.load_config(conf.official_examples_path("mnist_pytorch/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_native_parallel(config, False)
    config = conf.set_max_steps(config, 2)
    config = conf.set_tensor_auto_tuning(config, True)

    exp.run_basic_test_with_temp_config(config, conf.official_examples_path("mnist_pytorch"), 1)
