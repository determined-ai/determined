import json
import os
import subprocess
import time
from typing import Any, Dict, Iterator, List, Tuple, cast

import pytest

from tests import config as conf
from tests import experiment as exp

from .utils import get_command_info, run_command, wait_for_command_state

DEVCLUSTER_CONFIG_PATH = conf.PROJECT_ROOT_PATH.joinpath(
    ".circleci/devcluster/double.devcluster.yaml"
)


def _get_agent_data(master_url: str) -> List[Dict[str, Any]]:
    command = ["det", "-m", master_url, "agent", "list", "--json"]
    output = subprocess.check_output(command).decode()
    agent_data = cast(List[Dict[str, Any]], json.loads(output))
    return agent_data


class ManagedCluster:
    # This utility wrapper uses double agent yaml configurations,
    # but provides helpers to run/kill a single agent setup.

    def __init__(self) -> None:
        # Strategically only import devcluster on demand to avoid having it as a hard dependency.
        from devcluster import Devcluster

        self.dc = Devcluster(config=str(DEVCLUSTER_CONFIG_PATH))
        self.master_url = conf.make_master_url()

    def __enter__(self) -> "ManagedCluster":
        self.old_cd = os.getcwd()
        os.chdir(str(conf.PROJECT_ROOT_PATH))
        self.dc.__enter__()
        return self

    def __exit__(self, *_: Any) -> None:
        os.chdir(self.old_cd)
        self.dc.__exit__(*_)

    def initial_startup(self) -> None:
        self.dc.set_target("agent1", wait=True, timeout=3 * 60)

    def kill_agent(self) -> None:
        self.dc.kill_stage("agent1")

        WAIT_FOR_KILL = 5
        for _i in range(WAIT_FOR_KILL):
            agent_data = _get_agent_data(self.master_url)
            if len(agent_data) == 0:
                break
            if len(agent_data) == 1 and agent_data[0]["draining"] is True:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Agent is still present after {WAIT_FOR_KILL} seconds")

    def restart_agent(self) -> None:
        agent_data = _get_agent_data(self.master_url)
        if len(agent_data) == 1 and agent_data[0]["enabled"]:
            return

        # Currently, we've got to wait for master to "forget" the agent before reconnecting.
        WAIT_FOR_AMNESIA = 60
        for _i in range(WAIT_FOR_AMNESIA):
            agent_data = _get_agent_data(self.master_url)
            if len(agent_data) == 0:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Agent is still not forgotten after {WAIT_FOR_AMNESIA} seconds")

        self.dc.restart_stage("agent1", wait=True, timeout=10)

        WAIT_FOR_STARTUP = 10
        for _i in range(WAIT_FOR_STARTUP):
            agent_data = _get_agent_data(self.master_url)
            if (
                len(agent_data) == 1
                and agent_data[0]["enabled"] is True
                and agent_data[0]["draining"] is False
            ):
                break
            time.sleep(1)
        else:
            pytest.fail(f"Agent didn't restart after {WAIT_FOR_STARTUP} seconds")

    def ensure_agent_ok(self) -> None:
        agent_data = _get_agent_data(self.master_url)
        assert len(agent_data) == 1
        assert agent_data[0]["enabled"] is True
        assert agent_data[0]["draining"] is False


@pytest.fixture(scope="session")
def managed_cluster() -> Iterator[ManagedCluster]:
    with ManagedCluster() as mc:
        mc.initial_startup()
        yield mc


@pytest.mark.managed_devcluster
def test_managed_cluster_working(managed_cluster: ManagedCluster) -> None:
    try:
        managed_cluster.ensure_agent_ok()
        managed_cluster.kill_agent()
    finally:
        managed_cluster.restart_agent()


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
        if f"/experiments/{exp_id}/" in labels:
            yield container_id


def _local_container_ids_for_command(command_id: str) -> Iterator[str]:
    for container_id, labels in _local_container_ids_with_labels():
        if f"/commands/{command_id}/" in labels:
            yield container_id


def _task_list_json(master_url: str) -> Dict[str, Dict[str, Any]]:
    command = ["det", "-m", master_url, "task", "list", "--json"]
    tasks_data: Dict[str, Dict[str, Any]] = json.loads(subprocess.check_output(command).decode())
    return tasks_data


@pytest.mark.managed_devcluster
def test_agent_restart_exp_container_failure(managed_cluster: ManagedCluster) -> None:
    managed_cluster.ensure_agent_ok()
    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_workload_progress(exp_id)
        container_ids = list(_local_container_ids_for_experiment(exp_id))
        if len(container_ids) != 1:
            pytest.fail(
                f"unexpected number of local containers for the experiment: {len(container_ids)}"
            )
        # Get task id / allocation id
        tasks_data = _task_list_json(managed_cluster.master_url)
        assert len(tasks_data) == 1
        exp_task_before = list(tasks_data.values())[0]

        managed_cluster.kill_agent()
        subprocess.check_call(["docker", "kill", container_ids[0]])
    except Exception:
        managed_cluster.restart_agent()
        raise
    else:
        managed_cluster.restart_agent()
        # As soon as the agent is back, the original allocation should be considered dead,
        # but the new one should be allocated.
        state = exp.experiment_state(exp_id)
        assert state == "ACTIVE"
        tasks_data = _task_list_json(managed_cluster.master_url)
        assert len(tasks_data) == 1
        exp_task_after = list(tasks_data.values())[0]

        assert exp_task_before["task_id"] == exp_task_after["task_id"]
        assert exp_task_before["allocation_id"] != exp_task_after["allocation_id"]

        exp.wait_for_experiment_state(exp_id, "COMPLETED")


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("command_duration", [10, 20, 60])
def test_agent_restart_cmd_container_failure(
    managed_cluster: ManagedCluster, command_duration: int
) -> None:
    # Launch a cmd, kill agent, wait for reconnect timeout, check it's not marked as success.
    # Reconnect timeout is ~25 seconds. We'd like to both test tasks that take
    # longer (60 seconds) and shorter (10 seconds) than that.
    # I've also added the (20 seconds) run for extra insurance in case of some
    # flakiness of (10 second) run.
    managed_cluster.ensure_agent_ok()
    try:
        command_id = run_command(command_duration)
        wait_for_command_state(command_id, "RUNNING", 10)

        for _i in range(10):
            if len(list(_local_container_ids_for_command(command_id))) > 0:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Failed to find the command container id after {_i} ticks")

        managed_cluster.kill_agent()

        # Container should still be alive.
        assert list(_local_container_ids_for_command(command_id))
        for _i in range(60):
            if len(list(_local_container_ids_for_command(command_id))) == 0:
                break
            time.sleep(1)
        else:
            pytest.fail(f"command container didn't terminate after {_i} ticks")
        wait_for_command_state(command_id, "TERMINATED", 30)
        assert "success" not in get_command_info(command_id)["exitStatus"]
    except Exception:
        managed_cluster.restart_agent()
        raise
    else:
        managed_cluster.restart_agent()
