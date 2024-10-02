import contextlib
import time
from typing import Any, Dict, Iterator, List, Optional, cast

import pytest

from determined.common import api
from determined.common.api import bindings
from determined.experimental import client
from tests import api_utils, detproc
from tests import experiment as exp
from tests.cluster import utils
from tests.experiment import noop


@pytest.mark.e2e_cpu
def test_disable_and_enable_slots() -> None:
    sess = api_utils.admin_session()

    command = ["det", "slot", "list", "--json"]
    slots = detproc.check_json(sess, command)
    assert len(slots) == 1

    slot_id, agent_id = slots[0]["slot_id"], slots[0]["agent_id"]

    command = ["det", "slot", "disable", agent_id, slot_id]
    detproc.check_call(sess, command)

    slot = bindings.get_GetSlot(sess, agentId=agent_id, slotId=slot_id).slot
    assert slot is not None
    assert slot.enabled is False

    command = ["det", "slot", "enable", agent_id, slot_id]
    detproc.check_call(sess, command)

    slot = bindings.get_GetSlot(sess, agentId=agent_id, slotId=slot_id).slot
    assert slot is not None
    assert slot.enabled is True


def _fetch_slots(sess: api.Session) -> List[Dict[str, Any]]:
    command = ["det", "slot", "list", "--json"]
    slots = detproc.check_json(sess, command)
    return cast(List[Dict[str, str]], slots)


def _wait_for_slots(
    sess: api.Session, min_slots_expected: int, max_ticks: int = 60 * 2
) -> List[Dict[str, Any]]:
    for _ in range(max_ticks):
        slots = _fetch_slots(sess)
        if len(slots) >= min_slots_expected:
            return slots
        time.sleep(1)

    pytest.fail(f"Didn't detect {min_slots_expected} slots within {max_ticks} seconds")


@contextlib.contextmanager
def _disable_agent(
    sess: api.Session, agent_id: str, drain: bool = False, json: bool = False
) -> Iterator[str]:
    command = (
        ["det", "agent", "disable"]
        + (["--drain"] if drain else [])
        + (["--json"] if json else [])
        + [agent_id]
    )
    try:
        yield detproc.check_output(sess, command)
    finally:
        detproc.check_call(sess, ["det", "agent", "enable", agent_id])


@pytest.mark.e2e_cpu
@pytest.mark.e2e_k8s
def test_disable_agent_experiment_resume() -> None:
    """
    Start an experiment with max_restarts=0 and ensure that being killed due to an explicit agent
    disable/enable (without draining) does not count toward the number of restarts.
    """
    admin = api_utils.admin_session()
    sess = api_utils.user_session()
    slots = _fetch_slots(admin)
    assert len(slots) == 1
    agent_id = slots[0]["agent_id"]

    # Make the experiment preemptible.
    exp_ref = noop.create_experiment(sess, [noop.Sleep(100)], config={"max_restarts": 0})
    exp.wait_for_experiment_state(
        sess,
        exp_ref.id,
        bindings.experimentv1State.RUNNING,
        max_wait_secs=utils.KUBERNETES_EXPERIMENT_TIMEOUT,
    )

    with _disable_agent(admin, agent_id):
        # Wait for the allocation to go away.
        for _ in range(20):
            slots = _fetch_slots(admin)
            print(slots)
            if not any(s["allocation_id"] != "FREE" for s in slots):
                break
            time.sleep(1)
        else:
            pytest.fail("Experiment stayed scheduled after agent was disabled")

    # Wait for the experiment to be running again.
    exp.wait_for_experiment_state(
        sess,
        exp_ref.id,
        bindings.experimentv1State.RUNNING,
        max_wait_secs=utils.KUBERNETES_EXPERIMENT_TIMEOUT,
    )

    # Now just kill it off.
    exp_ref.kill()
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.CANCELED


@pytest.mark.e2e_cpu
@pytest.mark.e2e_k8s
def test_disable_agent_zero_slots() -> None:
    """
    Start a command, disable the agent it's running on. The command should
    then be terminated promptly.
    """
    admin = api_utils.admin_session()
    sess = api_utils.user_session()
    slots = _fetch_slots(admin)
    assert len(slots) == 1
    agent_id = slots[0]["agent_id"]

    command_id = utils.run_zero_slot_command(sess, sleep=180)
    # Wait for it to run.
    utils.wait_for_command_state(sess, command_id, "RUNNING", utils.KUBERNETES_EXPERIMENT_TIMEOUT)

    try:
        with _disable_agent(admin, agent_id):
            utils.wait_for_command_state(sess, command_id, "TERMINATED", 30)
    finally:
        # Kill the command before failing so it does not linger.
        command = ["det", "command", "kill", command_id]
        detproc.check_call(sess, command)


