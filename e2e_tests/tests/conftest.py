import json
import os
import pathlib
import subprocess
import time
import uuid
from typing import Any, Dict, Iterator, List, Optional, Tuple, cast

import _pytest.config.argparsing
import _pytest.fixtures
import boto3
import pytest
from botocore import exceptions as boto_exc

from tests import api_utils, cluster_log_manager
from tests import config as conf
from tests import detproc
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
    "e2e_multi_k8s",
    "e2e_single_k8s",
    "e2e_k8s",
    "e2e_pbs",
    "e2e_saml",
    "e2e_slurm",
    "e2e_slurm_restart",
    "e2e_slurm_internet_connected_cluster",
    "det_deploy_local",
    "test_oauth",
    "test_model_registry_rbac",
    "distributed",
    "parallel",
    "nightly",
    "deepspeed",
    "managed_devcluster",
    "managed_devcluster_resource_pools",
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
    parser.addoption(
        "--user-password",
        action="store",
        default=os.environ.get("INITIAL_USER_PASSWORD", ""),
        help="Password for the admin and determined users",
    )


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
    conf.USER_PASSWORD = config.getoption("--user-password")


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


@pytest.fixture(scope="session")
def is_multirm_cluster() -> bool:
    sess = api_utils.admin_session()
    multi_rm = True
    try:
        multi_rm = detproc.check_json(sess, ["det", "master", "config", "show", "--json"])[
            "additional_resource_managers"
        ]
    except KeyError:
        multi_rm = False
    return multi_rm


@pytest.fixture(scope="session")
def namespaces_created(is_multirm_cluster: bool) -> Iterator[Tuple[str, str]]:
    defaultrm_namespace = uuid.uuid4().hex[:8]
    additionalrm_namespace = uuid.uuid4().hex[:8]

    # Create a namespace in Kubernetes in the each resource manager's Kubernetes cluster.
    create_namespace_defaultrm_cmd = ["kubectl", "create", "namespace", defaultrm_namespace]

    if is_multirm_cluster:
        create_namespace_defaultrm_cmd += ["--kubeconfig", conf.DEFAULT_RM_KUBECONFIG]
        create_namespace_additionalrm_cmd = [
            "kubectl",
            "create",
            "namespace",
            additionalrm_namespace,
            "--kubeconfig",
            conf.ADDITIONAL_RM_KUBECONFIG,
        ]
        subprocess.run(create_namespace_additionalrm_cmd, check=True)

    subprocess.run(create_namespace_defaultrm_cmd, check=True)

    default_kubeconfig = []
    additionalrm_kubeconfig = ["--kubeconfig", conf.ADDITIONAL_RM_KUBECONFIG]
    if is_multirm_cluster:
        get_namespace(additionalrm_namespace, additionalrm_kubeconfig)
        default_kubeconfig += ["--kubeconfig", conf.DEFAULT_RM_KUBECONFIG]

    get_namespace(defaultrm_namespace, default_kubeconfig)

    # Make sure that we can successfully retrieve both namespaces.
    namespaces = [defaultrm_namespace]
    if is_multirm_cluster:
        namespaces.append(additionalrm_namespace)

    yield defaultrm_namespace, additionalrm_namespace

    delete_namespace(defaultrm_namespace, kubeconfig=default_kubeconfig)
    if is_multirm_cluster:
        delete_namespace(additionalrm_namespace, kubeconfig=additionalrm_kubeconfig)


def get_namespace(namespace: str, kubeconfig: List[str]) -> None:
    for _ in range(150):
        try:
            p = subprocess.run(["kubectl", "get", "namespace", namespace] + kubeconfig, check=True)
            if not p.returncode:
                break
        except subprocess.CalledProcessError:
            pass
        time.sleep(2)


def delete_namespace(namespace: str, kubeconfig: List[str]) -> None:
    for _ in range(150):
        try:
            p = subprocess.run(
                ["kubectl", "delete", "namespace", namespace] + kubeconfig, check=True
            )
            if not p.returncode:
                break
        except subprocess.CalledProcessError:
            pass
        time.sleep(2)
