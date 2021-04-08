import argparse
import base64
import re
import sys
from typing import Callable, Dict, Type, Union

import boto3
from botocore.exceptions import NoCredentialsError
from termcolor import colored

from determined.common.declarative_argparse import Arg, ArgGroup, Cmd
from determined.deploy.errors import MasterTimeoutExpired

from . import aws, constants
from .deployment_types import base, govcloud, secure, simple, vpc
from .preflight import check_quotas


def validate_spot_max_price() -> Callable:
    def validate(s: str) -> str:
        if s.count(".") > 1:
            raise argparse.ArgumentTypeError("must have one or zero decimal points")
        for char in s:
            if not (char.isdigit() or char == "."):
                raise argparse.ArgumentTypeError("must only contain digits and a decimal point")
        return s

    return validate


def validate_scheduler_type() -> Callable:
    def validate(s: str) -> str:
        supported_scheduler_types = ["fair_share", "priority", "round_robin"]
        if s not in supported_scheduler_types:
            raise argparse.ArgumentTypeError(
                f"supported schedulers are: {supported_scheduler_types}"
            )
        return s

    return validate


def error_no_credentials() -> None:
    print(
        colored("Unable to locate AWS credentials.", "red"),
        "Did you run %s?" % colored("aws configure", "yellow"),
    )
    print(
        "See the AWS Documentation for information on how to use AWS credentials:",
        "https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html",
    )
    sys.exit(1)


