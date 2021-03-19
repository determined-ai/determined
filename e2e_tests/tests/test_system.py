import json
import operator
import subprocess
import tempfile
import time
from typing import Dict, Set

import botocore.exceptions
import numpy as np
import pytest
import yaml

from determined.common import check, storage
from determined.experimental import Determined, ModelSortBy
from tests import config as conf
from tests import experiment as exp
from tests.fixtures.metric_maker.metric_maker import structure_equal, structure_to_metrics


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
        conf.fixtures_path("invalid_experiment/const.yaml"), conf.cv_examples_path("mnist_tf")
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

    scheduling_unit = 100

    # Check training metrics.
    full_trial_metrics = exp.trial_metrics(trials[0]["id"])
    for step in full_trial_metrics["steps"]:
        metrics = step["metrics"]
        assert metrics["num_inputs"] == scheduling_unit

        actual = metrics["batch_metrics"]
        assert len(actual) == scheduling_unit

        first_base_value = base_value + (step["id"] - 1) * scheduling_unit
        batch_values = first_base_value + gain_per_batch * np.arange(scheduling_unit)
        expected = [structure_to_metrics(value, training_structure) for value in batch_values]
        assert structure_equal(expected, actual)

    # Check validation metrics.
    for step in trials[0]["steps"]:
        validation = step["validation"]
        metrics = validation["metrics"]
        actual = metrics["validation_metrics"]

        value = base_value + step["id"] * scheduling_unit
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
            {"COMPLETED": {800, 900, 1000}, "DELETED": {100, 200, 300, 400, 500, 600, 700}},
        ),
        (
            conf.fixtures_path("no_op/gc_checkpoints_increasing.yaml"),
            {"COMPLETED": {100, 200, 300, 900, 1000}, "DELETED": {400, 500, 600, 700, 800}},
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
                (step["checkpoint"] for step in trials[0]["steps"]),
                key=operator.itemgetter("total_batches"),
            )
            assert len(checkpoints) == 10
            by_state = {}  # type: Dict[str, Set[int]]
            for checkpoint in checkpoints:
                by_state.setdefault(checkpoint["state"], set()).add(checkpoint["total_batches"])

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
    for i in range(max_checks):
        time.sleep(1)
        try:
            for config, checkpoints in all_checkpoints:
                checkpoint_config = config["checkpoint_storage"]

                if checkpoint_config["type"] == "shared_fs":
                    deleted_exception = check.CheckFailedError
                elif checkpoint_config["type"] == "s3":
                    deleted_exception = botocore.exceptions.ClientError
                else:
                    raise NotImplementedError(
                        f'unsupported storage type {checkpoint_config["type"]}'
                    )

                storage_manager = storage.build(checkpoint_config, container_path=None)
                for checkpoint in checkpoints:
                    metadata = storage.StorageMetadata.from_json(checkpoint)
                    if checkpoint["state"] == "COMPLETED":
                        with storage_manager.restore_path(metadata):
                            pass
                    elif checkpoint["state"] == "DELETED":
                        try:
                            with storage_manager.restore_path(metadata):
                                raise AssertionError("checkpoint not deleted")
                        except deleted_exception:
                            pass
        except AssertionError:
            if i == max_checks - 1:
                raise
        else:
            break


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


