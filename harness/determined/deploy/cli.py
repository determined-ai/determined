import argparse
import warnings
from typing import List, Union

from determined import __version__
from determined.cli.top_arg_descriptions import deploy_cmd
from determined.common.declarative_argparse import Arg, Cmd, add_args

from .aws.cli import args_description as aws_args_description
from .gcp.cli import args_description as gcp_args_description
from .gke.cli import args_description as gke_args_description
from .local.cli import args_description as local_args_description

args_subs: List[Union[Arg, Cmd]] = [
    # TODO(DET-5171): Remove --version flag when det-deploy is deprecated.
    Arg("--version", action="version", version="%(prog)s {}".format(__version__)),
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


def main() -> None:
    """Deprecated entry point for standalone `det-deploy`."""
    parser = argparse.ArgumentParser(
        description="Manage Determined deployments.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    add_args(parser, args_subs)
    parsed_args = parser.parse_args()

    v = vars(parsed_args)
    if not v.get("func"):
        parser.print_usage()
        parser.exit(2, "{}: no subcommand specified\n".format(parser.prog))

    warnings.warn(
        "`det-deploy` executable is deprecated, please use `det deploy` instead.", FutureWarning
    )

    parsed_args.func(parsed_args)


if __name__ == "__main__":
    main()
