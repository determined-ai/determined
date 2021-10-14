import contextlib
import json
import subprocess
import time
from typing import Any, Dict, Iterator, List, Optional, cast

import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
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


def _fetch_slots() -> List[Dict[str, Any]]:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "slot",
        "list",
        "--json",
    ]
    output = subprocess.check_output(command).decode()
    slots = cast(List[Dict[str, str]], json.loads(output))
    return slots


def _run_zero_slot_command(sleep: int = 30) -> str:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "command",
        "run",
        "-d",
        "--config",
        "resources.slots=0",
        "sleep",
        str(sleep),
    ]
    return subprocess.check_output(command).decode().strip()


@contextlib.contextmanager
def _disable_agent(agent_id: str, drain: bool = False, json: bool = False) -> Iterator[str]:
    command = (
        ["det", "-m", conf.make_master_url(), "agent", "disable"]
        + (["--drain"] if drain else [])
        + (["--json"] if json else [])
        + [agent_id]
    )
    try:
        yield subprocess.check_output(command).decode()
    finally:
        command = ["det", "-m", conf.make_master_url(), "agent", "enable", agent_id]
        subprocess.check_call(command)


def _get_command_info(command_id: str) -> Dict[str, Any]:
    command = ["det", "-m", conf.make_master_url(), "command", "list", "--json"]
    command_data = json.loads(subprocess.check_output(command).decode())
    return next((d for d in command_data if d["id"] == command_id), {})


def _wait_for_command_state(command_id: str, state: str, ticks: int = 60) -> None:
    for _ in range(ticks):
        info = _get_command_info(command_id)
        if info.get("state") == state:
            return
        time.sleep(1)

    pytest.fail(f"Command did't reach {state} state after {ticks} secs")


@pytest.mark.e2e_cpu
def test_disable_agent_zero_slots() -> None:
    """
    Start a command, disable the agent it's running on. The command should
    then be terminated promptly.
    """
    slots = _fetch_slots()
    assert len(slots) == 1
    agent_id = slots[0]["agent_id"]

    command_id = _run_zero_slot_command(sleep=60)
    # Wait for it to run.
    _wait_for_command_state(command_id, "RUNNING", 30)

    try:
        with _disable_agent(agent_id):
            _wait_for_command_state(command_id, "TERMINATED", 5)
    finally:
        # Kill the command before failing so it does not linger.
        command = ["det", "-m", conf.make_master_url(), "command", "kill", command_id]
        subprocess.check_call(command)


@pytest.mark.e2e_cpu
def test_drain_agent() -> None:
    """
    Start an experiment, `disable --drain` the agent once the trial is running,
    make sure the experiment still finishes, but the new ones won't schedule.
    """

    slots = _fetch_slots()
    assert len(slots) == 1
    agent_id = slots[0]["agent_id"]

    experiment_id = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )
    exp.wait_for_experiment_state(experiment_id, "ACTIVE")
    exp.wait_for_experiment_active_workload(experiment_id)
    exp.wait_for_experiment_workload_progress(experiment_id)

    # Disable and quickly enable it back.
    with _disable_agent(agent_id, drain=True):
        pass

    # Try to launch another experiment. It shouldn't get scheduled because the
    # slot is still busy with the first experiment.
    experiment_id_no_start = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )
    time.sleep(5)
    exp.wait_for_experiment_state(experiment_id_no_start, "ACTIVE")

    with _disable_agent(agent_id, drain=True):
        # Check for 15 seconds it doesn't get scheduled into the same slot.
        for _ in range(15):
            trials = exp.experiment_trials(experiment_id_no_start)
            assert len(trials) == 0

        # Ensure the first one has finished with the correct number of steps.
        exp.wait_for_experiment_state(experiment_id, "COMPLETED")
        trials = exp.experiment_trials(experiment_id)
        assert len(trials) == 1
        assert len(trials[0]["steps"]) == 5

        # Ensure the slot is empty.
        slots = _fetch_slots()
        assert len(slots) == 1
        assert slots[0]["enabled"] is False
        assert slots[0]["draining"] is True
        assert slots[0]["allocation_id"] == "FREE"

        # Check agent state.
        command = ["det", "-m", conf.make_master_url(), "agent", "list", "--json"]
        output = subprocess.check_output(command).decode()
        agent_data = cast(List[Dict[str, Any]], json.loads(output))[0]
        assert agent_data["id"] == agent_id
        assert agent_data["enabled"] is False
        assert agent_data["draining"] is True

        exp.cancel_single(experiment_id_no_start)


