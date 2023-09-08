import pathlib
import subprocess
import time
import uuid
from typing import Any, Dict, Iterator, Tuple

import pytest

from determined.common import api
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests.cluster import managed_cluster, utils

DEVCLUSTER_CONFIG_ROOT_PATH = conf.PROJECT_ROOT_PATH.joinpath(".circleci/devcluster")
DEVCLUSTER_REATTACH_OFF_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double.devcluster.yaml"
DEVCLUSTER_REATTACH_ON_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double-reattach.devcluster.yaml"
DEVCLUSTER_LOG_PATH = pathlib.Path("/tmp/devcluster")
DEVCLUSTER_MASTER_LOG_PATH = DEVCLUSTER_LOG_PATH / "master.log"


@pytest.mark.managed_devcluster
def test_managed_cluster_working(
    restartable_managed_cluster: managed_cluster.ManagedCluster,
) -> None:
    try:
        restartable_managed_cluster.ensure_agent_ok()
        restartable_managed_cluster.kill_agent()
    finally:
        restartable_managed_cluster.restart_agent()


def _local_container_ids_with_labels() -> Iterator[Tuple[str, str]]:
    lines = (
        subprocess.check_output(["docker", "ps", "--format", "{{.ID}}\t{{.Labels}}"])
        .decode("utf-8")
        .strip()
        .split("\n")
    )
    for line in lines:
        container_id, *labels = line.strip().split("\t")
        yield container_id, (labels[0] if labels else "")


def _local_container_ids_for_experiment(exp_id: int) -> Iterator[str]:
    for container_id, labels in _local_container_ids_with_labels():
        if f"exp-{exp_id}" in labels:
            yield container_id


def _local_container_ids_for_command(command_id: str) -> Iterator[str]:
    for container_id, labels in _local_container_ids_with_labels():
        if f"cmd-{command_id}" in labels:
            yield container_id


def _task_list_json(sess: api.Session) -> Dict[str, Dict[str, Any]]:
    command = ["det", "task", "list", "--json"]
    tasks_data: Dict[str, Dict[str, Any]] = detproc.check_json(sess, command)
    return tasks_data


