import argparse
import getpass
import os
import pathlib
import sys
from typing import Any, Callable, Tuple

import pkg_resources
import termcolor

import determined
from determined import cli
from determined.cli import errors
from determined.deploy import errors as deploy_errors
from determined.deploy.gcp import constants, gcp


def validate_cluster_id() -> Callable:
    def validate(s: str) -> str:
        if isinstance(s, str) and len(s) <= 35:
            return s
        raise argparse.ArgumentTypeError("must be at most 35 characters")

    return validate


def deploy_gcp(command: str, args: argparse.Namespace) -> None:
    # Preprocess the local path to store the states.

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

    # Set default tf state gcs bucket as '$PROJECT_NAME-determined-deploy` if local tf
    # state doesn't exist and user has not provided a gcs bucket.
    if (
        (not os.path.exists(os.path.join(args.local_state_path, "terraform.tfstate")))
        and hasattr(args, "tf_state_gcs_bucket_name")
        and not args.tf_state_gcs_bucket_name
    ):
        # if user has provided credentials using --keypath, use them
        if hasattr(args, "keypath") and args.keypath:
            gcp.check_or_create_gcsbucket(args.project_id, args.keypath)
        else:
            gcp.check_or_create_gcsbucket(args.project_id)

        args.tf_state_gcs_bucket_name = args.project_id + "-determined-deploy"

    # tf_state_gcs_bucket_name argument is not necessary for `down` operation, because TF reads it
    # from local tf files.
    if hasattr(args, "tf_state_gcs_bucket_name") and args.tf_state_gcs_bucket_name:
        print("Using GCS bucket for state:", args.tf_state_gcs_bucket_name)
    else:
        print("Using local state path:", args.local_state_path)

    # Set the TF_DATA_DIR where Terraform will store its supporting files
    env = os.environ.copy()
    env["TF_DATA_DIR"] = os.path.join(args.local_state_path, "terraform_data")

    # Initialize determined configurations.
    det_configs = {}
    args_dict = vars(args)
    for arg in args_dict:
        if args_dict[arg] is not None:
            det_configs[arg] = args_dict[arg]

    # Not all args will be passed to Terraform, list the ones that won't be
    # TODO(ilia): Switch to filtering variables_to_include instead, i.e.
    #             only pass the ones recognized by terraform.
    variables_to_exclude = [
        "command",
        "dry_run",
        "environment",
        "local_state_path",
        "master",
        "user",
        "no_preflight_checks",
        "no_wait_for_master",
        "yes",
        "no_prompt",
        "master_config_template_path",
        "tf_state_gcs_bucket_name",
        "func",
        "add_label",
        "_command",
        "_subcommand",
        "_subsubcommand",
    ]

    # Handle down subcommand.
    if command == "down" and args.cluster_id:
        if not args.project_id:
            raise errors.CliError(
                "Error: --project-id not provided. Please provide both project id"
                + " and cluster id to delete the cluster."
            )

        # TODO: Find a way to get config defaults using CLI Parser Pipeline.
        det_configs = {
            "no_preflight_checks": False,
            "no_wait_for_master": False,
            "image_repo_prefix": "determinedai",
            "cluster_id": args.cluster_id,
            "project_id": args.project_id,
            "network": "det-default",
            "filestore_address": "",
            "no_filestore": False,
            "det_version": determined.__version__,
            "region": constants.defaults.REGION,
            "disk_size": constants.defaults.BOOT_DISK_SIZE,
            "disk_type": constants.defaults.BOOT_DISK_TYPE,
            "environment_image": constants.defaults.ENVIRONMENT_IMAGE,
            "preemptible": False,
            "operation_timeout_period": constants.defaults.OPERATION_TIMEOUT_PERIOD,
            "master_instance_type": constants.defaults.MASTER_INSTANCE_TYPE,
            "compute_agent_instance_type": constants.defaults.COMPUTE_AGENT_INSTANCE_TYPE,
            "aux_agent_instance_type": constants.defaults.AUX_AGENT_INSTANCE_TYPE,
            "db_password": constants.defaults.AUX_AGENT_INSTANCE_TYPE,
            "max_aux_containers_per_agent": constants.defaults.MAX_AUX_CONTAINERS_PER_AGENT,
            "max_idle_agent_period": constants.defaults.MAX_IDLE_AGENT_PERIOD,
            "max_agent_starting_period": constants.defaults.MAX_AGENT_STARTING_PERIOD,
            "port": constants.defaults.PORT,
            "gpu_type": constants.defaults.GPU_TYPE,
            "gpu_num": constants.defaults.GPU_NUM,
            "min_dynamic_agents": constants.defaults.MIN_DYNAMIC_AGENTS,
            "max_dynamic_agents": constants.defaults.MAX_DYNAMIC_AGENTS,
            "min_cpu_platform_master": constants.defaults.MIN_CPU_PLATFORM_MASTER,
            "min_cpu_platform_agent": constants.defaults.MIN_CPU_PLATFORM_AGENT,
            "scheduler_type": constants.defaults.SCHEDULER_TYPE,
            "preemption_enabled": constants.defaults.PREEMPTION_ENABLED,
            "cpu_env_image": "",
            "gpu_env_image": "",
            "labels": {},
            "local_state_path": os.getcwd(),
            "tf_state_gcs_bucket_name": args.project_id + "-determined-deploy",
        }

        gcp.delete(det_configs, env, args.yes)
        print("Delete Successful")
        return
    elif command == "down":
        gcp.delete(det_configs, env, args.yes)
        print("Delete Successful")
        return

    det_configs["labels"] = dict(det_configs.get("add_label", []))
    reserved_labels = {
        "determined-master-host",
        "determined-master-port",
        "determined-resource-pool",
        "managed-by",
    }
    if reserved_labels.intersection(det_configs["labels"]):
        print(f"The labels {reserved_labels} are reserved for agents.")
        sys.exit(1)

    # Handle Up subcommand.
    if (args.cpu_env_image and not args.gpu_env_image) or (
        args.gpu_env_image and not args.cpu_env_image
    ):
        print("If a CPU or GPU image is specified, both should be.")
        sys.exit(1)

    if args.master_config_template_path:
        if not args.master_config_template_path.exists():
            raise ValueError(
                f"Input master config template doesn't exist: {args.master_config_template_path}"
            )
        with args.master_config_template_path.open("r") as fin:
            det_configs["master_config_template"] = fin.read()

    # Dry-run flag
    if args.dry_run:
        gcp.dry_run(det_configs, env, variables_to_exclude)
        print("Printed plan. To execute, run `det deploy gcp`")
        return

    # If local tf state exists this cluster isn't new anyway, so we can just check the bucket.
    if (
        hasattr(args, "tf_state_gcs_bucket_name")
        and args.tf_state_gcs_bucket_name
        and not gcp.cluster_exists(
            args.tf_state_gcs_bucket_name, det_configs["project_id"], det_configs["cluster_id"]
        )
    ):
        initial_user_password = args.initial_user_password
        if not initial_user_password:
            initial_user_password = getpass.getpass(
                "Please enter an initial password for the built-in `determined` and `admin` users: "
            )
            initial_user_password_check = getpass.getpass("Enter the password again: ")
            if initial_user_password != initial_user_password_check:
                raise ValueError("passwords did not match")
        det_configs["initial_user_password"] = initial_user_password

    print("Starting Determined deployment on GCP...\n")
    gcp.deploy(det_configs, env, variables_to_exclude)

    if not args.no_wait_for_master:
        try:
            gcp.wait_for_master(det_configs, env, timeout=5 * 60)
        except deploy_errors.MasterTimeoutExpired:
            print(
                termcolor.colored(
                    "Determined cluster has been deployed, but master health check has failed.",
                    "red",
                )
            )
            print(
                "For details, SSH to master instance and run "
                "`sudo journalctl -u google-startup-scripts.service`"
                " or check /var/log/cloud-init-output.log."
            )
            sys.exit(1)

    print("Determined Deployment Successful")

    if args.no_wait_for_master:
        print("Please allow 1-5 minutes for the master instance to be accessible via the web-ui\n")


