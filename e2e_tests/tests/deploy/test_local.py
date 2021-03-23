import os
import subprocess
from pathlib import Path
from typing import List

import docker
import pytest

from tests import config as conf
from tests import experiment as exp


def det_deploy(subcommand: List) -> None:
    command = [
        "det",
        "deploy",
        "local",
    ] + subcommand
    subprocess.run(command)


def cluster_up(arguments: List) -> None:
    command = ["cluster-up", "--no-gpu"]
    det_version = conf.DET_VERSION
    if det_version is not None:
        command += ["--det-version", det_version]
    command += arguments
    det_deploy(command)


def cluster_down(arguments: List) -> None:
    command = ["cluster-down"]
    command += arguments
    det_deploy(command)


def master_up(arguments: List) -> None:
    command = ["master-up"]
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


@pytest.mark.det_deploy_local  # type: ignore
def test_cluster_down() -> None:
    master_host = "localhost"
    master_port = "8080"
    name = "fixture_down_test"
    conf.MASTER_IP = master_host
    conf.MASTER_PORT = master_port

    cluster_up(["--cluster-name", name])

    container_name = name + "_determined-master_1"
    client = docker.from_env()

    containers = client.containers.list(filters={"name": container_name})
    assert len(containers) > 0

    cluster_down(["--cluster-name", name])

    containers = client.containers.list(filters={"name": container_name})
    assert len(containers) == 0


@pytest.mark.det_deploy_local  # type: ignore
def test_custom_etc() -> None:
    master_host = "localhost"
    master_port = "8080"
    conf.MASTER_IP = master_host
    conf.MASTER_PORT = master_port
    etc_path = str(Path(__file__).parent.joinpath("etc/master.yaml").resolve())
    cluster_up(["--master-config-path", etc_path])
    exp.run_basic_test(
        conf.fixtures_path("no_op/single-default-ckpt.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    assert os.path.exists("/tmp/ckpt-test/")
    cluster_down([])


@pytest.mark.det_deploy_local  # type: ignore
def test_custom_port() -> None:
    name = "port_test"
    master_host = "localhost"
    master_port = "12321"
    conf.MASTER_IP = master_host
    conf.MASTER_PORT = master_port
    arguments = [
        "--cluster-name",
        name,
        "--master-port",
        f"{master_port}",
    ]
    cluster_up(arguments)
    exp.run_basic_test(
        conf.fixtures_path("no_op/single-one-short-step.yaml"),
        conf.fixtures_path("no_op"),
        1,
    )
    cluster_down(["--cluster-name", name])


@pytest.mark.det_deploy_local  # type: ignore
def test_agents_made() -> None:
    master_host = "localhost"
    master_port = "8080"
    name = "agents_test"
    num_agents = 2
    conf.MASTER_IP = master_host
    conf.MASTER_PORT = master_port
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


@pytest.mark.det_deploy_local  # type: ignore
def test_master_up_down() -> None:
    master_host = "localhost"
    master_port = "8080"
    name = "determined"
    conf.MASTER_IP = master_host
    conf.MASTER_PORT = master_port

    master_up(["--master-name", name])

    container_name = name + "_determined-master_1"
    client = docker.from_env()

    containers = client.containers.list(filters={"name": container_name})
    assert len(containers) > 0

    master_down([])

    containers = client.containers.list(filters={"name": container_name})
    assert len(containers) == 0


@pytest.mark.det_deploy_local  # type: ignore
def test_agent_up_down() -> None:
    master_host = "localhost"
    master_port = "8080"
    agent_name = "determined-agent"
    conf.MASTER_IP = master_host
    conf.MASTER_PORT = master_port

    master_up([])
    agent_up(["--agent-name", agent_name])

    client = docker.from_env()
    containers = client.containers.list(filters={"name": agent_name})
    assert len(containers) > 0

    agent_down(["--agent-name", agent_name])
    containers = client.containers.list(filters={"name": agent_name})
    assert len(containers) == 0

    master_down([])
