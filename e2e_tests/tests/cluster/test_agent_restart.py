import json
import subprocess
import time
import uuid
from pathlib import Path
from typing import Any, Dict, Iterator, Tuple

import pytest

from determined.common.api.bindings import determinedexperimentv1State as EXP_STATE
from tests import config as conf
from tests import experiment as exp

from .managed_cluster import ManagedCluster
from .utils import get_command_info, run_command, wait_for_command_state

DEVCLUSTER_CONFIG_ROOT_PATH = conf.PROJECT_ROOT_PATH.joinpath(".circleci/devcluster")
DEVCLUSTER_REATTACH_OFF_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double.devcluster.yaml"
DEVCLUSTER_REATTACH_ON_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double-reattach.devcluster.yaml"
DEVCLUSTER_LOG_PATH = Path("/tmp/devcluster")
DEVCLUSTER_MASTER_LOG_PATH = DEVCLUSTER_LOG_PATH / "master.log"


@pytest.mark.managed_devcluster
def test_managed_cluster_working(managed_cluster_restarts: ManagedCluster) -> None:
    try:
        managed_cluster_restarts.ensure_agent_ok()
        managed_cluster_restarts.kill_agent()
    finally:
        managed_cluster_restarts.restart_agent()


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
def test_agent_restart_exp_container_failure(managed_cluster_restarts: ManagedCluster) -> None:
    managed_cluster_restarts.ensure_agent_ok()
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
        tasks_data = _task_list_json(conf.make_master_url())
        assert len(tasks_data) == 1
        exp_task_before = list(tasks_data.values())[0]

        managed_cluster_restarts.kill_agent()
        subprocess.run(["docker", "kill", container_ids[0]], check=True, stdout=subprocess.PIPE)
    except Exception:
        managed_cluster_restarts.restart_agent()
        raise
    else:
        managed_cluster_restarts.restart_agent()
        # As soon as the agent is back, the original allocation should be considered dead,
        # but the new one should be allocated.
        state = exp.experiment_state(exp_id)
        assert state == EXP_STATE.STATE_ACTIVE
        tasks_data = _task_list_json(conf.make_master_url())
        assert len(tasks_data) == 1
        exp_task_after = list(tasks_data.values())[0]

        assert exp_task_before["task_id"] == exp_task_after["task_id"]
        assert exp_task_before["allocation_id"] != exp_task_after["allocation_id"]

        exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_COMPLETED)


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("command_duration", [10, 60])
def test_agent_restart_cmd_container_failure(
    managed_cluster_restarts: ManagedCluster, command_duration: int
) -> None:
    # Launch a cmd, kill agent, wait for reconnect timeout, check it's not marked as success.
    # Reconnect timeout is ~25 seconds. We'd like to both test tasks that take
    # longer (60 seconds) and shorter (10 seconds) than that.
    managed_cluster_restarts.ensure_agent_ok()
    try:
        command_id = run_command(command_duration)
        wait_for_command_state(command_id, "RUNNING", 10)

        for _i in range(10):
            if len(list(_local_container_ids_for_command(command_id))) > 0:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Failed to find the command container id after {_i} ticks")

        managed_cluster_restarts.kill_agent()

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
        managed_cluster_restarts.restart_agent()
        raise
    else:
        managed_cluster_restarts.restart_agent()


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("command_duration", [20])
def test_agent_noreattach_restart_kills(
    managed_cluster_restarts: ManagedCluster, command_duration: int
) -> None:
    if managed_cluster_restarts.reattach:
        pytest.skip()

    managed_cluster_restarts.ensure_agent_ok()
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

        managed_cluster_restarts.kill_agent()
        managed_cluster_restarts.restart_agent(wait_for_amnesia=False)

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
        managed_cluster_restarts.restart_agent()
        raise


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("slots", [0, 1])
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_agent_restart_recover_cmd(
    managed_cluster_restarts: ManagedCluster, slots: int, downtime: int
) -> None:
    if not managed_cluster_restarts.reattach:
        pytest.skip()

    managed_cluster_restarts.ensure_agent_ok()
    try:
        command_id = run_command(30, slots=slots)
        wait_for_command_state(command_id, "RUNNING", 10)

        managed_cluster_restarts.kill_agent()
        time.sleep(downtime)
        managed_cluster_restarts.restart_agent(wait_for_amnesia=False)

        wait_for_command_state(command_id, "TERMINATED", 30)

        # Commands fail if they have finished while the agent was off.
        reattach_wait = managed_cluster_restarts.fetch_config_reattach_wait()
        succeeded = "success" in get_command_info(command_id)["exitStatus"]
        assert succeeded is (reattach_wait > downtime)
    except Exception:
        managed_cluster_restarts.restart_agent()
        raise


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [-1, 0, 20, 60])
def test_agent_restart_recover_experiment(
    managed_cluster_restarts: ManagedCluster, downtime: int
) -> None:
    if not managed_cluster_restarts.reattach:
        pytest.skip()

    managed_cluster_restarts.ensure_agent_ok()
    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_workload_progress(exp_id)

        if downtime >= 0:
            managed_cluster_restarts.kill_agent()
            time.sleep(downtime)
            managed_cluster_restarts.restart_agent(wait_for_amnesia=False)

        exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_COMPLETED)
        trials = exp.experiment_trials(exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        managed_cluster_restarts.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_agent_reconnect_keep_experiment(managed_cluster_restarts: ManagedCluster) -> None:
    managed_cluster_restarts.ensure_agent_ok()

    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_workload_progress(exp_id)

        managed_cluster_restarts.kill_proxy()
        time.sleep(1)
        managed_cluster_restarts.restart_proxy()

        exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_COMPLETED)
        trials = exp.experiment_trials(exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        managed_cluster_restarts.restart_proxy(wait_for_reconnect=False)
        managed_cluster_restarts.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_agent_reconnect_keep_cmd(managed_cluster_restarts: ManagedCluster) -> None:
    managed_cluster_restarts.ensure_agent_ok()

    try:
        command_id = run_command(20, slots=0)
        wait_for_command_state(command_id, "RUNNING", 10)

        managed_cluster_restarts.kill_proxy()
        time.sleep(1)
        managed_cluster_restarts.restart_proxy()

        wait_for_command_state(command_id, "TERMINATED", 30)

        assert "success" in get_command_info(command_id)["exitStatus"]
    except Exception:
        managed_cluster_restarts.restart_proxy(wait_for_reconnect=False)
        managed_cluster_restarts.restart_agent()
        raise


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("slots", [0, 1])
def test_agent_reconnect_trigger_schedule(
    managed_cluster_restarts: ManagedCluster, slots: int
) -> None:
    if managed_cluster_restarts.reattach:
        pytest.skip()

    managed_cluster_restarts.ensure_agent_ok()

    try:
        managed_cluster_restarts.kill_proxy()
        command_id = run_command(5, slots=slots)
        managed_cluster_restarts.restart_proxy()
        wait_for_command_state(command_id, "TERMINATED", 10)

        assert "success" in get_command_info(command_id)["exitStatus"]
    except Exception:
        managed_cluster_restarts.restart_proxy(wait_for_reconnect=False)
        managed_cluster_restarts.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_queued_experiment_restarts_with_correct_allocation_id(
    managed_cluster_restarts: ManagedCluster,
) -> None:
    exp_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        ["--config", "resources.slots_per_trial=9999"],
    )
    exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_ACTIVE)

    managed_cluster_restarts.kill_master()
    log_marker = str(uuid.uuid4())
    managed_cluster_restarts.log_marker(log_marker)
    managed_cluster_restarts.restart_master()

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
