import abc
from typing import Any, Dict, Iterable, List, Optional

import pkg_resources
from termcolor import colored

import determined
import determined.deploy
from determined.deploy.aws import constants


class DeterminedDeployment(metaclass=abc.ABCMeta):
    template_parameter_keys = []  # type: List[str]
    template = None  # type: Optional[str]

    master_info = "Configure the Determined CLI: " + colored(
        "export DET_MASTER={master_ip}", "yellow"
    )
    ui_info = "View the Determined UI: " + colored("http://{master_ip}:8080", "blue")
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
    def deploy(self) -> None:
        pass

    def print(self) -> None:
        with open(self.template_path) as f:
            print(f.read())

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
