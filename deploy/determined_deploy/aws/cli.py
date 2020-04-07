import argparse

import boto3

import determined_deploy
from determined_deploy.aws import aws, constants
from determined_deploy.aws.deployment_types import secure, simple, vpc


def make_aws_parser(subparsers: argparse._SubParsersAction):
    parser_aws = subparsers.add_parser(
        "aws", help="aws help", formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )

    parser_aws.add_argument("--delete", action="store_true", help="Delete Determined from account")
    parser_aws.add_argument(
        "--master-ami",
        type=str,
        default=constants.defaults.MASTER_AMI_ID,
        help="ami for determined master",
    )
    parser_aws.add_argument(
        "--agent-ami", type=str, default=constants.defaults.AGENT_AMI_ID, help="ami for det agent"
    )
    parser_aws.add_argument(
        "--keypair",
        type=str,
        default=constants.defaults.KEYPAIR_NAME,
        help="keypair for master and agent",
    )
    parser_aws.add_argument(
        "--master-instance-type",
        type=str,
        default=constants.defaults.MASTER_INSTANCE_TYPE,
        help="instance type for master",
    )
    parser_aws.add_argument(
        "--agent-instance-type",
        type=str,
        default=constants.defaults.AGENT_INSTANCE_TYPE,
        help="instance type for agent",
    )
    parser_aws.add_argument(
        "--deployment-type",
        type=str,
        choices=constants.deployment_types.DEPLOYMENT_TYPES,
        default=constants.defaults.DEPLOYMENT_TYPE,
        help=f"deployment type - "
        f'must be one of [{", ".join(constants.deployment_types.DEPLOYMENT_TYPES)}]',
    )
    parser_aws.add_argument(
        "--user", type=str, default=None, help="user to name stack and tag resources"
    )
    parser_aws.add_argument(
        "--aws-profile", type=str, default=None, help="aws profile for deploying"
    )
    parser_aws.add_argument(
        "--inbound-cidr",
        type=str,
        default=constants.defaults.INBOUND_CIDR,
        help="inbound IP Range in CIDR format",
    )
    parser_aws.add_argument(
        "--version",
        type=str,
        default=determined_deploy.__version__,
        help="Determined version or commit to deploy",
    )
    parser_aws.add_argument(
        "--db-password",
        type=str,
        default=constants.defaults.DB_PASSWORD,
        help="password for master database",
    )
    parser_aws.add_argument(
        "--hasura-secret",
        type=str,
        default=constants.defaults.HASURA_SECRET,
        help="password for hasura service",
    )
    parser_aws.add_argument(
        "--region-name", type=str, default=constants.defaults.REGION, help="aws region",
    )
    parser_aws.add_argument(
        "--max-idle-agent-period",
        type=str,
        default=constants.defaults.MAX_IDLE_AGENT_PERIOD,
        help="max agent idle time",
    )
    parser_aws.add_argument(
        "--max-instances", type=int, default=constants.defaults.MAX_INSTANCES, help="max instances",
    )

    parser_aws.add_argument(
        "--print", action="store_true", help="print deployment template",
    )


def deploy_aws(args: argparse.Namespace) -> None:
    if args.aws_profile:
        boto3_session = boto3.Session(profile_name=args.aws_profile, region_name=args.region_name)
    else:
        boto3_session = boto3.Session(region_name=args.region_name)

    user = args.user if args.user else aws.get_user(boto3_session)
    user = user.replace(".", "-").replace("_", "-")
    stack_name = constants.defaults.DET_STACK_NAME_BASE.format(user)
    if args.delete:
        aws.delete(stack_name, boto3_session)
        print("Delete Successful")
        return

    deployment_type_map = {
        constants.deployment_types.SIMPLE: simple.Simple,
        constants.deployment_types.SECURE: secure.Secure,
        constants.deployment_types.VPC: vpc.VPC,
    }

    det_configs = {
        constants.cloudformation.MASTER_AMI: args.master_ami,
        constants.cloudformation.AGENT_AMI: args.agent_ami,
        constants.cloudformation.KEYPAIR: args.keypair,
        constants.cloudformation.MASTER_INSTANCE_TYPE: args.master_instance_type,
        constants.cloudformation.AGENT_INSTANCE_TYPE: args.agent_instance_type,
        constants.cloudformation.USER_NAME: user,
        constants.cloudformation.DET_STACK_NAME: stack_name,
        constants.cloudformation.BOTO3_SESSION: boto3_session,
        constants.cloudformation.VERSION: args.version,
        constants.cloudformation.INBOUND_CIDR: args.inbound_cidr,
        constants.cloudformation.DB_PASSWORD: args.db_password,
        constants.cloudformation.HASURA_SECRET: args.hasura_secret,
        constants.cloudformation.MAX_IDLE_AGENT_PERIOD: args.max_idle_agent_period,
        constants.cloudformation.MAX_INSTANCES: args.max_instances,
    }

    deployment_object = deployment_type_map[args.deployment_type](det_configs)

    if args.print:
        deployment_object.print()
        return

    aws.check_keypair(args.keypair, boto3_session)

    print("Starting Determined Deployment")
    deployment_object.deploy()
    print("Determined Deployment Successful")
