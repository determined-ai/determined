import contextlib
import getpass
import os
import pathlib
import random
import re
import secrets
import socket
import string
import subprocess
import sys
import tempfile
import threading
import time
import warnings
from typing import Any, Callable, Dict, Generator, List, Optional, Sequence, Type

import appdirs
import docker

from determined.common import api, constants, util
from determined.common.api import authentication
from determined.deploy import errors, healthcheck
from determined.experimental import client as determined

AGENT_NAME_DEFAULT = f"det-agent-{socket.gethostname()}"
MASTER_PORT_DEFAULT = 8080

# Default names are consistent with previous docker-compose names to support migration.
DB_NAME = "determined-db_1"
NETWORK_NAME = "determined_default"
VOLUME_NAME = "determined-db-volume"
MASTER_NAME = "determined-master_1"
OSS_EDITION = "determined"
ENTERPRISE_EDITION = "hpe-mlde"

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

_docker_client = None


def docker_client() -> docker.client:
    global _docker_client
    if not _docker_client:
        _docker_client = docker.from_env()
    return _docker_client


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


def _wait_for_master(master: str, cluster_name: str) -> None:
    try:
        healthcheck.wait_for_master(master)
        return
    except errors.MasterTimeoutExpired:
        print("Timed out connecting to master, but attempting to dump logs from cluster...")
        logs(cluster_name=cluster_name, follow=False)
        raise ConnectionError("Timed out connecting to master")


def _wait_for_container(container_name: str, timeout: int = 100) -> None:
    """
    Waits for a Docker container's healthcheck (specified when the container is run) to reach a
    "healthy" status.
    """
    print(f"Waiting for {container_name}...")
    client = docker_client()
    container = client.containers.get(container_name)
    start_time = time.time()
    while time.time() - start_time < timeout:
        inspect = client.api.inspect_container(container.name)
        if inspect["State"]["Health"]["Status"] == "healthy":
            return
        time.sleep(1)
    raise TimeoutError


