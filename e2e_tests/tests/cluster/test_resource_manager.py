import time
from typing import List

import pytest

from determined.common.api import bindings, errors
from determined.experimental import client
from tests import api_utils
from tests import experiment as exp
from tests.cluster import test_agent_disable
from tests.experiment import noop

# How long we should for the Nth = 1 rank to free.
RANK_ONE_WAIT_TIME = 300


@pytest.mark.e2e_cpu_2a
@pytest.mark.timeout(600)
def test_allocation_resources_incremental_release() -> None:
    """
    Start an two container experiment and ensure one container exits before the other. Ensure
    resources are released and schedule-able without the other needing to be released.
    """
    admin = api_utils.admin_session()
    sess = api_utils.user_session()
    cleanup_exp_ids = []

    try:
        slots = test_agent_disable._wait_for_slots(admin, 2)
        assert len(slots) == 2

        # Launch a noop experiment with two workers (one will naturally exit immediately).
        exp_ref = noop.create_experiment(
            sess,
            [
                # Two Reports to meet the requirements of wait_for_workload_progress().
                noop.Report({"loss": 1}),
                noop.Report({"loss": 1}),
                noop.Sleep(1000),
            ],
            config={"resources": {"slots_per_trial": 2}},
        )
        cleanup_exp_ids.append(exp_ref.id)

        # Wait for the experiment to start and run some.
        exp.wait_for_experiment_state(
            sess,
            exp_ref.id,
            bindings.experimentv1State.RUNNING,
        )
        exp.wait_for_experiment_active_workload(sess, exp_ref.id)

        # And wait for exactly one of the resources to free, while one is still in use.
        confirmations = 0
        for _ in range(RANK_ONE_WAIT_TIME):
            free_agents = list_free_agents()
            if len(free_agents) == 1:
                confirmations += 1

            if confirmations == 2:
                # Just for the race where one container has exited and the other hasn't quite yet,
                # but is going to, make sure we see it at least twice.
                break

            # Still waiting on partial exit
            time.sleep(1)
        else:
            pytest.fail(
                "exactly one agent did not free after {} seconds".format(RANK_ONE_WAIT_TIME)
            )

        # Ensure we can schedule on the free slot, not only that the API says its available.
        exp_ref_2 = noop.create_experiment(sess)
        cleanup_exp_ids.append(exp_ref_2.id)
        assert exp_ref_2.wait(interval=0.01) == client.ExperimentState.COMPLETED
        cleanup_exp_ids = cleanup_exp_ids[:-1]

        # And check the hung experiment still is holding on to its hung slot.
        free_agents = list_free_agents()
        if len(free_agents) != 1:
            pytest.fail(f"should still have exactly one agent scheduled: {free_agents}")

    finally:
        for exp_id in cleanup_exp_ids:
            bindings.post_KillExperiment(sess, id=exp_id)
            # TODO(CM-542): Experiment should end up in CANCELED, not ERROR
            exp.wait_for_experiment_state(sess, exp_id, bindings.experimentv1State.ERROR)


def list_free_agents() -> List[bindings.v1Agent]:
    agents = bindings.get_GetAgents(api_utils.user_session())
    if not agents.agents:
        pytest.fail(f"missing agents: {agents}")

    return [a for a in agents.agents or [] if len(a.containers or {}) == 0]


@pytest.mark.e2e_cpu_2a
@pytest.mark.timeout(600)
def test_experiment_is_single_node() -> None:
    admin = api_utils.admin_session()
    sess = api_utils.user_session()
    slots = test_agent_disable._wait_for_slots(admin, 2)
    assert len(slots) == 2

    config = {
        "resources": {
            "slots_per_trial": 2,
            "is_single_node": True,
        },
    }
    with pytest.raises(errors.APIException, match="request unfulfillable"):
        noop.create_experiment(sess, config=config)
