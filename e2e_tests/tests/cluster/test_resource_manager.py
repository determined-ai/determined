import os
import shutil
import tempfile
import time
from typing import List

import pytest

from determined.common import yaml
from determined.common.api import bindings
from determined.common.api.bindings import experimentv1State
from tests import api_utils
from tests import config as conf
from tests import experiment as exp

from .test_agent_disable import _wait_for_slots

# How long we should for the Nth = 1 rank to free.
RANK_ONE_WAIT_TIME = 300


@pytest.mark.e2e_cpu_2a
@pytest.mark.timeout(600)
def test_allocation_resources_incremental_release() -> None:
    """
    Start an two container experiment and ensure one container exits before the other. Ensure
    resources are released and schedule-able without the other needing to be released.
    """
    cleanup_exp_ids = []

    try:
        slots = _wait_for_slots(2)
        assert len(slots) == 2

        with tempfile.TemporaryDirectory() as context_dir, open(
            os.path.join(context_dir, "const.yaml"), "w"
        ) as config_file:
            # Launch an experiment that has one resource (docker container) that exits immediately.
            config_obj = conf.load_config(conf.fixtures_path("no_op/single.yaml"))
            config_obj["resources"] = {**config_obj.get("resources", {}), **{"slots": 2}}
            config_obj["hyperparameters"] = {
                **config_obj.get("hyperparameters", {}),
                **{"non_chief_exit_immediately": True},
            }
            yaml.dump(config_obj, config_file)

            shutil.copy(
                conf.fixtures_path("no_op/model_def.py"), os.path.join(context_dir, "model_def.py")
            )

            exp_id = exp.create_experiment(config_file.name, context_dir, None)
            cleanup_exp_ids.append(exp_id)

        # Wait for the experiment to start and run some.
        exp.wait_for_experiment_state(
            exp_id,
            experimentv1State.STATE_RUNNING,
        )
        exp.wait_for_experiment_active_workload(exp_id)

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
        exp_id_2 = exp.create_experiment(
            conf.fixtures_path("no_op/single.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )
        cleanup_exp_ids.append(exp_id_2)

        exp.wait_for_experiment_workload_progress(exp_id_2)
        exp.wait_for_experiment_state(exp_id_2, experimentv1State.STATE_COMPLETED)
        cleanup_exp_ids = cleanup_exp_ids[:-1]

        # And check the hung experiment still is holding on to its hung slot.
        free_agents = list_free_agents()
        if len(free_agents) != 1:
            pytest.fail(f"should still have exactly one agent scheduled: {free_agents}")

    finally:
        for exp_id in cleanup_exp_ids:
            bindings.post_KillExperiment(api_utils.determined_test_session(), id=exp_id)
            exp.wait_for_experiment_state(exp_id, experimentv1State.STATE_CANCELED)


def list_free_agents() -> List[bindings.v1Agent]:
    agents = bindings.get_GetAgents(api_utils.determined_test_session())
    if not agents.agents:
        pytest.fail(f"missing agents: {agents}")

    return [a for a in agents.agents or [] if len(a.containers or {}) == 0]
