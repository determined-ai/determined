import argparse
import os
import sys
from typing import Callable

from termcolor import colored

import determined
import determined.deploy
from determined.common.declarative_argparse import Arg, ArgGroup, Cmd
from determined.deploy.errors import MasterTimeoutExpired
from determined.deploy.gcp import constants, gcp


def validate_cluster_id() -> Callable:
    def validate(s: str) -> str:
        if isinstance(s, str) and len(s) <= 35:
            return s
        raise argparse.ArgumentTypeError("must be at most 35 characters")

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


def deploy_gcp(command: str, args: argparse.Namespace) -> None:
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
    args_dict = vars(args)
    for arg in args_dict:
        if args_dict[arg] is not None:
            det_configs[arg] = args_dict[arg]

    # Not all args will be passed to Terraform, list the ones that won't be
    # TODO(ilia): Switch to filtering variables_to_include instead, i.e.
    # only pass the ones recognized by terraform.
    variables_to_exclude = [
        "command",
        "dry_run",
        "environment",
        "local_state_path",
        "master",
        "user",
        "no_preflight_checks",
        "no_wait_for_master",
        "func",
        "_command",
        "_subcommand",
        "_subsubcommand",
    ]

    # Delete
    if command == "down":
        gcp.delete(det_configs, env)
        print("Delete Successful")
        return

    if (args.cpu_env_image and not args.gpu_env_image) or (
        args.gpu_env_image and not args.cpu_env_image
    ):
        print("If a CPU or GPU image is specified, both should be.")
        sys.exit(1)

    # Dry-run flag
    if args.dry_run:
        gcp.dry_run(det_configs, env, variables_to_exclude)
        print("Printed plan. To execute, run `det deploy gcp`")
        return

    print("Starting Determined Deployment")
    gcp.deploy(det_configs, env, variables_to_exclude)

    if not args.no_wait_for_master:
        try:
            gcp.wait_for_master(det_configs, env, timeout=5 * 60)
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

    if args.no_wait_for_master:
        print("Please allow 1-5 minutes for the master instance to be accessible via the web-ui\n")


def handle_down(args: argparse.Namespace) -> None:
    return deploy_gcp("down", args)


def handle_up(args: argparse.Namespace) -> None:
    return deploy_gcp("up", args)