@pytest.mark.e2e_cpu  # type: ignore
def test_trial_logs() -> None:
    experiment_id = exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    trial_id = exp.experiment_trials(experiment_id)[0]["id"]
    subprocess.check_call(["det", "-m", conf.make_master_url(), "trial", "logs", str(trial_id)])
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "trial", "logs", "--head", "10", str(trial_id)],
    )
    subprocess.check_call(
        ["det", "-m", conf.make_master_url(), "trial", "logs", "--tail", "10", str(trial_id)],
    )


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
        conf.tutorials_path("mnist_pytorch"),
        None,
    )

    # Check that validation accuracy look sane (more than 93% on MNIST).
    trials = exp.experiment_trials(exp_id)
    best = None
    for trial in trials:
        assert len(trial["steps"])
        last_step = trial["steps"][-1]
        accuracy = last_step["validation"]["metrics"]["validation_metrics"]["accuracy"]
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
    metrics = [c.validation["metrics"]["validationMetrics"]["validation_loss"] for c in top_k]

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
    assert checkpoint.metadata == {"testing": "metadata"}
    assert checkpoint.metadata == db_check.metadata

    checkpoint.add_metadata({"some_key": "some_value"})
    db_check = d.get_checkpoint(checkpoint.uuid)
    assert checkpoint.metadata == {"testing": "metadata", "some_key": "some_value"}
    assert checkpoint.metadata == db_check.metadata

    checkpoint.add_metadata({"testing": "override"})
    db_check = d.get_checkpoint(checkpoint.uuid)
    assert checkpoint.metadata == {"testing": "override", "some_key": "some_value"}
    assert checkpoint.metadata == db_check.metadata

    checkpoint.remove_metadata(["some_key"])
    db_check = d.get_checkpoint(checkpoint.uuid)
    assert checkpoint.metadata == {"testing": "override"}
    assert checkpoint.metadata == db_check.metadata


@pytest.mark.e2e_cpu  # type: ignore
def test_model_registry() -> None:
    exp_id = exp.run_basic_test(
        conf.fixtures_path("mnist_pytorch/const-pytorch11.yaml"),
        conf.tutorials_path("mnist_pytorch"),
        None,
    )

    d = Determined(conf.make_master_url())

    mnist = d.create_model("mnist", "simple computer vision model")
    assert mnist.metadata == {}

    mnist.add_metadata({"testing": "metadata"})
    db_model = d.get_model("mnist")
    # Make sure the model metadata is correct and correctly saved to the db.
    assert mnist.metadata == db_model.metadata
    assert mnist.metadata == {"testing": "metadata"}

    mnist.add_metadata({"some_key": "some_value"})
    db_model = d.get_model("mnist")
    assert mnist.metadata == db_model.metadata
    assert mnist.metadata == {"testing": "metadata", "some_key": "some_value"}

    mnist.add_metadata({"testing": "override"})
    db_model = d.get_model("mnist")
    assert mnist.metadata == db_model.metadata
    assert mnist.metadata == {"testing": "override", "some_key": "some_value"}

    mnist.remove_metadata(["some_key"])
    db_model = d.get_model("mnist")
    assert mnist.metadata == db_model.metadata
    assert mnist.metadata == {"testing": "override"}

    checkpoint = d.get_experiment(exp_id).top_checkpoint()
    model_version = mnist.register_version(checkpoint.uuid)

    assert model_version.model_version == 1

    latest_version = mnist.get_version()
    assert latest_version is not None
    assert latest_version.uuid == checkpoint.uuid

    d.create_model("transformer", "all you need is attention")
    d.create_model("object-detection", "a bounding box model")

    models = d.get_models(sort_by=ModelSortBy.NAME)
    assert [m.name for m in models] == ["mnist", "object-detection", "transformer"]


@pytest.mark.e2e_cpu  # type: ignore
def test_log_null_bytes() -> None:
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["write_null"] = True
    config_obj["max_restarts"] = 0
    config_obj["searcher"]["max_length"] = {"batches": 1}
    experiment_id = exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 1)

    trials = exp.experiment_trials(experiment_id)
    assert len(trials) == 1
    logs = exp.trial_logs(trials[0]["id"])
    assert len(logs) > 0


@pytest.mark.e2e_cpu  # type: ignore
def test_graceful_trial_termination() -> None:
    config_obj = conf.load_config(conf.fixtures_path("no_op/grid-graceful-trial-termination.yaml"))
    exp.run_basic_test_with_temp_config(config_obj, conf.fixtures_path("no_op"), 2)


