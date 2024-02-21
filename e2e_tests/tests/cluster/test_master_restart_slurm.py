import logging
from typing import Iterator

import pytest

from tests.cluster import managed_slurm_cluster, test_master_restart

logger = logging.getLogger(__name__)


# Create a pytest fixture that returns a restartable instance of ManagedSlurmCluster.
@pytest.fixture
def restartable_managed_slurm_cluster(
    managed_slurm_cluster_restarts: managed_slurm_cluster.ManagedSlurmCluster,
) -> Iterator[managed_slurm_cluster.ManagedSlurmCluster]:
    try:
        yield managed_slurm_cluster_restarts
    except Exception:
        managed_slurm_cluster_restarts.restart_master()
        raise


# Test to ensure master restarts successfully.
@pytest.mark.e2e_slurm_restart
def test_master_restart_ok_slurm(
    managed_slurm_cluster_restarts: managed_slurm_cluster.ManagedSlurmCluster,
) -> None:
    test_master_restart._test_master_restart_ok(managed_slurm_cluster_restarts)


# Test to ensure that master can reattach to the experiment and resume it, after the determined
# master has restarted.
@pytest.mark.e2e_slurm_restart
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_reattach_recover_experiment_slurm(
    managed_slurm_cluster_restarts: managed_slurm_cluster.ManagedSlurmCluster, downtime: int
) -> None:
    test_master_restart._test_master_restart_reattach_recover_experiment(
        managed_slurm_cluster_restarts, downtime
    )


# Test to ensure that master can recover and complete a command that was in running state
# when the master has restarted.
@pytest.mark.e2e_slurm_restart
@pytest.mark.parametrize("slots", [0, 1])
@pytest.mark.parametrize("downtime", [0, 20, 60])
def test_master_restart_cmd_slurm(
    restartable_managed_slurm_cluster: managed_slurm_cluster.ManagedSlurmCluster,
    slots: int,
    downtime: int,
) -> None:
    test_master_restart._test_master_restart_cmd(restartable_managed_slurm_cluster, slots, downtime)
