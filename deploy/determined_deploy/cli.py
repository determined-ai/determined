import argparse
import sys

import determined_deploy.aws.cli
import determined_deploy.local.cli


def main() -> None:
    environment_map = {
        "aws": determined_deploy.aws.cli.deploy_aws,
        "local": determined_deploy.local.cli.deploy_local,
    }

    parser = argparse.ArgumentParser(
        description="Manage Determined deployments.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    subparsers = parser.add_subparsers(help="environment", dest="environment")

    determined_deploy.local.cli.make_local_parser(subparsers)
    determined_deploy.aws.cli.make_aws_parser(subparsers)

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
