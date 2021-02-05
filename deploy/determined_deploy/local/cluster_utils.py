import os
import re
import subprocess
import sys
import time
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence

import docker
import requests

import determined_deploy
from determined_common import api

# This object, when included in the host config in a container creation request, tells Docker to
# expose all host GPUs inside a container.
GPU_DEVICE_REQUEST = {"Driver": "nvidia", "Count": -1, "Capabilities": [["gpu", "utility"]]}


# Patch the Docker library to support device requests, since it has yet to support them natively
# (see https://github.com/docker/docker-py/issues/2395).
def _patch_docker_for_device_requests() -> None:
    _old_create_container_args = docker.models.containers._create_container_args

    def _create_container_args(kwargs: Any) -> Any:
        device_requests = kwargs.pop("device_requests", None)
        create_kwargs = _old_create_container_args(kwargs)
        if device_requests:
            create_kwargs["host_config"]["DeviceRequests"] = device_requests
        return create_kwargs

    docker.models.containers._create_container_args = _create_container_args


_patch_docker_for_device_requests()


def get_shell_id() -> str:
    args = ["id", "-u", "-n"]
    byte_str: str = subprocess.check_output(args, encoding="utf-8")
    return byte_str.rstrip("\n").strip("'").strip()


def _make_master_url(master_host: str, master_port: int, suffix: str = "") -> str:
    return "http://{}:{}/{}".format(master_host, master_port, suffix)


def get_proxy_addr() -> str:
    # The Determined proxying code relies on docker port-mapping container ports to host
    # ports, and it uses the IP address of the agent as a way to address spawned
    # docker containers. This breaks down when running in a docker compose
    # environment, because the address of the agent is not the address of the
    # docker host. As a work-around, force agents to report their IP address as the
    # IP address of the host machine.
    if "darwin" in sys.platform:
        # On macOS, docker runs in a VM and host.docker.internal points to the IP
        # address of this VM.
        return "host.docker.internal"
    else:
        # On non-macOS, host.docker.internal does not exist. Instead, grab the source IP
        # address we would use if we had to talk to the internet. The sed command
        # searches the first line of its input for "src" and prints the first field
        # after that.
        proxy_addr_args = ["ip", "route", "get", "8.8.8.8"]
        pattern = r"s|.* src +(\S+).*|\1|"
        s = subprocess.check_output(proxy_addr_args, encoding="utf-8")
        matches = re.match(pattern, s)
        if matches is not None:
            groups: Sequence[str] = matches.groups()
            if len(groups) != 0:
                return groups[0]
        return ""


def docker_compose(
    args: List[str],
    cluster_name: str,
    env: Optional[Dict] = None,
    extra_files: Optional[List[str]] = None,
) -> None:
    path = Path(__file__).parent.joinpath("docker-compose.yaml")
    # Start with the user's environment to ensure that Docker and Docker Compose work correctly.
    process_env = dict(os.environ)
    if env is not None:
        # raise ValueError(str(env))
        process_env.update(env)
    process_env["INTEGRATIONS_PROXY_ADDR"] = get_proxy_addr()
    base_command = ["docker-compose", "-f", str(path), "-p", cluster_name]
    if extra_files is not None:
        for extra_file in extra_files:
            base_command += ["-f", extra_file]
    args = base_command + args
    subprocess.run(args, env=process_env)


def _wait_for_master(master_host: str, master_port: int, cluster_name: str) -> None:
    for _ in range(50):
        try:
            r = api.get(_make_master_url(master_host, master_port), "info", authenticated=False)
            if r.status_code == requests.codes.ok:
                return
        except api.errors.MasterNotFoundException:
            pass
        print("Waiting for master to be available...")
        time.sleep(2)

    print("Timed out connecting to master, but attempting to dump logs from cluster...")
    docker_compose(["logs"], cluster_name)
    raise ConnectionError("Timed out connecting to master")


def master_up(
    port: int,
    master_config_path: Path,
    master_name: str,
    version: Optional[str],
    db_password: str,
    delete_db: bool,
    autorestart: bool,
    cluster_name: str,
) -> None:
    command = ["up", "-d"]
    extra_files = []
    if master_config_path is not None:
        master_config_path = Path(master_config_path).resolve()
        mount_yaml = Path(__file__).parent.joinpath("mount.yaml").resolve()
        extra_files.append(str(mount_yaml))
    if version is None:
        version = determined_deploy.__version__
    if autorestart:
        restart_policy = "unless-stopped"
    else:
        restart_policy = "no"
    env = {
        "INTEGRATIONS_HOST_PORT": str(port),
        "DET_MASTER_CONFIG": str(master_config_path),
        "DET_DB_PASSWORD": db_password,
        "DET_VERSION": version,
        "DET_RESTART_POLICY": restart_policy,
    }
    master_down(master_name, delete_db)
    docker_compose(command, master_name, env, extra_files=extra_files)
    _wait_for_master("localhost", port, cluster_name)


