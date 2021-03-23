import argparse
import sys

import determined
import determined.deploy.aws.cli
import determined.deploy.gcp.cli
import determined.deploy.local.cli


def main() -> None:
    environment_map = {
        "aws": determined.deploy.aws.cli.deploy_aws,
        "gcp": determined.deploy.gcp.cli.deploy_gcp,
        "local": determined.deploy.local.cli.deploy_local,
    }

    parser = argparse.ArgumentParser(
        description="Manage Determined deployments.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--version", action="version", version="%(prog)s {}".format(determined.__version__)
    )
    subparsers = parser.add_subparsers(help="environment", dest="environment")

    determined.deploy.local.cli.make_local_parser(subparsers)
    determined.deploy.aws.cli.make_aws_parser(subparsers)
    determined.deploy.gcp.cli.make_gcp_parser(subparsers)

    args = parser.parse_args()
    environment = args.environment
    if environment:
        environment_map[environment](args)
    else:
        print("environment is required")
        parser.print_help()
        sys.exit(1)


if __name__ == "__main__":
    main()