def handle_list(args: argparse.Namespace) -> Any:
    if hasattr(args, "tf_state_gcs_bucket_name") and args.tf_state_gcs_bucket_name:
        bucket_name = args.tf_state_gcs_bucket_name
    else:
        bucket_name = args.project_id + "-determined-deploy"

    if args.json:
        return gcp.list_clusters(bucket_name, args.project_id, "json")
    elif args.yaml:
        return gcp.list_clusters(bucket_name, args.project_id, "yaml")
    else:
        return gcp.list_clusters(bucket_name, args.project_id)


def handle_down(args: argparse.Namespace) -> None:
    return deploy_gcp("down", args)


def handle_up(args: argparse.Namespace) -> None:
    deploy_errors.warn_version_mismatch(args.det_version)
    if args.det_version is None:
        # keep the existing default value behavior of the cli.
        args.det_version = determined.__version__
    return deploy_gcp("up", args)


def handle_dump_master_config_template(args: argparse.Namespace) -> None:
    fn = pkg_resources.resource_filename("determined.deploy.gcp", "terraform/master.yaml.tmpl")
    with open(fn, "r") as fin:
        print(fin.read())


def parse_add_label() -> Callable:
    def parse(s: str) -> Tuple[str, str]:
        try:
            key, value = s.split("=", 1)
        except ValueError:
            raise argparse.ArgumentTypeError("key=value format requires both a key and a value")

        if not key or not value:
            raise argparse.ArgumentTypeError(
                "both key and value must be defined in key=value format"
            )
        return key, value

    return parse


