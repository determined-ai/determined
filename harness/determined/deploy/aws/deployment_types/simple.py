from determined.deploy.aws import aws, constants
from determined.deploy.aws.deployment_types import base


class Simple(base.DeterminedDeployment):
    template = "simple.yaml"
    deployment_type = constants.deployment_types.SIMPLE

    template_parameter_keys = base.COMMON_TEMPLATE_PARAMETER_KEYS + [
        constants.cloudformation.PREEMPTION_ENABLED,
        constants.cloudformation.RETAIN_LOG_GROUP,
        constants.cloudformation.SCHEDULER_TYPE,
        constants.cloudformation.SUBNET_ID_KEY,
    ]

    def deploy(self, no_prompt: bool, update_terminate_agents: bool) -> None:
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
            update_terminate_agents=update_terminate_agents,
        )
        self.print_results()
