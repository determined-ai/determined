import operator
import tempfile
import time
from typing import Dict, Set

import pytest
import yaml

from determined import errors
from determined.common import storage
from tests import config as conf
from tests import experiment as exp


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

                storage_manager = storage.build(checkpoint_config, container_path=None)
                for checkpoint in checkpoints:
                    storage_id = checkpoint["uuid"]
                    if checkpoint["state"] == "COMPLETED":
                        with storage_manager.restore_path(storage_id):
                            pass
                    elif checkpoint["state"] == "DELETED":
                        try:
                            with storage_manager.restore_path(storage_id):
                                raise AssertionError("checkpoint not deleted")
                        except errors.CheckpointNotFound:
                            pass
        except AssertionError:
            if i == max_checks - 1:
                raise
        else:
            break


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
