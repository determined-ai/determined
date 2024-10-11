import subprocess

import pytest

from determined.experimental import client
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp
from tests.cluster import managed_cluster, utils
from tests.experiment import noop


@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_experiment(
    managed_cluster_priority_scheduler: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    managed_cluster_priority_scheduler.ensure_agent_ok()
    assert str(conf.MASTER_PORT) == str(8082)
    # uses the default priority set in cluster config
    exp_ref = noop.create_experiment(sess)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED
    # uses explicit priority
    exp_ref = noop.create_experiment(sess)
    exp.set_priority(sess, exp_ref.id, 50)
    assert exp_ref.wait(interval=0.01) == client.ExperimentState.COMPLETED


@pytest.mark.managed_devcluster
def test_priortity_scheduler_noop_command(
    managed_cluster_priority_scheduler: managed_cluster.ManagedCluster,
) -> None:
    sess = api_utils.user_session()
    managed_cluster_priority_scheduler.ensure_agent_ok()
    assert str(conf.MASTER_PORT) == "8082"
    # without slots (and default priority)
    command_id = utils.run_command(sess, 0, slots=0)
    utils.wait_for_command_state(sess, command_id, "TERMINATED", 40)
    utils.assert_command_succeeded(sess, command_id)
    # with slots (and default priority)
    command_id = utils.run_command(sess, 0, slots=1)
    utils.wait_for_command_state(sess, command_id, "TERMINATED", 60)
    utils.assert_command_succeeded(sess, command_id)
    # explicity priority
    command_id = utils.run_command_set_priority(sess, 0, slots=0, priority=60)
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