def deploy_aws(command: str, args: argparse.Namespace) -> None:
    if args.profile:
        boto3_session = boto3.Session(profile_name=args.profile, region_name=args.region)
    else:
        boto3_session = boto3.Session(region_name=args.region)

    if boto3_session.region_name not in constants.misc.SUPPORTED_REGIONS:
        print(
            f"`det deploy` is only supported in {constants.misc.SUPPORTED_REGIONS} - "
            f"tried to deploy to {boto3_session.region_name}"
        )
        print("use the --region argument to deploy to a supported region")
        sys.exit(1)

    if command == "list":
        try:
            output = aws.list_stacks(boto3_session)
        except NoCredentialsError:
            error_no_credentials()
        except Exception as e:
            print(e)
            print("Listing stacks failed. Check the AWS CloudFormation Console for details.")
            sys.exit(1)
        for item in output:
            print(item["StackName"])
        return

    # TODO(DET-4258) Uncomment this when we fully support all P3 regions.
    # if boto3_session.region_name == "eu-west-2" and args.agent_instance_type is None:
    #     print(
    #         "the default agent instance type for `det deploy` (p2.8xlarge) is not available in "
    #         "eu-west-2 (London).  Please specify an --agent-instance-type argument."
    #     )
    #     sys.exit(1)

    if not re.match(constants.misc.CLOUDFORMATION_REGEX, args.cluster_id):
        print("Deployment Failed - cluster-id much match ^[a-zA-Z][-a-zA-Z0-9]*$")
        sys.exit(1)

    if command == "down":
        try:
            aws.delete(args.cluster_id, boto3_session)
        except NoCredentialsError:
            error_no_credentials()
        except Exception as e:
            print(e)
            print("Stack Deletion Failed. Check the AWS CloudFormation Console for details.")
            sys.exit(1)

        print("Delete Successful")
        return

    if (args.cpu_env_image and not args.gpu_env_image) or (
        args.gpu_env_image and not args.cpu_env_image
    ):
        print("If a CPU or GPU environment image is specified, both should be.")
        sys.exit(1)

    deployment_type_map = {
        constants.deployment_types.SIMPLE: simple.Simple,
        constants.deployment_types.SECURE: secure.Secure,
        constants.deployment_types.VPC: vpc.VPC,
        constants.deployment_types.EFS: vpc.EFS,
        constants.deployment_types.FSX: vpc.FSx,
        constants.deployment_types.GOVCLOUD: govcloud.Govcloud,
    }  # type: Dict[str, Union[Type[base.DeterminedDeployment]]]

    if args.deployment_type != constants.deployment_types.SIMPLE:
        if args.agent_subnet_id is not None:
            raise ValueError(
                f"The agent-subnet-id can only be set if the deployment-type=simple. "
                f"The agent-subnet-id was set to '{args.agent_subnet_id}', but the "
                f"deployment-type={args.deployment_type}."
            )

    if args.deployment_type == constants.deployment_types.GOVCLOUD:
        if args.region not in ["us-gov-east-1", "us-gov-west-1"]:
            raise ValueError(
                "When deploying to GovCloud, set the region to either us-gov-east-1 "
                "or us-gov-west-1."
            )

    master_tls_cert = master_tls_key = ""
    if args.master_tls_cert:
        with open(args.master_tls_cert, "rb") as f:
            master_tls_cert = base64.b64encode(f.read()).decode()
    if args.master_tls_key:
        with open(args.master_tls_key, "rb") as f:
            master_tls_key = base64.b64encode(f.read()).decode()

    det_configs = {
        constants.cloudformation.KEYPAIR: args.keypair,
        constants.cloudformation.ENABLE_CORS: args.enable_cors,
        constants.cloudformation.MASTER_TLS_CERT: master_tls_cert,
        constants.cloudformation.MASTER_TLS_KEY: master_tls_key,
        constants.cloudformation.MASTER_CERT_NAME: args.master_cert_name,
        constants.cloudformation.MASTER_INSTANCE_TYPE: args.master_instance_type,
        constants.cloudformation.CPU_AGENT_INSTANCE_TYPE: args.cpu_agent_instance_type,
        constants.cloudformation.GPU_AGENT_INSTANCE_TYPE: args.gpu_agent_instance_type,
        constants.cloudformation.CLUSTER_ID: args.cluster_id,
        constants.cloudformation.BOTO3_SESSION: boto3_session,
        constants.cloudformation.VERSION: args.det_version,
        constants.cloudformation.INBOUND_CIDR: args.inbound_cidr,
        constants.cloudformation.DB_PASSWORD: args.db_password,
        constants.cloudformation.MAX_IDLE_AGENT_PERIOD: args.max_idle_agent_period,
        constants.cloudformation.MAX_AGENT_STARTING_PERIOD: args.max_agent_starting_period,
        constants.cloudformation.MAX_CPU_CONTAINERS_PER_AGENT: args.max_cpu_containers_per_agent,
        constants.cloudformation.MIN_DYNAMIC_AGENTS: args.min_dynamic_agents,
        constants.cloudformation.MAX_DYNAMIC_AGENTS: args.max_dynamic_agents,
        constants.cloudformation.SPOT_ENABLED: args.spot,
        constants.cloudformation.SPOT_MAX_PRICE: args.spot_max_price,
        constants.cloudformation.SUBNET_ID_KEY: args.agent_subnet_id,
        constants.cloudformation.SCHEDULER_TYPE: args.scheduler_type,
        constants.cloudformation.PREEMPTION_ENABLED: args.preemption_enabled,
        constants.cloudformation.CPU_ENV_IMAGE: args.cpu_env_image,
        constants.cloudformation.GPU_ENV_IMAGE: args.gpu_env_image,
        constants.cloudformation.LOG_GROUP_PREFIX: args.log_group_prefix,
        constants.cloudformation.RETAIN_LOG_GROUP: args.retain_log_group,
    }

    deployment_object = deployment_type_map[args.deployment_type](det_configs)

    if not args.no_preflight_checks:
        check_quotas(det_configs, deployment_object)

    if args.dry_run:
        deployment_object.print()
        return

    print("Starting Determined Deployment")
    try:
        deployment_object.deploy()
    except NoCredentialsError:
        error_no_credentials()
    except Exception as e:
        print(e)
        print(
            colored(
                "Stack Deployment Failed. Check the AWS CloudFormation Console for details.", "red"
            )
        )
        sys.exit(1)

    if not args.no_wait_for_master:
        try:
            deployment_object.wait_for_master(timeout=5 * 60)
        except MasterTimeoutExpired:
            print(
                colored(
                    "Determined cluster has been deployed, but master health check has failed.",
                    "red",
                )
            )
            print("For details, SSH to master instance and check /var/log/cloud-init-output.log.")
            sys.exit(1)

    print("Determined Deployment Successful")


def handle_list(args: argparse.Namespace) -> None:
    return deploy_aws("list", args)


def handle_up(args: argparse.Namespace) -> None:
    return deploy_aws("up", args)


def handle_down(args: argparse.Namespace) -> None:
    return deploy_aws("down", args)


