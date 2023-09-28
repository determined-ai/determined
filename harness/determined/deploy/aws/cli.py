import argparse
import base64
import json
import re
from pathlib import Path
from typing import Callable, Dict, Tuple, Type

import boto3
from botocore.exceptions import NoCredentialsError
from termcolor import colored

from determined.cli.errors import CliError
from determined.common.declarative_argparse import Arg, ArgGroup, BoolOptArg, Cmd
from determined.deploy.aws import aws, constants
from determined.deploy.errors import MasterTimeoutExpired

from .deployment_types import base, govcloud, secure, simple, vpc
from .preflight import check_quotas, get_default_cf_parameter


def validate_spot_max_price() -> Callable:
    def validate(s: str) -> str:
        if s.count(".") > 1:
            raise argparse.ArgumentTypeError("must have one or zero decimal points")
        for char in s:
            if not (char.isdigit() or char == "."):
                raise argparse.ArgumentTypeError("must only contain digits and a decimal point")
        return s

    return validate


def parse_add_tag() -> Callable:
    def parse(s: str) -> Tuple[str, str]:
        try:
            key, value = s.split("=", 1)
        except ValueError:
            raise argparse.ArgumentTypeError("key=value format requires both a key and a value")

        if not key or not value:
            raise argparse.ArgumentTypeError(
                "both key and value must be defined in key=value format"
            )

        if key in ["deployment-type", "managed-by"]:
            raise argparse.ArgumentTypeError("cannot us a reserved tag name: %s" % key)

        return (key, value)

    return parse


def error_no_credentials() -> None:
    print(
        colored("Unable to locate AWS credentials.", "red"),
        "Did you run %s?" % colored("aws configure", "yellow"),
    )
    raise CliError(
        "See the AWS Documentation for information on how to use AWS credentials: "
        "https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html",
    )


