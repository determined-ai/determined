import argparse
import socket
import sys
from pathlib import Path
from typing import Callable, Dict

from determined.common.declarative_argparse import Arg, BoolOptArg, Cmd, Group

from . import cluster_utils
from .preflight import check_docker_install

AGENT_NAME_DEFAULT = "det-agent-<hostname>"


def handle_cluster_up(args: argparse.Namespace) -> None:
    if not args.no_preflight_checks:
        check_docker_install()

    cluster_utils.cluster_up(
        num_agents=args.agents,
        port=args.master_port,
        master_config_path=args.master_config_path,
        storage_host_path=args.storage_host_path,
        cluster_name=args.cluster_name,
        image_repo_prefix=args.image_repo_prefix,
        version=args.det_version,
        db_password=args.db_password,
        delete_db=args.delete_db,
        gpu=args.gpu,
        autorestart=(not args.no_autorestart),
        auto_work_dir=args.auto_work_dir,
    )


def handle_cluster_down(args: argparse.Namespace) -> None:
    cluster_utils.cluster_down(cluster_name=args.cluster_name, delete_db=args.delete_db)


def handle_logs(args: argparse.Namespace) -> None:
    cluster_utils.logs(cluster_name=args.cluster_name, no_follow=args.no_follow)


def handle_master_up(args: argparse.Namespace) -> None:
    cluster_utils.master_up(
        port=args.master_port,
        master_config_path=args.master_config_path,
        storage_host_path=args.storage_host_path,
        master_name=args.master_name,
        image_repo_prefix=args.image_repo_prefix,
        version=args.det_version,
        db_password=args.db_password,
        delete_db=args.delete_db,
        autorestart=(not args.no_autorestart),
        cluster_name=args.cluster_name,
        auto_work_dir=args.auto_work_dir,
    )


def handle_master_down(args: argparse.Namespace) -> None:
    cluster_utils.master_down(master_name=args.master_name, delete_db=args.delete_db)


def handle_agent_up(args: argparse.Namespace) -> None:

    agent_name = args.agent_name
    if args.agent_name == AGENT_NAME_DEFAULT:
        agent_name = f"det-agent-{socket.gethostname()}"

    cluster_utils.agent_up(
        master_host=args.master_host,
        master_port=args.master_port,
        gpu=args.gpu,
        agent_name=agent_name,
        agent_label=args.agent_label,
        agent_resource_pool=args.agent_resource_pool,
        image_repo_prefix=args.image_repo_prefix,
        version=args.det_version,
        labels=None,
        autorestart=(not args.no_autorestart),
        cluster_name=args.cluster_name,
    )


def handle_agent_down(args: argparse.Namespace) -> None:

    agent_name = args.agent_name
    if args.agent_name == AGENT_NAME_DEFAULT:
        agent_name = f"det-agent-{socket.gethostname()}"

    if args.all:
        cluster_utils.stop_all_agents()
    else:
        cluster_utils.stop_agent(agent_name=agent_name)


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


args_description = Cmd(
    "local",
    None,
    "local help",
    [
        Cmd(
            "cluster-up",
            handle_cluster_up,
            "Create a Determined cluster",
            [
                Group(
                    Arg(
                        "--master-config-path",
                        type=Path,
                        default=None,
                        help="path to master configuration",
                    ),
                    Arg(
                        "--storage-host-path",
                        type=Path,
                        default=None,
                        help="Storage location for cluster data (e.g. checkpoints)",
                    ),
                ),
                Arg(
                    "--agents",
                    type=int,
                    default=1,
                    help="number of agents to start (on this machine)",
                ),
                Arg("--master-port", type=int, default=8080, help="port to expose master on"),
                Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                Arg("--det-version", type=str, default=None, help="version or commit to use"),
                Arg(
                    "--db-password",
                    type=str,
                    default="postgres",
                    help="password for master database",
                ),
                Arg(
                    "--delete-db",
                    action="store_true",
                    help="remove current master database",
                ),
                BoolOptArg(
                    "--gpu",
                    "--no-gpu",
                    dest="gpu",
                    default=("darwin" not in sys.platform),
                    true_help="enable GPU support for agent",
                    false_help="disable GPU support for agent",
                ),
                Arg(
                    "--no-autorestart",
                    help="disable container auto-restart (recommended for local development)",
                    action="store_true",
                ),
                Arg(
                    "--auto-work-dir",
                    type=Path,
                    default=None,
                    help="the default work dir, used for interactive jobs",
                ),
            ],
        ),
        Cmd(
            "cluster-down",
            handle_cluster_down,
            "Stop a Determined cluster",
            [
                Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                Arg(
                    "--delete-db",
                    action="store_true",
                    help="remove current master database",
                ),
            ],
        ),
        Cmd(
            "master-up",
            handle_master_up,
            "Start a Determined master",
            [
                Group(
                    Arg(
                        "--master-config-path",
                        type=Path,
                        default=None,
                        help="path to master configuration",
                    ),
                    Arg(
                        "--storage-host-path",
                        type=Path,
                        default=None,
                        help="Storage location for cluster data (e.g. checkpoints)",
                    ),
                ),
                Arg("--master-port", type=int, default=8080, help="port to expose master on"),
                Arg(
                    "--master-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                Arg("--det-version", type=str, default=None, help="version or commit to use"),
                Arg(
                    "--db-password",
                    type=str,
                    default="postgres",
                    help="password for master database",
                ),
                Arg(
                    "--delete-db",
                    action="store_true",
                    help="remove current master database",
                ),
                Arg(
                    "--no-autorestart",
                    help="disable container auto-restart (recommended for local development)",
                    action="store_true",
                ),
                Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                Arg(
                    "--auto-work-dir",
                    type=Path,
                    default=None,
                    help="the default work dir, used for interactive jobs",
                ),
            ],
        ),
        Cmd(
            "master-down",
            handle_master_down,
            "Stop a Determined master",
            [
                Arg(
                    "--master-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                Arg(
                    "--delete-db",
                    action="store_true",
                    help="remove current master database",
                ),
                Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
            ],
        ),
        Cmd(
            "logs",
            handle_logs,
            "Show the logs of a Determined cluster",
            [
                Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                Arg("--no-follow", help="disable following logs", action="store_true"),
            ],
        ),
        Cmd(
            "agent-up",
            handle_agent_up,
            "Start a Determined agent",
            [
                Arg("master_host", type=str, help="master hostname"),
                Arg("--master-port", type=int, default=8080, help="master port"),
                Arg("--det-version", type=str, default=None, help="version or commit to use"),
                Arg("--agent-name", type=str, default=AGENT_NAME_DEFAULT, help="agent name"),
                Arg("--agent-label", type=str, default=None, help="agent label"),
                Arg("--agent-resource-pool", type=str, default=None, help="agent resource pool"),
                BoolOptArg(
                    "--gpu",
                    "--no-gpu",
                    dest="gpu",
                    default=("darwin" not in sys.platform),
                    true_help="enable GPU support for agent",
                    false_help="disable GPU support for agent",
                ),
                Arg(
                    "--no-autorestart",
                    help="disable container auto-restart (recommended for local development)",
                    action="store_true",
                ),
                Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
            ],
        ),
        Cmd(
            "agent-down",
            handle_agent_down,
            "Stop a Determined agent",
            [
                Arg("--agent-name", type=str, default=AGENT_NAME_DEFAULT, help="agent name"),
                Arg("--all", help="stop all running agents", action="store_true"),
                Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
            ],
        ),
    ],
)