@pytest.mark.e2e_gpu  # type: ignore
def test_s3_no_creds(secrets: Dict[str, str]) -> None:
    pytest.skip("Temporarily skipping this until we find a more secure way of testing this.")
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config["checkpoint_storage"] = exp.s3_checkpoint_config_no_creds()
    config.setdefault("environment", {})
    config["environment"].setdefault("environment_variables", [])
    config["environment"]["environment_variables"] += [
        f"AWS_ACCESS_KEY_ID={secrets['INTEGRATIONS_S3_ACCESS_KEY']}",
        f"AWS_SECRET_ACCESS_KEY={secrets['INTEGRATIONS_S3_SECRET_KEY']}",
    ]
    exp.run_basic_test_with_temp_config(config, conf.tutorials_path("mnist_pytorch"), 1)


@pytest.mark.parallel  # type: ignore
def test_pytorch_parallel() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_tensor_auto_tuning(config, True)
    config = conf.set_perform_initial_validation(config, True)

    exp_id = exp.run_basic_test_with_temp_config(
        config, conf.tutorials_path("mnist_pytorch"), 1, has_zeroth_step=True
    )
    exp.assert_performed_initial_validation(exp_id)


@pytest.mark.e2e_cpu  # type: ignore
def test_fail_on_first_validation() -> None:
    error_log = "failed on first validation"
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["fail_on_first_validation"] = error_log
    exp.run_failure_test_with_temp_config(
        config_obj,
        conf.fixtures_path("no_op"),
        error_log,
    )


@pytest.mark.e2e_cpu  # type: ignore
def test_fail_on_chechpoint_save() -> None:
    error_log = "failed on checkpoint save"
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["fail_on_chechpoint_save"] = error_log
    exp.run_failure_test_with_temp_config(
        config_obj,
        conf.fixtures_path("no_op"),
        error_log,
    )


@pytest.mark.e2e_cpu  # type: ignore
def test_fail_on_preclose_chechpoint_save() -> None:
    error_log = "failed on checkpoint save"
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["fail_on_chechpoint_save"] = error_log
    config_obj["searcher"]["max_length"] = {"batches": 1}
    config_obj["min_validation_period"] = {"batches": 1}
    config_obj["max_restarts"] = 1
    exp.run_failure_test_with_temp_config(
        config_obj,
        conf.fixtures_path("no_op"),
        error_log,
    )


@pytest.mark.e2e_cpu  # type: ignore
def test_perform_initial_validation() -> None:
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config = conf.set_max_length(config, {"batches": 1})
    config = conf.set_perform_initial_validation(config, True)
    exp_id = exp.run_basic_test_with_temp_config(
        config, conf.fixtures_path("no_op"), 1, has_zeroth_step=True
    )
    exp.assert_performed_initial_validation(exp_id)


@pytest.mark.parallel  # type: ignore
def test_distributed_logging() -> None:
    config = conf.load_config(conf.fixtures_path("pytorch_no_op/const.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_max_length(config, {"batches": 1})

    e_id = exp.run_basic_test_with_temp_config(config, conf.fixtures_path("pytorch_no_op"), 1)
    t_id = exp.experiment_trials(e_id)[0]["id"]

    for i in range(config["resources"]["slots_per_trial"]):
        assert exp.check_if_string_present_in_trial_logs(
            t_id, "finished train_batch for rank {}".format(i)
        )


@pytest.mark.e2e_cpu  # type: ignore
def test_disable_and_enable_slots() -> None:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "slot",
        "list",
        "--json",
    ]
    output = subprocess.check_output(command).decode()
    slots = json.loads(output)
    assert len(slots) == 1

    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "slot",
        "disable",
        slots[0]["agent_id"],
        slots[0]["slot_id"],
    ]
    subprocess.check_call(command)

    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "slot",
        "enable",
        slots[0]["agent_id"],
        slots[0]["slot_id"],
    ]
    subprocess.check_call(command)