def get_deployment_class(deployment_type: str) -> Type[base.DeterminedDeployment]:
    deployment_type_map = {
        constants.deployment_types.SIMPLE: simple.Simple,
        constants.deployment_types.SIMPLE_RDS: simple.SimpleRDS,
        constants.deployment_types.SECURE: secure.Secure,
        constants.deployment_types.EFS: vpc.EFS,
        constants.deployment_types.FSX: vpc.FSx,
        constants.deployment_types.GOVCLOUD: govcloud.Govcloud,
    }  # type: Dict[str, Type[base.DeterminedDeployment]]
    return deployment_type_map[deployment_type]


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
        raise CliError("use the --region argument to deploy to a supported region")

    if command == "list":
        try:
            output = aws.list_stacks(boto3_session)
        except NoCredentialsError:
            error_no_credentials()
        except Exception as e:
            raise CliError(
                "Listing stacks failed. Check the AWS CloudFormation Console for details.",
                e_stack=e,
            )
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
        raise CliError("Deployment Failed - cluster-id much match ^[a-zA-Z][-a-zA-Z0-9]*$")

    if command == "down":
        if not args.yes:
            val = input(
                "Deleting an AWS stack will lose all your data, including the created network "
                "file system. Please back up the file system before deleting it. Do you still "
                "want to delete the stack? [y/N]"
            )
            if val.lower() != "y":
                print("Delete cancelled.")
                return

        try:
            aws.delete(args.cluster_id, boto3_session)
        except NoCredentialsError:
            error_no_credentials()
        except Exception as e:
            raise CliError(
                "Stack Deletion Failed. Check the AWS CloudFormation Console for details.",
                e_stack=e,
            )

        print("Delete Successful")
        return

    if (args.cpu_env_image and not args.gpu_env_image) or (
        args.gpu_env_image and not args.cpu_env_image
    ):
        raise CliError("If a CPU or GPU environment image is specified, both should be.")

    if (
        args.deployment_type != constants.deployment_types.SIMPLE
        or args.deployment_type != constants.deployment_types.SIMPLE_RDS
    ):
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

    if args.deployment_type != constants.deployment_types.SIMPLE_RDS:
        if args.db_instance_type != constants.defaults.DB_INSTANCE_TYPE:
            raise ValueError(
                f"--db-instance-size cannot be specified for deployment types other than "
                f"{constants.deployment_types.SIMPLE_RDS} got {args.deployment_type}"
            )
        if args.db_size != constants.defaults.DB_SIZE:
            raise ValueError(
                f"--db-size cannot be specified for deployment types other than "
                f"{constants.deployment_types.SIMPLE_RDS} got {args.deployment_type}"
            )
    else:
        if args.db_size is not None and args.db_size < 20:
            raise ValueError("--db-size must be greater than or equal to 20 GB")

    if args.deployment_type != constants.deployment_types.EFS:
        if args.efs_id is not None:
            raise ValueError("--efs-id can only be specified for 'efs' deployments")

    if args.deployment_type != constants.deployment_types.FSX:
        if args.fsx_id is not None:
            raise ValueError("--fsx-id can only be specified for 'fsx' deployments")

    master_tls_cert = master_tls_key = ""
    if args.master_tls_cert:
        with open(args.master_tls_cert, "rb") as f:
            master_tls_cert = base64.b64encode(f.read()).decode()
    if args.master_tls_key:
        with open(args.master_tls_key, "rb") as f:
            master_tls_key = base64.b64encode(f.read()).decode()

    config_file_contents = {}

    if args.shut_down_on_connection_loss:
        config_file_contents["hooks"] = {"on_connection_lost": ["shutdown", "now"]}

    master_image_name, agent_image_name = "determined-master", "determined-agent"
    if args.enterprise_edition:
        master_image_name, agent_image_name = "hpe-mlde-master", "hpe-mlde-agent"

    det_configs = {
        constants.cloudformation.KEYPAIR: args.keypair,
        constants.cloudformation.ENABLE_CORS: args.enable_cors,
        constants.cloudformation.MASTER_TLS_CERT: master_tls_cert,
        constants.cloudformation.MASTER_TLS_KEY: master_tls_key,
        constants.cloudformation.MASTER_CERT_NAME: args.master_cert_name,
        constants.cloudformation.MASTER_INSTANCE_TYPE: args.master_instance_type,
        constants.cloudformation.AUX_AGENT_INSTANCE_TYPE: args.aux_agent_instance_type,
        constants.cloudformation.COMPUTE_AGENT_INSTANCE_TYPE: args.compute_agent_instance_type,
        constants.cloudformation.CLUSTER_ID: args.cluster_id,
        constants.cloudformation.EXTRA_TAGS: args.add_tag,
        constants.cloudformation.BOTO3_SESSION: boto3_session,
        constants.cloudformation.VERSION: args.det_version,
        constants.cloudformation.INBOUND_CIDR: args.inbound_cidr,
        constants.cloudformation.DB_PASSWORD: args.db_password,
        constants.cloudformation.DB_INSTANCE_TYPE: args.db_instance_type,
        constants.cloudformation.DB_SIZE: args.db_size,
        constants.cloudformation.MAX_IDLE_AGENT_PERIOD: args.max_idle_agent_period,
        constants.cloudformation.MAX_AGENT_STARTING_PERIOD: args.max_agent_starting_period,
        constants.cloudformation.MAX_AUX_CONTAINERS_PER_AGENT: args.max_aux_containers_per_agent,
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
        constants.cloudformation.IMAGE_REPO_PREFIX: args.image_repo_prefix,
        constants.cloudformation.MOUNT_EFS_ID: args.efs_id,
        constants.cloudformation.MOUNT_FSX_ID: args.fsx_id,
        constants.cloudformation.AGENT_REATTACH_ENABLED: args.agent_reattach_enabled,
        constants.cloudformation.AGENT_RECONNECT_ATTEMPTS: args.agent_reconnect_attempts,
        constants.cloudformation.AGENT_RECONNECT_BACKOFF: args.agent_reconnect_backoff,
        constants.cloudformation.AGENT_CONFIG_FILE_CONTENTS: json.dumps(config_file_contents),
        constants.cloudformation.MASTER_IMAGE_NAME: master_image_name,
        constants.cloudformation.AGENT_IMAGE_NAME: agent_image_name,
        constants.cloudformation.DOCKER_USER: args.docker_user,
        constants.cloudformation.DOCKER_PASS: args.docker_pass,
        constants.cloudformation.NOTEBOOK_TIMEOUT: args.notebook_timeout,
    }

    if args.master_config_template_path:
        if not args.master_config_template_path.exists():
            raise ValueError(
                f"Input master config template doesn't exist: {args.master_config_template_path}"
            )
        with args.master_config_template_path.open("r") as fin:
            det_configs[constants.cloudformation.MASTER_CONFIG_TEMPLATE] = fin.read()

    deployment_object = get_deployment_class(args.deployment_type)(det_configs)

    if not args.no_preflight_checks:
        check_quotas(det_configs, deployment_object)

    if args.dry_run:
        deployment_object.print()
        return

    print("Starting Determined Deployment")
    try:
        deployment_object.deploy(args.yes, args.update_terminate_agents)
    except NoCredentialsError:
        error_no_credentials()
    except Exception as e:
        raise CliError(
            "Stack Deployment Failed. Check the AWS CloudFormation Console for details.", e_stack=e
        )

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
            raise CliError(
                "For details, SSH to master instance and check /var/log/cloud-init-output.log."
            )

    print("Determined Deployment Successful")


def handle_list(args: argparse.Namespace) -> None:
    return deploy_aws("list", args)


def handle_up(args: argparse.Namespace) -> None:
    return deploy_aws("up", args)


def handle_down(args: argparse.Namespace) -> None:
    return deploy_aws("down", args)