def master_up(
    port: int,
    initial_user_password: Optional[str],
    master_config_path: Optional[pathlib.Path],
    storage_host_path: pathlib.Path,
    master_name: Optional[str],
    image_repo_prefix: str,
    version: str,
    db_password: str,
    delete_db: bool,
    autorestart: bool,
    cluster_name: str,
    auto_work_dir: Optional[pathlib.Path],
    enterprise_edition: bool = False,
) -> None:
    # Some cli flags for det deploy local will cause us to write a temporary master.yaml.
    make_temp_conf = False
    # If we have to generate a password for the user, we should print it later.
    generated_user_password = None

    if master_name is None:
        master_name = f"{cluster_name}_{MASTER_NAME}"

    if master_config_path is not None:
        with master_config_path.open() as f:
            master_conf = util.yaml_safe_load(f)
    else:
        master_conf = MASTER_CONF_DEFAULT
        make_temp_conf = True

    if master_conf.get("security") is None:
        master_conf["security"] = {}

    if initial_user_password is not None:
        master_conf["security"]["initial_user_password"] = initial_user_password
        make_temp_conf = True

    try:
        authentication.check_password_complexity(
            master_conf["security"].get("initial_user_password")
        )
    except ValueError:
        random_password_characters = string.ascii_uppercase + string.ascii_lowercase + string.digits
        random_password = [
            secrets.choice(string.ascii_lowercase),
            secrets.choice(string.ascii_uppercase),
            secrets.choice(string.digits),
        ]
        random_password.extend([secrets.choice(random_password_characters) for _ in range(13)])
        random.shuffle(random_password)
        generated_user_password = "".join(random_password)

        master_conf["security"]["initial_user_password"] = generated_user_password
        make_temp_conf = True

    if storage_host_path is not None:
        master_conf["checkpoint_storage"] = {
            "type": "shared_fs",
            "host_path": str(storage_host_path.resolve()),
        }
        make_temp_conf = True

    # Ensure checkpoint storage directory exists.
    storage_info = master_conf.get("checkpoint_storage", {})
    final_storage_host_path = storage_info.get("host_path")
    container_storage_path = (
        storage_info.get("container_path") or constants.SHARED_FS_CONTAINER_PATH
    )

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
            util.yaml_safe_dump(master_conf, f)
        master_config_path = pathlib.Path(temp_path)

    # This is always true by now, but mypy needs help.
    assert master_config_path is not None
    restart_policy = "unless-stopped" if autorestart else "no"
    env = {
        "DET_MASTER_HTTP_PORT": str(port),
        "DET_DB_PASSWORD": db_password,
        "DET_LOG_INFO": "info",
    }

    # Kill existing master container if exists.
    master_down(master_name=master_name, delete_db=delete_db, cluster_name=cluster_name)

    # Create network.
    client = docker_client()

    # Start db and wait for healthcheck.
    db_name = f"{cluster_name}_{DB_NAME}"
    volume_name = f"{cluster_name}_{VOLUME_NAME}"

    @contextlib.contextmanager
    def defer_cleanup(fn: Callable[[], None]) -> Generator:
        """
        Defer cleanup tasks for each resource if Exceptions are caught.
        """
        try:
            yield
        except Exception as ex:
            fn()
            raise ex

    with contextlib.ExitStack() as exit_stack:
        # Create network used by DB and master.
        print(f"Creating network {NETWORK_NAME}...")
        exit_stack.enter_context(defer_cleanup(lambda: remove_network(NETWORK_NAME)))
        client.networks.create(name=NETWORK_NAME, attachable=True)

        # Start up db.
        exit_stack.enter_context(defer_cleanup(lambda: db_down(db_name, VOLUME_NAME, delete_db)))
        db_up(
            name=db_name,
            password=db_password,
            network_name=NETWORK_NAME,
            cluster_name=cluster_name,
            volume_name=volume_name,
        )

        # Wait for db to reach a healthy state.
        _wait_for_container(db_name, timeout=5)

        # Remove cleanup methods from ExitStack after DB successfully starts.
        exit_stack.pop_all()

        # Start master instance.
        print(f"Creating {master_name}...")
        exit_stack.enter_context(
            defer_cleanup(lambda: master_down(master_name, delete_db, cluster_name))  # type: ignore
        )
        volumes = [
            f"{os.path.abspath(master_config_path)}:/etc/determined/master.yaml",
            f"{final_storage_host_path}:{container_storage_path}",
        ]
        det_edition = ENTERPRISE_EDITION if enterprise_edition else OSS_EDITION
        client.containers.run(
            image=f"{image_repo_prefix}/{det_edition}-master:{version}",
            environment=env,
            init=True,
            mounts=[],
            volumes=volumes,
            name=master_name,
            detach=True,
            labels={},
            restart_policy={"Name": restart_policy},
            device_requests=None,
            ports={"8080": f"{port}"},
            network=NETWORK_NAME,
            hostname="determined-master",
        )

        _wait_for_master(f"http://localhost:{port}", cluster_name)

        if generated_user_password is not None:
            try:
                sess = authentication.login(
                    f"http://localhost:{port}", "admin", generated_user_password
                ).with_retry(util.get_max_retries_config())
                sess.get("/api/v1/me")

                # No exception was raised, so this generated password is the way to log in.
                print(
                    "Determined Master was launched without a strong initial_user_password set. "
                    + "Please set a strong password by following prompts, or by logging in "
                    + "with generated passwords and following the password change process."
                )

                try:
                    if not sys.stdin.isatty() or sys.stdin.closed:
                        raise EOFError
                    # getpass raises this warning instead of an exception if it can't find the
                    # terminal, which almost always means we're not going to be able to receive
                    # a password interactively and securely, so should abort quickly rather
                    # than time out.
                    with warnings.catch_warnings():
                        warnings.filterwarnings(action="error", category=getpass.GetPassWarning)
                        prompt = (
                            "Please enter a password for the built-in "
                            + "`determined` and `admin` users: "
                        )
                        new_password = getpass.getpass(prompt)
                        # Give one more chance if this password is too weak
                        try:
                            authentication.check_password_complexity(new_password)
                        except ValueError as e:
                            print(e)
                            new_password = getpass.getpass(prompt)

                        authentication.check_password_complexity(new_password)
                        new_password_check = getpass.getpass("Enter the password again: ")
                        if new_password != new_password_check:
                            raise ValueError("passwords did not match")

                        d = determined.Determined._from_session(sess)
                        user = d.get_user_by_name("determined")
                        user.change_password(new_password)
                        user = d.get_user_by_name("admin")
                        user.change_password(new_password)

                except Exception:
                    # User could exit, or might be unable to pass validation,
                    # or this might not even be interactive; none of these
                    # are problems with the deployment itself, so just print
                    # the password so users aren't locked out
                    print(
                        "A password has been created for you. The admin and determined users "
                        + f"can log in with this password:\n\t{generated_user_password}\n"
                    )

            except api.errors.UnauthenticatedException:
                # There was a non-generated password there already; carry on
                pass

        # Remove all cleanup methods from ExitStack.
        exit_stack.pop_all()


def db_up(name: str, password: str, network_name: str, cluster_name: str, volume_name: str) -> None:
    print(f"Creating {name}...")
    env = {"PGUSER": "postgres", "POSTGRES_DB": "determined", "POSTGRES_PASSWORD": password}
    client = docker_client()
    client.containers.run(
        image="postgres:10.14",
        environment=env,
        mounts=[],
        volumes=[f"{volume_name}:/var/lib/postgresql/data"],
        name=name,
        detach=True,
        labels={},
        restart_policy={"Name": "unless-stopped"},
        device_requests=None,
        command="--max_connections=96 --shared_buffers=512MB",
        healthcheck={
            "test": ["CMD-SHELL", "pg_isready", "-d", "determined"],
            "interval": 1000000,
        },
        network=network_name,
        hostname="determined-db",
    )


