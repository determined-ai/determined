import os
import pathlib
import re
import socket
import subprocess
import sys
import tempfile
import threading
from typing import Any, Dict, List, Optional, Sequence, Type

import appdirs
import docker

from determined.common import yaml
from determined.deploy.errors import MasterTimeoutExpired
from determined.deploy.healthcheck import wait_for_master

AGENT_NAME_DEFAULT = f"det-agent-{socket.gethostname()}"
MASTER_PORT_DEFAULT = 8080
DB_NAME = "determined-db"
NETWORK_NAME = "determined-network"
VOLUME_NAME = "determined-db-volume"

# This object, when included in the host config in a container creation request, tells Docker to
# expose all host GPUs inside a container.
GPU_DEVICE_REQUEST = {"Driver": "nvidia", "Count": -1, "Capabilities": [["gpu", "utility"]]}

Container = Type[docker.models.containers.Container]

# These defaults come from master/packaging/master.yaml (except for host_path).
MASTER_CONF_DEFAULT = {
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

docker_client = docker.from_env()


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


def _wait_for_master(master_host: str, master_port: int, cluster_name: str) -> None:
    try:
        wait_for_master(master_host, master_port, timeout=100)
        return
    except MasterTimeoutExpired:
        print("Timed out connecting to master, but attempting to dump logs from cluster...")
        logs(cluster_name=cluster_name, no_follow=True)
        raise ConnectionError("Timed out connecting to master")


def master_up(
    port: int,
    master_config_path: Optional[pathlib.Path],
    storage_host_path: pathlib.Path,
    master_name: str,
    image_repo_prefix: str,
    version: str,
    db_password: str,
    delete_db: bool,
    autorestart: bool,
    cluster_name: str,
    auto_work_dir: Optional[pathlib.Path],
) -> None:
    # Some cli flags for det deploy local will cause us to write a temporary master.yaml.
    make_temp_conf = False

    if master_config_path is not None:
        with master_config_path.open() as f:
            master_conf = yaml.safe_load(f)
    else:
        master_conf = MASTER_CONF_DEFAULT
        make_temp_conf = True

    if storage_host_path is not None:
        master_conf["checkpoint_storage"] = {
            "type": "shared_fs",
            "host_path": str(storage_host_path.resolve()),
        }
        make_temp_conf = True

    # Ensure checkpoint storage directory exists.
    final_storage_host_path = master_conf.get("checkpoint_storage", {}).get("host_path")
    if final_storage_host_path is not None:
        final_storage_host_path = pathlib.Path(final_storage_host_path)
        if not final_storage_host_path.exists():
            final_storage_host_path.mkdir(parents=True)

    if auto_work_dir is not None:
        work_dir = str(auto_work_dir.resolve())
        master_conf.setdefault("task_container_defaults", {})["work_dir"] = work_dir
        master_conf["task_container_defaults"].setdefault("bind_mounts", []).append(
            {"host_path": work_dir, "container_path": work_dir}
        )
        make_temp_conf = True

    if make_temp_conf:
        fd, temp_path = tempfile.mkstemp(prefix="det-deploy-local-master-config-")
        with open(fd, "w") as f:
            yaml.dump(master_conf, f)
        master_config_path = pathlib.Path(temp_path)

    # This is always true by now, but mypy needs help.
    assert master_config_path is not None
    restart_policy = "unless-stopped" if autorestart else "no"
    env = {
        "INTEGRATIONS_HOST_PORT": str(port),
        "DET_DB_PASSWORD": db_password,
        "IMAGE_REPO_PREFIX": image_repo_prefix,
        "DET_VERSION": version,
        "DET_RESTART_POLICY": restart_policy,
        "DET_MASTER_CONFIG": str(master_config_path.resolve()),
    }

    # Kill existing master container if exists.
    master_down(master_name=master_name, delete_db=delete_db)

    # Create network.
    docker_client.networks.create(NETWORK_NAME)

    # Start db.
    db_up(password=db_password, network=NETWORK_NAME)

    # Start master instance.
    volumes = [f"{os.path.abspath(master_config_path)}:/etc/determined/master.yaml"]
    docker_client.containers.run(
        image=f"{image_repo_prefix}/determined-master:{version}",
        environment=env,
        init=True,
        mounts=[],
        volumes=volumes,
        name=master_name,
        detach=True,
        labels={},
        restart_policy={"Name": restart_policy},
        device_requests=None,
        ports={f"{port}": "8080"},
        network=NETWORK_NAME,
    )
    _wait_for_master("localhost", port, cluster_name)


def db_up(password: str, network: str) -> None:
    env = {"POSTGRES_DB": "determined", "POSTGRES_PASSWORD": password}

    docker_client.containers.run(
        image="postgres:10.14",
        environment=env,
        init=True,
        mounts=[],
        volumes=[f"{VOLUME_NAME}:/var/lib/postgresql/data"],
        name=DB_NAME,
        detach=True,
        network=network,
        labels={},
        restart_policy={"Name": "unless-stopped"},
        device_requests=None,
    )


def master_down(master_name: str, delete_db: bool) -> None:
    # Kill master instance.
    _kill_container(master_name)

    # Remove existing db container if exists.
    _kill_container(DB_NAME)

    # Remove the volume if specified.
    if delete_db:
        print(f"Removing db volume {VOLUME_NAME}")
        volume = docker_client.volumes.get(VOLUME_NAME)
        volume.remove()

    # Remove network if exists.
    networks = docker_client.networks.list(names=[NETWORK_NAME])
    for network in networks:
        print(f"Removing network {network.name}")
        network.remove()


def cluster_up(
    num_agents: int,
    port: int,
    master_config_path: Optional[pathlib.Path],
    storage_host_path: pathlib.Path,
    cluster_name: str,
    image_repo_prefix: str,
    version: str,
    db_password: str,
    delete_db: bool,
    gpu: bool,
    autorestart: bool,
    auto_work_dir: Optional[pathlib.Path],
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
    def docker_logs(container_name: str) -> None:
        container = docker_client.containers.get(container_name)
        log_stream = container.logs(stream=not no_follow)
        if no_follow:
            print(log_stream.decode("utf-8"))
            return
        log_line = next(log_stream)
        while log_line:
            print(log_line.decode("utf-8").strip())
            log_line = next(log_stream)

    master_thread = threading.Thread(target=docker_logs, args=(cluster_name,))
    db_thread = threading.Thread(target=docker_logs, args=(DB_NAME,))
    db_thread.start()
    master_thread.start()


def agent_up(
    master_host: str,
    master_port: int,
    agent_config_path: Optional[pathlib.Path],
    agent_name: str,
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

    if agent_resource_pool is not None:
        environment["DET_RESOURCE_POOL"] = agent_resource_pool

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


def _kill_container(container_name: str) -> None:
    try:
        container = docker_client.containers.get(container_name)
        print(f"Stopping {container.name}")
        container.stop(timeout=20)
        print(f"Removing {container.name}")
        container.remove()
    except docker.errors.NotFound:
        return


def _kill_containers(containers: List[Container]) -> None:
    for container in containers:
        print(f"Stopping {container.name}")
        container.stop(timeout=20)
        print(f"Removing {container.name}")
        container.remove()


def stop_all_agents() -> None:
    filters = {"label": ["ai.determined.type=agent"]}
    to_stop = docker_client.containers.list(all=True, filters=filters)
    _kill_containers(to_stop)


def stop_cluster_agents(cluster_name: str) -> None:
    labels = [f"determined.cluster={cluster_name}"]
    filters = {"label": labels}
    to_stop = docker_client.containers.list(all=True, filters=filters)
    _kill_containers(to_stop)


def stop_agent(agent_name: str) -> None:
    filters = {"name": [agent_name]}
    to_stop = docker_client.containers.list(all=True, filters=filters)
    _kill_containers(to_stop)
