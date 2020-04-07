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

    def __init__(self, parameters: List) -> None:
        template_path = pkg_resources.resource_filename(constants.misc.TEMPLATE_PATH, self.template)
        super().__init__(template_path, parameters)

    def deploy(self) -> None:
        cfn_parameters = [
            {
                "ParameterKey": constants.cloudformation.USER_NAME,
                "ParameterValue": self.parameters[constants.cloudformation.USER_NAME],
            },
            {
                "ParameterKey": constants.cloudformation.KEYPAIR,
                "ParameterValue": self.parameters[constants.cloudformation.KEYPAIR],
            },
            {
                "ParameterKey": constants.cloudformation.BASTION_AMI,
                "ParameterValue": constants.defaults.BASTION_AMI,
            },
            {
                "ParameterKey": constants.cloudformation.MASTER_AMI,
                "ParameterValue": self.parameters[constants.cloudformation.MASTER_AMI],
            },
            {
                "ParameterKey": constants.cloudformation.MASTER_INSTANCE_TYPE,
                "ParameterValue": self.parameters[constants.cloudformation.MASTER_INSTANCE_TYPE],
            },
            {
                "ParameterKey": constants.cloudformation.AGENT_AMI,
                "ParameterValue": self.parameters[constants.cloudformation.AGENT_AMI],
            },
            {
                "ParameterKey": constants.cloudformation.AGENT_INSTANCE_TYPE,
                "ParameterValue": self.parameters[constants.cloudformation.AGENT_INSTANCE_TYPE],
            },
            {
                "ParameterKey": constants.cloudformation.INBOUND_CIDR,
                "ParameterValue": self.parameters[constants.cloudformation.INBOUND_CIDR],
            },
            {
                "ParameterKey": constants.cloudformation.VERSION,
                "ParameterValue": self.parameters[constants.cloudformation.VERSION],
            },
            {
                "ParameterKey": constants.cloudformation.DB_PASSWORD,
                "ParameterValue": self.parameters[constants.cloudformation.DB_PASSWORD],
            },
            {
                "ParameterKey": constants.cloudformation.HASURA_SECRET,
                "ParameterValue": self.parameters[constants.cloudformation.HASURA_SECRET],
            },
            {
                "ParameterKey": constants.cloudformation.MAX_IDLE_AGENT_PERIOD,
                "ParameterValue": self.parameters[constants.cloudformation.MAX_IDLE_AGENT_PERIOD],
            },
            {
                "ParameterKey": constants.cloudformation.MAX_INSTANCES,
                "ParameterValue": str(self.parameters[constants.cloudformation.MAX_INSTANCES]),
            },
        ]

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