def handle_dump_master_config_template(args: argparse.Namespace) -> None:
    deployment_object = get_deployment_class(args.deployment_type)({})
    default_template = get_default_cf_parameter(
        deployment_object, constants.cloudformation.MASTER_CONFIG_TEMPLATE
    )
    print(default_template)


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
                Arg(
                    "--no-prompt",
                    dest="yes",
                    action="store_true",
                    help=argparse.SUPPRESS,
                ),
                Arg(
                    "--yes",
                    action="store_true",
                    help="no prompt when deleting resources",
                ),
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
                Arg(
                    "--add-tag",
                    type=parse_add_tag(),
                    action="append",
                    default=None,
                    help="Stack tag to in key=value format, declare repeatedly to add more flags",
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
                    "--compute-agent-instance-type",
                    "--gpu-agent-instance-type",
                    type=str,
                    help="instance type for agents in the compute resource pool",
                ),
                Arg(
                    "--aux-agent-instance-type",
                    "--cpu-agent-instance-type",
                    type=str,
                    help="instance type for agents in the auxiliary resource pool",
                ),
                Arg(
                    "--deployment-type",
                    type=str,
                    choices=constants.deployment_types.DEPLOYMENT_TYPES,
                    default=constants.defaults.DEPLOYMENT_TYPE,
                    help="deployment type",
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
                    "--db-instance-type",
                    type=str,
                    default=constants.defaults.DB_INSTANCE_TYPE,
                    help="instance type for master database (only for simple-rds)",
                ),
                Arg(
                    "--db-size",
                    type=int,
                    default=constants.defaults.DB_SIZE,
                    help="storage size in GB for master database (only for simple-rds)",
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
                    "--max-aux-containers-per-agent",
                    "--max-cpu-containers-per-agent",
                    type=int,
                    help="maximum number of containers on agents in the auxiliary resource pool",
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
                    help="maximum hourly price for spot instances "
                    "(do not include the dollar sign)",
                ),
                Arg(
                    "--scheduler-type",
                    type=str,
                    choices=["fair_share", "priority", "round_robin"],
                    default="fair_share",
                    help="scheduler to use",
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
                Arg(
                    "--master-config-template-path",
                    type=Path,
                    default=None,
                    help="path to master yaml template",
                ),
                Arg(
                    "--efs-id",
                    type=str,
                    help="preexisting EFS that will be mounted into the task containers; "
                    "if not provided, a new EFS instance will be created. The agents must be "
                    "able to connect to the EFS instance.",
                ),
                Arg(
                    "--fsx-id",
                    type=str,
                    help="preexisting FSx that will be mounted into the task containers; "
                    "if not provided, a new FSx instance will be created. The agents must be "
                    "able to connect to the FSx instance.",
                ),
                Arg(
                    "--agent-reattach-enabled",
                    type=str,
                    help="whether master & agent try to recover running containers after a restart",
                ),
                Arg(
                    "--agent-reconnect-attempts",
                    type=int,
                    default=5,
                    help="max attempts an agent has to reconnect",
                ),
                Arg(
                    "--agent-reconnect-backoff",
                    type=int,
                    default=5,
                    help="time between reconnect attempts, with the exception of the first.",
                ),
                BoolOptArg(
                    "--shut-down-agents-on-connection-loss",
                    "--no-shut-down-agents-on-connection-loss",
                    dest="shut_down_on_connection_loss",
                    default=True,
                    true_help="shut down agent instances on connection loss",
                    false_help="do not shut down agent instances on connection loss",
                ),
                BoolOptArg(
                    "--update-terminate-agents",
                    "--no-update-terminate-agents",
                    dest="update_terminate_agents",
                    default=True,
                    true_help="terminate running agents on stack update",
                    false_help="do not terminate running agents on stack update",
                ),
                Arg(
                    "--yes",
                    action="store_true",
                    help="no prompt when deployment would delete existing database",
                ),
                Arg(
                    "--no-prompt",
                    dest="yes",
                    action="store_true",
                    help=argparse.SUPPRESS,
                ),
                Arg(
                    "--enterprise-edition",
                    action="store_true",
                    help="Deploy the enterprise edition of Determined",
                ),
                Arg(
                    "--docker-user",
                    type=str,
                    help="Docker user to pull the Determined master and agent images",
                ),
                Arg(
                    "--docker-pass",
                    type=str,
                    help="Docker password used to pull the Determined master and agent images",
                ),
                Arg(
                    "--notebook-timeout",
                    type=int,
                    help="Specifies the duration in seconds before idle notebook instances "
                    "are automatically terminated",
                ),
            ],
        ),
        Cmd(
            "dump-master-config-template",
            handle_dump_master_config_template,
            "dump default master config template",
            [
                Arg(
                    "--deployment-type",
                    type=str,
                    choices=constants.deployment_types.DEPLOYMENT_TYPES,
                    default=constants.defaults.DEPLOYMENT_TYPE,
                    help="deployment type",
                ),
            ],
        ),
    ],
)