args_description = Cmd(
    "aws",
    None,
    "AWS help",
    [
        Cmd(
            "list",
            handle_list,
            "list CloudFormation stacks",
            [
                Arg(
                    "--region",
                    type=str,
                    default=None,
                    help="AWS region",
                ),
                Arg("--profile", type=str, default=None, help="AWS profile"),
            ],
        ),
        Cmd(
            "down",
            handle_down,
            "delete CloudFormation stack",
            [
                ArgGroup(
                    "required named arguments",
                    None,
                    [
                        Arg(
                            "--cluster-id",
                            type=str,
                            help="stack name for CloudFormation cluster",
                            required=True,
                        ),
                    ],
                ),
                Arg(
                    "--region",
                    type=str,
                    default=None,
                    help="AWS region",
                ),
                Arg("--profile", type=str, default=None, help="AWS profile"),
            ],
        ),
        Cmd(
            "up",
            handle_up,
            "deploy/update CloudFormation stack",
            [
                ArgGroup(
                    "required named arguments",
                    None,
                    [
                        Arg(
                            "--cluster-id",
                            type=str,
                            help="stack name for CloudFormation cluster",
                            required=True,
                        ),
                        Arg(
                            "--keypair",
                            type=str,
                            help="aws ec2 keypair for master and agent",
                            required=True,
                        ),
                    ],
                ),
                Arg(
                    "--region",
                    type=str,
                    default=None,
                    help="AWS region",
                ),
                Arg("--profile", type=str, default=None, help="AWS profile"),
                Arg(
                    "--master-instance-type",
                    type=str,
                    help="instance type for master",
                ),
                Arg(
                    "--enable-cors",
                    action="store_true",
                    help="allow CORS requests or not: true/false",
                ),
                Arg("--master-tls-cert"),
                Arg("--master-tls-key"),
                Arg("--master-cert-name"),
                Arg(
                    "--gpu-agent-instance-type",
                    type=str,
                    help="instance type for agent in the GPU resource pool",
                ),
                Arg(
                    "--cpu-agent-instance-type",
                    type=str,
                    help="instance type for agent in the CPU resource pool",
                ),
                Arg(
                    "--deployment-type",
                    type=str,
                    choices=constants.deployment_types.DEPLOYMENT_TYPES,
                    default=constants.defaults.DEPLOYMENT_TYPE,
                    help=f"deployment type - "
                    f'must be one of [{", ".join(constants.deployment_types.DEPLOYMENT_TYPES)}]',
                ),
                Arg(
                    "--inbound-cidr",
                    type=str,
                    help="inbound IP Range in CIDR format",
                ),
                Arg(
                    "--agent-subnet-id",
                    type=str,
                    help="subnet to deploy agents into. Optional. "
                    "Only used with simple deployment type",
                ),
                Arg(
                    "--det-version",
                    type=str,
                    help=argparse.SUPPRESS,
                ),
                Arg(
                    "--db-password",
                    type=str,
                    default=constants.defaults.DB_PASSWORD,
                    help="password for master database",
                ),
                Arg(
                    "--max-idle-agent-period",
                    type=str,
                    help="max agent idle time",
                ),
                Arg(
                    "--max-agent-starting-period",
                    type=str,
                    help="max agent starting time",
                ),
                Arg(
                    "--max-cpu-containers-per-agent",
                    type=int,
                    help="maximum number of cpu containers on agent in the CPU resource pool",
                ),
                Arg(
                    "--min-dynamic-agents",
                    type=int,
                    help="minimum number of dynamic agent instances at one time",
                ),
                Arg(
                    "--max-dynamic-agents",
                    type=int,
                    help="maximum number of dynamic agent instances at one time",
                ),
                Arg(
                    "--spot",
                    action="store_true",
                    help="whether to use spot instances or not",
                ),
                Arg(
                    "--spot-max-price",
                    type=validate_spot_max_price(),
                    help="maximum hourly price for the spot instance "
                    "(do not include the dollar sign)",
                ),
                Arg(
                    "--scheduler-type",
                    type=validate_scheduler_type(),
                    default="fair_share",
                    help="scheduler to use (defaults to fair_share).",
                ),
                Arg(
                    "--preemption-enabled",
                    type=str,
                    default="false",
                    help="whether task preemption is supported in the scheduler "
                    "(only configurable for priority scheduler).",
                ),
                Arg(
                    "--dry-run",
                    action="store_true",
                    help="print deployment template",
                ),
                Arg(
                    "--cpu-env-image",
                    type=str,
                    help="Docker image for CPU tasks",
                ),
                Arg(
                    "--gpu-env-image",
                    type=str,
                    help="Docker image for GPU tasks",
                ),
                Arg(
                    "--log-group-prefix",
                    type=str,
                    help="prefix for output CloudWatch log group",
                ),
                Arg(
                    "--retain-log-group",
                    action="store_const",
                    const="true",
                    help="whether to retain CloudWatch log group after the stack is deleted"
                    " (only available for the simple template)",
                ),
            ],
        ),
    ],
)
