import json
import os
import random
import sys
import tempfile
import time
from typing import Any, Dict, List, Set, Tuple

import pytest
import yaml

from determined import errors
from determined.common import api, storage
from determined.common.api import authentication, bindings, certs
from determined.common.api.bindings import determinedcheckpointv1State
from tests import config as conf
from tests import experiment as exp


def wait_for_gc_to_finish(experiment_id: int) -> None:
    certs.cli_cert = certs.default_load(conf.make_master_url())
    authentication.cli_auth = authentication.Authentication(conf.make_master_url(), try_reauth=True)
    saw_gc = False
    # Don't wait longer than 5 minutes (as 600 half-seconds to improve our sampling resolution).
    for _ in range(600):
        r = api.get(conf.make_master_url(), "tasks").json()
        names = [task["name"] for task in r.values()]
        gc_name = f"Checkpoint GC (Experiment {experiment_id})"
        if gc_name in names:
            saw_gc = True
        elif saw_gc:
            # We previously saw checkpoint gc but now we don't, so it must have finished.
            return
        time.sleep(0.5)

    # It's possible that it ran really fast and we missed it, so just log this.
    print("Did not observe checkpoint gc start or finish!", file=sys.stderr)


@pytest.mark.e2e_gpu
def test_gc_checkpoints_s3(secrets: Dict[str, str]) -> None:
    config = exp.s3_checkpoint_config(secrets)
    run_gc_checkpoints_test(config)


@pytest.mark.e2e_cpu
def test_gc_checkpoints_lfs() -> None:
    run_gc_checkpoints_test(exp.shared_fs_checkpoint_config())


@pytest.mark.e2e_cpu
def test_delete_checkpoints() -> None:
    base_conf_path = conf.fixtures_path("no_op/single-default-ckpt.yaml")
    config = conf.load_config(str(base_conf_path))
    checkpoint_storage_delete = {
        "type": "shared_fs",
        "host_path": "/tmp",
        "storage_path": "delete-checkpoints-e2etest",
    }
    config["checkpoint_storage"] = {}
    config["checkpoint_storage"].update(checkpoint_storage_delete)
    config["min_checkpoint_period"] = {}
    config["min_checkpoint_period"]["batches"] = 10

    exp_id = exp.run_basic_test_with_temp_config(
        config, model_def_path=conf.fixtures_path("no_op"), expected_trials=1
    )

    t_s = exp.test_session()
    exp_checkpoints = bindings.get_GetExperimentCheckpoints(session=t_s, id=exp_id).checkpoints

    assert len(exp_checkpoints) > 0
    print(f"len of checkpoints: {len(exp_checkpoints)}")

    d_index = random.randint(0, len(exp_checkpoints) - 1)
    d_CheckpointUuid = exp_checkpoints[d_index].uuid
    delete_body = bindings.v1DeleteCheckpointsRequest(checkpointUuids=[d_CheckpointUuid])
    bindings.delete_DeleteCheckpoints(session=t_s, body=delete_body)
    wait_for_gc_to_finish(exp_id)

    d_checkpoint = bindings.get_GetCheckpoint(
        session=t_s, checkpointUuid=d_CheckpointUuid
    ).checkpoint
    if d_checkpoint is not None:
        assert d_checkpoint.state == determinedcheckpointv1State.STATE_DELETED
    else:
        raise Exception("Failed to get checkpoint to validate correct deletion")

    checkpoint_config = config["checkpoint_storage"]
    storage_manager = storage.build(checkpoint_config, container_path=None)
    checkpoint_file = os.path.join(storage_manager._base_path, d_CheckpointUuid)

    if os.path.exists(checkpoint_file):
        raise AssertionError(f"Checkpoint file with path {checkpoint_file} was not deleted")


