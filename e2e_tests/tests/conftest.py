import json
import pathlib
import subprocess
import time
from typing import Any, Callable, Dict, Iterator, Optional, cast

import _pytest.config.argparsing
import _pytest.fixtures
import boto3
import pytest
from botocore import exceptions as boto_exc

from tests import api_utils, cluster_log_manager
from tests import config as conf
from tests import detproc
from tests.experiment import record_profiling
from tests.nightly import compute_stats

_INTEG_MARKERS = {
    "tensorflow1_cpu",
    "tensorflow2_cpu",
    "tensorflow2",
    "e2e_cpu",
    "e2e_cpu_rbac",
    "e2e_cpu_2a",
    "e2e_cpu_agent_connection_loss",
    "e2e_cpu_elastic",
    "e2e_cpu_rbac",
    "e2e_gpu",
    "e2e_k8s",
    "e2e_pbs",
    "e2e_slurm",
    "e2e_slurm_restart",
    "e2e_slurm_preemption",
    "e2e_slurm_internet_connected_cluster",
    "e2e_slurm_misconfigured",
    "det_deploy_local",
    "stress_test",
    "test_oauth",
    "test_model_registry_rbac",
    "distributed",
    "parallel",
    "nightly",
    "model_hub_transformers",
    "model_hub_transformers_amp",
    "model_hub_mmdetection",
    "deepspeed",
    "managed_devcluster",
    "port_registry",
    "distributed_quarantine",
    "det_deploy_local_quarantine",
}


def pytest_addoption(parser: _pytest.config.argparsing.Parser) -> None:
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
        pathlib.Path(__file__)
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


def pytest_configure(config: _pytest.config.Config) -> None:
    """
    pytest_configure is a pytest hook which runs before all fixtures and test decorators.

    It is important we use this hook to capture information related to accessing the master, so that
    our various skipif decorators can access the master.
    """

    conf.MASTER_SCHEME = config.getoption("--master-scheme")
    conf.MASTER_IP = config.getoption("--master-host")
    conf.MASTER_PORT = config.getoption("--master-port")
    conf.DET_VERSION = config.getoption("--det-version")


@pytest.fixture(scope="session", autouse=True)
def cluster_log_manager_fixture(
    request: _pytest.fixtures.SubRequest,
) -> Iterator[Optional[cluster_log_manager.ClusterLogManager]]:
    follow_local_logs = request.config.getoption("--follow-local-logs")
    compare_stats_enabled = not request.config.getoption("--no-compare-stats")

    if conf.MASTER_IP == "localhost" and follow_local_logs:
        project_name = request.config.getoption("--compose-project-name")
        with cluster_log_manager.ClusterLogManager(
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
        sess = api_utils.admin_session()
        compute_stats.compare_stats(sess)


def pytest_itemcollected(item: Any) -> None:
    if _INTEG_MARKERS.isdisjoint({marker.name for marker in item.iter_markers()}):
        pytest.exit(f"{item.nodeid} is missing an integration test mark (any of {_INTEG_MARKERS})")


def s3_secrets(request: _pytest.fixtures.SubRequest) -> Dict[str, str]:
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
def secrets(request: _pytest.fixtures.SubRequest) -> Dict[str, str]:
    response = {}

    try:
        response = s3_secrets(request)
    except boto_exc.NoCredentialsError:
        if request.config.getoption("--require-secrets"):
            raise
        pytest.skip("No S3 access")

    return response


@pytest.fixture(scope="session")
def checkpoint_storage_config(request: _pytest.fixtures.SubRequest) -> Dict[str, Any]:
    command = ["det", "master", "config", "--json"]

    sess = api_utils.admin_session()
    output = detproc.check_json(sess, command)

    checkpoint_config = output["checkpoint_storage"]

    if checkpoint_config["type"] == "s3":
        secret_conf = s3_secrets(request)
        checkpoint_config["bucket"] = secret_conf["INTEGRATIONS_S3_BUCKET"]
        checkpoint_config["access_key"] = secret_conf["INTEGRATIONS_S3_ACCESS_KEY"]
        checkpoint_config["secret_key"] = secret_conf["INTEGRATIONS_S3_SECRET_KEY"]

    return cast(Dict[str, Any], checkpoint_config)


@pytest.fixture(autouse=True)
def test_start_timer(request: _pytest.fixtures.SubRequest) -> Iterator[None]:
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

    Note: this must be a fixture in order to use the record_property fixture provided by pytest.
    """

    return record_profiling.profile_test(record_property=record_property)
