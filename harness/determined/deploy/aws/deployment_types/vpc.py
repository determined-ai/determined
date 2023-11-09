from typing import Iterable

import termcolor

from determined.deploy.aws import aws, constants
from determined.deploy.aws.deployment_types import base


class VPCBase(base.DeterminedDeployment):
    deployment_type = None  # type: str

    template_parameter_keys = base.COMMON_TEMPLATE_PARAMETER_KEYS + [
        constants.cloudformation.MOUNT_EFS_ID,
        constants.cloudformation.MOUNT_FSX_ID,
    ]

    def deploy(self, no_prompt: bool, update_terminate_agents: bool) -> None:
        self.before_deploy_print()
        cfn_parameters = self.consolidate_parameters()
        with open(self.template_path) as f:
            template = f.read()

        aws.deploy_stack(
            stack_name=self.parameters[constants.cloudformation.CLUSTER_ID],
            template_body=template,
            keypair=self.parameters[constants.cloudformation.KEYPAIR],
            boto3_session=self.parameters[constants.cloudformation.BOTO3_SESSION],
            parameters=cfn_parameters,
            extra_tags=self.parameters[constants.cloudformation.EXTRA_TAGS],
            no_prompt=no_prompt,
            deployment_type=self.deployment_type,
            update_terminate_agents=update_terminate_agents,
        )
        self.print_results()


class FSx(VPCBase):
    template = "fsx.yaml"
    deployment_type = constants.deployment_types.FSX


class EFS(VPCBase):
    template = "efs.yaml"
    deployment_type = constants.deployment_types.EFS


class Lore(VPCBase):
    template = "lore.yaml"
    deployment_type = constants.deployment_types.LORE
    lore_info = "View the Lore UI: " + termcolor.colored("{master_url}/lore", "blue")

    def before_deploy_print(self) -> None:
        super().before_deploy_print()
        lore_tag = self.parameters[constants.cloudformation.LORE_VERSION] or "latest"
        print(f"Lore Version: {lore_tag}")
        print(f"Lore Image: determinedai/environments-dev:lore-backend-image-{lore_tag}")

    @property
    def info_partials(self) -> Iterable[str]:
        return (
            self.lore_info,
            self.master_info,
            self.ui_info,
            self.logs_info,
            self.ssh_info,
        )
