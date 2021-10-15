from typing import Iterable

from termcolor import colored

from determined.deploy.aws import aws, constants
from determined.deploy.aws.deployment_types import base


class Secure(base.DeterminedDeployment):
    bastion_info = (
        "To View Determined UI:\n"
        "Add Keypair: " + colored("ssh-add <keypair>", "yellow") + "\n"
        "Open SSH Tunnel through Bastion: "
        + colored("ssh -N -L 8080:{master_ip}:8080 ubuntu@{bastion_ip}", "yellow")
    )

    master_info = "Configure the Determined CLI: " + colored(
        "export DET_MASTER=localhost:8080", "yellow"
    )
    ui_info = "View the Determined UI: " + colored("http://localhost:8080", "blue")
    ssh_info = "SSH to Determined Master: " + colored(
        "ssh -i  <keypair> ubuntu@{master_ip} -o "
        '"proxycommand ssh -W %h:%p -i <keypair> ubuntu@{bastion_ip}"',
        "yellow",
    )
    template = "secure.yaml"

    template_parameter_keys = [
        constants.cloudformation.ENABLE_CORS,
        constants.cloudformation.MASTER_TLS_CERT,
        constants.cloudformation.MASTER_TLS_KEY,
        constants.cloudformation.MASTER_CERT_NAME,
        constants.cloudformation.KEYPAIR,
        constants.cloudformation.BASTION_ID,
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
    ]

    def deploy(self) -> None:
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
        )
        self.print_results()

    def print_results(self) -> None:
        stack_name = self.parameters[constants.cloudformation.CLUSTER_ID]
        boto3_session = self.parameters[constants.cloudformation.BOTO3_SESSION]

        output = aws.get_output(stack_name, boto3_session)

        bastion_ip = aws.get_ec2_info(output["BastionId"], boto3_session)[
            constants.cloudformation.PUBLIC_IP_ADDRESS
        ]
        master_ip = aws.get_ec2_info(output["MasterId"], boto3_session)[
            constants.cloudformation.PRIVATE_IP_ADDRESS
        ]
        region = output[constants.cloudformation.REGION]
        log_group = output[constants.cloudformation.LOG_GROUP]

        self.print_output_info(
            master_ip=master_ip, bastion_ip=bastion_ip, region=region, log_group=log_group
        )

    @property
    def info_partials(self) -> Iterable[str]:
        return (
            self.bastion_info,
            self.master_info,
            self.ui_info,
            self.logs_info,
            self.ssh_info,
        )

    def wait_for_master(self, timeout: int = 0) -> None:
        print("Skipping automated master health check due to bastion host usage.")
        return
