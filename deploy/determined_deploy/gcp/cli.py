import argparse
import json
import os

import determined_deploy
from determined_deploy.gcp import constants, gcp


def make_gcp_parser(subparsers: argparse._SubParsersAction) -> None:
    parser_aws = subparsers.add_parser("gcp", help="gcp help")
    parser_aws.add_argument("--delete", action="store_true", help="delete Determined from account")
    parser_aws.add_argument(
        "--plan",
        action="store_true",
        help="return the Terraform infrastructure plan for the current configuration",
    )
    parser_aws.add_argument(
        "--keypath",
        type=str,
        default=None,
        required=True,
        help="Path to service account key, or `gcloud` if using default credentials",
    )
    parser_aws.add_argument(
        "--identifier",
        type=str,
        default=None,
        required=True,
        help="unique identifier to name and tag resources",
    )
    parser_aws.add_argument(
        "--network",
        type=str,
        default=None,
        required=True,
        help="network name to use (the network will be created if it doesn't exist)",
    )
    parser_aws.add_argument(
        "--project_id",
        type=str,
        default=None,
        required=True,
        help="project ID to create the cluster in",
    )
    parser_aws.add_argument(
        "--det_version",
        type=str,
        default=determined_deploy.__version__,
        help="Determined version or commit to deploy",
    )
    parser_aws.add_argument(
        "--region",
        type=str,
        default=constants.defaults.REGION,
        help="region to create the cluster in (defaults to us-central1)",
    )
    parser_aws.add_argument(
        "--zone",
        type=str,
        default=constants.defaults.ZONE,
        help="zone to create the cluster in (defaults to us-central1-a)",
    )
    parser_aws.add_argument(
        "--environment_image",
        type=str,
        default=constants.defaults.ENVIRONMENT_IMAGE,
        help="base environment image for agents",
    )
    parser_aws.add_argument(
        "--local_state_path",
        type=str,
        default=os.getcwd(),
        help="base path to store the .tfstate file; defaults to the current directory",
    )
    parser_aws.add_argument(
        "--preemptible",
        type=str,
        default="false",
        help="whether to use preemptible instances for agents",
    )
    parser_aws.add_argument(
        "--master_instance_type",
        type=str,
        default=constants.defaults.MASTER_INSTANCE_TYPE,
        help="instance type for master",
    )
    parser_aws.add_argument(
        "--agent_instance_type",
        type=str,
        default=constants.defaults.AGENT_INSTANCE_TYPE,
        help="instance type for agent",
    )
    parser_aws.add_argument(
        "--db_password",
        type=str,
        default=constants.defaults.DB_PASSWORD,
        help="password for master database",
    )
    parser_aws.add_argument(
        "--hasura_secret",
        type=str,
        default=constants.defaults.HASURA_SECRET,
        help="password for hasura service",
    )
    parser_aws.add_argument(
        "--max_idle_agent_period",
        type=str,
        default=constants.defaults.MAX_IDLE_AGENT_PERIOD,
        help="max agent idle time before it is shut down, e.g. 30m for 30 minutes",
    )
    parser_aws.add_argument(
        "--port",
        type=str,
        default=constants.defaults.PORT,
        help="port to use for communication on master instance",
    )
    parser_aws.add_argument(
        "--gpu_type", type=str, default=constants.defaults.GPU_TYPE, help="max instances",
    )
    parser_aws.add_argument(
        "--gpu_num",
        type=int,
        default=constants.defaults.GPU_NUM,
        help="number of GPUs per agent instance",
    )
    parser_aws.add_argument(
        "--max_instances",
        type=int,
        default=constants.defaults.MAX_INSTANCES,
        help="Maximum number of agent instances at one time",
    )


def deploy_gcp(args: argparse.Namespace) -> None:

    det_configs = {}

    # Get project_id from keyfile or args
    if args.keypath != "gcloud":
        with open(args.keypath) as keyfile:
            keyfile_dict = json.load(keyfile)
            project_id_from_key = keyfile_dict["project_id"]

        # Set det_configs based on arguments
        det_configs["project_id"] = project_id_from_key

    # Add args to det_configs dict
    for arg in vars(args):
        if vars(args)[arg] is not None:
            det_configs[arg] = vars(args)[arg]

    # Set the TF_DATA_DIR where Terraform will store its supporting files
    env = os.environ.copy()
    env["TF_DATA_DIR"] = os.path.join(args.local_state_path, "terraform_data")

    # Not all args will be passed to Terraform, list the ones that won't be
    variables_to_exclude = [
        "delete",
        "plan",
        "environment",
        "local_state_path",
    ]

    if args.delete:
        gcp.delete(det_configs, env, variables_to_exclude)
        print("Delete Successful")
        return

    if args.plan:
        gcp.plan(det_configs, env, variables_to_exclude)
        return

    print("Starting Determined Deployment")
    gcp.deploy(det_configs, env, variables_to_exclude)
    print("Determined Deployment Successful")
