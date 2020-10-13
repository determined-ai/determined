import argparse
import os
from typing import Callable

import determined_deploy
from determined_deploy.gcp import constants, gcp


def validate_cluster_id() -> Callable:
    def validate(s: str) -> str:
        if isinstance(s, str) and len(s) <= 35:
            return s
        raise argparse.ArgumentTypeError("must be at most 35 characters")

    return validate


def make_down_subparser(subparsers: argparse._SubParsersAction) -> None:
    subparser = subparsers.add_parser("down", help="delete gcp cluster")

    optional_named = subparser.add_argument_group("optional named arguments")
    optional_named.add_argument(
        "--local-state-path",
        type=str,
        default=os.getcwd(),
        help="local directory for storing cluster state",
    )


def make_up_subparser(subparsers: argparse._SubParsersAction) -> None:
    parser_gcp = subparsers.add_parser("up", help="create gcp cluster")

    required_named = parser_gcp.add_argument_group("required named arguments")
    required_named.add_argument(
        "--cluster-id",
        type=validate_cluster_id(),
        default=None,
        required=True,
        help="unique identifier to name and tag resources",
    )
    required_named.add_argument(
        "--project-id",
        type=str,
        default=None,
        required=True,
        help="project ID to create the cluster in",
    )

    optional_named = parser_gcp.add_argument_group("optional named arguments")
    optional_named.add_argument(
        "--dry-run",
        action="store_true",
        help="return the infrastructure plan to be executed based on your arguments",
    )
    optional_named.add_argument(
        "--keypath",
        type=str,
        default=None,
        help="path to service account key if not using default credentials",
    )
    optional_named.add_argument(
        "--network",
        type=str,
        default="det-default",
        help="network name to create (the network should not already exist in the project)",
    )
    optional_named.add_argument(
        "--det-version",
        type=str,
        default=determined_deploy.__version__,
        help=argparse.SUPPRESS,
    )
    optional_named.add_argument(
        "--region",
        type=str,
        default=constants.defaults.REGION,
        help="region to create the cluster in (defaults to us-west1)",
    )
    optional_named.add_argument(
        "--zone",
        type=str,
        default=None,
        help="zone to create the cluster in (defaults to `region`-b)",
    )
    optional_named.add_argument(
        "--environment-image",
        type=str,
        default=constants.defaults.ENVIRONMENT_IMAGE,
        help=argparse.SUPPRESS,
    )
    optional_named.add_argument(
        "--local-state-path",
        type=str,
        default=os.getcwd(),
        help="local directory for storing cluster state",
    )
    optional_named.add_argument(
        "--preemptible",
        type=str,
        default="false",
        help="whether to use preemptible instances for agents",
    )
    optional_named.add_argument(
        "--operation-timeout-period",
        type=str,
        default=constants.defaults.OPERATION_TIMEOUT_PERIOD,
        help="operation timeout before retrying, e.g. 5m for 5 minutes",
    )
    optional_named.add_argument(
        "--master-instance-type",
        type=str,
        default=constants.defaults.MASTER_INSTANCE_TYPE,
        help="instance type for master",
    )
    optional_named.add_argument(
        "--agent-instance-type",
        type=str,
        default=constants.defaults.AGENT_INSTANCE_TYPE,
        help="instance type for agent",
    )
    optional_named.add_argument(
        "--db-password",
        type=str,
        default=constants.defaults.DB_PASSWORD,
        help="password for master database",
    )
    optional_named.add_argument(
        "--max-idle-agent-period",
        type=str,
        default=constants.defaults.MAX_IDLE_AGENT_PERIOD,
        help="max agent idle time before it is shut down, e.g. 30m for 30 minutes",
    )
    optional_named.add_argument(
        "--max-agent-starting-period",
        type=str,
        default=constants.defaults.MAX_AGENT_STARTING_PERIOD,
        help="max agent starting time before retrying, e.g. 30m for 30 minutes",
    )
    optional_named.add_argument(
        "--port",
        type=int,
        default=constants.defaults.PORT,
        help="port to use for communication on master instance",
    )
    optional_named.add_argument(
        "--gpu-type",
        type=str,
        default=constants.defaults.GPU_TYPE,
        help="type of GPU to use on agents",
    )
    optional_named.add_argument(
        "--gpu-num",
        type=int,
        default=constants.defaults.GPU_NUM,
        help="number of GPUs per agent instance",
    )
    optional_named.add_argument(
        "--min-dynamic-agents",
        type=int,
        default=constants.defaults.MIN_DYNAMIC_AGENTS,
        help="minimum number of dynamic agent instances at one time",
    )
    optional_named.add_argument(
        "--max-dynamic-agents",
        type=int,
        default=constants.defaults.MAX_DYNAMIC_AGENTS,
        help="maximum number of dynamic agent instances at one time",
    )
    optional_named.add_argument(
        "--static-agents",
        type=int,
        default=constants.defaults.STATIC_AGENTS,
        help=argparse.SUPPRESS,
    )
    optional_named.add_argument(
        "--min-cpu-platform-master",
        type=str,
        default=constants.defaults.MIN_CPU_PLATFORM_MASTER,
        help="minimum cpu platform for master instances",
    )
    optional_named.add_argument(
        "--min-cpu-platform-agent",
        type=str,
        default=constants.defaults.MIN_CPU_PLATFORM_AGENT,
        help="minimum cpu platform for agent instances",
    )


def make_gcp_parser(subparsers: argparse._SubParsersAction) -> None:
    parser_gcp = subparsers.add_parser("gcp", help="gcp help")
    gcp_subparsers = parser_gcp.add_subparsers(help="command", dest="command")
    make_up_subparser(gcp_subparsers)
    make_down_subparser(gcp_subparsers)
    gcp_subparsers.required = True


def deploy_gcp(args: argparse.Namespace) -> None:

    # Set local state path as our current working directory. This is a no-op
    # when the --local-state-path arg isn't used. We do this because Terraform
    # module directories are populated with relative paths, and we want to
    # support users running gcp up and down commands from different directories.
    # Also, because we change the working directory, we ensure that
    # local_state_path is an absolute path.
    args.local_state_path = os.path.abspath(args.local_state_path)
    if not os.path.exists(args.local_state_path):
        os.makedirs(args.local_state_path)
    os.chdir(args.local_state_path)

    # Set the TF_DATA_DIR where Terraform will store its supporting files
    env = os.environ.copy()
    env["TF_DATA_DIR"] = os.path.join(args.local_state_path, "terraform_data")

    # Create det_configs dictionary
    det_configs = {}

    # Add args to det_configs dict
    for arg in vars(args):
        if vars(args)[arg] is not None:
            det_configs[arg] = vars(args)[arg]

    # Not all args will be passed to Terraform, list the ones that won't be
    variables_to_exclude = [
        "command",
        "dry_run",
        "environment",
        "local_state_path",
    ]

    # Delete
    if args.command == "down":
        gcp.delete(det_configs, env)
        print("Delete Successful")
        return

    # Dry-run flag
    if args.dry_run:
        gcp.dry_run(det_configs, env, variables_to_exclude)
        print("Printed plan. To execute, run `det-deploy gcp`")
        return

    print("Starting Determined Deployment")
    gcp.deploy(det_configs, env, variables_to_exclude)
    print("\nDetermined Deployment Successful")
    print("Please allow 1-5 minutes for the master instance to be accessible via the web-ui\n")
