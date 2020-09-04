import abc
from typing import Any, Dict, List, Optional

import pkg_resources

import determined_deploy
from determined_deploy.aws import constants


class DeterminedDeployment(metaclass=abc.ABCMeta):
    template_parameter_keys = []  # type: List[str]
    template = None  # type: Optional[str]

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
            else determined_deploy.__version__
        )
        keypair = self.parameters[constants.cloudformation.KEYPAIR]

        print(f"Determined Version: {version}")
        print(f"Stack Name: {cluster_id}")
        print(f"AWS Region: {aws_region}")
        print(f"Keypair: {keypair}")