@pytest.mark.managed_devcluster
def test_agent_restart_exp_container_failure(
    restartable_managed_cluster: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    restartable_managed_cluster.ensure_agent_ok()
    try:
        exp_id = exp.create_experiment(
            sess,
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_workload_progress(sess, exp_id)
        container_ids = list(_local_container_ids_for_experiment(exp_id))
        if len(container_ids) != 1:
            pytest.fail(
                f"unexpected number of local containers for the experiment: {len(container_ids)}"
            )
        # Get task id / allocation id
        tasks_data = _task_list_json(sess)
        assert len(tasks_data) == 1
        exp_task_before = list(tasks_data.values())[0]

        restartable_managed_cluster.kill_agent()
        subprocess.check_call(["docker", "kill", container_ids[0]])
    except Exception:
        restartable_managed_cluster.restart_agent()
        raise
    else:
        restartable_managed_cluster.restart_agent()
        # As soon as the agent is back, the original allocation should be considered dead,
        # but the new one should be allocated.
        state = exp.experiment_state(sess, exp_id)
        # old STATE_ACTIVE got divided into three states
        assert state in [
            bindings.experimentv1State.ACTIVE,
            bindings.experimentv1State.QUEUED,
            bindings.experimentv1State.PULLING,
            bindings.experimentv1State.STARTING,
            bindings.experimentv1State.RUNNING,
        ]
        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.RUNNING)
        tasks_data = _task_list_json(sess)
        assert len(tasks_data) == 1
        exp_task_after = list(tasks_data.values())[0]

        assert exp_task_before["taskId"] == exp_task_after["taskId"]
        assert exp_task_before["allocationId"] != exp_task_after["allocationId"]

        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.COMPLETED)


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("command_duration", [10, 60])
def test_agent_restart_cmd_container_failure(
    restartable_managed_cluster: managed_cluster.ManagedCluster, command_duration: int
) -> None:
    sess = api_utils.user_session()
    # Launch a cmd, kill agent, wait for reconnect timeout, check it's not marked as success.
    # Reconnect timeout is ~25 seconds. We'd like to both test tasks that take
    # longer (60 seconds) and shorter (10 seconds) than that.
    restartable_managed_cluster.ensure_agent_ok()
    try:
        command_id = utils.run_command(sess, command_duration)
        utils.wait_for_command_state(sess, command_id, "RUNNING", 10)

        for _i in range(10):
            if len(list(_local_container_ids_for_command(command_id))) > 0:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Failed to find the command container id after {_i} ticks")

        restartable_managed_cluster.kill_agent()

        # Container should still be alive.
        assert list(_local_container_ids_for_command(command_id))
        for _i in range(command_duration + 10):
            if len(list(_local_container_ids_for_command(command_id))) == 0:
                break
            time.sleep(1)
        else:
            pytest.fail(f"command container didn't terminate after {_i} ticks")
        utils.wait_for_command_state(sess, command_id, "TERMINATED", 30)
        assert "success" not in utils.get_command_info(sess, command_id)["exitStatus"]
    except Exception:
        restartable_managed_cluster.restart_agent()
        raise
    else:
        restartable_managed_cluster.restart_agent()


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime, slots", [(0, 0), (20, 1), (60, 1)])
def test_agent_restart_recover_cmd(
    restartable_managed_cluster: managed_cluster.ManagedCluster, slots: int, downtime: int
) -> None:
    sess = api_utils.user_session()
    restartable_managed_cluster.ensure_agent_ok()
    try:
        command_id = utils.run_command(sess, 30, slots=slots)
        utils.wait_for_command_state(sess, command_id, "RUNNING", 10)

        restartable_managed_cluster.kill_agent()
        time.sleep(downtime)
        restartable_managed_cluster.restart_agent(wait_for_amnesia=False)

        utils.wait_for_command_state(sess, command_id, "TERMINATED", 30)

        # If the reattach_wait <= downtime, master would have considered agent
        # to be dead marking the experiment fail. We can ignore such scenarios.
        # We only need to check if the command succeeded when
        # reattach_wait > downtime, which ensures that the agent would have
        # reconnected in time.
        reattach_wait = restartable_managed_cluster.fetch_config_reattach_wait()
        if reattach_wait > downtime:
            utils.assert_command_succeeded(sess, command_id)
    except Exception:
        restartable_managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_agent_restart_recover_experiment(
    restartable_managed_cluster: managed_cluster.ManagedCluster, downtime: int
) -> None:
    sess = api_utils.user_session()
    restartable_managed_cluster.ensure_agent_ok()
    try:
        exp_id = exp.create_experiment(
            sess,
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_workload_progress(sess, exp_id)

        if downtime >= 0:
            restartable_managed_cluster.kill_agent()
            time.sleep(downtime)
            restartable_managed_cluster.restart_agent(wait_for_amnesia=False)

        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.COMPLETED)
        trials = exp.experiment_trials(sess, exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        restartable_managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_agent_reconnect_keep_experiment(
    restartable_managed_cluster: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    try:
        exp_id = exp.create_experiment(
            sess,
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_workload_progress(sess, exp_id)

        restartable_managed_cluster.kill_proxy()
        time.sleep(1)
        restartable_managed_cluster.restart_proxy()

        exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.COMPLETED)
        trials = exp.experiment_trials(sess, exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        restartable_managed_cluster.restart_proxy(wait_for_reconnect=False)
        restartable_managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_agent_reconnect_keep_cmd(
    restartable_managed_cluster: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    try:
        command_id = utils.run_command(sess, 20, slots=0)
        utils.wait_for_command_state(sess, command_id, "RUNNING", 10)

        restartable_managed_cluster.kill_proxy()
        time.sleep(1)
        restartable_managed_cluster.restart_proxy()

        utils.wait_for_command_state(sess, command_id, "TERMINATED", 30)

        utils.assert_command_succeeded(sess, command_id)
    except Exception:
        restartable_managed_cluster.restart_proxy(wait_for_reconnect=False)
        restartable_managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_agent_reconnect_trigger_schedule(
    restartable_managed_cluster: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    restartable_managed_cluster.ensure_agent_ok()

    try:
        restartable_managed_cluster.kill_proxy()
        command_id = utils.run_command(sess, 5, slots=1)
        restartable_managed_cluster.restart_proxy()
        utils.wait_for_command_state(sess, command_id, "TERMINATED", 10)

        utils.assert_command_succeeded(sess, command_id)
    except Exception:
        restartable_managed_cluster.restart_proxy(wait_for_reconnect=False)
        restartable_managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_queued_experiment_restarts_with_correct_allocation_id(
    restartable_managed_cluster: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    exp_id = exp.create_experiment(
        sess,
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "resources.slots_per_trial=9999"],
    )
    exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.QUEUED)

    restartable_managed_cluster.kill_master()
    log_marker = str(uuid.uuid4())
    restartable_managed_cluster.log_marker(log_marker)
    restartable_managed_cluster.restart_master()

    err = 'duplicate key value violates unique constraint \\"allocations_allocation_id_key\\"'
    past_marker = False
    with open(DEVCLUSTER_MASTER_LOG_PATH) as lf:
        for line in lf.readlines():
            if not past_marker:
                if log_marker in line:
                    past_marker = True
                continue

            if err in line:
                pytest.fail(f"allocation id save failure after restart: {line}")
