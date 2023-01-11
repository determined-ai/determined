import abc
import json
import os
import subprocess
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, Iterator, List, Union, cast

import pytest

from tests import config as conf

from .test_users import ADMIN_CREDENTIALS, logged_in_user
from .utils import get_master_port


class Cluster(metaclass=abc.ABCMeta):
    @abc.abstractmethod
    def __init__(self) -> None:
        pass

    @abc.abstractmethod
    def kill_master(self) -> None:
        pass

    @abc.abstractmethod
    def restart_master(self) -> None:
        pass

    @abc.abstractmethod
    def restart_agent(self, wait_for_amnesia: bool = True, wait_for_agent: bool = True) -> None:
        pass

    @abc.abstractmethod
    def ensure_agent_ok(self) -> None:
        pass


DEVCLUSTER_CONFIG_ROOT_PATH = conf.PROJECT_ROOT_PATH.joinpath(".circleci/devcluster")
DEVCLUSTER_REATTACH_OFF_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double.devcluster.yaml"
DEVCLUSTER_REATTACH_ON_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "double-reattach.devcluster.yaml"
DEVCLUSTER_PRIORITY_SCHEDULER_CONFIG_PATH = DEVCLUSTER_CONFIG_ROOT_PATH / "priority.devcluster.yaml"
DEVCLUSTER_LOG_PATH = Path("/tmp/devcluster")


def get_agent_data(master_url: str) -> List[Dict[str, Any]]:
    command = ["det", "-m", master_url, "agent", "list", "--json"]
    output = subprocess.check_output(command).decode()
    agent_data = cast(List[Dict[str, Any]], json.loads(output))
    return agent_data


class ManagedCluster(Cluster):
    # This utility wrapper uses double agent yaml configurations,
    # but provides helpers to run/kill a single agent setup.

    def __init__(self, config: Union[str, Dict[str, Any]], reattach: bool) -> None:
        # Strategically only import devcluster on demand to avoid having it as a hard dependency.
        from devcluster import Devcluster

        self.dc = Devcluster(config=config)
        self.reattach = reattach

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
        for _i in range(WAIT_FOR_KILL):
            agent_data = get_agent_data(conf.make_master_url())
            if len(agent_data) == 0:
                break
            if len(agent_data) == 1 and agent_data[0]["draining"] is True:
                break
            time.sleep(1)
        else:
            pytest.fail(f"Agent is still present after {WAIT_FOR_KILL} seconds")

    def restart_agent(self, wait_for_amnesia: bool = True, wait_for_agent: bool = True) -> None:
        agent_data = get_agent_data(conf.make_master_url())
        if len(agent_data) == 1 and agent_data[0]["enabled"]:
            return

        if wait_for_amnesia:
            # Currently, we've got to wait for master to "forget" the agent before reconnecting.
            WAIT_FOR_AMNESIA = 60
            for _i in range(WAIT_FOR_AMNESIA):
                agent_data = get_agent_data(conf.make_master_url())
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
        self.dc.restart_stage("proxy")
        if wait_for_reconnect:
            for _i in range(25):
                agent_data = get_agent_data(conf.make_master_url())
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
        agent_data = get_agent_data(conf.make_master_url())
        assert len(agent_data) == 1
        assert agent_data[0]["enabled"] is True
        assert agent_data[0]["draining"] is False

    def wait_for_agent_ok(self, ticks: int) -> None:
        for _i in range(ticks):
            agent_data = get_agent_data(conf.make_master_url())
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
        with logged_in_user(ADMIN_CREDENTIALS):
            master_config = json.loads(
                subprocess.run(
                    ["det", "-m", conf.make_master_url(), "master", "config", "--json"],
                    stdout=subprocess.PIPE,
                    check=True,
                ).stdout.decode()
            )
        return cast(Dict, master_config)

    def fetch_config_reattach_wait(self) -> float:
        s = self.fetch_config()["resource_pools"][0]["agent_reconnect_wait"]
        return float(s.rstrip("s"))

    def log_marker(self, marker: str) -> None:
        for log_path in DEVCLUSTER_LOG_PATH.glob("*.log"):
            with log_path.open("a") as fout:
                fout.write(marker)


@pytest.fixture(scope="session", params=[True, False], ids=["reattach-on", "reattach-off"])
def managed_cluster_session(request: Any) -> Iterator[ManagedCluster]:
    reattach = cast(bool, request.param)
    if reattach:
        config = str(DEVCLUSTER_REATTACH_ON_CONFIG_PATH)
    else:
        config = str(DEVCLUSTER_REATTACH_OFF_CONFIG_PATH)

    with ManagedCluster(config, reattach=reattach) as mc:
        mc.initial_startup()
        yield mc


@pytest.fixture(scope="session")
def managed_cluster_session_priority_scheduler(request: Any) -> Iterator[ManagedCluster]:
    config = str(DEVCLUSTER_PRIORITY_SCHEDULER_CONFIG_PATH)

    with ManagedCluster(config, reattach=False) as mc:
        mc.initial_startup()
        yield mc


def _now_ts() -> str:
    return datetime.now(timezone.utc).astimezone().isoformat()


@pytest.fixture
def managed_cluster_priority_scheduler(
    managed_cluster_session_priority_scheduler: ManagedCluster, request: Any
) -> Iterator[ManagedCluster]:
    config = str(DEVCLUSTER_PRIORITY_SCHEDULER_CONFIG_PATH)
    lc = conf.load_config(config_path=config)
    port = get_master_port(lc)
    set_master_port_conf(port)

    nodeid = request.node.nodeid
    managed_cluster_session_priority_scheduler.log_marker(f"pytest [{_now_ts()}] {nodeid} setup\n")
    yield managed_cluster_session_priority_scheduler
    managed_cluster_session_priority_scheduler.log_marker(
        f"pytest [{_now_ts()}] {nodeid} teardown\n"
    )


@pytest.fixture
def managed_cluster_restarts(
    managed_cluster_session: ManagedCluster, request: Any
) -> Iterator[ManagedCluster]:  # check if priority scheduler or not using config.
    config = str(DEVCLUSTER_REATTACH_ON_CONFIG_PATH)
    # port number is same for both reattach on and off config files so you can use either.
    lc = conf.load_config(config_path=config)
    port = get_master_port(lc)
    set_master_port_conf(port)

    nodeid = request.node.nodeid
    managed_cluster_session.log_marker(f"pytest [{_now_ts()}] {nodeid} setup\n")
    yield managed_cluster_session
    managed_cluster_session.log_marker(f"pytest [{_now_ts()}] {nodeid} teardown\n")


def set_master_port_conf(port: str) -> None:
    conf.MASTER_PORT = port
