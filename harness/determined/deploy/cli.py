from determined.cli.top_arg_descriptions import deploy_cmd
from determined.common.declarative_argparse import Arg, ArgsDescription

from .aws.cli import args_description as aws_args_description
from .gcp.cli import args_description as gcp_args_description
from .gke.cli import args_description as gke_args_description
from .local.cli import args_description as local_args_description

args_subs: ArgsDescription = [
    Arg("--no-preflight-checks", action="store_true", help="Disable preflight checks"),
    Arg(
        "--no-wait-for-master",
        action="store_true",
        help="Do not wait for master to come up after AWS or GCP clusters are deployed",
    ),
    Arg(
        "--image-repo-prefix",
        type=str,
        default="determinedai",
        help="Docker image repository to use for determined-master and determined-agent images",
    ),
    local_args_description,
    aws_args_description,
    gcp_args_description,
    gke_args_description,
]

deploy_cmd.subs = args_subs
args_description = deploy_cmd
