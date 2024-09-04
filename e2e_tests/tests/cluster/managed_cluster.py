import os
import subprocess
import time
from typing import Any, Dict, Iterator, List, Union, cast

import pytest

from determined.common import api
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests.cluster import abstract_cluster, utils

DEVCLUSTER_CONFIG_ROOT_PATH = conf.PROJECT_ROOT_PATH.joinpath(".circleci/devcluster")
DEVCLUSTER_REATTACH_OFF_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double.devcluster.yaml"
DEVCLUSTER_REATTACH_ON_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double-reattach.devcluster.yaml"
DEVCLUSTER_PRIORITY_SCHEDULER_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "priority.devcluster.yaml"
DEVCLUSTER_MULTI_RP_CONFIG_PATH = (
    DEVCLUSTER_CONFIG_ROOT_PATH / "multi-resource-pools.devcluster.yaml"
)


def get_agent_data(sess: api.Session) -> List[Dict[str, Any]]:
    command = ["det", "agent", "list", "--json"]
    output = detproc.check_json(sess, command)
    agent_data = cast(List[Dict[str, Any]], output)
    return agent_data


class ManagedCluster(abstract_cluster.Cluster):
    # This utility wrapper uses double agent yaml configurations,
    # but provides helpers to run/kill a single agent setup.

    def __init__(self, config: Union[str, Dict[str, Any]]) -> None:
        # Strategically only import devcluster on demand to avoid having it as a hard dependency.
        import devcluster  # noqa: I2000

        self.dc = devcluster.Devcluster(config=config)

    def __enter__(self) -> "ManagedCluster":
        self.old_cd = os.getcwd()
        os.chdir(str(conf.PROJECT_ROOT_PATH))
        self.dc.__enter__()
        return self

    def __exit__(self, *_: Any) -> None:
        os.chdir(self.old_cd)
        self.dc.__exit__(*_)

    def initial_startup(self) -> None:
        self.dc.set_target("agent1", wait=True, timeout=3 * 60)

    def kill_master(self) -> None:
        self.dc.kill_stage("master")

    def restart_master(self) -> None:
        self.dc.restart_stage("master", wait=True, timeout=20)

    def kill_agent(self) -> None:
        self.dc.kill_stage("agent1")

        WAIT_FOR_KILL = 5
        sess = api_utils.user_session()
        for _i in range(WAIT_FOR_KILL):
            agent_data = get_agent_data(sess)
            if len(agent_data) == 0:
                break
            if len(agent_data) == 1 and agent_data[0]["draining"] is True:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Agent is still present after {WAIT_FOR_KILL} seconds")

    def restart_agent(self, wait_for_amnesia: bool = True, wait_for_agent: bool = True) -> None:
        sess = api_utils.user_session()
        agent_data = get_agent_data(sess)
        if len(agent_data) == 1 and agent_data[0]["enabled"]:
            return

        if wait_for_amnesia:
            print(f"Agent is in state {agent_data}, waiting for amnesia")
            # Currently, we've got to wait for master to "forget" the agent before reconnecting.
            WAIT_FOR_AMNESIA = 60
            for _i in range(WAIT_FOR_AMNESIA):
                agent_data = get_agent_data(sess)
                if len(agent_data) == 0:
                    break
                time.sleep(1)
            else:
                pytest.fail(f"Agent is still not forgotten after {WAIT_FOR_AMNESIA} seconds")

        self.dc.restart_stage("agent1", wait=wait_for_agent, timeout=10)

        WAIT_FOR_STARTUP = 10
        if wait_for_agent:
            self.wait_for_agent_ok(WAIT_FOR_STARTUP)

    def kill_proxy(self) -> None:
        subprocess.run(["killall", "socat"])

    def restart_proxy(self, wait_for_reconnect: bool = True) -> None:
        sess = api_utils.user_session()
        self.dc.restart_stage("proxy")
        if wait_for_reconnect:
            for _i in range(25):
                agent_data = get_agent_data(sess)
                if (
                    len(agent_data) == 1
                    and agent_data[0]["enabled"] is True
                    and agent_data[0]["draining"] is False
                ):
                    break
                time.sleep(1)
            else:
                pytest.fail(f"Agent didn't reconnect after {_i} seconds")

    def ensure_agent_ok(self) -> None:
        sess = api_utils.user_session()
        agent_data = get_agent_data(sess)
        assert (
            len(agent_data) == 1
        ), f"expected agent_data for 1, instead found {len(agent_data)} agents:\n{agent_data}\n"
        assert agent_data[0]["enabled"] is True
        assert agent_data[0]["draining"] is False

    def wait_for_agent_ok(self, ticks: int) -> None:
        """
        Each tick is >= 1 second
        """
        sess = api_utils.user_session()
        for _i in range(ticks):
            agent_data = get_agent_data(sess)
            if (
                len(agent_data) == 1
                and agent_data[0]["enabled"] is True
                and agent_data[0]["draining"] is False
            ):
                break
            time.sleep(1)
        else:
            pytest.fail(f"Agent didn't restart after {ticks} seconds")

    def fetch_config(self) -> Dict:
        admin = api_utils.admin_session()
        master_config = detproc.check_json(admin, ["det", "master", "config", "show", "--json"])
        return cast(Dict, master_config)

    def fetch_config_reattach_wait(self) -> float:
        s = self.fetch_config()["resource_pools"][0]["agent_reconnect_wait"]
        return float(s.rstrip("s"))


