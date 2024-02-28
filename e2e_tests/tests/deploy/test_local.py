import contextlib
import enum
import os
import pathlib
import subprocess
from typing import Dict, Iterator, List, Optional

import docker
import pytest

from determined.common import api
from determined.common.api import authentication
from tests import config as conf
from tests import detproc
from tests import experiment as exp


class Resource(enum.Enum):
    CLUSTER = "cluster"
    MASTER = "master"
    AGENT = "agent"


def mksess(host: str, port: int, username: str = "determined", password: str = "") -> api.Session:
    """
    Since this file frequently creates new masters, always create a fresh Session.
    """

    master_url = api.canonicalize_master_url(f"http://{host}:{port}")
    utp = authentication.login(master_url, username=username, password=password)
    return api.Session(master_url, utp, cert=None, max_retries=0)


def det_deploy(subcommand: List) -> None:
    command = [
        "det",
        "deploy",
        "local",
    ] + subcommand
    subprocess.run(command, check=True)


def resource_up(
    resource: Resource,
    name,
    kwflags: Optional[Dict[str, str]] = None,
    flags: Optional[List[str]] = None,
    positional_arguments: Optional[List[str]] = None,
) -> None:
    """Issue a `det deploy local` command to bring up a resource.

    Ex:
      det deploy local cluster-up --cluster-name test_cluster --det-version 0.15.0

    Arguments:
        resource: The resource to bring up, e.g. "cluster", "master", "agent".
        name: The name to give the resource.
        kwflags: A dictionary of keyword flags (and their values) to pass to the command.
        flags: A list of flags to pass to the command.
        positional_arguments: A list of positional arguments to pass to the command.

    This additionally sets a --det-version flag if DET_VERSION is set in the config.
    """
    command = [f"{resource}-up", "--no-gpu", f"--{resource}-name", name]
    if kwflags:
        for k, v in kwflags.items():
            command += [f"--{k}", v]
    if flags:
        for flag in flags:
            command += [f"--{flag}"]
    if positional_arguments:
        command += positional_arguments

    det_version = conf.DET_VERSION
    if det_version is not None:
        command += ["--det-version", det_version]
    det_deploy(command)


def resource_down(resource: Resource, name: str) -> None:
    command = [f"{resource}-down", f"--{resource}-name", name]
    det_deploy(command)


@contextlib.contextmanager
def resource_manager(
    resource: Resource,
    name,
    kwflags: Optional[Dict[str, str]] = None,
    boolean_flags: Optional[List[str]] = None,
) -> Iterator[None]:
    """Context manager to bring resources up and down."""
    resource_up(resource, name, kwflags, boolean_flags)
    try:
        yield
    finally:
        resource_down(resource, name)


def agent_enable(sess: api.Session, arguments: List) -> None:
    detproc.check_output(sess, ["det", "agent", "enable"] + arguments)


def agent_disable(sess: api.Session, arguments: List) -> None:
    detproc.check_output(sess, ["det", "agent", "disable"] + arguments)


@pytest.mark.det_deploy_local
def test_cluster_down() -> None:
    name = "test_cluster_down"

    resource_up(Resource.CLUSTER, name, {}, ["no-gpu"])

    container_name = name + "_determined-master_1"
    client = docker.from_env()

    containers = client.containers.list(filters={"name": container_name})
    assert len(containers) > 0

    resource_down(Resource.CLUSTER, name)

    containers = client.containers.list(filters={"name": container_name})
    assert len(containers) == 0


@pytest.mark.det_deploy_local
@pytest.mark.skip("Skipping until logic to get scheduler type is fixed")
def test_custom_etc() -> None:
    name = "test_custom_etc"
    etc_path = str(pathlib.Path(__file__).parent.joinpath("etc/master.yaml").resolve())
    resource_up(Resource.CLUSTER, name, {"master-config-path": etc_path}, ["no-gpu"])
    sess = mksess("localhost", 8080)
    exp.run_basic_test(
        sess,
        conf.fixtures_path("no_op/single-default-ckpt.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    assert os.path.exists("/tmp/ckpt-test/")
    resource_down(Resource.CLUSTER, name)


@pytest.mark.det_deploy_local
def test_agent_config_path() -> None:
    cluster_name = "test_agent_config_path"
    master_name = f"{cluster_name}_determined-master_1"
    resource_up(Resource.MASTER, master_name)

    # Config makes it unmodified.
    etc_path = str(pathlib.Path(__file__).parent.joinpath("etc/agent.yaml").resolve())
    agent_name = "test-path-agent"
    resource_up(
        Resource.AGENT, agent_name, {"agent-config-path": etc_path}, ["no-gpu"], [conf.MASTER_IP]
    )

    client = docker.from_env()
    agent_container = client.containers.get(agent_name)
    exit_code, out = agent_container.exec_run(["cat", "/etc/determined/agent.yaml"])
    assert exit_code == 0
    with open(etc_path) as f:
        assert f.read() == out.decode("utf-8")
    resource_down(Resource.AGENT, agent_name)

    # Validate CLI flags overwrite config file options.
    agent_name += "-2"
    resource_up(
        Resource.AGENT, agent_name, {"agent-config-path": etc_path}, ["no-gpu"], [conf.MASTER_IP]
    )
    sess = mksess("localhost", 8080)
    agent_list = detproc.check_json(sess, ["det", "a", "list", "--json"])
    agent_list = [el for el in agent_list if el["id"] == agent_name]
    assert len(agent_list) == 1
    resource_down(Resource.AGENT, agent_name)

    resource_down(Resource.MASTER, master_name)


@pytest.mark.det_deploy_local
@pytest.mark.skip("Skipping until logic to get scheduler type is fixed")
def test_custom_port() -> None:
    name = "port_test"
    custom_port = 12321
    resource_up(Resource.CLUSTER, name, {"master-port": str(custom_port)}, ["no-gpu"])

    sess = mksess("localhost", custom_port)

    exp.run_basic_test(
        sess,
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    resource_down(Resource.CLUSTER, name)


@pytest.mark.det_deploy_local
def test_agents_made() -> None:
    name = "agents_test"
    num_agents = 2
    resource_up(Resource.CLUSTER, name, {"agents": str(num_agents)}, ["no-gpu"])
    container_names = [name + f"-agent-{i}" for i in range(0, num_agents)]
    client = docker.from_env()

    for container_name in container_names:
        containers = client.containers.list(filters={"name": container_name})
        assert len(containers) > 0

    resource_down(Resource.CLUSTER, name)


@pytest.mark.det_deploy_local
def test_master_up_down() -> None:
    cluster_name = "test_master_up_down"
    master_name = f"{cluster_name}_determined-master_1"

    resource_up(Resource.MASTER, master_name)

    client = docker.from_env()

    containers = client.containers.list(filters={"name": master_name})
    assert len(containers) > 0

    resource_down(Resource.MASTER, master_name)

    containers = client.containers.list(filters={"name": master_name})
    assert len(containers) == 0


@pytest.mark.det_deploy_local
def test_agent_up_down() -> None:
    agent_name = "test_agent-determined-agent"
    cluster_name = "test_agent_up_down"
    master_name = f"{cluster_name}_determined-master_1"

    resource_up(Resource.MASTER, master_name)
    resource_up(Resource.AGENT, agent_name, {}, ["no-gpu"], [conf.MASTER_IP])

    client = docker.from_env()
    containers = client.containers.list(filters={"name": agent_name})
    assert len(containers) > 0

    resource_down(Resource.AGENT, agent_name)
    containers = client.containers.list(filters={"name": agent_name})
    assert len(containers) == 0

    resource_down(Resource.MASTER, master_name)
