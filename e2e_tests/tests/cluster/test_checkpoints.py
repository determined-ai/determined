import json
import os
import random
import sys
import tempfile
import time
from typing import Any, Dict, List, Set, Tuple

import pytest

from determined import errors
from determined.common import api, storage, util
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp

EXPECT_TIMEOUT = 5


def wait_for_gc_to_finish(sess: api.Session, experiment_ids: List[int]) -> None:
    seen_gc_experiment_ids = set()
    done_gc_experiment_ids = set()
    # Don't wait longer than 5 minutes (as 600 half-seconds to improve our sampling resolution).
    for _ in range(600):
        r = sess.get("tasks").json()
        names = [task["name"] for task in r.values()]

        for experiment_id in experiment_ids:
            gc_name = f"Checkpoint GC (Experiment {experiment_id})"
            if gc_name in names:
                seen_gc_experiment_ids.add(experiment_id)
            elif experiment_id in seen_gc_experiment_ids:
                # We saw the gc before but now don't so we assume it is done.
                done_gc_experiment_ids.add(experiment_id)

        if len(done_gc_experiment_ids) == len(experiment_ids):
            return
        time.sleep(0.5)

    # It's possible that it ran really fast and we missed it, so just log this.
    print("Did not observe checkpoint gc start or finish!", file=sys.stderr)


@pytest.mark.e2e_gpu
@pytest.mark.e2e_slurm_gpu
def test_set_gc_policy() -> None:
    sess = api_utils.user_session()
    exp_id = exp.run_basic_test(
        sess,
        config_file=conf.fixtures_path("no_op/gc_checkpoints_decreasing.yaml"),
        model_def_file=conf.fixtures_path("no_op"),
        expected_trials=1,
    )

    config = conf.load_config(str(conf.fixtures_path("no_op/gc_checkpoints_decreasing.yaml")))
    save_exp_best = config["checkpoint_storage"]["save_experiment_best"]
    save_trial_latest = config["checkpoint_storage"]["save_trial_latest"]
    save_trial_best = 1  # default because not set in this config

    # Command that uses the same gc policy as initial policy used for the experiment.
    run_command_gc_policy(
        sess, str(save_exp_best), str(save_trial_latest), str(save_trial_best), str(exp_id)
    )

    # Command that uses a diff gc policy from the initial policy used for the experiment.
    save_exp_best = 0
    save_trial_latest = 1
    save_trial_best = 1
    run_command_gc_policy(
        sess, str(save_exp_best), str(save_trial_latest), str(save_trial_best), str(exp_id)
    )


def run_command_gc_policy(
    sess: api.Session, save_exp_best: str, save_trial_latest: str, save_trial_best: str, exp_id: str
) -> None:
    command = [
        "det",
        "e",
        "set",
        "gc-policy",
        "--yes",
        "--save-experiment-best",
        str(save_exp_best),
        "--save-trial-best",
        str(save_trial_best),
        "--save-trial-latest",
        str(save_trial_latest),
        str(exp_id),
    ]
    detproc.check_output(sess, command)


def run_command_master_checkpoint_download(sess: api.Session, uuid: str) -> None:
    with tempfile.TemporaryDirectory() as dirpath:
        outdir = dirpath + "/checkpoint"
        command = [
            "det",
            "checkpoint",
            "download",
            "--mode",
            "master",
            "--output-dir",
            outdir,
            uuid,
        ]

        detproc.check_call(sess, command)
        assert os.path.exists(outdir + "/metadata.json")


@pytest.mark.e2e_gpu
def test_gc_checkpoints(checkpoint_storage_config: Dict[str, Any]) -> None:
    run_gc_checkpoints_test(checkpoint_storage_config)


@pytest.mark.e2e_cpu
def test_gc_checkpoints_lfs() -> None:
    run_gc_checkpoints_test(exp.shared_fs_checkpoint_config())