def master_down(master_name: str, delete_db: bool) -> None:
    if delete_db:
        docker_compose(["down", "--volumes", "-t", "1"], master_name)
    else:
        docker_compose(["down", "-t", "1"], master_name)


def cluster_up(
    num_agents: int,
    port: int,
    master_config_path: Path,
    cluster_name: str,
    version: Optional[str],
    db_password: str,
    delete_db: bool,
    no_gpu: bool,
    autorestart: bool,
) -> None:
    cluster_down(cluster_name, delete_db)
    master_up(
        port=port,
        master_config_path=master_config_path,
        master_name=cluster_name,
        version=version,
        db_password=db_password,
        delete_db=delete_db,
        autorestart=autorestart,
        cluster_name=cluster_name,
    )
    for agent_number in range(num_agents):
        agent_name = cluster_name + f"-agent-{agent_number}"
        labels = {"determined.cluster": cluster_name}
        agent_up(
            master_host="localhost",
            master_port=port,
            agent_name=agent_name,
            agent_label=None,
            agent_resource_pool=None,
            version=version,
            labels=labels,
            no_gpu=no_gpu,
            autorestart=autorestart,
            cluster_name=cluster_name,
        )


def cluster_down(cluster_name: str, delete_db: bool) -> None:
    master_down(master_name=cluster_name, delete_db=delete_db)
    stop_cluster_agents(cluster_name=cluster_name)


def logs(cluster_name: str, no_follow: bool) -> None:
    docker_compose(["logs"] if no_follow else ["logs", "-f"], cluster_name)


def agent_up(
    master_host: str,
    master_port: int,
    agent_name: str,
    agent_label: Optional[str],
    agent_resource_pool: Optional[str],
    version: Optional[str],
    no_gpu: bool,
    autorestart: bool,
    cluster_name: str,
    labels: Optional[Dict] = None,
) -> None:
    if version is None:
        version = determined_deploy.__version__

    _wait_for_master(master_host, master_port, cluster_name)

    if master_host == "localhost":
        master_host = get_proxy_addr()
    image = "determinedai/determined-agent:{}".format(version)
    environment = {
        "DET_MASTER_HOST": master_host,
        "DET_MASTER_PORT": master_port,
        "DET_AGENT_ID": agent_name,
        "DET_LABEL": agent_label,
        "DET_RESOURCE_POOL": agent_resource_pool,
    }
    init = True
    volumes = ["/var/run/docker.sock:/var/run/docker.sock"]
    mounts = []  # type: List[str]
    if labels is None:
        labels = {}
    labels["ai.determined.type"] = "agent"
    if autorestart:
        restart_policy = {"Name": "unless-stopped"}  # type: Optional[Dict[str, str]]
    else:
        restart_policy = None
    if no_gpu:
        device_requests = None
    else:
        device_requests = [GPU_DEVICE_REQUEST]

    docker_client = docker.from_env()

    print(f"Starting {agent_name}")
    docker_client.containers.run(
        image=image,
        environment=environment,
        init=init,
        mounts=mounts,
        volumes=volumes,
        network_mode="host",
        name=agent_name,
        detach=True,
        labels=labels,
        restart_policy=restart_policy,
        device_requests=device_requests,
    )


def _kill_containers(containers: docker.models.containers.Container) -> None:
    for container in containers:
        print(f"Stopping {container.name}")
        container.stop(timeout=20)
        print(f"Removing {container.name}")
        container.remove()


def stop_all_agents() -> None:
    docker_client = docker.from_env()
    filters = {"label": ["ai.determined.type=agent"]}
    to_stop = docker_client.containers.list(all=True, filters=filters)
    _kill_containers(to_stop)


def stop_cluster_agents(cluster_name: str) -> None:
    docker_client = docker.from_env()
    labels = [f"determined.cluster={cluster_name}"]
    filters = {"label": labels}
    to_stop = docker_client.containers.list(all=True, filters=filters)
    _kill_containers(to_stop)


def stop_agent(agent_name: str) -> None:
    docker_client = docker.from_env()
    filters = {"name": [agent_name]}
    to_stop = docker_client.containers.list(all=True, filters=filters)
    _kill_containers(to_stop)
