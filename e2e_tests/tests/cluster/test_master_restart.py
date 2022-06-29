import logging
import subprocess
import time
from typing import Iterator

import pytest

from determined.common.api.bindings import determinedexperimentv1State as EXP_STATE
from tests import config as conf
from tests import experiment as exp

from .managed_cluster import ManagedCluster
from .utils import command_succeeded, run_command, wait_for_command_state

logger = logging.getLogger(__name__)


def _sanity_check(managed_cluster_restarts: ManagedCluster) -> None:
    if not managed_cluster_restarts.reattach:
        pytest.skip()

    managed_cluster_restarts.ensure_agent_ok()


@pytest.fixture
def restartable_managed_cluster(managed_cluster: ManagedCluster) -> Iterator[ManagedCluster]:
    _sanity_check(managed_cluster)

    try:
        yield managed_cluster
        managed_cluster.wait_for_agent_ok(20)
    except Exception:
        managed_cluster.restart_master()
        managed_cluster.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_master_restart_ok(managed_cluster_restarts: ManagedCluster) -> None:
    # - Kill master
    # - Restart master
    # - Schedule something.
    # Do it twice to ensure nothing gets stuck.
    _sanity_check(managed_cluster_restarts)

    try:
        for i in range(3):
            print("test_master_restart_ok stage %s start" % i)
            cmd_ids = [run_command(1, slots) for slots in [0, 1]]

            for cmd_id in cmd_ids:
                wait_for_command_state(cmd_id, "TERMINATED", 10)
                assert command_succeeded(cmd_id)

            managed_cluster_restarts.kill_master()
            managed_cluster_restarts.restart_master()

            print("test_master_restart_ok stage %s done" % i)
    except Exception:
        managed_cluster_restarts.restart_master()
        managed_cluster_restarts.restart_agent()
        raise
    managed_cluster_restarts.restart_agent(wait_for_amnesia=False)
    _sanity_check(managed_cluster_restarts)


@pytest.mark.managed_devcluster
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_reattach_recover_experiment(
    managed_cluster_restarts: ManagedCluster, downtime: int
) -> None:
    _sanity_check(managed_cluster_restarts)

    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-medium-train-step.yaml"),
            conf.fixtures_path("no_op"),
            None,
        )

        # TODO(ilia): don't wait for progress.
        exp.wait_for_experiment_workload_progress(exp_id)

        if downtime >= 0:
            managed_cluster_restarts.kill_master()
            time.sleep(downtime)
            managed_cluster_restarts.restart_master()

        exp.wait_for_experiment_state(
            exp_id, EXP_STATE.STATE_COMPLETED, max_wait_secs=downtime + 60
        )
        trials = exp.experiment_trials(exp_id)

        assert len(trials) == 1
        train_wls = exp.workloads_with_training(trials[0].workloads)
        assert len(train_wls) == 5
    except Exception:
        managed_cluster_restarts.restart_master()
        managed_cluster_restarts.restart_agent()
        raise


@pytest.mark.managed_devcluster
def test_master_restart_kill_works(managed_cluster_restarts: ManagedCluster) -> None:
    _sanity_check(managed_cluster_restarts)

    try:
        exp_id = exp.create_experiment(
            conf.fixtures_path("no_op/single-many-long-steps.yaml"),
            conf.fixtures_path("no_op"),
            ["--config", "searcher.max_length.batches=10000", "--config", "max_restarts=0"],
        )

        exp.wait_for_experiment_workload_progress(exp_id)

        managed_cluster_restarts.kill_master()
        time.sleep(0)
        managed_cluster_restarts.restart_master()

        command = ["det", "-m", conf.make_master_url(), "e", "kill", str(exp_id)]
        subprocess.check_call(command)

        exp.wait_for_experiment_state(exp_id, EXP_STATE.STATE_CANCELED, max_wait_secs=10)

        managed_cluster_restarts.ensure_agent_ok()
    except Exception:
        managed_cluster_restarts.restart_master()
        managed_cluster_restarts.restart_agent()