args_description = cli.Cmd(
    "gcp",
    None,
    "GCP help",
    [
        cli.Cmd(
            "list ls",
            handle_list,
            "list gcp cluster",
            [
                cli.Arg(
                    "--project-id",
                    type=str,
                    default=None,
                    required=True,
                    help="project id to list clusters from",
                ),
                cli.Arg(
                    "--tf-state-gcs-bucket-name",
                    type=str,
                    help="use a particular GCS bucket to retreive clusters "
                    "instead of the default GCS bucket",
                ),
                cli.Group(
                    cli.Arg("--json", action="store_true", help="print as CSV"),
                    cli.Arg("--yaml", action="store_true", help="print as JSON"),
                ),
            ],
        ),
        cli.Cmd(
            "down",
            handle_down,
            "delete gcp cluster",
            [
                cli.ArgGroup(
                    "optional named arguments",
                    None,
                    [
                        cli.Arg(
                            "--cluster-id",
                            type=str,
                            help="cluster id of the cluster to be deleted",
                        ),
                        cli.Arg(
                            "--project-id",
                            type=str,
                            help="project id that the cluster belongs to",
                        ),
                        cli.Arg(
                            "--local-state-path",
                            type=str,
                            default=os.getcwd(),
                            help="local directory for storing cluster state",
                        ),
                        cli.Arg(
                            "--yes",
                            action="store_true",
                            help="no prompt when deleting resources",
                        ),
                        cli.Arg(
                            "--no-prompt",
                            dest="yes",
                            action="store_true",
                            help=argparse.SUPPRESS,
                        ),
                    ],
                ),
            ],
        ),
        cli.Cmd(
            "up",
            handle_up,
            "create gcp cluster",
            [
                cli.ArgGroup(
                    "required named arguments",
                    None,
                    [
                        cli.Arg(
                            "--cluster-id",
                            type=validate_cluster_id(),
                            default=None,
                            required=True,
                            help="unique identifier to name and tag resources",
                        ),
                        cli.Arg(
                            "--project-id",
                            type=str,
                            default=None,
                            required=True,
                            help="project id to create the cluster in",
                        ),
                    ],
                ),
                cli.ArgGroup(
                    "optional named arguments",
                    None,
                    [
                        cli.Arg(
                            "--dry-run",
                            action="store_true",
                            help="return the infrastructure plan to be executed "
                            "based on your arguments",
                        ),
                        cli.Arg(
                            "--keypath",
                            type=str,
                            default=None,
                            help="path to service account key if not using default credentials",
                        ),
                        cli.Arg(
                            "--network",
                            type=str,
                            default="det-default",
                            help="network name to create "
                            "(the network should not already exist in the project)",
                        ),
                        cli.Arg(
                            "--filestore-address",
                            type=str,
                            default="",
                            help="the address of an existing Filestore in the format of "
                            "'ip-address:/file-share'; if not provided and the no-filestore "
                            "flag is not set, a new Filestore instance will be created",
                        ),
                        cli.Arg(
                            "--no-filestore",
                            help="whether to create a new Filestore if filestore-address "
                            "parameter is not set",
                            action="store_true",
                        ),
                        cli.Arg(
                            "--det-version",
                            type=str,
                            help=argparse.SUPPRESS,
                        ),
                        cli.Arg(
                            "--region",
                            type=str,
                            default=constants.defaults.REGION,
                            help="region to create the cluster in (defaults to us-west1)",
                        ),
                        cli.Arg(
                            "--zone",
                            type=str,
                            default=None,
                            help="zone to create the cluster in (defaults to `region`-b)",
                        ),
                        cli.Arg(
                            "--disk-size",
                            type=int,
                            default=constants.defaults.BOOT_DISK_SIZE,
                            help="Boot disk size for cluster agents, in GB",
                        ),
                        cli.Arg(
                            "--disk-type",
                            type=str,
                            default=constants.defaults.BOOT_DISK_TYPE,
                            help="Boot disk type for cluster agents",
                        ),
                        cli.Arg(
                            "--environment-image",
                            type=str,
                            default=constants.defaults.ENVIRONMENT_IMAGE,
                            help=argparse.SUPPRESS,
                        ),
                        cli.Arg(
                            "--local-state-path",
                            type=str,
                            default=os.getcwd(),
                            help="local directory for storing cluster state",
                        ),
                        cli.Arg(
                            "--preemptible",
                            type=cli.string_to_bool,
                            default=False,
                            help="whether to use preemptible instances for dynamic agents",
                        ),
                        cli.Arg(
                            "--operation-timeout-period",
                            type=str,
                            default=constants.defaults.OPERATION_TIMEOUT_PERIOD,
                            help="operation timeout before retrying, e.g. 5m for 5 minutes",
                        ),
                        cli.Arg(
                            "--master-instance-type",
                            type=str,
                            default=constants.defaults.MASTER_INSTANCE_TYPE,
                            help="instance type for master",
                        ),
                        cli.Arg(
                            "--compute-agent-instance-type",
                            "--gpu-agent-instance-type",
                            type=str,
                            default=constants.defaults.COMPUTE_AGENT_INSTANCE_TYPE,
                            help="instance type for agents in the compute resource pool",
                        ),
                        cli.Arg(
                            "--aux-agent-instance-type",
                            "--cpu-agent-instance-type",
                            type=str,
                            default=constants.defaults.AUX_AGENT_INSTANCE_TYPE,
                            help="instance type for agents in the auxiliary resource pool",
                        ),
                        cli.Arg(
                            "--db-password",
                            type=str,
                            default=constants.defaults.DB_PASSWORD,
                            help="password for master database",
                        ),
                        cli.Arg(
                            "--max-aux-containers-per-agent",
                            "--max-cpu-containers-per-agent",
                            type=int,
                            default=constants.defaults.MAX_AUX_CONTAINERS_PER_AGENT,
                            help="maximum number of containers on agents in the "
                            "auxiliary resource pool",
                        ),
                        cli.Arg(
                            "--max-idle-agent-period",
                            type=str,
                            default=constants.defaults.MAX_IDLE_AGENT_PERIOD,
                            help="max agent idle time before it is shut down, "
                            "e.g. 30m for 30 minutes",
                        ),
                        cli.Arg(
                            "--max-agent-starting-period",
                            type=str,
                            default=constants.defaults.MAX_AGENT_STARTING_PERIOD,
                            help="max agent starting time before retrying, e.g. 30m for 30 minutes",
                        ),
                        cli.Arg(
                            "--port",
                            type=int,
                            default=constants.defaults.PORT,
                            help="port to use for communication on master instance",
                        ),
                        cli.Arg(
                            "--gpu-type",
                            type=str,
                            default=constants.defaults.GPU_TYPE,
                            help="type of GPU to use on agents",
                        ),
                        cli.Arg(
                            "--gpu-num",
                            type=int,
                            default=constants.defaults.GPU_NUM,
                            help="number of GPUs per agent instance",
                        ),
                        cli.Arg(
                            "--min-dynamic-agents",
                            type=int,
                            default=constants.defaults.MIN_DYNAMIC_AGENTS,
                            help="minimum number of dynamic agent instances at one time",
                        ),
                        cli.Arg(
                            "--max-dynamic-agents",
                            type=int,
                            default=constants.defaults.MAX_DYNAMIC_AGENTS,
                            help="maximum number of dynamic agent instances at one time",
                        ),
                        cli.Arg(
                            "--min-cpu-platform-master",
                            type=str,
                            default=constants.defaults.MIN_CPU_PLATFORM_MASTER,
                            help="minimum cpu platform for master instances",
                        ),
                        cli.Arg(
                            "--min-cpu-platform-agent",
                            type=str,
                            default=constants.defaults.MIN_CPU_PLATFORM_AGENT,
                            help="minimum cpu platform for agent instances",
                        ),
                        cli.Arg(
                            "--scheduler-type",
                            type=str,
                            choices=["fair_share", "priority", "round_robin"],
                            default=constants.defaults.SCHEDULER_TYPE,
                            help="scheduler to use",
                        ),
                        cli.Arg(
                            "--preemption-enabled",
                            type=cli.string_to_bool,
                            default=constants.defaults.PREEMPTION_ENABLED,
                            help="whether task preemption is supported in the scheduler "
                            "(only configurable for priority scheduler).",
                        ),
                        cli.Arg(
                            "--cpu-env-image",
                            type=str,
                            default="",
                            help="Docker image for CPU tasks",
                        ),
                        cli.Arg(
                            "--gpu-env-image",
                            type=str,
                            default="",
                            help="Docker image for GPU tasks",
                        ),
                        cli.Arg(
                            "--master-config-template-path",
                            type=pathlib.Path,
                            default=None,
                            help="path to master yaml template",
                        ),
                        cli.Arg(
                            "--tf-state-gcs-bucket-name",
                            type=str,
                            default=None,
                            help="use the GCS bucket to store the terraform state "
                            "instead of a local directory",
                        ),
                        cli.Arg(
                            "--add-label",
                            type=parse_add_label(),
                            action="append",
                            default=None,
                            help="apply label to master instance in key=value format, "
                            "can be repeated",
                        ),
                        cli.Arg(
                            "--initial-user-password",
                            type=str,
                            help="Password for the default 'determined' and 'admin' users.",
                        ),
                    ],
                ),
            ],
        ),
        cli.Cmd(
            "dump-master-config-template",
            handle_dump_master_config_template,
            "dump default master config template",
            [],
        ),
    ],
)
