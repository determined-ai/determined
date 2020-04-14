from typing import List

import boto3
import pkg_resources

from determined_deploy.aws import aws, constants
from determined_deploy.aws.deployment_types import base


class Secure(base.DeterminedDeployment):
    ssh_command = (
        "SSH to Determined Master: ssh -i  <keypair> ubuntu@{master_ip} -o "
        '"proxycommand ssh -W %h:%p -i <keypair> ubuntu@{bastion_ip}"'
    )
    det_ui = (
        "To View Determined UI:\n"
        "Add Keypair: ssh-add <keypair>\n"
        "Open SSH Tunnel through Bastion:  ssh -N -L 8080:{master_ip}:8080 ubuntu@{bastion_ip}\n"
        "Access Determined through cli: det -m {master_ip}\n"
        "View the Determined UI: http://localhost:8080"
    )
    template = "secure.yaml"

    template_parameter_keys = [
        constants.cloudformation.USER_NAME,
        constants.cloudformation.KEYPAIR,
        constants.cloudformation.BASTION_ID,
        constants.cloudformation.MASTER_AMI,
        constants.cloudformation.MASTER_INSTANCE_TYPE,
        constants.cloudformation.AGENT_AMI,
        constants.cloudformation.AGENT_INSTANCE_TYPE,
        constants.cloudformation.INBOUND_CIDR,
        constants.cloudformation.VERSION,
        constants.cloudformation.DB_PASSWORD,
        constants.cloudformation.HASURA_SECRET,
        constants.cloudformation.MAX_IDLE_AGENT_PERIOD,
        constants.cloudformation.MAX_INSTANCES,
    ]

    def __init__(self, parameters: List) -> None:
        template_path = pkg_resources.resource_filename(constants.misc.TEMPLATE_PATH, self.template)
        super().__init__(template_path, parameters)

    def deploy(self) -> None:
        cfn_parameters = self.consolidate_parameters()
        with open(self.template_path) as f:
            template = f.read()

        aws.deploy_stack(
            self.parameters[constants.cloudformation.DET_STACK_NAME],
            template,
            self.parameters[constants.cloudformation.BOTO3_SESSION],
            parameters=cfn_parameters,
        )
        self.print_results(
            self.parameters[constants.cloudformation.DET_STACK_NAME],
            self.parameters[constants.cloudformation.BOTO3_SESSION],
        )

    def print_results(self, stack_name: str, boto3_session: boto3.session.Session) -> None:
        output = aws.get_output(stack_name, boto3_session)

        bastion_ip = aws.get_ec2_info(output["BastionId"], boto3_session)[
            constants.cloudformation.PUBLIC_IP_ADDRESS
        ]
        master_ip = aws.get_ec2_info(output["MasterId"], boto3_session)[
            constants.cloudformation.PRIVATE_IP_ADDRESS
        ]

        ui_command = self.det_ui.format(master_ip=master_ip, bastion_ip=bastion_ip)
        print(ui_command)
        print()

        ssh_command = self.ssh_command.format(master_ip=master_ip, bastion_ip=bastion_ip)
        print(ssh_command)
        print()
