import subprocess

import pytest

from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests.cluster import managed_cluster, utils


@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_experiment(
    managed_cluster_priority_scheduler: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    managed_cluster_priority_scheduler.ensure_agent_ok()
    assert str(conf.MASTER_PORT) == str(8082)
    # uses the default priority set in cluster config
    exp.run_basic_test(
        sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1
    )
    # uses explicit priority
    exp.run_basic_test(
        sess, conf.fixtures_path("no_op/single.yaml"), conf.fixtures_path("no_op"), 1, priority=50
    )


@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_command(
    managed_cluster_priority_scheduler: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    managed_cluster_priority_scheduler.ensure_agent_ok()
    assert str(conf.MASTER_PORT) == "8082"
    # without slots (and default priority)
    command_id = utils.run_command(sess, slots=0)
    utils.wait_for_command_state(sess, command_id, "TERMINATED", 40)
    utils.assert_command_succeeded(sess, command_id)
    # with slots (and default priority)
    command_id = utils.run_command(sess, slots=1)
    utils.wait_for_command_state(sess, command_id, "TERMINATED", 60)
    utils.assert_command_succeeded(sess, command_id)
    # explicity priority
    command_id = utils.run_command_set_priority(sess, slots=0, priority=60)
    utils.wait_for_command_state(sess, command_id, "TERMINATED", 60)
    utils.assert_command_succeeded(sess, command_id)


@pytest.mark.managed_devcluster
def test_slots_list_command(
    managed_cluster_priority_scheduler: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    managed_cluster_priority_scheduler.ensure_agent_ok()
    assert str(conf.MASTER_PORT) == "8082"
    command = ["det", "slot", "list"]
    p = detproc.run(
        sess, command, universal_newlines=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )
    assert p.returncode == 0, f"\nstdout:\n{p.stdout} \nstderr:\n{p.stderr}"
