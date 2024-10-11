import json
import pathlib
from typing import Any, Dict, Optional

import pytest

from determined.common import api, util
from determined.experimental import client
from tests import api_utils
from tests import command as cmd
from tests import detproc
from tests.experiment import noop

num_trials = 1


SHARED_FS_CONFIG = {
    "checkpoint_storage": {
        "type": "shared_fs",
        "host_path": "/tmp",
        "storage_path": "test-tensorboard",
    },
}


def s3_config(secrets: Dict[str, str], prefix: Optional[str] = None) -> Dict[str, Any]:
    config = {
        "checkpoint_storage": {
            "type": "s3",
            "access_key": secrets["INTEGRATIONS_S3_ACCESS_KEY"],
            "secret_key": secrets["INTEGRATIONS_S3_SECRET_KEY"],
            "bucket": secrets["INTEGRATIONS_S3_BUCKET"],
        },
    }
    if prefix is not None:
        config["checkpoint_storage"]["prefix"] = prefix

    return config


@pytest.mark.slow
@pytest.mark.e2e_cpu
def test_start_tensorboard_for_shared_fs_experiment(tmp_path: pathlib.Path) -> None:
    """
    Start a random experiment configured with the shared_fs backend, start a
    TensorBoard instance pointed to the experiment, and kill the TensorBoard
    instance.
    """
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, [noop.Report({"x": 1})], config=SHARED_FS_CONFIG)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    command = ["tensorboard", "start", str(exp_ref.id), "--no-browser"]
    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err


@pytest.mark.slow
@pytest.mark.e2e_gpu
@pytest.mark.tensorflow2
@pytest.mark.parametrize("prefix", [None, "my/test/prefix"])
def test_start_tensorboard_for_s3_experiment(
    tmp_path: pathlib.Path, secrets: Dict[str, str], prefix: Optional[str]
) -> None:
    """
    Start a random experiment configured with the s3 backend, start a
    TensorBoard instance pointed to the experiment, and kill the TensorBoard
    instance.
    """
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, [noop.Report({"x": 1})], config=s3_config(secrets))
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    command = ["tensorboard", "start", str(exp_ref.id), "--no-browser"]
    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err


@pytest.mark.slow
@pytest.mark.e2e_cpu
@pytest.mark.e2e_slurm
@pytest.mark.e2e_pbs
@pytest.mark.tensorflow2
def test_start_tensorboard_for_multi_experiment(
    tmp_path: pathlib.Path, secrets: Dict[str, str]
) -> None:
    """
    Start 3 random experiments configured with the s3 and shared_fs backends,
    start a TensorBoard instance pointed to the experiments and some select
    trials, and kill the TensorBoard instance.
    """
    sess = api_utils.user_session()
    exp1 = noop.create_experiment(sess, [noop.Report({"x": 1})], config=SHARED_FS_CONFIG)
    exp2 = noop.create_experiment(sess, [noop.Report({"x": 1})], config=s3_config(secrets))
    exp3 = noop.create_experiment(
        sess,
        [noop.Report({"x": 1})],
        config={
            **SHARED_FS_CONFIG,
            "searcher": {
                "name": "random",
                "max_trials": 2,
            },
        },
    )
    assert exp1.wait(interval=0.01) == client.ExperimentState.COMPLETED
    assert exp2.wait(interval=0.01) == client.ExperimentState.COMPLETED
    assert exp3.wait(interval=0.01) == client.ExperimentState.COMPLETED
    trial_ids = [str(t.id) for t in exp3.get_trials()]

    command = [
        "tensorboard",
        "start",
        str(exp1.id),
        str(exp2.id),
        str(exp3.id),
        "-t",
        *trial_ids,
        "--no-browser",
    ]

    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err


@pytest.mark.e2e_cpu
def test_start_tensorboard_with_custom_image() -> None:
    """
    Start a random experiment, start a TensorBoard instance pointed
    to the experiment with custom image, verify the image has been set.
    """
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess)
    command = [
        "det",
        "tensorboard",
        "start",
        str(exp_ref.id),
        "--no-browser",
        "--detach",
        "--config",
        "environment.image=python:3.8.16",
    ]
    t_id = detproc.check_output(sess, command).strip()
    command = ["det", "tensorboard", "config", t_id]
    res = detproc.check_output(sess, command)
    config = util.yaml_safe_load(res)
    assert (
        config["environment"]["image"]["cpu"] == "python:3.8.16"
        and config["environment"]["image"]["cuda"] == "python:3.8.16"
        and config["environment"]["image"]["rocm"] == "python:3.8.16"
    ), config
    # Kill experiment and tensorbaord.
    exp_ref.kill()
    detproc.check_call(sess, ["det", "tensorboard", "kill", t_id])


@pytest.mark.e2e_cpu
def test_tensorboard_inherit_image_pull_secrets() -> None:
    """
    Start a random experiment with image_pull_secrets, start a TensorBoard
    instance pointed to the experiment, verify the secrets are inherited.
    """
    sess = api_utils.user_session()
    exp_secrets = [{"name": "ips"}]
    config = {"environment": {"pod_spec": {"spec": {"imagePullSecrets": exp_secrets}}}}
    exp_ref = noop.create_experiment(sess, config=config)

    command = [
        "det",
        "tensorboard",
        "start",
        str(exp_ref.id),
        "--no-browser",
        "--detach",
    ]
    t_id = detproc.check_output(sess, command).strip()
    command = ["det", "tensorboard", "config", t_id]
    res = detproc.check_output(sess, command)
    config = util.yaml_safe_load(res)

    ips = config["environment"]["pod_spec"]["spec"]["imagePullSecrets"]
    assert ips == exp_secrets, (ips, exp_secrets)

    # Kill experiment and tensorbaord.
    exp_ref.kill()
    detproc.check_call(sess, ["det", "tensorboard", "kill", t_id])


@pytest.mark.e2e_cpu
def test_delete_tensorboard_for_experiment() -> None:
    """
    Start a random experiment, start a TensorBoard instance pointed to
    the experiment, delete tensorboard and verify deletion.
    """
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, [noop.Report({"x": 1})])
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    command = ["det", "e", "delete-tb-files", str(exp_ref.id)]
    detproc.check_call(sess, command)

    # Ensure Tensorboard files are deleted.
    assert exp_ref.config
    host_path = exp_ref.config["checkpoint_storage"]["host_path"]
    storage_path = exp_ref.config["checkpoint_storage"]["storage_path"]
    cluster_id = sess.get("info").json()["cluster_id"]
    tb_path = (
        pathlib.Path(host_path)
        / storage_path
        / cluster_id
        / "tensorboard"
        / "experiment"
        / str(exp_ref.id)
    )
    assert not tb_path.exists()


@pytest.mark.e2e_cpu
def test_tensorboard_directory_storage() -> None:
    sess = api_utils.user_session()
    bind_mounts = [{"host_path": "/tmp", "container_path": "/tmp/somepath"}]
    config = {
        "checkpoint_storage": {
            "type": "directory",
            "container_path": "/tmp/somepath",
        },
        "bind_mounts": bind_mounts,
    }
    exp_ref = noop.create_experiment(sess, [noop.Report({"x": 1})], config=config)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    command = [
        "tensorboard",
        "start",
        str(exp_ref.id),
        "--no-browser",
        f"--config=bind_mounts={json.dumps(bind_mounts)}",
    ]

    with cmd.interactive_command(sess, command) as tensorboard:
        assert tensorboard.task_id is not None
        err = api.wait_for_task_ready(sess, tensorboard.task_id)
        assert err is None, err
