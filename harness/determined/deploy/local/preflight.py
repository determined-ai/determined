import shutil
import sys

import docker
from termcolor import colored


def _print_see_more() -> None:
    print(
        "For more details, please see Determined installation docs: "
        "https://docs.determined.ai/latest/how-to/installation/requirements.html#install-docker"
    )


def check_docker_install() -> None:
    # Do we have `docker` executable available?
    if shutil.which("docker") is None:
        print(
            colored(
                "Docker is required for local Determined cluster. "
                "Please ensure it is properly installed.",
                "red",
            )
        )
        _print_see_more()
        sys.exit(1)

    # Can we talk to the Docker daemon?
    try:
        docker.from_env()
    except docker.errors.DockerException as ex:
        print(colored("Failed to connect to Docker daemon: %s" % ex, "red"))
        print(
            colored(
                "Please ensure that the Docker daemon is running "
                "and that the current user has access permissions.",
                "red",
            )
        )
        _print_see_more()
        sys.exit(1)