def run_gc_checkpoints_test(checkpoint_storage: Dict[str, str]) -> None:
    fixtures = [
        (
            conf.fixtures_path("no_op/gc_checkpoints_decreasing.yaml"),
            {
                (bindings.determinedexperimentv1State.STATE_COMPLETED.value): {800, 900, 1000},
                (bindings.determinedexperimentv1State.STATE_DELETED.value): {
                    100,
                    200,
                    300,
                    400,
                    500,
                    600,
                    700,
                },
            },
        ),
        (
            conf.fixtures_path("no_op/gc_checkpoints_increasing.yaml"),
            {
                (bindings.determinedexperimentv1State.STATE_COMPLETED.value): {
                    100,
                    200,
                    300,
                    900,
                    1000,
                },
                (bindings.determinedexperimentv1State.STATE_DELETED.value): {
                    400,
                    500,
                    600,
                    700,
                    800,
                },
            },
        ),
    ]

    all_checkpoints: List[Tuple[Any, List[bindings.v1CheckpointWorkload]]] = []
    for base_conf_path, result in fixtures:
        config = conf.load_config(str(base_conf_path))
        config["checkpoint_storage"].update(checkpoint_storage)

        with tempfile.NamedTemporaryFile() as tf:
            with open(tf.name, "w") as f:
                yaml.dump(config, f)

            experiment_id = exp.create_experiment(tf.name, conf.fixtures_path("no_op"))

        exp.wait_for_experiment_state(
            experiment_id, bindings.determinedexperimentv1State.STATE_COMPLETED
        )

        # In some configurations, checkpoint GC will run on an auxillary machine, which may have to
        # be spun up still.  So we'll wait for it to run.
        wait_for_gc_to_finish(experiment_id)

        # Checkpoints are not marked as deleted until gc_checkpoint task starts.
        retries = 5
        for retry in range(retries):
            trials = exp.experiment_trials(experiment_id)
            assert len(trials) == 1

            cpoints = exp.workloads_with_checkpoint(trials[0].workloads)
            sorted_checkpoints = sorted(
                cpoints,
                key=lambda ckp: int(ckp.totalBatches),
            )
            assert len(sorted_checkpoints) == 10
            by_state = {}  # type: Dict[str, Set[int]]
            for ckpt in sorted_checkpoints:
                by_state.setdefault(ckpt.state.value, set()).add(ckpt.totalBatches)

            if by_state == result:
                all_checkpoints.append((config, sorted_checkpoints))
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
            storage_states = []
            for config, checkpoints in all_checkpoints:
                checkpoint_config = config["checkpoint_storage"]

                storage_manager = storage.build(checkpoint_config, container_path=None)
                storage_state = {}  # type: Dict[str, Any]
                for checkpoint in checkpoints:
                    assert checkpoint.uuid is not None
                    storage_id = checkpoint.uuid
                    storage_state[storage_id] = {}
                    if checkpoint.state == bindings.determinedcheckpointv1State.STATE_COMPLETED:
                        storage_state[storage_id]["found"] = False
                        try:
                            with storage_manager.restore_path(storage_id):
                                storage_state[storage_id]["found"] = True
                        except errors.CheckpointNotFound:
                            pass
                    elif checkpoint.state == bindings.determinedcheckpointv1State.STATE_DELETED:
                        storage_state[storage_id] = {"deleted": False, "checkpoint": checkpoint}
                        try:
                            with storage_manager.restore_path(storage_id):
                                pass
                        except errors.CheckpointNotFound:
                            storage_state[storage_id]["deleted"] = True
                        storage_states.append(storage_state)

            for storage_state in storage_states:
                for state in storage_state.values():
                    if state.get("deleted", None) is False:
                        json_states = json.dumps(storage_states)
                        raise AssertionError(
                            f"Some checkpoints were not deleted: JSON:{json_states}"
                        )
                    if state.get("found", None) is False:
                        json_states = json.dumps(storage_states)
                        raise AssertionError(f"Some checkpoints were not found: JSON:{json_states}")
        except AssertionError:
            if i == max_checks - 1:
                raise
        else:
            break


@pytest.mark.e2e_gpu
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


@pytest.mark.e2e_cpu
def test_fail_on_chechpoint_save() -> None:
    error_log = "failed on checkpoint save"
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["fail_on_chechpoint_save"] = error_log
    exp.run_failure_test_with_temp_config(
        config_obj,
        conf.fixtures_path("no_op"),
        error_log,
    )


@pytest.mark.e2e_cpu
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