@pytest.fixture(scope="session")
def managed_cluster_session(request: Any) -> Iterator[ManagedCluster]:
    config = str(DEVCLUSTER_REATTACH_ON_CONFIG_PATH)
    with ManagedCluster(config) as mc:
        mc.initial_startup()
        yield mc


@pytest.fixture(scope="session")
def managed_cluster_session_priority_scheduler(request: Any) -> Iterator[ManagedCluster]:
    config = str(DEVCLUSTER_PRIORITY_SCHEDULER_CONFIG_PATH)
    with ManagedCluster(config) as mc:
        mc.initial_startup()
        yield mc


@pytest.fixture
def managed_cluster_priority_scheduler(
    managed_cluster_session_priority_scheduler: ManagedCluster, request: Any
) -> Iterator[ManagedCluster]:
    config = str(DEVCLUSTER_PRIORITY_SCHEDULER_CONFIG_PATH)
    utils.set_master_port(config)
    nodeid = request.node.nodeid
    managed_cluster_session_priority_scheduler.log_marker(
        f"pytest [{utils.now_ts()}] {nodeid} setup\n"
    )
    yield managed_cluster_session_priority_scheduler
    managed_cluster_session_priority_scheduler.log_marker(
        f"pytest [{utils.now_ts()}] {nodeid} teardown\n"
    )


@pytest.fixture
def managed_cluster_restarts(
    managed_cluster_session: ManagedCluster, request: Any
) -> Iterator[ManagedCluster]:  # check if priority scheduler or not using config.
    config = str(DEVCLUSTER_REATTACH_ON_CONFIG_PATH)
    # port number is same for both reattach on and off config files so you can use either.
    utils.set_master_port(config)
    nodeid = request.node.nodeid
    managed_cluster_session.log_marker(f"pytest [{utils.now_ts()}] {nodeid} setup\n")
    yield managed_cluster_session
    managed_cluster_session.log_marker(f"pytest [{utils.now_ts()}] {nodeid} teardown\n")


@pytest.fixture
def restartable_managed_cluster(
    managed_cluster_restarts: ManagedCluster,
) -> Iterator[ManagedCluster]:
    managed_cluster_restarts.wait_for_agent_ok(20)
    try:
        yield managed_cluster_restarts
        managed_cluster_restarts.wait_for_agent_ok(20)
    except Exception:
        managed_cluster_restarts.restart_master()
        managed_cluster_restarts.restart_agent()
        raise


@pytest.fixture(scope="session")
def managed_cluster_session_multi_resource_pools(request: Any) -> Iterator[ManagedCluster]:
    config = str(DEVCLUSTER_MULTI_RP_CONFIG_PATH)
    with ManagedCluster(config) as mc:
        mc.initial_startup()
        yield mc


@pytest.fixture
def managed_cluster_multi_resource_pools(
    managed_cluster_session_multi_resource_pools: ManagedCluster, request: Any
) -> Iterator[ManagedCluster]:
    config = str(DEVCLUSTER_MULTI_RP_CONFIG_PATH)
    utils.set_master_port(config)
    nodeid = request.node.nodeid
    managed_cluster_session_multi_resource_pools.log_marker(
        f"pytest [{utils.now_ts()}] {nodeid} setup\n"
    )
    yield managed_cluster_session_multi_resource_pools
    managed_cluster_session_multi_resource_pools.log_marker(
        f"pytest [{utils.now_ts()}] {nodeid} teardown\n"
    )


@pytest.fixture
def restartable_managed_cluster_multi_resource_pools(
    managed_cluster_multi_resource_pools: ManagedCluster,
) -> Iterator[ManagedCluster]:
    managed_cluster_multi_resource_pools.wait_for_agent_ok(20)
    try:
        yield managed_cluster_multi_resource_pools
        managed_cluster_multi_resource_pools.wait_for_agent_ok(20)
    except Exception:
        managed_cluster_multi_resource_pools.restart_master()
        managed_cluster_multi_resource_pools.restart_agent()
        raise
