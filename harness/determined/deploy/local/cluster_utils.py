import os
import re
import socket
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence

import appdirs
import docker

import determined
import determined.deploy
from determined.common import yaml
from determined.deploy.errors import MasterTimeoutExpired
from determined.deploy.healthcheck import wait_for_master

AGENT_NAME_DEFAULT = f"det-agent-{socket.gethostname()}"
MASTER_PORT_DEFAULT = 8080

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
        process_env.update(env)
    process_env["INTEGRATIONS_PROXY_ADDR"] = get_proxy_addr()
    base_command = ["docker-compose", "-f", str(path), "-p", cluster_name]
    if extra_files is not None:
        for extra_file in extra_files:
            base_command += ["-f", extra_file]
    args = base_command + args
    subprocess.check_call(args, env=process_env)


def _wait_for_master(master_host: str, master_port: int, cluster_name: str) -> None:
    try:
        wait_for_master(master_host, master_port, timeout=100)
        return
    except MasterTimeoutExpired:
        print("Timed out connecting to master, but attempting to dump logs from cluster...")
        docker_compose(["logs"], cluster_name)
        raise ConnectionError("Timed out connecting to master")


def master_up(
    port: int,
    master_config_path: Optional[Path],
    storage_host_path: Path,
    master_name: str,
    image_repo_prefix: Optional[str],
    version: Optional[str],
    db_password: str,
    delete_db: bool,
    autorestart: bool,
    cluster_name: str,
    auto_work_dir: Optional[Path],
) -> None:
    command = ["up", "-d"]
    if image_repo_prefix is None:
        image_repo_prefix = "determinedai"
    if version is None:
        version = determined.__version__
    if autorestart:
        restart_policy = "unless-stopped"
    else:
        restart_policy = "no"

    env = {
        "INTEGRATIONS_HOST_PORT": str(port),
        "DET_DB_PASSWORD": db_password,
        "IMAGE_REPO_PREFIX": image_repo_prefix,
        "DET_VERSION": version,
        "DET_RESTART_POLICY": restart_policy,
    }

    # Some cli flags for det deploy local will cause us to write a temporary master.yaml.
    master_conf = {}
    make_temp_conf = False

    if master_config_path is not None:
        with master_config_path.open() as f:
            master_conf = yaml.safe_load(f)
    else:
        # These defaults come from master/packaging/master.yaml (except for host_path).
        master_conf = {
            "db": {
                "user": "postgres",
                "host": "determined-db",
                "port": 5432,
                "name": "determined",
            },
            "checkpoint_storage": {
                "type": "shared_fs",
                "host_path": appdirs.user_data_dir("determined"),
                "save_experiment_best": 0,
                "save_trial_best": 1,
                "save_trial_latest": 1,
            },
        }
        make_temp_conf = True

    if storage_host_path is not None:
        master_conf["checkpoint_storage"] = {
            "type": "shared_fs",
            "host_path": str(storage_host_path.resolve()),
        }
        make_temp_conf = True

    if auto_work_dir is not None:
        work_dir = str(auto_work_dir.resolve())
        master_conf.setdefault("task_container_defaults", {})["work_dir"] = work_dir
        master_conf["task_container_defaults"].setdefault("bind_mounts", []).append(
            {"host_path": work_dir, "container_path": work_dir}
        )
        make_temp_conf = True

    # Ensure checkpoint storage directory exists.
    final_storage_host_path = master_conf.get("checkpoint_storage", {}).get("host_path")
    if final_storage_host_path is not None:
        final_storage_host_path = Path(final_storage_host_path)
        if not final_storage_host_path.exists():
            final_storage_host_path.mkdir(parents=True)

    if make_temp_conf:
        fd, temp_path = tempfile.mkstemp(prefix="det-deploy-local-master-config-")
        with open(fd, "w") as f:
            yaml.dump(master_conf, f)
        master_config_path = Path(temp_path)

    # This is always true by now, but mypy needs help.
    assert master_config_path is not None

    env["DET_MASTER_CONFIG"] = str(master_config_path.resolve())

    master_down(master_name, delete_db)
    docker_compose(command, master_name, env)
    _wait_for_master("localhost", port, cluster_name)


def master_down(master_name: str, delete_db: bool) -> None:
    if delete_db:
        docker_compose(["down", "--volumes", "-t", "1"], master_name)
    else:
        docker_compose(["down", "-t", "1"], master_name)


def cluster_up(
    num_agents: int,
    port: int,
    master_config_path: Optional[Path],
    storage_host_path: Path,
    cluster_name: str,
    image_repo_prefix: Optional[str],
    version: Optional[str],
    db_password: str,
    delete_db: bool,
    gpu: bool,
    autorestart: bool,
    auto_work_dir: Optional[Path],
) -> None:
    cluster_down(cluster_name, delete_db)
    master_up(
        port=port,
        master_config_path=master_config_path,
        storage_host_path=storage_host_path,
        master_name=cluster_name,
        image_repo_prefix=image_repo_prefix,
        version=version,
        db_password=db_password,
        delete_db=delete_db,
        autorestart=autorestart,
        cluster_name=cluster_name,
        auto_work_dir=auto_work_dir,
    )
    for agent_number in range(num_agents):
        agent_name = cluster_name + f"-agent-{agent_number}"
        labels = {"determined.cluster": cluster_name}
        agent_up(
            master_host="localhost",
            master_port=port,
            agent_config_path=None,
            agent_name=agent_name,
            agent_label=None,
            agent_resource_pool=None,
            image_repo_prefix=image_repo_prefix,
            version=version,
            labels=labels,
            gpu=gpu,
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
    agent_config_path: Optional[Path],
    agent_name: str,
    agent_label: Optional[str],
    agent_resource_pool: Optional[str],
    image_repo_prefix: Optional[str],
    version: Optional[str],
    gpu: bool,
    autorestart: bool,
    cluster_name: str,
    labels: Optional[Dict] = None,
) -> None:
    agent_conf = {}
    volumes = ["/var/run/docker.sock:/var/run/docker.sock"]
    if agent_config_path is not None:
        with agent_config_path.open() as f:
            agent_conf = yaml.safe_load(f)
        volumes += [f"{os.path.abspath(agent_config_path)}:/etc/determined/agent.yaml"]

    # Fallback on agent config for options not specified as flags.
    environment = {}
    if agent_name == AGENT_NAME_DEFAULT:
        agent_name = agent_conf.get("agent_id", agent_name)
    else:
        environment["DET_AGENT_ID"] = agent_name
    environment["DET_MASTER_PORT"] = str(master_port)
    if master_port == MASTER_PORT_DEFAULT:
        if "master_port" in agent_conf:
            del environment["DET_MASTER_PORT"]
            master_port = agent_conf["master_port"]

    if agent_label is not None:
        environment["DET_LABEL"] = agent_label
    if agent_resource_pool is not None:
        environment["DET_RESOURCE_POOL"] = agent_resource_pool
    if image_repo_prefix is None:
        image_repo_prefix = "determinedai"
    if version is None:
        version = determined.__version__

    _wait_for_master(master_host, master_port, cluster_name)

    if master_host == "localhost":
        master_host = get_proxy_addr()
    environment["DET_MASTER_HOST"] = master_host

    image = f"{image_repo_prefix}/determined-agent:{version}"
    init = True
    mounts = []  # type: List[str]
    if labels is None:
        labels = {}
    labels["ai.determined.type"] = "agent"

    restart_policy = {"Name": "unless-stopped"} if autorestart else None
    device_requests = [GPU_DEVICE_REQUEST] if gpu else None

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
