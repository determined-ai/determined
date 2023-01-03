import abc
from typing import Any, Dict, Iterable, List, Optional

import pkg_resources
from termcolor import colored

import determined
import determined.deploy
from determined.common.api import certs
from determined.deploy import healthcheck
from determined.deploy.aws import aws, constants

COMMON_TEMPLATE_PARAMETER_KEYS = [
    constants.cloudformation.ENABLE_CORS,
    constants.cloudformation.MASTER_TLS_CERT,
    constants.cloudformation.MASTER_TLS_KEY,
    constants.cloudformation.MASTER_CERT_NAME,
    constants.cloudformation.KEYPAIR,
    constants.cloudformation.MASTER_INSTANCE_TYPE,
    constants.cloudformation.AUX_AGENT_INSTANCE_TYPE,
    constants.cloudformation.COMPUTE_AGENT_INSTANCE_TYPE,
    constants.cloudformation.INBOUND_CIDR,
    constants.cloudformation.VERSION,
    constants.cloudformation.DB_PASSWORD,
    constants.cloudformation.MAX_IDLE_AGENT_PERIOD,
    constants.cloudformation.MAX_AGENT_STARTING_PERIOD,
    constants.cloudformation.MAX_AUX_CONTAINERS_PER_AGENT,
    constants.cloudformation.MAX_DYNAMIC_AGENTS,
    constants.cloudformation.SPOT_ENABLED,
    constants.cloudformation.SPOT_MAX_PRICE,
    constants.cloudformation.CPU_ENV_IMAGE,
    constants.cloudformation.GPU_ENV_IMAGE,
    constants.cloudformation.LOG_GROUP_PREFIX,
    constants.cloudformation.IMAGE_REPO_PREFIX,
    constants.cloudformation.MASTER_CONFIG_TEMPLATE,
    constants.cloudformation.AGENT_REATTACH_ENABLED,
    constants.cloudformation.AGENT_RECONNECT_ATTEMPTS,
    constants.cloudformation.AGENT_RECONNECT_BACKOFF,
    constants.cloudformation.AGENT_CONFIG_FILE_CONTENTS,
    constants.cloudformation.MASTER_IMAGE_NAME,
    constants.cloudformation.AGENT_IMAGE_NAME,
    constants.cloudformation.DOCKER_USER,
    constants.cloudformation.DOCKER_PASS,
    constants.cloudformation.NOTEBOOK_TIMEOUT,
]  # type: List[str]


class DeterminedDeployment(metaclass=abc.ABCMeta):
    template_parameter_keys = []  # type: List[str]
    template = None  # type: Optional[str]

    master_info = "Configure the Determined CLI: " + colored(
        "export DET_MASTER={master_url}", "yellow"
    )
    ui_info = "View the Determined UI: " + colored("{master_url}", "blue")
    logs_info = "View Logs at: " + colored(
        "https://{region}.console.aws.amazon.com/cloudwatch/home?"
        "region={region}#logStream:group={log_group}",
        "blue",
    )
    ssh_info = "SSH to master Instance: " + colored(
        "ssh -i <pem-file> ubuntu@{master_ip}", "yellow"
    )

    def __init__(self, parameters: Dict[str, Any]) -> None:
        assert self.template is not None
        self.template_path = pkg_resources.resource_filename(
            constants.misc.TEMPLATE_PATH, self.template
        )
        self.parameters = parameters

    @abc.abstractmethod
    def deploy(self, no_prompt: bool, update_terminate_agents: bool) -> None:
        pass

    def print(self) -> None:
        with open(self.template_path) as f:
            print(f.read())

    def wait_for_master(self, timeout: int = 5 * 60) -> None:
        cert = None
        if self.parameters[constants.cloudformation.MASTER_TLS_CERT]:
            cert = certs.Cert(noverify=True)
        master_url = self._get_master_url()
        return healthcheck.wait_for_master_url(master_url, timeout=timeout, cert=cert)

    def consolidate_parameters(self) -> List[Dict[str, Any]]:
        return [
            {"ParameterKey": k, "ParameterValue": str(self.parameters[k])}
            for k in self.parameters.keys()
            if self.parameters[k] and k in self.template_parameter_keys
        ]

    def before_deploy_print(self) -> None:
        cluster_id = self.parameters[constants.cloudformation.CLUSTER_ID]
        aws_region = self.parameters[constants.cloudformation.BOTO3_SESSION].region_name
        version = (
            self.parameters[constants.cloudformation.VERSION]
            if self.parameters[constants.cloudformation.VERSION]
            else determined.__version__
        )
        keypair = self.parameters[constants.cloudformation.KEYPAIR]

        print(f"Determined Version: {version}")
        print(f"Stack Name: {cluster_id}")
        print(f"AWS Region: {aws_region}")
        print(f"Keypair: {keypair}")

    @property
    def info_partials(self) -> Iterable[str]:
        return (
            self.master_info,
            self.ui_info,
            self.logs_info,
            self.ssh_info,
        )

    def print_output_info(self, **kwargs: str) -> None:
        print("\n".join(self.info_partials).format(**kwargs))

    def _get_aws_output(self) -> Dict[str, str]:
        stack_name = self.parameters[constants.cloudformation.CLUSTER_ID]
        boto3_session = self.parameters[constants.cloudformation.BOTO3_SESSION]
        return aws.get_output(stack_name, boto3_session)

    def print_results(self) -> None:
        output = self._get_aws_output()
        master_ip = output[constants.cloudformation.DET_ADDRESS]
        region = output[constants.cloudformation.REGION]
        log_group = output[constants.cloudformation.LOG_GROUP]
        master_url = self._get_master_url()

        self.print_output_info(
            master_ip=master_ip, master_url=master_url, region=region, log_group=log_group
        )

    def _get_master_url(self) -> str:
        output = self._get_aws_output()

        master_ip = output[constants.cloudformation.DET_ADDRESS]
        master_port = output[constants.cloudformation.MASTER_PORT]
        master_scheme = output[constants.cloudformation.MASTER_SCHEME]

        master_url = f"{master_scheme}://{master_ip}:{master_port}"

        return master_url
