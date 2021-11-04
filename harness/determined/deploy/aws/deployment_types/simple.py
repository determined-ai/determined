from determined.deploy.aws import aws, constants
from determined.deploy.aws.deployment_types import base


class Simple(base.DeterminedDeployment):
    template = "simple.yaml"
    deployment_type = constants.deployment_types.SIMPLE

    template_parameter_keys = [
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
        constants.cloudformation.SUBNET_ID_KEY,
        constants.cloudformation.SCHEDULER_TYPE,
        constants.cloudformation.PREEMPTION_ENABLED,
        constants.cloudformation.CPU_ENV_IMAGE,
        constants.cloudformation.GPU_ENV_IMAGE,
        constants.cloudformation.LOG_GROUP_PREFIX,
        constants.cloudformation.RETAIN_LOG_GROUP,
        constants.cloudformation.IMAGE_REPO_PREFIX,
        constants.cloudformation.MASTER_CONFIG_TEMPLATE,
    ]

    def deploy(self, no_prompt: bool) -> None:
        cfn_parameters = self.consolidate_parameters()
        self.before_deploy_print()
        with open(self.template_path) as f:
            template = f.read()

        aws.deploy_stack(
            stack_name=self.parameters[constants.cloudformation.CLUSTER_ID],
            template_body=template,
            keypair=self.parameters[constants.cloudformation.KEYPAIR],
            boto3_session=self.parameters[constants.cloudformation.BOTO3_SESSION],
            parameters=cfn_parameters,
            no_prompt=no_prompt,
            deployment_type=self.deployment_type,
        )
        self.print_results()