@pytest.mark.e2e_cpu
@pytest.mark.e2e_k8s
def test_drain_agent() -> None:
    """
    Start an experiment, `disable --drain` the agent once the trial is running,
    make sure the experiment still finishes, but the new ones won't schedule.
    """
    admin = api_utils.admin_session()
    sess = api_utils.user_session()

    slots = _fetch_slots(admin)
    assert len(slots) == 1
    agent_id = slots[0]["agent_id"]

    exp_ref = noop.create_experiment(
        sess,
        [
            # Two Reports to meet the requirements of wait_for_workload_progress().
            noop.Report({"loss": 1}),
            noop.Report({"loss": 1}),
            noop.Sleep(5),
            # A third Report to prove we finished successfully.
            noop.Report({"loss": 1}),
        ],
    )
    exp.wait_for_experiment_state(
        sess,
        exp_ref.id,
        bindings.experimentv1State.RUNNING,
        max_wait_secs=utils.KUBERNETES_EXPERIMENT_TIMEOUT,
    )
    exp.wait_for_experiment_active_workload(sess, exp_ref.id)
    exp.wait_for_experiment_workload_progress(sess, exp_ref.id)

    # Disable and quickly enable it back.
    with _disable_agent(admin, agent_id, drain=True):
        pass

    # Try to launch another experiment. It shouldn't get scheduled because the
    # slot is still busy with the first experiment.
    no_start = noop.create_experiment(sess)
    time.sleep(2)
    no_start.reload()
    assert no_start.state == client.ExperimentState.QUEUED, no_start.state

    with _disable_agent(admin, agent_id, drain=True):
        # Ensure the first one has finished with the correct number of workloads.
        assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED
        trials = exp.experiment_trials(sess, exp_ref.id)
        assert len(trials) == 1
        assert len(trials[0].workloads) == 3

        # Make sure it doesn't get scheduled into the same slot.
        time.sleep(2)
        no_start.reload()
        assert no_start.state == client.ExperimentState.QUEUED, no_start.state

        # Ensure the slot is empty.
        slots = _fetch_slots(admin)
        assert len(slots) == 1
        assert slots[0]["enabled"] is False
        assert slots[0]["draining"] is True
        assert slots[0]["allocation_id"] == "FREE"

        # Check agent state.
        command = ["det", "agent", "list", "--json"]
        output = detproc.check_json(admin, command)
        agent_data = cast(List[Dict[str, Any]], output)[0]
        assert agent_data["id"] == agent_id
        assert agent_data["enabled"] is False
        assert agent_data["draining"] is True

        no_start.kill()


@pytest.mark.e2e_cpu_2a
def test_drain_agent_sched() -> None:
    """
    Start an experiment, drain it. Start a second one and make sure it schedules
    on the second agent *before* the first one has finished.
    """
    admin = api_utils.admin_session()
    sess = api_utils.user_session()
    slots = _wait_for_slots(admin, 2)
    assert len(slots) == 2

    exp_ref1 = noop.create_experiment(
        sess,
        [
            # Two Reports to meet the requirements of wait_for_workload_progress().
            noop.Report({"loss": 1}),
            noop.Report({"loss": 1}),
            noop.Sleep(100),
        ],
    )
    exp.wait_for_experiment_workload_progress(sess, exp_ref1.id)

    slots = _fetch_slots(admin)
    used_slots = [s for s in slots if s["allocation_id"] != "FREE"]
    assert len(used_slots) == 1
    agent_id1 = used_slots[0]["agent_id"]

    with _disable_agent(admin, agent_id1, drain=True):
        exp_ref2 = noop.create_experiment(sess, [noop.Sleep(100)])

        # Wait for a state when *BOTH* experiments are scheduled.
        for _ in range(200):
            slots = _fetch_slots(admin)
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

        exp_ref1.kill()
        exp_ref2.kill()
        assert exp_ref1.wait(interval=0.01) == client.ExperimentState.CANCELED
        assert exp_ref2.wait(interval=0.01) == client.ExperimentState.CANCELED


def _task_data(sess: api.Session, task_id: str) -> Optional[Dict[str, Any]]:
    command = ["det", "task", "list", "--json"]
    tasks_data: Dict[str, Dict[str, Any]] = detproc.check_json(sess, command)
    matches = [t for t in tasks_data.values() if t["taskId"] == task_id]
    return matches[0] if matches else None


def _task_agents(sess: api.Session, task_id: str) -> List[str]:
    task_data = _task_data(sess, task_id)
    if not task_data:
        return []
    return [a for r in task_data.get("resources", []) for a in r["agentDevices"]]


@pytest.mark.e2e_cpu_2a
def test_drain_agent_sched_zeroslot() -> None:
    """
    Start a command, drain it, start another one on the second node, drain it
    as well. Wait for them to finish, reenable both agents, and make sure
    next command schedules and succeeds.
    """
    admin = api_utils.admin_session()
    sess = api_utils.user_session()
    slots = _wait_for_slots(admin, 2)
    assert len(slots) == 2

    command_id1 = utils.run_zero_slot_command(sess, 60)
    utils.wait_for_command_state(sess, command_id1, "RUNNING", 10)
    agent_id1 = _task_agents(sess, command_id1)[0]

    with _disable_agent(admin, agent_id1, drain=True):
        command_id2 = utils.run_zero_slot_command(sess, 60)
        utils.wait_for_command_state(sess, command_id2, "RUNNING", 10)
        agent_id2 = _task_agents(sess, command_id2)[0]
        assert agent_id1 != agent_id2

        with _disable_agent(admin, agent_id2, drain=True):
            for command_id in [command_id1, command_id2]:
                utils.wait_for_command_state(sess, command_id, "TERMINATED", 60)
                utils.assert_command_succeeded(sess, command_id)

    command_id3 = utils.run_zero_slot_command(sess, 1)
    utils.wait_for_command_state(sess, command_id3, "TERMINATED", 60)
    utils.assert_command_succeeded(sess, command_id3)
