import os
import re
import subprocess
import sys
import time
from pathlib import Path
from typing import Dict, List, Optional, Sequence

import docker
import requests

import determined_deploy
from determined_common import api
from determined_deploy import config


def get_shell_id() -> str:
    args = ["id", "-u", "-n"]
    byte_str: str = subprocess.check_output(args, encoding="utf-8")
    return byte_str.rstrip("\n").strip("'").strip()


def get_proxy_addr() -> str:
    # The Determined proxying code relies on docker port-mapping container ports to host
    # ports, and it uses the IP address of the agent as a way to address spawned
    # docker containers. This breaks down when running in a docker compose
    # environment, because the address of the agent is not the address of the
    # docker host. As a work-around, force agents to report their IP address as the
    # IP address of the host machine.
    if "darwin" in sys.platform:
        # On MacOS, docker runs in a VM and host.docker.internal points to the IP
        # address of this VM.
        return "host.docker.internal"
    else:
        # On non-MacOS, host.docker.internal does not exist. Instead, grab the source IP
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
    args: List[str], cluster_name: str, env: Optional[Dict] = None, extra_files: List[str] = None
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


def _wait_for_master() -> None:
    for _ in range(50):
        try:
            r = api.get(config.make_master_url(), "info", authenticated=False)
            if r.status_code == requests.codes.ok:
                return
        except api.errors.MasterNotFoundException:
            pass
        print("Waiting for master to be available...")
        time.sleep(2)
    raise ConnectionError("Timed out connecting to Master")


def master_up(
    port: Optional[int],
    etc_path: Path,
    master_name: str,
    version: str,
    db_password: str,
    hasura_secret: str,
    delete_db: bool,
):
    config.MASTER_PORT = port
    command = ["up", "-d"]
    extra_files = []
    if etc_path is not None:
        etc_path = Path(etc_path).resolve()
        mount_yaml = Path(__file__).parent.joinpath("mount.yaml").resolve()
        extra_files.append(str(mount_yaml))
    if version is None:
        version = determined_deploy.__version__
    env = {
        "INTEGRATIONS_HOST_PORT": str(port),
        "DET_ETC_ROOT": str(etc_path),
        "DET_DB_PASSWORD": db_password,
        "DET_HASURA_SECRET": hasura_secret,
        "DET_VERSION": version,
    }
    master_down(master_name, delete_db)
    docker_compose(command, master_name, env, extra_files=extra_files)
    _wait_for_master()


def master_down(master_name: str, delete_db: bool) -> None:
    if delete_db:
        docker_compose(["down", "--volumes", "-t", "1"], master_name)
    else:
        docker_compose(["down", "-t", "1"], master_name)


def fixture_up(
    num_agents: Optional[int],
    port: Optional[int],
    etc_path: Path,
    cluster_name: str,
    version: str,
    db_password: str,
    hasura_secret: str,
    delete_db: bool,
    no_gpu: bool,
):
    master_up(
        port=port,
        etc_path=etc_path,
        master_name=cluster_name,
        version=version,
        db_password=db_password,
        hasura_secret=hasura_secret,
        delete_db=delete_db,
    )
    for agent_number in range(num_agents):
        agent_name = cluster_name + f"-agent-{agent_number}"
        labels = {"determined.cluster": cluster_name}
        agent_up(
            master_host=get_proxy_addr(),
            master_port=port,
            agent_name=agent_name,
            version=version,
            labels=labels,
            no_gpu=no_gpu,
        )


def fixture_down(cluster_name: str, delete_db: bool) -> None:
    master_down(master_name=cluster_name, delete_db=delete_db)
    stop_cluster_agents(cluster_name=cluster_name)


def logs(cluster_name: str) -> None:
    docker_compose(["logs", "-f"], cluster_name)


def agent_up(
    master_host: Optional[str],
    master_port: Optional[int],
    agent_name: Optional[str],
    version: Optional[str],
    no_gpu: bool,
    labels: Dict = None,
) -> None:
    if version is None:
        version = determined_deploy.__version__
    image = "determinedai/determined-agent:{}".format(version)
    environment = {
        "DET_MASTER_HOST": master_host,
        "DET_MASTER_PORT": master_port,
        "DET_AGENT_ID": agent_name,
    }
    init = True
    volumes = ["/var/run/docker.sock:/var/run/docker.sock"]
    mounts = []
    if labels is None:
        labels = {}
    labels["ai.determined.type"] = "agent"
    config.MASTER_HOST = master_host
    config.MASTER_PORT = master_port
    _wait_for_master()
    docker_client = docker.from_env()
    if "nvidia" in docker_client.info()["Runtimes"] and not no_gpu:
        runtime = "nvidia"
    else:
        runtime = None
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
        runtime=runtime,
        labels=labels,
    )


def _kill_containers(containers):
    for container in containers:
        print(f"Stopping {container.name}")
        container.stop(timeout=20)
        print(f"Removing {container.name}")
        container.remove()


def stop_all_agents():
    docker_client = docker.from_env()
    filters = {"label": ["ai.determined.type=agent"]}
    to_stop = docker_client.containers.list(all=True, filters=filters)
    _kill_containers(to_stop)


def stop_cluster_agents(cluster_name: str):
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
