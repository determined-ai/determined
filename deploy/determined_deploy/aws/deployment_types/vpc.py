import boto3

from determined_deploy.aws import aws, constants
from determined_deploy.aws.deployment_types import base


class VPC(base.DeterminedDeployment):
    ssh_command = "SSH to master Instance: ssh -i <pem-file> ubuntu@{master_ip}"
    det_ui = (
        "Configure the Determined CLI: export DET_MASTER={master_ip}\n"
        "View the Determined UI: http://{master_ip}:8080\n"
        "View Logs at: https://{region}.console.aws.amazon.com/cloudwatch/home?"
        "region={region}#logStream:group={log_group}"
    )
    template = "vpc.yaml"

    template_parameter_keys = [
        constants.cloudformation.ENABLE_CORS,
        constants.cloudformation.MASTER_TLS_CERT,
        constants.cloudformation.MASTER_TLS_KEY,
        constants.cloudformation.MASTER_CERT_NAME,
        constants.cloudformation.KEYPAIR,
        constants.cloudformation.MASTER_INSTANCE_TYPE,
        constants.cloudformation.AGENT_INSTANCE_TYPE,
        constants.cloudformation.INBOUND_CIDR,
        constants.cloudformation.VERSION,
        constants.cloudformation.DB_PASSWORD,
        constants.cloudformation.MAX_IDLE_AGENT_PERIOD,
        constants.cloudformation.MAX_AGENT_STARTING_PERIOD,
        constants.cloudformation.MAX_DYNAMIC_AGENTS,
        constants.cloudformation.SPOT_ENABLED,
        constants.cloudformation.SPOT_MAX_PRICE,
        constants.cloudformation.CPU_ENV_IMAGE,
        constants.cloudformation.GPU_ENV_IMAGE,
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
        self.print_results(
            self.parameters[constants.cloudformation.CLUSTER_ID],
            self.parameters[constants.cloudformation.BOTO3_SESSION],
        )

    def print_results(self, stack_name: str, boto3_session: boto3.session.Session) -> None:
        output = aws.get_output(stack_name, boto3_session)
        master_ip = output[constants.cloudformation.DET_ADDRESS]
        region = output[constants.cloudformation.REGION]
        log_group = output[constants.cloudformation.LOG_GROUP]

        ui_command = self.det_ui.format(master_ip=master_ip, region=region, log_group=log_group)
        print(ui_command)

        ssh_command = self.ssh_command.format(master_ip=master_ip)
        print(ssh_command)


class FSx(VPC):
    template = "fsx.yaml"


class EFS(VPC):
    template = "efs.yaml"
