import argparse

import boto3

from determined_deploy.aws import aws, constants
from determined_deploy.aws.deployment_types import secure, simple, vpc


def make_down_subparser(subparsers: argparse._SubParsersAction):
    subparser = subparsers.add_parser("down", help="delete CloudFormation stack")
    require_named = subparser.add_argument_group("required named arguments")
    require_named.add_argument(
        "--cluster-id", type=str, help="stack name for CloudFormation cluster", required=True
    )

    subparser.add_argument(
        "--region", type=str, default=constants.defaults.REGION, help="AWS region",
    )
    subparser.add_argument("--aws-profile", type=str, default=None, help=argparse.SUPPRESS)


def make_up_subparser(subparsers: argparse._SubParsersAction):
    subparser = subparsers.add_parser("up", help="deploy/update CloudFormation stack")
    require_named = subparser.add_argument_group("required named arguments")
    require_named.add_argument(
        "--cluster-id", type=str, help="stack name for CloudFormation cluster", required=True
    )
    require_named.add_argument(
        "--keypair", type=str, help="keypair for master and agent", required=True
    )
    subparser.add_argument(
        "--master-ami", type=str, help=argparse.SUPPRESS,
    )
    subparser.add_argument(
        "--agent-ami", type=str, help=argparse.SUPPRESS,
    )
    subparser.add_argument(
        "--master-instance-type", type=str, help="instance type for master",
    )
    subparser.add_argument(
        "--agent-instance-type", type=str, help="instance type for agent",
    )
    subparser.add_argument(
        "--deployment-type",
        type=str,
        choices=constants.deployment_types.DEPLOYMENT_TYPES,
        default=constants.defaults.DEPLOYMENT_TYPE,
        help=f"deployment type - "
        f'must be one of [{", ".join(constants.deployment_types.DEPLOYMENT_TYPES)}]',
    )
    subparser.add_argument("--aws-profile", type=str, default=None, help=argparse.SUPPRESS)
    subparser.add_argument(
        "--inbound-cidr", type=str, help="inbound IP Range in CIDR format",
    )
    subparser.add_argument(
        "--det-version", type=str, help=argparse.SUPPRESS,
    )
    subparser.add_argument(
        "--db-password",
        type=str,
        default=constants.defaults.DB_PASSWORD,
        help="password for master database",
    )
    subparser.add_argument(
        "--hasura-secret",
        type=str,
        default=constants.defaults.HASURA_SECRET,
        help="password for Hasura service",
    )
    subparser.add_argument(
        "--region", type=str, default=constants.defaults.REGION, help="AWS region",
    )
    subparser.add_argument(
        "--max-idle-agent-period", type=str, help="max agent idle time",
    )
    subparser.add_argument(
        "--max-instances", type=int, help="max instances",
    )

    subparser.add_argument(
        "--dry-run", action="store_true", help="print deployment template",
    )


def make_aws_parser(subparsers: argparse._SubParsersAction):
    parser_aws = subparsers.add_parser("aws", help="AWS help")

    aws_subparsers = parser_aws.add_subparsers(help="command", dest="command")
    make_down_subparser(aws_subparsers)
    make_up_subparser(aws_subparsers)


def deploy_aws(args: argparse.Namespace) -> None:
    if args.aws_profile:
        boto3_session = boto3.Session(profile_name=args.aws_profile, region_name=args.region)
    else:
        boto3_session = boto3.Session(region_name=args.region)

    if args.command == "down":
        aws.delete(args.cluster_id, boto3_session)
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
        constants.cloudformation.CLUSTER_ID: args.cluster_id,
        constants.cloudformation.BOTO3_SESSION: boto3_session,
        constants.cloudformation.VERSION: args.det_version,
        constants.cloudformation.INBOUND_CIDR: args.inbound_cidr,
        constants.cloudformation.DB_PASSWORD: args.db_password,
        constants.cloudformation.HASURA_SECRET: args.hasura_secret,
        constants.cloudformation.MAX_IDLE_AGENT_PERIOD: args.max_idle_agent_period,
        constants.cloudformation.MAX_INSTANCES: args.max_instances,
    }

    deployment_object = deployment_type_map[args.deployment_type](det_configs)

    if args.dry_run:
        deployment_object.print()
        return

    print("Starting Determined Deployment")
    deployment_object.deploy()
    print("Determined Deployment Successful")
