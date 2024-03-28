from determined import cli
from determined.cli import top_arg_descriptions
from determined.deploy.aws import cli as aws_cli
from determined.deploy.gcp import cli as gcp_cli
from determined.deploy.gke import cli as gke_cli
from determined.deploy.local import cli as local_cli

args_subs: cli.ArgsDescription = [
    cli.Arg("--no-preflight-checks", action="store_true", help="Disable preflight checks"),
    cli.Arg(
        "--no-wait-for-master",
        action="store_true",
        help="Do not wait for master to come up after AWS or GCP clusters are deployed",
    ),
    cli.Arg(
        "--image-repo-prefix",
        type=str,
        default="determinedai",
        help="Docker image repository to use for determined-master and determined-agent images",
    ),
    local_cli.args_description,
    aws_cli.args_description,
    gcp_cli.args_description,
    gke_cli.args_description,
]

top_arg_descriptions.deploy_cmd.subs = args_subs
args_description = top_arg_descriptions.deploy_cmd
