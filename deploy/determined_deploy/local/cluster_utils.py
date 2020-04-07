import os
import re
import subprocess
import sys
import time
from pathlib import Path
from typing import Dict, List, Optional, Sequence

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


def docker_compose(args: List[str], cluster_name: str, env: Optional[Dict] = None) -> None:
    path = Path(__file__).parent.joinpath("docker-compose.yaml")
    # Start with the user's environment to ensure that Docker and Docker Compose work correctly.
    process_env = dict(os.environ)
    if env is not None:
        # raise ValueError(str(env))
        process_env.update(env)
    process_env["DET_VERSION"] = determined_deploy.__version__
    process_env["INTEGRATIONS_PROXY_ADDR"] = get_proxy_addr()
    args = ["docker-compose", "-f", str(path), "-p", cluster_name] + args
    subprocess.run(args, env=process_env)


def _wait_for_master(port: int) -> None:
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


def fixture_up(
    num_agents: Optional[int],
    port: Optional[int],
    etc_path: Path,
    cluster_name: str,
    db_password: str,
    hasura_secret: str,
) -> str:
    env = {
        "INTEGRATIONS_HOST_PORT": str(port),
        "DET_ETC_ROOT": str(etc_path),
        "DET_DB_PASSWORD": db_password,
        "DET_HASURA_SECRET": hasura_secret,
    }
    config.MASTER_PORT = port
    fixture_down(cluster_name)
    docker_compose(["up", "-d", "--scale", f"determined-agent={num_agents}"], cluster_name, env)
    _wait_for_master(port)


def fixture_down(cluster_name: str) -> None:
    docker_compose(["down", "--volumes", "-t", "1"], cluster_name)


def logs(cluster_name: str) -> None:
    docker_compose(["logs", "-f"], cluster_name)
