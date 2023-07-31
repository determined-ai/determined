import json
import subprocess
import time
from pathlib import Path
from typing import Any, Callable, Dict, Iterator, Optional, cast

import boto3
import pytest
from _pytest.config.argparsing import Parser
from _pytest.fixtures import SubRequest
from botocore import exceptions as boto_exc

from determined.experimental import client as _client
from tests import config
from tests.experiment import profile_test
from tests.nightly.compute_stats import compare_stats

from .cluster.test_users import ADMIN_CREDENTIALS, logged_in_user
from .cluster_log_manager import ClusterLogManager

_INTEG_MARKERS = {
    "tensorflow1_cpu",
    "tensorflow2_cpu",
    "tensorflow2",
    "e2e_cpu",
    "e2e_cpu_2a",
    "e2e_cpu_agent_connection_loss",
    "e2e_cpu_elastic",
    "e2e_cpu_rbac",
    "e2e_gpu",
    "e2e_k8s",
    "det_deploy_local",
    "stress_test",
    "distributed",
    "parallel",
    "nightly",
    "model_hub_transformers",
    "model_hub_transformers_amp",
    "model_hub_mmdetection",
    "deepspeed",
    "managed_devcluster",
    "port_registry",
    "model_hub_mmdetection_quarantine",
    "nightly_quarantine",
    "det_deploy_local_quarantine",
}


def pytest_addoption(parser: Parser) -> None:
    parser.addoption(
        "--master-config-path", action="store", default=None, help="Path to master config path"
    )
    parser.addoption(
        "--master-scheme",
        action="store",
        default="http",
        help="Master scheme for integration tests",
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
        "--det-version",
        action="store",
        default=None,
        help="Determined version for det deploy tests",
    )
    parser.addoption(
        "--require-secrets", action="store_true", help="fail tests when s3 access fails"
    )
    path = (
        Path(__file__)
        .parents[2]
        .joinpath("deploy", "determined.deploy", "local", "docker-compose.yaml")
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
    parser.addoption("--no-compare-stats", action="store_true", help="Disable usage stats check")


@pytest.fixture(scope="session", autouse=True)
def instantiate_gpu() -> None:
    command = ["det", "cmd", "--config", "resources.slots=1", "'sleep 30'"]

    subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


@pytest.fixture(scope="session", autouse=True)
def cluster_log_manager(request: SubRequest) -> Iterator[Optional[ClusterLogManager]]:
    master_scheme = request.config.getoption("--master-scheme")
    master_host = request.config.getoption("--master-host")
    master_port = request.config.getoption("--master-port")
    det_version = request.config.getoption("--det-version")
    follow_local_logs = request.config.getoption("--follow-local-logs")
    compare_stats_enabled = not request.config.getoption("--no-compare-stats")

    config.MASTER_SCHEME = master_scheme
    config.MASTER_IP = master_host
    config.MASTER_PORT = master_port
    config.DET_VERSION = det_version

    if master_host == "localhost" and follow_local_logs:
        project_name = request.config.getoption("--compose-project-name")
        with ClusterLogManager(
            lambda: subprocess.run(
                ["det", "deploy", "local", "logs", "--cluster-name", project_name]
            )
        ) as clm:
            # Yield instead of return so that `__exit__` is called when the
            # testing session is finished.
            yield clm
    else:
        # Yield `None` so that pytest handles the no log manager case correctly.
        yield None

    if compare_stats_enabled:
        compare_stats()


def pytest_itemcollected(item: Any) -> None:
    if _INTEG_MARKERS.isdisjoint({marker.name for marker in item.iter_markers()}):
        pytest.exit(f"{item.nodeid} is missing an integration test mark (any of {_INTEG_MARKERS})")


def s3_secrets(request: SubRequest) -> Dict[str, str]:
    """
    Connect to S3 secretsmanager to get the secret values used in integrations tests.
    """
    secret_name = "integrations-s3"
    region_name = "us-west-2"

    # Create a Secrets Manager client
    session = boto3.session.Session()
    client = session.client(service_name="secretsmanager", region_name=region_name)
    response = client.get_secret_value(SecretId=secret_name)

    return cast(Dict[str, str], json.loads(response["SecretString"]))


@pytest.fixture(scope="session")
def secrets(request: SubRequest) -> Dict[str, str]:
    response = {}

    try:
        response = s3_secrets(request)
    except boto_exc.NoCredentialsError:
        if request.config.getoption("--require-secrets"):
            raise
        pytest.skip("No S3 access")

    return response


@pytest.fixture(scope="session")
def checkpoint_storage_config(request: SubRequest) -> Dict[str, Any]:
    command = [
        "det",
        "-m",
        config.make_master_url(),
        "master",
        "config",
        "--json",
    ]

    with logged_in_user(ADMIN_CREDENTIALS):
        output = subprocess.check_output(command, universal_newlines=True, stderr=subprocess.PIPE)

    checkpoint_config = json.loads(output)["checkpoint_storage"]

    if checkpoint_config["type"] == "s3":
        secret_conf = s3_secrets(request)
        checkpoint_config["bucket"] = secret_conf["INTEGRATIONS_S3_BUCKET"]
        checkpoint_config["access_key"] = secret_conf["INTEGRATIONS_S3_ACCESS_KEY"]
        checkpoint_config["secret_key"] = secret_conf["INTEGRATIONS_S3_SECRET_KEY"]

    return cast(Dict[str, Any], checkpoint_config)


@pytest.fixture(scope="session")
def using_k8s(request: SubRequest) -> bool:
    command = [
        "det",
        "-m",
        config.make_master_url(),
        "master",
        "config",
        "--json",
    ]

    with logged_in_user(ADMIN_CREDENTIALS):
        output = subprocess.check_output(command, universal_newlines=True, stderr=subprocess.PIPE)

    rp = json.loads(output)["resource_manager"]["type"]
    return bool(rp == "kubernetes")


@pytest.fixture(autouse=True)
def test_start_timer(request: SubRequest) -> Iterator[None]:
    # If pytest is run with minimal verbosity, individual test names are not printed and the output
    # of this would look funny.
    if request.config.option.verbose >= 1:
        # This ends up concatenated to the line pytest prints containing the test file and name.
        print("starting at", time.strftime("%Y-%m-%d %H:%M:%S"))
    yield


@pytest.fixture
def collect_trial_profiles(record_property: Callable[[str, object], None]) -> Callable[[int], None]:
    """
    Returns a method that allows profiling of test run for certain system metrics
    and records to JUnit report.

    Currently retrieves metrics by trial (assumes one trial per experiment) using
    profiler API.
    """

    return profile_test(record_property=record_property)


@pytest.fixture(scope="session")
def client() -> _client.Determined:
    """
    Reduce logins by having one session-level fixture do the login.
    """
    return _client.Determined(config.make_master_url())
