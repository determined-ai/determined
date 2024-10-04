import pathlib
import subprocess
import time

import pytest

from determined.common import api
from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils, detproc
from tests import experiment as exp
from tests.experiment import noop


@pytest.mark.e2e_cpu
def test_delete_experiment_removes_tensorboard_files() -> None:
    """
    Start a random experiment, delete the experiment and verify that TensorBoard files are deleted.
    """
    sess = api_utils.user_session()
    host_path = "/tmp"
    storage_path = "determined-integration-checkpoints"
    config = {
        "checkpoint_storage": {
            "type": "shared_fs",
            "host_path": host_path,
            "storage_path": storage_path,
        }
    }
    # TODO(CM-540): you should not need a checkpoint for tensorboard files to get deleted.
    exp_ref = noop.create_experiment(
        sess, [noop.Report({"x": 1}), noop.Checkpoint()], config=config
    )
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED

    cluster_id = sess.get("info").json()["cluster_id"]

    # Check if Tensorboard files are created
    tb_path = (
        pathlib.Path(host_path)
        / storage_path
        / cluster_id
        / "tensorboard"
        / "experiment"
        / str(exp_ref.id)
    )
    assert pathlib.Path(tb_path).exists()

    detproc.check_call(sess, ["det", "e", "delete", str(exp_ref.id), "--yes"])

    ticks = 60
    for i in range(ticks):
        try:
            state = exp.experiment_state(sess, exp_ref.id)
            if i % 5 == 0:
                print(f"experiment in state {state} waiting to be deleted")
            time.sleep(1)
        except api.errors.NotFoundException:
            # Check if Tensorboard files are deleted
            assert not pathlib.Path(tb_path).exists()
            return

    pytest.fail(f"experiment failed to be deleted after {ticks} seconds")


@pytest.mark.e2e_k8s
@pytest.mark.e2e_single_k8s
def test_cancel_experiment_remove_k8s_pod() -> None:
    # Start a random experiment.
    sess = api_utils.user_session()
    exp_ref = noop.create_experiment(sess, [noop.Sleep(100)])
    exp.wait_for_experiment_state(sess, exp_ref.id, bindings.experimentv1State.RUNNING)

    # Find the trial, and kill the pod.
    cmd = f'kubectl delete $(kubectl get pods -o name | grep -i "exp-{exp_ref.id}")'
    subprocess.run(cmd, shell=True)

    # Cancel the experiment
    detproc.check_call(sess, ["det", "e", "cancel", str(exp_ref.id)])

    # Assert that the experiment fails.
    assert exp.experiment_state(sess, exp_ref.id) == bindings.experimentv1State.ERROR
