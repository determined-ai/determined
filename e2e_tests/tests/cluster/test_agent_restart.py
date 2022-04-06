import json
import os
import subprocess
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, Iterator, List, Tuple, Union, cast

import pytest

from determined.common.api.bindings import determinedexperimentv1State
from tests import config as conf
from tests import experiment as exp

from .utils import get_command_info, run_command, wait_for_command_state

DEVCLUSTER_CONFIG_ROOT_PATH = conf.PROJECT_ROOT_PATH.joinpath(".circleci/devcluster")
DEVCLUSTER_REATTACH_OFF_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double.devcluster.yaml"
DEVCLUSTER_REATTACH_ON_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double-reattach.devcluster.yaml"
DEVCLUSTER_LOG_PATH = Path("/tmp/devcluster")


def _get_agent_data(master_url: str) -> List[Dict[str, Any]]:
    command = ["det", "-m", master_url, "agent", "list", "--json"]
    output = subprocess.check_output(command).decode()
    agent_data = cast(List[Dict[str, Any]], json.loads(output))
    return agent_data


class ManagedCluster:
    # This utility wrapper uses double agent yaml configurations,
    # but provides helpers to run/kill a single agent setup.

    def __init__(self, config: Union[str, Dict[str, Any]], reattach: bool) -> None:
        # Strategically only import devcluster on demand to avoid having it as a hard dependency.
        from devcluster import Devcluster

        self.dc = Devcluster(config=config)
        self.master_url = conf.make_master_url()
        self.reattach = reattach

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

    def restart_agent(self, wait_for_amnesia: bool = True) -> None:
        agent_data = _get_agent_data(self.master_url)
        if len(agent_data) == 1 and agent_data[0]["enabled"]:
            return

        if wait_for_amnesia:
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

    def kill_proxy(self) -> None:
        subprocess.run(["killall", "socat"])

    def restart_proxy(self, wait_for_reconnect: bool = True) -> None:
        self.dc.restart_stage("proxy")
        if wait_for_reconnect:
            for _i in range(25):
                agent_data = _get_agent_data(self.master_url)
                if (
                    len(agent_data) == 1
                    and agent_data[0]["enabled"] is True
                    and agent_data[0]["draining"] is False
                ):
                    break
                time.sleep(1)
            else:
                pytest.fail(f"Agent didn't reconnect after {_i} seconds")

    def ensure_agent_ok(self) -> None:
        agent_data = _get_agent_data(self.master_url)
        assert len(agent_data) == 1
        assert agent_data[0]["enabled"] is True
        assert agent_data[0]["draining"] is False

    def fetch_config(self) -> Dict:
        master_config = json.loads(
            subprocess.run(
                ["det", "-m", self.master_url, "master", "config", "--json"],
                stdout=subprocess.PIPE,
                check=True,
            ).stdout.decode()
        )
        return cast(Dict, master_config)

    def fetch_config_reattach_wait(self) -> float:
        s = self.fetch_config()["resource_pools"][0]["agent_reconnect_wait"]
        return float(s.rstrip("s"))

    def log_marker(self, marker: str) -> None:
        for log_path in DEVCLUSTER_LOG_PATH.glob("*.log"):
            with log_path.open("a") as fout:
                fout.write(marker)


@pytest.fixture(scope="session", params=[False, True], ids=["reattach-off", "reattach-on"])
def managed_cluster_session(request: Any) -> Iterator[ManagedCluster]:
    reattach = cast(bool, request.param)
    if reattach:
        config = str(DEVCLUSTER_REATTACH_ON_CONFIG_PATH)
    else:
        config = str(DEVCLUSTER_REATTACH_OFF_CONFIG_PATH)

    with ManagedCluster(config, reattach=reattach) as mc:
        mc.initial_startup()
        yield mc