args_description = Cmd(
    "gcp",
    None,
    "gcp_help",
    [
        Cmd(
            "down",
            handle_down,
            "delete gcp cluster",
            [
                ArgGroup(
                    "optional named arguments",
                    None,
                    [
                        Arg(
                            "--local-state-path",
                            type=str,
                            default=os.getcwd(),
                            help="local directory for storing cluster state",
                        ),
                    ],
                ),
            ],
        ),
        Cmd(
            "up",
            handle_up,
            "create gcp cluster",
            [
                ArgGroup(
                    "required named arguments",
                    None,
                    [
                        Arg(
                            "--cluster-id",
                            type=validate_cluster_id(),
                            default=None,
                            required=True,
                            help="unique identifier to name and tag resources",
                        ),
                        Arg(
                            "--project-id",
                            type=str,
                            default=None,
                            required=True,
                            help="project ID to create the cluster in",
                        ),
                    ],
                ),
                ArgGroup(
                    "optional named arguments",
                    None,
                    [
                        Arg(
                            "--dry-run",
                            action="store_true",
                            help="return the infrastructure plan to be executed "
                            "based on your arguments",
                        ),
                        Arg(
                            "--keypath",
                            type=str,
                            default=None,
                            help="path to service account key if not using default credentials",
                        ),
                        Arg(
                            "--network",
                            type=str,
                            default="det-default",
                            help="network name to create "
                            "(the network should not already exist in the project)",
                        ),
                        Arg(
                            "--det-version",
                            type=str,
                            default=determined.__version__,
                            help=argparse.SUPPRESS,
                        ),
                        Arg(
                            "--region",
                            type=str,
                            default=constants.defaults.REGION,
                            help="region to create the cluster in (defaults to us-west1)",
                        ),
                        Arg(
                            "--zone",
                            type=str,
                            default=None,
                            help="zone to create the cluster in (defaults to `region`-b)",
                        ),
                        Arg(
                            "--environment-image",
                            type=str,
                            default=constants.defaults.ENVIRONMENT_IMAGE,
                            help=argparse.SUPPRESS,
                        ),
                        Arg(
                            "--local-state-path",
                            type=str,
                            default=os.getcwd(),
                            help="local directory for storing cluster state",
                        ),
                        Arg(
                            "--preemptible",
                            type=bool,
                            default=False,
                            help="whether to use preemptible instances for dynamic agents",
                        ),
                        Arg(
                            "--operation-timeout-period",
                            type=str,
                            default=constants.defaults.OPERATION_TIMEOUT_PERIOD,
                            help="operation timeout before retrying, e.g. 5m for 5 minutes",
                        ),
                        Arg(
                            "--master-instance-type",
                            type=str,
                            default=constants.defaults.MASTER_INSTANCE_TYPE,
                            help="instance type for master",
                        ),
                        Arg(
                            "--cpu-agent-instance-type",
                            type=str,
                            default=constants.defaults.CPU_AGENT_INSTANCE_TYPE,
                            help="instance type for agens in the CPU resource pool",
                        ),
                        Arg(
                            "--gpu-agent-instance-type",
                            type=str,
                            default=constants.defaults.GPU_AGENT_INSTANCE_TYPE,
                            help="instance type for agents in the GPU resource pool",
                        ),
                        Arg(
                            "--db-password",
                            type=str,
                            default=constants.defaults.DB_PASSWORD,
                            help="password for master database",
                        ),
                        Arg(
                            "--max-cpu-containers-per-agent",
                            type=str,
                            default=constants.defaults.MAX_CPU_CONTAINERS_PER_AGENT,
                            help="max CPU containers running for agents in the CPU resource pool",
                        ),
                        Arg(
                            "--max-idle-agent-period",
                            type=str,
                            default=constants.defaults.MAX_IDLE_AGENT_PERIOD,
                            help="max agent idle time before it is shut down, "
                            "e.g. 30m for 30 minutes",
                        ),
                        Arg(
                            "--max-agent-starting-period",
                            type=str,
                            default=constants.defaults.MAX_AGENT_STARTING_PERIOD,
                            help="max agent starting time before retrying, e.g. 30m for 30 minutes",
                        ),
                        Arg(
                            "--port",
                            type=int,
                            default=constants.defaults.PORT,
                            help="port to use for communication on master instance",
                        ),
                        Arg(
                            "--gpu-type",
                            type=str,
                            default=constants.defaults.GPU_TYPE,
                            help="type of GPU to use on agents",
                        ),
                        Arg(
                            "--gpu-num",
                            type=int,
                            default=constants.defaults.GPU_NUM,
                            help="number of GPUs per agent instance",
                        ),
                        Arg(
                            "--min-dynamic-agents",
                            type=int,
                            default=constants.defaults.MIN_DYNAMIC_AGENTS,
                            help="minimum number of dynamic agent instances at one time",
                        ),
                        Arg(
                            "--max-dynamic-agents",
                            type=int,
                            default=constants.defaults.MAX_DYNAMIC_AGENTS,
                            help="maximum number of dynamic agent instances at one time",
                        ),
                        Arg(
                            "--static-agents",
                            type=int,
                            default=constants.defaults.STATIC_AGENTS,
                            help=argparse.SUPPRESS,
                        ),
                        Arg(
                            "--min-cpu-platform-master",
                            type=str,
                            default=constants.defaults.MIN_CPU_PLATFORM_MASTER,
                            help="minimum cpu platform for master instances",
                        ),
                        Arg(
                            "--min-cpu-platform-agent",
                            type=str,
                            default=constants.defaults.MIN_CPU_PLATFORM_AGENT,
                            help="minimum cpu platform for agent instances",
                        ),
                        Arg(
                            "--scheduler-type",
                            type=validate_scheduler_type(),
                            default=constants.defaults.SCHEDULER_TYPE,
                            help="scheduler to use (defaults to fair_share).",
                        ),
                        Arg(
                            "--preemption-enabled",
                            type=bool,
                            default=constants.defaults.PREEMPTION_ENABLED,
                            help="whether task preemption is supported in the scheduler "
                            "(only configurable for priority scheduler).",
                        ),
                        Arg(
                            "--cpu-env-image",
                            type=str,
                            default="",
                            help="Docker image for CPU tasks",
                        ),
                        Arg(
                            "--gpu-env-image",
                            type=str,
                            default="",
                            help="Docker image for GPU tasks",
                        ),
                    ],
                ),
            ],
        ),
    ],
)
