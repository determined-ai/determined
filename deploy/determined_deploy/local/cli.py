import argparse
from typing import Callable, Dict

from determined_deploy.local import cluster_utils


def add_cluster_up_subparser(subparsers: argparse._SubParsersAction) -> None:
    parser = subparsers.add_parser(
        "cluster-up",
        help="Create a Determined cluster",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--master-config-path", type=str, default=None, help="path to master configuration"
    )
    parser.add_argument(
        "--agents", type=int, default=1, help="number of agents to start (on this machine)"
    )
    parser.add_argument("--master-port", type=int, default=8080, help="port to expose master on")
    parser.add_argument(
        "--cluster-name", type=str, default="determined", help="name for the cluster resources"
    )
    parser.add_argument("--det-version", type=str, default=None, help="version or commit to use")
    parser.add_argument(
        "--db-password",
        type=str,
        default="postgres",
        help="password for master database",
    )
    parser.add_argument(
        "--delete-db",
        action="store_true",
        help="remove current master database",
    )
    parser.add_argument("--no-gpu", help="disable GPU support for agent", action="store_true")
    parser.add_argument(
        "--no-autorestart",
        help="disable container auto-restart (recommended for local development)",
        action="store_true",
    )


def add_cluster_down_subparser(subparsers: argparse._SubParsersAction) -> None:
    parser = subparsers.add_parser(
        "cluster-down",
        help="Stop a Determined cluster",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--cluster-name", type=str, default="determined", help="name for the cluster resources"
    )
    parser.add_argument(
        "--delete-db",
        action="store_true",
        help="remove current master database",
    )


def add_master_up_subparser(subparsers: argparse._SubParsersAction) -> None:
    parser = subparsers.add_parser(
        "master-up",
        help="Start a Determined master",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--master-config-path", type=str, default=None, help="path to master configuration"
    )
    parser.add_argument("--master-port", type=int, default=8080, help="port to expose master on")
    parser.add_argument(
        "--master-name", type=str, default="determined", help="name for the cluster resources"
    )
    parser.add_argument("--det-version", type=str, default=None, help="version or commit to use")
    parser.add_argument(
        "--db-password",
        type=str,
        default="postgres",
        help="password for master database",
    )
    parser.add_argument(
        "--delete-db",
        action="store_true",
        help="remove current master database",
    )
    parser.add_argument(
        "--no-autorestart",
        help="disable container auto-restart (recommended for local development)",
        action="store_true",
    )
    parser.add_argument(
        "--cluster-name", type=str, default="determined", help="name for the cluster resources"
    )


def add_master_down_subparser(subparsers: argparse._SubParsersAction) -> None:
    parser = subparsers.add_parser(
        "master-down",
        help="Stop a Determined master",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--master-name", type=str, default="determined", help="name for the cluster resources"
    )
    parser.add_argument(
        "--delete-db",
        action="store_true",
        help="remove current master database",
    )
    parser.add_argument(
        "--cluster-name", type=str, default="determined", help="name for the cluster resources"
    )


def add_logs_subparser(subparsers: argparse._SubParsersAction) -> None:
    parser = subparsers.add_parser(
        "logs",
        help="Show the logs of a Determined cluster",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--cluster-name", type=str, default="determined", help="name for the cluster resources"
    )
    parser.add_argument("--no-follow", help="disable following logs", action="store_true")


def add_agent_up_subparser(subparsers: argparse._SubParsersAction) -> None:
    parser = subparsers.add_parser(
        "agent-up",
        help="Start a Determined agent",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument("master_host", type=str, help="master hostname")
    parser.add_argument("--master-port", type=int, default=8080, help="master port")
    parser.add_argument("--det-version", type=str, default=None, help="version or commit to use")
    parser.add_argument("--agent-name", type=str, default="det-agent", help="agent name")
    parser.add_argument("--agent-label", type=str, default=None, help="agent label")
    parser.add_argument("--agent-resource-pool", type=str, default=None, help="agent resource pool")
    parser.add_argument("--no-gpu", help="disable GPU support", action="store_true")
    parser.add_argument(
        "--no-autorestart",
        help="disable container auto-restart (recommended for local development)",
        action="store_true",
    )
    parser.add_argument(
        "--cluster-name", type=str, default="determined", help="name for the cluster resources"
    )


def add_agent_down_subparser(subparsers: argparse._SubParsersAction) -> None:
    parser = subparsers.add_parser(
        "agent-down",
        help="Stop a Determined agent",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument("--agent-name", type=str, default="det-agent", help="agent name")
    parser.add_argument("--all", help="stop all running agents", action="store_true")
    parser.add_argument(
        "--cluster-name", type=str, default="determined", help="name for the cluster resources"
    )


def make_local_parser(subparsers: argparse._SubParsersAction) -> None:
    parser_local = subparsers.add_parser(
        "local", help="local help", formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    subparsers = parser_local.add_subparsers(help="command", dest="command")
    add_cluster_up_subparser(subparsers)
    add_cluster_down_subparser(subparsers)
    add_logs_subparser(subparsers)
    add_master_up_subparser(subparsers)
    add_master_down_subparser(subparsers)
    add_agent_up_subparser(subparsers)
    add_agent_down_subparser(subparsers)
    subparsers.required = True


def handle_cluster_up(args: argparse.Namespace) -> None:
    cluster_utils.cluster_up(
        num_agents=args.agents,
        port=args.master_port,
        master_config_path=args.master_config_path,
        cluster_name=args.cluster_name,
        version=args.det_version,
        db_password=args.db_password,
        delete_db=args.delete_db,
        no_gpu=args.no_gpu,
        autorestart=(not args.no_autorestart),
    )


def handle_cluster_down(args: argparse.Namespace) -> None:
    cluster_utils.cluster_down(cluster_name=args.cluster_name, delete_db=args.delete_db)


def handle_logs(args: argparse.Namespace) -> None:
    cluster_utils.logs(cluster_name=args.cluster_name, no_follow=args.no_follow)


def handle_master_up(args: argparse.Namespace) -> None:
    cluster_utils.master_up(
        port=args.master_port,
        master_config_path=args.master_config_path,
        master_name=args.master_name,
        version=args.det_version,
        db_password=args.db_password,
        delete_db=args.delete_db,
        autorestart=(not args.no_autorestart),
        cluster_name=args.cluster_name,
    )


def handle_master_down(args: argparse.Namespace) -> None:
    cluster_utils.master_down(master_name=args.master_name, delete_db=args.delete_db)


def handle_agent_up(args: argparse.Namespace) -> None:
    cluster_utils.agent_up(
        master_host=args.master_host,
        master_port=args.master_port,
        no_gpu=args.no_gpu,
        agent_name=args.agent_name,
        agent_label=args.agent_label,
        agent_resource_pool=args.agent_resource_pool,
        version=args.det_version,
        labels=None,
        autorestart=(not args.no_autorestart),
        cluster_name=args.cluster_name,
    )


def handle_agent_down(args: argparse.Namespace) -> None:
    if args.all:
        cluster_utils.stop_all_agents()
    else:
        cluster_utils.stop_agent(agent_name=args.agent_name)


def deploy_local(args: argparse.Namespace) -> None:
    OPERATION_TO_FN = {
        "agent-up": handle_agent_up,
        "agent-down": handle_agent_down,
        "cluster-up": handle_cluster_up,
        "cluster-down": handle_cluster_down,
        "logs": handle_logs,
        "master-up": handle_master_up,
        "master-down": handle_master_down,
    }  # type: Dict[str, Callable[[argparse.Namespace], None]]
    OPERATION_TO_FN[args.command](args)
