import subprocess

import pytest

from tests import config as conf
from tests import experiment as exp

from .managed_cluster import ManagedCluster
from .utils import (
    assert_command_succeeded,
    run_command,
    run_command_set_priority,
    wait_for_command_state,
)


@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_experiment(
    managed_cluster_priority_scheduler: ManagedCluster,
) -> None:
    managed_cluster_priority_scheduler.ensure_agent_ok()
    assert str(conf.MASTER_PORT) == str(8082)
    # uses the default priority set in cluster config
    exp.run_basic_test(conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1)
    # uses explicit priority
    exp.run_basic_test(
        conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1, priority=50
    )


@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_command(
    managed_cluster_priority_scheduler: ManagedCluster,
) -> None:
    managed_cluster_priority_scheduler.ensure_agent_ok()
    assert str(conf.MASTER_PORT) == "8082"
    # without slots (and default priority)
    command_id = run_command(slots=0)
    wait_for_command_state(command_id, "TERMINATED", 40)
    assert_command_succeeded(command_id)
    # with slots (and default priority)
    command_id = run_command(slots=1)
    wait_for_command_state(command_id, "TERMINATED", 60)
    assert_command_succeeded(command_id)
    # explicity priority
    command_id = run_command_set_priority(slots=0, priority=60)
    wait_for_command_state(command_id, "TERMINATED", 60)
    assert_command_succeeded(command_id)


@pytest.mark.managed_devcluster
def test_slots_list_command(managed_cluster_priority_scheduler: ManagedCluster) -> None:
    managed_cluster_priority_scheduler.ensure_agent_ok()
    assert str(conf.MASTER_PORT) == "8082"
    command = ["det", "slot", "list"]
    completed_process = subprocess.run(
        command, universal_newlines=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    assert completed_process.returncode == 0, "\nstdout:\n{} \nstderr:\n{}".format(
        completed_process.stdout, completed_process.stderr
    )