@pytest.mark.e2e_cpu_2a
def test_drain_agent_sched() -> None:
    """
    Start an experiment, drain it. Start a second one and make sure it schedules
    on the second agent *before* the first one has finished.
    """
    slots = _fetch_slots()
    assert len(slots) == 2

    exp_id1 = exp.create_experiment(
        conf.fixtures_path("no_op/single-medium-train-step.yaml"),
        conf.fixtures_path("no_op"),
        None,
    )
    exp.wait_for_experiment_workload_progress(exp_id1)

    slots = _fetch_slots()
    used_slots = [s for s in slots if s["allocation_id"] != "FREE"]
    assert len(used_slots) == 1
    agent_id1 = used_slots[0]["agent_id"]

    with _disable_agent(agent_id1, drain=True):
        exp_id2 = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        exp.wait_for_experiment_state(exp_id2, "ACTIVE")

        # Wait for a state when *BOTH* experiments are scheduled.
        for _ in range(20):
            slots = _fetch_slots()
            assert len(slots) == 2
            used_slots = [s for s in slots if s["allocation_id"] != "FREE"]
            if len(used_slots) == 2:
                # All good.
                break
        else:
            pytest.fail(
                "Second experiment didn't schedule on the second agent "
                "while the first agent was draining"
            )

        exp.wait_for_experiment_state(exp_id1, "COMPLETED")
        exp.wait_for_experiment_state(exp_id2, "COMPLETED")

        trials1 = exp.experiment_trials(exp_id1)
        trials2 = exp.experiment_trials(exp_id2)
        assert len(trials1) == len(trials2) == 1
        assert len(trials1[0]["steps"]) == len(trials2[0]["steps"]) == 5


def _task_data(task_id: str) -> Optional[Dict[str, Any]]:
    command = ["det", "-m", conf.make_master_url(), "task", "list", "--json"]
    tasks_data: Dict[str, Dict[str, Any]] = json.loads(subprocess.check_output(command).decode())
    matches = [t for t in tasks_data.values() if t["task_id"] == task_id]
    return matches[0] if matches else None


def _task_agents(task_id: str) -> List[str]:
    task_data = _task_data(task_id)
    if not task_data:
        return []
    return [c["agent"] for c in task_data.get("containers", [])]


@pytest.mark.e2e_cpu_2a
def test_drain_agent_sched_zeroslot() -> None:
    """
    Start a command, drain it, start another one on the second node, drain it
    as well. Wait for them to finish, reenable both agents, and make sure
    next command schedules and succeeds.
    """
    slots = _fetch_slots()
    assert len(slots) == 2

    command_id1 = _run_zero_slot_command(60)
    _wait_for_command_state(command_id1, "RUNNING", 10)
    agent_id1 = _task_agents(command_id1)[0]

    with _disable_agent(agent_id1, drain=True):
        command_id2 = _run_zero_slot_command(60)
        _wait_for_command_state(command_id2, "RUNNING", 10)
        agent_id2 = _task_agents(command_id2)[0]
        assert agent_id1 != agent_id2

        with _disable_agent(agent_id2, drain=True):
            for command_id in [command_id1, command_id2]:
                _wait_for_command_state(command_id, "TERMINATED", 60)
                assert "success" in _get_command_info(command_id)["exitStatus"]

    command_id3 = _run_zero_slot_command(1)
    _wait_for_command_state(command_id3, "TERMINATED", 60)
    assert "success" in _get_command_info(command_id3)["exitStatus"]
