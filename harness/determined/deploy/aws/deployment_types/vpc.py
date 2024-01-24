from determined.common.api import certs
from determined.deploy import healthcheck
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
    deployment_type = constants.deployment_types.GENAI

    def before_deploy_print(self) -> None:
        super().before_deploy_print()
        genai_tag = self.parameters[constants.cloudformation.GENAI_VERSION] or "latest"
        # Lore is renamed to GenAI and we are changing the user visible text to GenAI
        # Interal references will be updated in a later stage if prioritized
        print(f"GenAI Version: {genai_tag}")
        print(f"GenAI Image: determinedai/genai:{genai_tag}")

    def wait_for_genai(self, timeout: int = 60) -> None:
        self.wait_for_master()
        cert = None
        if self.parameters[constants.cloudformation.MASTER_TLS_CERT]:
            cert = certs.Cert(noverify=True)
        master_url = self._get_master_url()
        return healthcheck.wait_for_genai_url(master_url, timeout=timeout, cert=cert)
