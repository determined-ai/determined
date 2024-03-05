from typing import Iterable

import termcolor

from determined.deploy.aws import aws, constants
from determined.deploy.aws.deployment_types import base


class Secure(base.DeterminedDeployment):
    bastion_info = (
        "To View Determined UI:\n"
        "Add Keypair: " + termcolor.colored("ssh-add <keypair>", "yellow") + "\n"
        "Open SSH Tunnel through Bastion: "
        + termcolor.colored("ssh -N -L 8080:{master_ip}:8080 ubuntu@{bastion_ip}", "yellow")
    )

    master_info = "Configure the Determined CLI: " + termcolor.colored(
        "export DET_MASTER=localhost:8080", "yellow"
    )
    ui_info = "View the Determined UI: " + termcolor.colored("http://localhost:8080", "blue")
    ssh_info = "SSH to Determined Master: " + termcolor.colored(
        "ssh -i  <keypair> ubuntu@{master_ip} -o "
        '"proxycommand ssh -W %h:%p -i <keypair> ubuntu@{bastion_ip}"',
        "yellow",
    )
    template = "secure.yaml"
    deployment_type = constants.deployment_types.SECURE

    template_parameter_keys = base.COMMON_TEMPLATE_PARAMETER_KEYS + [
        constants.cloudformation.BASTION_ID,
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