@pytest.mark.e2e_cpu
def test_delete_checkpoints() -> None:
    sess = api_utils.user_session()
    base_conf_path = conf.fixtures_path("no_op/single-default-ckpt.yaml")

    config = conf.load_config(str(base_conf_path))
    config["checkpoint_storage"] = {
        "type": "shared_fs",
        "host_path": "/tmp",
        "storage_path": "delete-checkpoints-e2etest",
        "save_trial_latest": 10,
    }
    config["min_checkpoint_period"] = {"batches": 10}

    exp_id_1 = exp.run_basic_test_with_temp_config(
        sess, config, model_def_path=conf.fixtures_path("no_op"), expected_trials=1
    )

    exp_id_2 = exp.run_basic_test_with_temp_config(
        sess, config, model_def_path=conf.fixtures_path("no_op"), expected_trials=1
    )

    sess = api_utils.user_session()
    exp_1_checkpoints = bindings.get_GetExperimentCheckpoints(session=sess, id=exp_id_1).checkpoints
    exp_2_checkpoints = bindings.get_GetExperimentCheckpoints(session=sess, id=exp_id_2).checkpoints
    assert len(exp_1_checkpoints) > 0, f"no checkpoints found in experiment with ID:{exp_id_1}"
    assert len(exp_2_checkpoints) > 0, f"no checkpoints found in experiment with ID:{exp_id_2}"

    assert all(
        checkpoint.storageId is not None
        for checkpoint in list(exp_1_checkpoints) + list(exp_2_checkpoints)
    ), "Storage ID cannot be None for any checkpoints in both experiments"

    d_exp_1_checkpoint_uuids = [
        exp_1_checkpoints[d_index].uuid
        for d_index in random.sample(range(len(exp_1_checkpoints)), 2)
    ]
    d_exp_2_checkpoint_uuids = [
        exp_2_checkpoints[d_index].uuid
        for d_index in random.sample(range(len(exp_2_checkpoints)), 2)
    ]

    d_checkpoint_uuids = d_exp_1_checkpoint_uuids + d_exp_2_checkpoint_uuids
    print(f"checkpoints uuids to be deleteted: {d_checkpoint_uuids}")
    # ensure checkpoint directories exist:
    checkpoint_config = config["checkpoint_storage"]
    storage_manager = storage.build(checkpoint_config, container_path=None)

    for uuid in d_checkpoint_uuids:
        try:
            storage_manager.restore_path(uuid)
        except errors.CheckpointNotFound:
            pytest.fail(f"checkpoint directory with uuid: {uuid} was not created.")

    delete_body = bindings.v1DeleteCheckpointsRequest(checkpointUuids=d_checkpoint_uuids)
    bindings.delete_DeleteCheckpoints(session=sess, body=delete_body)

    wait_for_gc_to_finish(sess, [exp_id_1, exp_id_2])

    for d_c in d_checkpoint_uuids:
        ensure_checkpoint_deleted(sess, d_c, storage_manager)


def ensure_checkpoint_deleted(sess: Any, d_checkpoint_uuid: Any, storage_manager: Any) -> None:
    d_checkpoint = bindings.get_GetCheckpoint(
        session=sess, checkpointUuid=d_checkpoint_uuid
    ).checkpoint

    if d_checkpoint is not None:
        assert (
            d_checkpoint.state == bindings.checkpointv1State.DELETED
        ), f"checkpoint with uuid {d_checkpoint_uuid} does not have a deleted state"
    else:
        pytest.fail(
            f"Failed to get checkpoint with uuid {d_checkpoint_uuid} to validate correct deletion"
        )
    checkpoint_file = os.path.join(storage_manager._base_path, d_checkpoint_uuid)

    if os.path.exists(checkpoint_file):
        raise AssertionError(f"Checkpoint file with path {checkpoint_file} was not deleted")