@pytest.fixture
def managed_cluster(
    managed_cluster_session: ManagedCluster, request: Any
) -> Iterator[ManagedCluster]:
    ts = datetime.now(timezone.utc).astimezone().isoformat()
    nodeid = request.node.nodeid
    managed_cluster_session.log_marker(f"pytest [{ts}] {nodeid} setup\n")
    yield managed_cluster_session
    managed_cluster_session.log_marker(f"pytest [{ts}] {nodeid} teardown\n")


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
        subprocess.run(["docker", "kill", container_ids[0]], check=True, stdout=subprocess.PIPE)
    except Exception:
        managed_cluster.restart_agent()
        raise
    else:
        managed_cluster.restart_agent()
        # As soon as the agent is back, the original allocation should be considered dead,
        # but the new one should be allocated.
        state = exp.experiment_state(exp_id)
        assert state == determinedexperimentv1State.STATE_ACTIVE
        tasks_data = _task_list_json(managed_cluster.master_url)
        assert len(tasks_data) == 1
        exp_task_after = list(tasks_data.values())[0]

        assert exp_task_before["task_id"] == exp_task_after["task_id"]
        assert exp_task_before["allocation_id"] != exp_task_after["allocation_id"]

        exp.wait_for_experiment_state(exp_id, determinedexperimentv1State.STATE_COMPLETED)


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("command_duration", [10, 60])
def test_agent_restart_cmd_container_failure(
    managed_cluster: ManagedCluster, command_duration: int
) -> None:
    # Launch a cmd, kill agent, wait for reconnect timeout, check it's not marked as success.
    # Reconnect timeout is ~25 seconds. We'd like to both test tasks that take
    # longer (60 seconds) and shorter (10 seconds) than that.
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
        for _i in range(command_duration + 10):
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


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("command_duration", [20])
def test_agent_noreattach_restart_kills(
    managed_cluster: ManagedCluster, command_duration: int
) -> None:
    if managed_cluster.reattach:
        pytest.skip()

    managed_cluster.ensure_agent_ok()
    try:
        command_id = run_command(command_duration)
        wait_for_command_state(command_id, "RUNNING", 10)

        for _i in range(10):
            container_ids = list(_local_container_ids_for_command(command_id))
            if len(container_ids) > 0:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Failed to find the command container id after {_i} ticks")

        managed_cluster.kill_agent()
        managed_cluster.restart_agent(wait_for_amnesia=False)

        # That command container should be killed right away.
        for _i in range(3):
            container_ids = list(_local_container_ids_for_command(command_id))
            if len(container_ids) == 0:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Command container wasn't killed after {_i} ticks")

        assert "success" not in get_command_info(command_id)["exitStatus"]
    except Exception:
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("slots", [0, 1])
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_agent_restart_recover_cmd(
    managed_cluster: ManagedCluster, slots: int, downtime: int
) -> None:
    if not managed_cluster.reattach:
        pytest.skip()

    managed_cluster.ensure_agent_ok()
    try:
        command_id = run_command(30, slots=slots)
        wait_for_command_state(command_id, "RUNNING", 10)

        managed_cluster.kill_agent()
        time.sleep(downtime)
        managed_cluster.restart_agent(wait_for_amnesia=False)

        wait_for_command_state(command_id, "TERMINATED", 30)

        # Commands fail if they have finished while the agent was off.
        reattach_wait = managed_cluster.fetch_config_reattach_wait()
        succeeded = "success" in get_command_info(command_id)["exitStatus"]
        assert succeeded is (reattach_wait > downtime)
    except Exception:
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [-1, 0, 20, 60])
def test_agent_restart_recover_experiment(managed_cluster: ManagedCluster, downtime: int) -> None:
    if not managed_cluster.reattach:
        pytest.skip()

    managed_cluster.ensure_agent_ok()
    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_workload_progress(exp_id)

        if downtime >= 0:
            managed_cluster.kill_agent()
            time.sleep(downtime)
            managed_cluster.restart_agent(wait_for_amnesia=False)

        exp.wait_for_experiment_state(exp_id, determinedexperimentv1State.STATE_COMPLETED)
        trials = exp.experiment_trials(exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_agent_reconnect_keep_experiment(managed_cluster: ManagedCluster) -> None:
    managed_cluster.ensure_agent_ok()

    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_workload_progress(exp_id)

        managed_cluster.kill_proxy()
        time.sleep(1)
        managed_cluster.restart_proxy()

        exp.wait_for_experiment_state(exp_id, determinedexperimentv1State.STATE_COMPLETED)
        trials = exp.experiment_trials(exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        managed_cluster.restart_proxy(wait_for_reconnect=False)
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_agent_reconnect_keep_cmd(managed_cluster: ManagedCluster) -> None:
    managed_cluster.ensure_agent_ok()

    try:
        command_id = run_command(30, slots=0)
        wait_for_command_state(command_id, "RUNNING", 10)

        managed_cluster.kill_proxy()
        time.sleep(1)
        managed_cluster.restart_proxy()

        wait_for_command_state(command_id, "TERMINATED", 30)

        assert "success" in get_command_info(command_id)["exitStatus"]
    except Exception:
        managed_cluster.restart_proxy(wait_for_reconnect=False)
        managed_cluster.restart_agent()
        raise
