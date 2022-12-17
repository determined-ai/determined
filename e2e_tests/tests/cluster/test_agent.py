import os
import time

import pytest

import determined
from determined.common import api
from tests import config as conf

from .managed_cluster import ManagedCluster


# TODO: This should be marked as a cross-version test, but it can't actually be at the time of
# writing, since older agent versions don't report their versions.
@pytest.mark.e2e_cpu
def test_agent_version() -> None:
    # DET_AGENT_VERSION is available and specifies the agent version in cross-version tests; for
    # other tests, this evaluates to the current version.
    target_version = os.environ.get("DET_AGENT_VERSION") or determined.__version__
    agents = api.get(conf.make_master_url(), "agents").json()

    assert all(agent["version"] == target_version for agent in agents.values())


@pytest.mark.e2e_cpu_agent_connection_loss
def test_agent_never_connect() -> None:
    for _ in range(15):
        if os.path.exists("/tmp/agent-connection-lost"):
            break
        time.sleep(1)
    else:
        pytest.fail("Did not find expected file from agent connection loss hook")


@pytest.mark.managed_devcluster
def test_agent_fail_reconnect(restartable_managed_cluster: ManagedCluster) -> None:
    restartable_managed_cluster.kill_proxy()

    found = False
    for _ in range(30):
        if os.path.exists("/tmp/agent1-connection-lost"):
            found = True
            break
        time.sleep(1)

    restartable_managed_cluster.restart_agent(wait_for_agent=False, wait_for_amnesia=False)
    restartable_managed_cluster.restart_proxy()
    if not found:
        pytest.fail("Did not find expected file from agent connection loss hook")