def run_gc_checkpoints_test(checkpoint_storage: Dict[str, str]) -> None:
    sess = api_utils.user_session()
    fixtures = [
        (
            conf.fixtures_path("no_op/gc_checkpoints_decreasing.yaml"),
            {
                (bindings.experimentv1State.COMPLETED.value): {800, 900, 1000},
                (bindings.experimentv1State.DELETED.value): {
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
                (bindings.experimentv1State.COMPLETED.value): {
                    100,
                    200,
                    300,
                    900,
                    1000,
                },
                (bindings.experimentv1State.DELETED.value): {
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
        experiment_storage = checkpoint_storage.copy()
        experiment_storage.update(config["checkpoint_storage"])
        config["checkpoint_storage"].update(experiment_storage)

        with tempfile.NamedTemporaryFile() as tf:
            with open(tf.name, "w") as f:
                util.yaml_safe_dump(config, f)

            experiment_id = exp.create_experiment(sess, tf.name, conf.fixtures_path("no_op"))

        exp.wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.COMPLETED)

        # In some configurations, checkpoint GC will run on an auxillary machine, which may have to
        # be spun up still.  So we'll wait for it to run.
        wait_for_gc_to_finish(sess, [experiment_id])

        # Checkpoints are not marked as deleted until gc_checkpoint task starts.
        retries = 5
        for retry in range(retries):
            trials = exp.experiment_trials(sess, experiment_id)
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

    # Check that the actual checkpoint storage reflects the
    # deletions. We want to wait for the GC containers to exit, so check
    # repeatedly with a timeout.
    max_checks = 30
    last_checkpoint_uuid = None
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
                    last_checkpoint_uuid = storage_id = checkpoint.uuid
                    storage_state[storage_id] = {}
                    if checkpoint.state == bindings.checkpointv1State.COMPLETED:
                        storage_state[storage_id]["found"] = False
                        try:
                            with storage_manager.restore_path(storage_id):
                                storage_state[storage_id]["found"] = True
                        except errors.CheckpointNotFound:
                            pass
                    elif checkpoint.state == bindings.checkpointv1State.DELETED:
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

    cs_type = checkpoint_storage["type"]
    if cs_type != "azure":
        assert type(last_checkpoint_uuid) == str
        run_command_master_checkpoint_download(sess, str(last_checkpoint_uuid))


@pytest.mark.e2e_gpu
def test_s3_no_creds(secrets: Dict[str, str]) -> None:
    pytest.skip("Temporarily skipping this until we find a more secure way of testing this.")
    sess = api_utils.user_session()
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    config["checkpoint_storage"] = exp.s3_checkpoint_config_no_creds()
    config.setdefault("environment", {})
    config["environment"].setdefault("environment_variables", [])
    config["environment"]["environment_variables"] += [
        f"AWS_ACCESS_KEY_ID={secrets['INTEGRATIONS_S3_ACCESS_KEY']}",
        f"AWS_SECRET_ACCESS_KEY={secrets['INTEGRATIONS_S3_SECRET_KEY']}",
    ]
    exp.run_basic_test_with_temp_config(sess, config, conf.tutorials_path("mnist_pytorch"), 1)


@pytest.mark.e2e_cpu
def test_delete_experiment_with_no_checkpoints() -> None:
    sess = api_utils.user_session()
    # Experiment will intentionally fail.
    config = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config["checkpoint_storage"] = {
        "type": "s3",
        "bucket": "dettestthisbucketdoesntexist",
    }
    config["max_restarts"] = 0
    exp_id = exp.run_failure_test_with_temp_config(
        sess,
        config,
        conf.fixtures_path("no_op"),
        None,
    )

    # Still able to delete this since it will have no checkpoints meaning no checkpoint gc task.
    bindings.delete_DeleteExperiment(session=sess, experimentId=exp_id)
    ticks = 60
    for i in range(ticks):
        try:
            state = exp.experiment_state(sess, exp_id)
            if i % 5 == 0:
                print(f"experiment in state {state} waiting to be deleted")
            time.sleep(1)
        except api.errors.NotFoundException:
            return

    pytest.fail(f"experiment failed to be deleted after {ticks} seconds")


@pytest.mark.e2e_cpu
def test_checkpoint_partial_delete() -> None:
    sess = api_utils.user_session()
    base_conf_path = conf.fixtures_path("no_op/single-default-ckpt.yaml")

    host_path = "/tmp"
    storage_path = "partial-delete-checkpoints-e2etest"
    config = conf.load_config(str(base_conf_path))
    config["checkpoint_storage"] = {
        "type": "shared_fs",
        "host_path": host_path,
        "storage_path": storage_path,
        "save_trial_latest": 10,
    }
    config["min_checkpoint_period"] = {"batches": 10}

    exp_id = exp.run_basic_test_with_temp_config(
        sess, config, model_def_path=conf.fixtures_path("no_op"), expected_trials=1
    )

    checkpoints = bindings.get_GetExperimentCheckpoints(
        session=sess,
        id=exp_id,
    ).checkpoints
    completed_checkpoints = []
    for c in checkpoints:
        if c.state == bindings.checkpointv1State.COMPLETED:
            completed_checkpoints.append(c)
            if len(completed_checkpoints) >= 2:
                break
    else:
        pytest.fail("did not find two checkpoints in state completed")

    s = bindings.get_GetExperiment(
        sess,
        experimentId=exp_id,
    ).experiment.checkpointSize
    assert s is not None
    starting_size = int(s)

    def assert_checkpoint_state(
        uuid: str,
        exp_size: int,
        trial_size: int,
        resources: Dict[str, Any],
        state: bindings.checkpointv1State,
    ) -> None:
        s = bindings.get_GetExperiment(
            sess,
            experimentId=exp_id,
        ).experiment.checkpointSize
        assert s is not None and int(s) == exp_size

        trials = bindings.get_GetExperimentTrials(sess, experimentId=exp_id).trials
        assert len(trials) == 1
        assert (
            trials[0].totalCheckpointSize is not None
            and int(trials[0].totalCheckpointSize) == trial_size
        )

        ckpt = bindings.get_GetCheckpoint(
            sess,
            checkpointUuid=uuid,
        ).checkpoint
        assert ckpt.resources == resources
        assert ckpt.state == state
        assert ckpt.storageId is not None

    # Unchanged and empty glob causes checkpoint state says the same.
    remove_body = bindings.v1CheckpointsRemoveFilesRequest(
        checkpointGlobs=[],
        checkpointUuids=[completed_checkpoints[0].uuid],
    )
    bindings.post_CheckpointsRemoveFiles(sess, body=remove_body)
    wait_for_gc_to_finish(sess, [exp_id])

    assert_checkpoint_state(
        completed_checkpoints[0].uuid,
        starting_size,
        starting_size,
        completed_checkpoints[0].resources,
        bindings.checkpointv1State.COMPLETED,
    )

    # Delete from shared_fs with no glob => update state.
    # metadata.json is being creatd somehow. This is the difference we are getting.
    new_resources = {}
    new_size = starting_size
    for file_name, size in completed_checkpoints[0].resources.items():
        if "pkl" in file_name:
            os.remove(f"{host_path}/{storage_path}/{completed_checkpoints[0].uuid}/{file_name}")
            new_size -= int(size)
        else:
            new_resources[file_name] = size

    remove_body = bindings.v1CheckpointsRemoveFilesRequest(
        checkpointGlobs=[],
        checkpointUuids=[completed_checkpoints[0].uuid],
    )
    bindings.post_CheckpointsRemoveFiles(sess, body=remove_body)
    wait_for_gc_to_finish(sess, [exp_id])

    assert_checkpoint_state(
        completed_checkpoints[0].uuid,
        new_size,
        new_size,
        new_resources,
        bindings.checkpointv1State.PARTIALLY_DELETED,
    )

    # Competly delete checkpoint.
    new_size -= sum(int(s) for s in new_resources.values())
    # new_resources stays the same since we don't delete resources when we delete checkpoints..
    remove_body = bindings.v1CheckpointsRemoveFilesRequest(
        checkpointGlobs=["**/*"],
        checkpointUuids=[completed_checkpoints[0].uuid],
    )
    bindings.post_CheckpointsRemoveFiles(sess, body=remove_body)
    wait_for_gc_to_finish(sess, [exp_id])

    assert_checkpoint_state(
        completed_checkpoints[0].uuid,
        new_size,
        new_size,
        new_resources,
        bindings.checkpointv1State.DELETED,
    )

    # Matching glob => update state and trials.
    new_resources = {}
    for file_name, size in completed_checkpoints[1].resources.items():
        if "pkl" in file_name:
            new_size -= int(size)
        else:
            new_resources[file_name] = size

    remove_body = bindings.v1CheckpointsRemoveFilesRequest(
        checkpointGlobs=["**/*.pkl"],
        checkpointUuids=[completed_checkpoints[1].uuid],
    )
    bindings.post_CheckpointsRemoveFiles(sess, body=remove_body)
    wait_for_gc_to_finish(sess, [exp_id])

    assert_checkpoint_state(
        completed_checkpoints[1].uuid,
        new_size,
        new_size,
        new_resources,
        bindings.checkpointv1State.PARTIALLY_DELETED,
    )


@pytest.mark.e2e_cpu
def test_fail_on_chechpoint_save() -> None:
    sess = api_utils.user_session()
    error_log = "failed on checkpoint save"
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["fail_on_chechpoint_save"] = error_log
    exp.run_failure_test_with_temp_config(
        sess,
        config_obj,
        conf.fixtures_path("no_op"),
        error_log,
    )


@pytest.mark.e2e_cpu
def test_fail_on_preclose_chechpoint_save() -> None:
    sess = api_utils.user_session()
    error_log = "failed on checkpoint save"
    config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
    config_obj["hyperparameters"]["fail_on_chechpoint_save"] = error_log
    config_obj["searcher"]["max_length"] = {"batches": 1}
    config_obj["min_validation_period"] = {"batches": 1}
    config_obj["max_restarts"] = 1
    exp.run_failure_test_with_temp_config(
        sess,
        config_obj,
        conf.fixtures_path("no_op"),
        error_log,
    )
