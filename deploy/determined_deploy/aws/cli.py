import argparse
import re
import sys
from typing import Callable, Dict, Type, Union

import boto3

from determined_deploy.aws import aws, constants
from determined_deploy.aws.deployment_types import base, secure, simple, vpc


def validate_spot_max_price() -> Callable:
    def validate(s: str) -> str:
        if s.count(".") > 1:
            raise argparse.ArgumentTypeError("must have one or zero decimal points")
        for char in s:
            if not (char.isdigit() or char == "."):
                raise argparse.ArgumentTypeError("must only contain digits and a decimal point")
        return s

    return validate


def make_down_subparser(subparsers: argparse._SubParsersAction) -> None:
    subparser = subparsers.add_parser("down", help="delete CloudFormation stack")
    require_named = subparser.add_argument_group("required named arguments")
    require_named.add_argument(
        "--cluster-id", type=str, help="stack name for CloudFormation cluster", required=True
    )

    subparser.add_argument(
        "--region",
        type=str,
        default=None,
        help="AWS region",
    )
    subparser.add_argument("--aws-profile", type=str, default=None, help=argparse.SUPPRESS)


def make_up_subparser(subparsers: argparse._SubParsersAction) -> None:
    subparser = subparsers.add_parser("up", help="deploy/update CloudFormation stack")
    require_named = subparser.add_argument_group("required named arguments")
    require_named.add_argument(
        "--cluster-id", type=str, help="stack name for CloudFormation cluster", required=True
    )
    require_named.add_argument(
        "--keypair", type=str, help="aws ec2 keypair for master and agent", required=True
    )
    subparser.add_argument(
        "--master-instance-type",
        type=str,
        help="instance type for master",
    )
    subparser.add_argument(
        "--enable-cors",
        action="store_true",
        help="allow CORS requests or not: true/false",
    )
    subparser.add_argument(
        "--agent-instance-type",
        type=str,
        help="instance type for agent",
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
        "--inbound-cidr",
        type=str,
        help="inbound IP Range in CIDR format",
    )
    subparser.add_argument(
        "--agent-subnet-id",
        type=str,
        help="subnet to deploy agents into. Optional. Only used with simple deployment type",
    )
    subparser.add_argument(
        "--det-version",
        type=str,
        help=argparse.SUPPRESS,
    )
    subparser.add_argument(
        "--db-password",
        type=str,
        default=constants.defaults.DB_PASSWORD,
        help="password for master database",
    )
    subparser.add_argument(
        "--region",
        type=str,
        default=None,
        help="AWS region",
    )
    subparser.add_argument(
        "--max-idle-agent-period",
        type=str,
        help="max agent idle time",
    )
    subparser.add_argument(
        "--max-agent-starting-period",
        type=str,
        help="max agent starting time",
    )
    subparser.add_argument(
        "--min-dynamic-agents",
        type=int,
        help="minimum number of dynamic agent instances at one time",
    )
    subparser.add_argument(
        "--max-dynamic-agents",
        type=int,
        help="maximum number of dynamic agent instances at one time",
    )
    subparser.add_argument(
        "--spot",
        action="store_true",
        help="whether to use spot instances or not",
    )
    subparser.add_argument(
        "--spot-max-price",
        type=validate_spot_max_price(),
        help="maximum hourly price for the spot instance (do not include the dollar sign)",
    )
    subparser.add_argument(
        "--dry-run",
        action="store_true",
        help="print deployment template",
    )


def make_aws_parser(subparsers: argparse._SubParsersAction) -> None:
    parser_aws = subparsers.add_parser("aws", help="AWS help")

    aws_subparsers = parser_aws.add_subparsers(help="command", dest="command")
    make_down_subparser(aws_subparsers)
    make_up_subparser(aws_subparsers)
    aws_subparsers.required = True


