import json
import subprocess
from pathlib import Path
from typing import Any, Dict, Optional, cast

import boto3
import pytest
from _pytest.config.argparsing import Parser
from _pytest.fixtures import SubRequest
from botocore import exceptions as boto_exc

from tests import config

from .cluster_log_manager import ClusterLogManager

_INTEG_MARKERS = {
    "tensorflow1_cpu",
    "tensorflow2_cpu",
    "e2e_cpu",
    "e2e_gpu",
    "distributed",
    "cloud",
    "performance",
    "parallel",
    "nightly",
}


def pytest_addoption(parser: Parser) -> None:
    parser.addoption(
        "--master-config-path", action="store", default=None, help="Path to master config path"
    )
    parser.addoption(
        "--master-host",
        action="store",
        default="localhost",
        help="Master host for integration tests",
    )
    parser.addoption(
        "--master-port", action="store", default="8080", help="Master port for integration tests"
    )
    parser.addoption(
        "--require-secrets", action="store_true", help="fail tests when s3 access fails"
    )
    path = (
        Path(__file__)
        .parents[2]
        .joinpath("deploy", "determined_deploy", "local", "docker-compose.yaml")
    )
    parser.addoption(
        "--compose-file", action="store", default=str(path), help="Docker compose file"
    )
    parser.addoption(
        "--compose-project-name",
        action="store",
        default="determined",
        help="Docker compose project name",
    )
    parser.addoption("--follow-local-logs", action="store_true", help="Follow local docker logs")


@pytest.fixture(scope="session", autouse=True)  # type: ignore
def cluster_log_manager(request: SubRequest) -> Optional[ClusterLogManager]:
    master_config_path = request.config.getoption("--master-config-path")
    master_config_path = Path(master_config_path) if master_config_path else None
    master_host = request.config.getoption("--master-host")
    master_port = request.config.getoption("--master-port")
    follow_local_logs = request.config.getoption("--follow-local-logs")
    compose_file = request.config.getoption("--compose-file")

    config.MASTER_IP = master_host
    config.MASTER_PORT = master_port

    if master_host == "localhost" and follow_local_logs:
        project_name = request.config.getoption("--compose-project-name")
        project = ["-p", project_name] if project_name else []
        with ClusterLogManager(
            lambda: subprocess.run(["docker-compose", "-f", compose_file, *project, "logs", "-f"])
        ) as clm:
            # Yield instead of return so that `__exit__` is called when the
            # testing session is finished.
            yield clm
    else:
        # Yield `None` so that pytest handles the no log manager case correctly.
        yield None


def pytest_itemcollected(item: Any) -> None:
    if _INTEG_MARKERS.isdisjoint({marker.name for marker in item.iter_markers()}):
        pytest.exit(f"{item.nodeid} is missing an integration test mark (any of {_INTEG_MARKERS})")


@pytest.fixture(scope="session")  # type: ignore
def secrets(request: SubRequest) -> Dict[str, str]:
    """
    Connect to S3 secretsmanager to get the secret values used in integrations tests.
    """

    secret_name = "integrations-s3"
    region_name = "us-west-2"

    # Create a Secrets Manager client
    session = boto3.session.Session()
    client = session.client(service_name="secretsmanager", region_name=region_name)

    try:
        response = client.get_secret_value(SecretId=secret_name)
    except boto_exc.NoCredentialsError:
        if request.config.getoption("--require-secrets"):
            raise
        pytest.skip("No S3 access")

    return cast(Dict[str, str], json.loads(response["SecretString"]))
