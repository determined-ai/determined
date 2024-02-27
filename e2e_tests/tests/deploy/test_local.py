import contextlib
import os
import pathlib
import subprocess
from typing import List

import docker
import pytest

from determined.common import api
from determined.common.api import authentication
from tests import config as conf
from tests import detproc
from tests import experiment as exp


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


def cluster_up(arguments: List, delete_db: bool = True) -> None:
    command = ["cluster-up", "--no-gpu"]
    if delete_db:
        command += ["--delete-db"]
    det_version = conf.DET_VERSION
    if det_version is not None:
        command += ["--det-version", det_version]
    command += arguments
    det_deploy(command)


def cluster_down(arguments: List) -> None:
    command = ["cluster-down"]
    command += arguments
    det_deploy(command)


@contextlib.contextmanager
def manage_cluster(arguments: List, delete_db: bool = True) -> None:
    cluster_up(arguments, delete_db)
    try:
        yield
    finally:
        cluster_down(arguments)


def master_up(arguments: List, delete_db: bool = True) -> None:
    command = ["master-up"]
    if delete_db:
        command += ["--delete-db"]
    det_version = conf.DET_VERSION
    if det_version is not None:
        command += ["--det-version", det_version]
    command += arguments
    det_deploy(command)


def master_down(arguments: List) -> None:
    command = ["master-down"]
    command += arguments
    det_deploy(command)


def agent_up(arguments: List) -> None:
    command = ["agent-up", conf.MASTER_IP, "--no-gpu"]
    det_version = conf.DET_VERSION
    if det_version is not None:
        command += ["--det-version", det_version]
    command += arguments
    det_deploy(command)


def agent_down(arguments: List) -> None:
    command = ["agent-down"]
    command += arguments
    det_deploy(command)


def agent_enable(sess: api.Session, arguments: List) -> None:
    detproc.check_output(sess, ["det", "agent", "enable"] + arguments)


def agent_disable(sess: api.Session, arguments: List) -> None:
    detproc.check_output(sess, ["det", "agent", "disable"] + arguments)


@pytest.mark.det_deploy_local
def test_cluster_down() -> None:
    name = "test_cluster_down"

    cluster_up(["--cluster-name", name])

    container_name = name + "_determined-master_1"
    client = docker.from_env()

    containers = client.containers.list(filters={"name": container_name})
    assert len(containers) > 0

    cluster_down(["--cluster-name", name])

    containers = client.containers.list(filters={"name": container_name})
    assert len(containers) == 0


@pytest.mark.det_deploy_local
@pytest.mark.skip("Skipping until logic to get scheduler type is fixed")
def test_custom_etc() -> None:
    name = "test_custom_etc"
    etc_path = str(pathlib.Path(__file__).parent.joinpath("etc/master.yaml").resolve())
    cluster_up(["--master-config-path", etc_path, "--cluster-name", name])
    sess = mksess("localhost", 8080)
    exp.run_basic_test(
        sess,
        conf.fixtures_path("no_op/single-default-ckpt.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    assert os.path.exists("/tmp/ckpt-test/")
    cluster_down(["--cluster-name", name])


@pytest.mark.det_deploy_local
def test_agent_config_path() -> None:
    cluster_name = "test_agent_config_path"
    master_name = f"{cluster_name}_determined-master_1"
    master_up(["--master-name", master_name])

    # Config makes it unmodified.
    etc_path = str(pathlib.Path(__file__).parent.joinpath("etc/agent.yaml").resolve())
    agent_name = "test-path-agent"
    agent_up(["--agent-config-path", etc_path])

    client = docker.from_env()
    agent_container = client.containers.get(agent_name)
    exit_code, out = agent_container.exec_run(["cat", "/etc/determined/agent.yaml"])
    assert exit_code == 0
    with open(etc_path) as f:
        assert f.read() == out.decode("utf-8")
    agent_down(["--agent-name", agent_name])

    # Validate CLI flags overwrite config file options.
    agent_name += "-2"
    agent_up(["--agent-name", agent_name, "--agent-config-path", etc_path])
    sess = mksess("localhost", 8080)
    agent_list = detproc.check_json(sess, ["det", "a", "list", "--json"])
    agent_list = [el for el in agent_list if el["id"] == agent_name]
    assert len(agent_list) == 1
    agent_down(["--agent-name", agent_name])

    master_down(["--master-name", master_name])


@pytest.mark.det_deploy_local
@pytest.mark.skip("Skipping until logic to get scheduler type is fixed")
def test_custom_port() -> None:
    name = "port_test"
    custom_port = 12321
    arguments = [
        "--cluster-name",
        name,
        "--master-port",
        f"{custom_port}",
    ]
    cluster_up(arguments)

    sess = mksess("localhost", custom_port)

    exp.run_basic_test(
        sess,
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    cluster_down(["--cluster-name", name])


@pytest.mark.det_deploy_local
def test_agents_made() -> None:
    name = "agents_test"
    num_agents = 2
    arguments = [
        "--cluster-name",
        name,
        "--agents",
        f"{num_agents}",
    ]
    cluster_up(arguments)
    container_names = [name + f"-agent-{i}" for i in range(0, num_agents)]
    client = docker.from_env()

    for container_name in container_names:
        containers = client.containers.list(filters={"name": container_name})
        assert len(containers) > 0

    cluster_down(["--cluster-name", name])


@pytest.mark.det_deploy_local
def test_master_up_down() -> None:
    cluster_name = "test_master_up_down"
    master_name = f"{cluster_name}_determined-master_1"

    master_up(["--master-name", master_name])

    client = docker.from_env()

    containers = client.containers.list(filters={"name": master_name})
    assert len(containers) > 0

    master_down(["--master-name", master_name])

    containers = client.containers.list(filters={"name": master_name})
    assert len(containers) == 0


@pytest.mark.det_deploy_local
def test_agent_up_down() -> None:
    agent_name = "test_agent-determined-agent"
    cluster_name = "test_agent_up_down"
    master_name = f"{cluster_name}_determined-master_1"

    master_up(["--master-name", master_name])
    agent_up(["--agent-name", agent_name])

    client = docker.from_env()
    containers = client.containers.list(filters={"name": agent_name})
    assert len(containers) > 0

    agent_down(["--agent-name", agent_name])
    containers = client.containers.list(filters={"name": agent_name})
    assert len(containers) == 0

    master_down(["--master-name", master_name])