def deploy_aws(args: argparse.Namespace) -> None:
    if args.aws_profile:
        boto3_session = boto3.Session(profile_name=args.aws_profile, region_name=args.region)
    else:
        boto3_session = boto3.Session(region_name=args.region)

    if boto3_session.region_name not in constants.misc.SUPPORTED_REGIONS:
        print(
            f"det-deploy is only supported in {constants.misc.SUPPORTED_REGIONS} - "
            f"tried to deploy to {boto3_session.region_name}"
        )
        print("use the --region argument to deploy to a supported region")
        sys.exit(1)

    # TODO(DET-4258) Uncomment this when we fully support all P3 regions.
    # if boto3_session.region_name == "eu-west-2" and args.agent_instance_type is None:
    #     print(
    #         "the default agent instance type for det-deploy (p2.8xlarge) is not available in "
    #         "eu-west-2 (London).  Please specify an --agent-instance-type argument."
    #     )
    #     sys.exit(1)

    if not re.match(constants.misc.CLOUDFORMATION_REGEX, args.cluster_id):
        print("Deployment Failed - cluster-id much match ^[a-zA-Z][-a-zA-Z0-9]*$")
        sys.exit(1)

    if args.command == "down":
        try:
            aws.delete(args.cluster_id, boto3_session)
        except Exception as e:
            print(e)
            print("Stack Deletion Failed. Check the AWS CloudFormation Console for details.")
            sys.exit(1)

        print("Delete Successful")
        return

    deployment_type_map = {
        constants.deployment_types.SIMPLE: simple.Simple,
        constants.deployment_types.SECURE: secure.Secure,
        constants.deployment_types.VPC: vpc.VPC,
        constants.deployment_types.EFS: vpc.EFS,
        constants.deployment_types.FSX: vpc.FSx,
    }  # type: Dict[str, Union[Type[base.DeterminedDeployment]]]

    if args.deployment_type != constants.deployment_types.SIMPLE:
        if args.agent_subnet_id != "":
            raise ValueError(
                f"The agent-subnet-id can only be set if the deployment-type=simple. "
                f"The agent-subnet-id was set to '{args.agent_subnet_id}', but the "
                f"deployment-type={args.deployment_type}."
            )

    det_configs = {
        constants.cloudformation.KEYPAIR: args.keypair,
        constants.cloudformation.ENABLE_CORS: args.enable_cors,
        constants.cloudformation.MASTER_INSTANCE_TYPE: args.master_instance_type,
        constants.cloudformation.AGENT_INSTANCE_TYPE: args.agent_instance_type,
        constants.cloudformation.CLUSTER_ID: args.cluster_id,
        constants.cloudformation.BOTO3_SESSION: boto3_session,
        constants.cloudformation.VERSION: args.det_version,
        constants.cloudformation.INBOUND_CIDR: args.inbound_cidr,
        constants.cloudformation.DB_PASSWORD: args.db_password,
        constants.cloudformation.MAX_IDLE_AGENT_PERIOD: args.max_idle_agent_period,
        constants.cloudformation.MAX_AGENT_STARTING_PERIOD: args.max_agent_starting_period,
        constants.cloudformation.MIN_DYNAMIC_AGENTS: args.min_dynamic_agents,
        constants.cloudformation.MAX_DYNAMIC_AGENTS: args.max_dynamic_agents,
        constants.cloudformation.SPOT_ENABLED: args.spot,
        constants.cloudformation.SPOT_MAX_PRICE: args.spot_max_price,
        constants.cloudformation.SUBNET_ID_KEY: args.agent_subnet_id,
    }

    deployment_object = deployment_type_map[args.deployment_type](det_configs)

    if args.dry_run:
        deployment_object.print()
        return

    print("Starting Determined Deployment")
    try:
        deployment_object.deploy()
    except Exception as e:
        print(e)
        print("Stack Deployment Failed. Check the AWS CloudFormation Console for details.")
        sys.exit(1)

    print("Determined Deployment Successful")
