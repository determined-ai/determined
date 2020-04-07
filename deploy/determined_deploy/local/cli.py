import argparse
import pathlib

from determined_deploy.local import cluster_utils

OPERATION_TO_FN = {
    "fixture-up": cluster_utils.fixture_up,
    "fixture-down": cluster_utils.fixture_down,
    "logs": cluster_utils.logs,
}


def make_local_parser(subparsers: argparse._SubParsersAction) -> None:
    parser_local = subparsers.add_parser(
        "local", help="local help", formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    parser_local.add_argument(
        "operations",
        type=str,
        choices=OPERATION_TO_FN.keys(),
        nargs="+",
        help="a list of operations",
    )
    default_etc_path = pathlib.Path(__file__).parent.joinpath("configuration/")
    parser_local.add_argument(
        "--etc-root", type=str, default=default_etc_path, help="path to etc directory"
    )
    parser_local.add_argument(
        "--agents", type=int, default=0, help="number of agents to start (on this machine)"
    )
    parser_local.add_argument(
        "--master-port", type=int, default=8080, help="port to expose master on"
    )
    parser_local.add_argument(
        "--cluster-name", type=str, default="determined", help="name for the cluster resources"
    )
    parser_local.add_argument(
        "--db-password", type=str, default="postgres", help="password for master database",
    )
    parser_local.add_argument(
        "--hasura-secret", type=str, default="hasura", help="password for hasura service",
    )


def deploy_local(args: argparse.Namespace) -> None:
    for op in args.operations:
        fn = OPERATION_TO_FN.get(op)
        if fn is cluster_utils.fixture_up:
            fn(
                num_agents=args.agents,
                etc_path=args.etc_root,
                port=args.master_port,
                cluster_name=args.cluster_name,
                db_password="postgres",
                hasura_secret="hasura",
            )
        else:
            fn(cluster_name=args.cluster_name)