def master_down(master_name: str, delete_db: bool, cluster_name: str) -> None:
    if master_name is None:
        master_name = f"{cluster_name}_{MASTER_NAME}"

    _kill_containers(names=[master_name])

    volume_name = f"{cluster_name}_{VOLUME_NAME}"
    db_name = f"{cluster_name}_{DB_NAME}"

    db_down(db_name=db_name, volume_name=volume_name, delete_volume=delete_db)
    remove_network(NETWORK_NAME)


def db_down(db_name: str, volume_name: str, delete_volume: bool) -> None:
    client = docker_client()

    _kill_containers([db_name])
    if delete_volume:
        try:
            volume = client.volumes.get(volume_name)
            print(f"Removing db volume {volume_name}")
            volume.remove()
        except docker.errors.NotFound:
            print(f"Volume {volume_name} not found.")


def remove_network(network_name: str) -> None:
    client = docker_client()
    networks = client.networks.list(names=[network_name])
    for network in networks:
        print(f"Removing network {network.name}")
        network.remove()


def cluster_up(
    num_agents: int,
    port: int,
    initial_user_password: Optional[str],
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
    enterprise_edition: bool = False,
) -> None:
    cluster_down(cluster_name, delete_db)
    master_up(
        port=port,
        initial_user_password=initial_user_password,
        master_config_path=master_config_path,
        storage_host_path=storage_host_path,
        master_name=f"{cluster_name}_{MASTER_NAME}",
        image_repo_prefix=image_repo_prefix,
        version=version,
        db_password=db_password,
        delete_db=delete_db,
        autorestart=autorestart,
        cluster_name=cluster_name,
        auto_work_dir=auto_work_dir,
        enterprise_edition=enterprise_edition,
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
            enterprise_edition=enterprise_edition,
        )


def cluster_down(cluster_name: str, delete_db: bool) -> None:
    master_down(
        master_name=f"{cluster_name}_{MASTER_NAME}", delete_db=delete_db, cluster_name=cluster_name
    )
    stop_cluster_agents(cluster_name=cluster_name)


def logs(cluster_name: str, follow: bool) -> None:
    def docker_logs(container_name: str) -> None:
        client = docker_client()
        try:
            container = client.containers.get(container_name)
        except docker.errors.NotFound:
            return
        log_stream = container.logs(stream=follow)
        if not follow:
            print(log_stream.decode("utf-8"))
            return
        log_line = next(log_stream)
        while log_line:
            print(log_line.decode("utf-8").strip())
            log_line = next(log_stream)

    master_name = f"{cluster_name}_{MASTER_NAME}"
    db_name = f"{cluster_name}_{DB_NAME}"
    master_thread = threading.Thread(target=docker_logs, args=(master_name,))
    db_thread = threading.Thread(target=docker_logs, args=(db_name,))
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
    enterprise_edition: bool = False,
) -> None:
    agent_conf = {}
    volumes = ["/var/run/docker.sock:/var/run/docker.sock"]
    if agent_config_path is not None:
        with agent_config_path.open() as f:
            agent_conf = util.yaml_safe_load(f)
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

    _wait_for_master(f"http://{master_host}:{master_port}", cluster_name)

    if master_host == "localhost":
        master_host = get_proxy_addr()
    environment["DET_MASTER_HOST"] = master_host

    det_edition = ENTERPRISE_EDITION if enterprise_edition else OSS_EDITION
    image = f"{image_repo_prefix}/{det_edition}-agent:{version}"
    init = True
    mounts = []  # type: List[str]
    if labels is None:
        labels = {}
    labels["ai.determined.type"] = "agent"

    restart_policy = {"Name": "unless-stopped"} if autorestart else None
    device_requests = [GPU_DEVICE_REQUEST] if gpu else None

    client = docker_client()
    print(f"Starting {agent_name}")
    client.containers.run(
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


def _kill_containers(names: Optional[List[str]] = None, labels: Optional[List[str]] = None) -> None:
    filters = {}
    if names:
        filters["name"] = names
    if labels:
        filters["label"] = labels
    client = docker_client()
    containers = client.containers.list(all=True, filters=filters)
    for container in containers:
        # Docker will match container names containing the string instead of strictly matching.
        if names and container.name not in names:
            continue
        print(f"Stopping {container.name}")
        container.stop(timeout=20)
        print(f"Removing {container.name}")
        container.remove()


def stop_all_agents() -> None:
    _kill_containers(labels=["ai.determined.type=agent"])


def stop_cluster_agents(cluster_name: str) -> None:
    _kill_containers(labels=[f"determined.cluster={cluster_name}"])


def stop_agent(agent_name: str) -> None:
    _kill_containers(names=[agent_name])
