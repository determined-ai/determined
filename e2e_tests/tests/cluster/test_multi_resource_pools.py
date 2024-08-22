# import subprocess

import pytest

from tests import api_utils

# from tests import config as conf
# from tests import detproc
# from tests import experiment as exp
from tests.cluster import managed_cluster


@pytest.mark.managed_devcluster
def test_agent_resource_pool_change(
    restartable_managed_cluster_multi_resource_pools: managed_cluster.ManagedCluster,
) -> None:
    admin = api_utils.user_session()  # api_utils.admin_session()
    try:
        restartable_managed_cluster_multi_resource_pools.kill_agent()
        restartable_managed_cluster_multi_resource_pools.dc.restart_stage("agent10")

        for _i in range(5):
            agent_data = managed_cluster.get_agent_data(admin)
            if len(agent_data) == 0:
                # Agent has exploded and been wiped due to resource pool mismatch, as expected.
                break
        else:
            pytest.fail(
                f"agent with different resource pool is still present after {_i} ticks:{agent_data}"
            )
    finally:
        restartable_managed_cluster_multi_resource_pools.dc.kill_stage("agent10")
        restartable_managed_cluster_multi_resource_pools.restart_agent()


@pytest.mark.managed_devcluster
def test_agent_resource_pool_unchanged(
    restartable_managed_cluster_multi_resource_pools: managed_cluster.ManagedCluster,
) -> None:
    admin = api_utils.user_session()  # api_utils.admin_session()
    try:
        restartable_managed_cluster_multi_resource_pools.kill_agent()
        restartable_managed_cluster_multi_resource_pools.dc.restart_stage("agent20")

        for _i in range(5):
            agent_data = managed_cluster.get_agent_data(admin)
            if len(agent_data) == 0:
                # Agent has exploded and been wiped due to resource pool mismatch,
                # which is not expected.
                pytest.fail("agent exploded even with the same resource pool")
    finally:
        restartable_managed_cluster_multi_resource_pools.dc.kill_stage("agent20")
        restartable_managed_cluster_multi_resource_pools.restart_agent()
