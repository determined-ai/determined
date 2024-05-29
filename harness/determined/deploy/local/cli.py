import argparse
import pathlib
import sys
from typing import Callable, Dict

import determined
from determined import cli
from determined.deploy import errors
from determined.deploy.local import cluster_utils, preflight


def handle_cluster_up(args: argparse.Namespace) -> None:
    if not args.no_preflight_checks:
        preflight.check_docker_install()

    errors.warn_version_mismatch(args.det_version)
    if args.det_version is None:
        args.det_version = determined.__version__

    cluster_utils.cluster_up(
        num_agents=args.agents,
        port=args.master_port,
        initial_user_password=args.initial_user_password,
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
        enterprise_edition=args.enterprise_edition,
    )


def handle_cluster_down(args: argparse.Namespace) -> None:
    cluster_utils.cluster_down(cluster_name=args.cluster_name, delete_db=args.delete_db)


def handle_logs(args: argparse.Namespace) -> None:
    cluster_utils.logs(cluster_name=args.cluster_name, follow=not args.no_follow)


def handle_master_up(args: argparse.Namespace) -> None:
    errors.warn_version_mismatch(args.det_version)
    if args.det_version is None:
        args.det_version = determined.__version__

    cluster_utils.master_up(
        port=args.master_port,
        initial_user_password=args.initial_user_password,
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
        enterprise_edition=args.enterprise_edition,
    )


def handle_master_down(args: argparse.Namespace) -> None:
    cluster_utils.master_down(
        master_name=args.master_name, delete_db=args.delete_db, cluster_name=args.cluster_name
    )


def handle_agent_up(args: argparse.Namespace) -> None:
    errors.warn_version_mismatch(args.det_version)
    if args.det_version is None:
        args.det_version = determined.__version__
    cluster_utils.agent_up(
        master_host=args.master_host,
        master_port=args.master_port,
        agent_config_path=args.agent_config_path,
        gpu=args.gpu,
        agent_name=args.agent_name,
        agent_resource_pool=args.agent_resource_pool,
        image_repo_prefix=args.image_repo_prefix,
        version=args.det_version,
        labels=None,
        autorestart=(not args.no_autorestart),
        cluster_name=args.cluster_name,
        enterprise_edition=args.enterprise_edition,
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


args_description = cli.Cmd(
    "local",
    None,
    "local help",
    [
        cli.Cmd(
            "cluster-up",
            handle_cluster_up,
            "Create a Determined cluster",
            [
                cli.Group(
                    cli.Arg(
                        "--master-config-path",
                        type=pathlib.Path,
                        default=None,
                        help="path to master configuration",
                    ),
                    cli.Arg(
                        "--storage-host-path",
                        type=pathlib.Path,
                        default=None,
                        help="Storage location for cluster data (e.g. checkpoints)",
                    ),
                ),
                cli.Arg(
                    "--initial-user-password",
                    type=str,
                    default=None,
                    help="Initial password for admin/determined users",
                ),
                cli.Arg(
                    "--agents",
                    type=int,
                    default=1,
                    help=argparse.SUPPRESS,
                ),
                cli.Arg(
                    "--master-port",
                    type=int,
                    default=cluster_utils.MASTER_PORT_DEFAULT,
                    help="port to expose master on",
                ),
                cli.Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                cli.Arg(
                    "--det-version",
                    type=str,
                    help="version or commit to use",
                ),
                cli.Arg(
                    "--db-password",
                    type=str,
                    default="postgres",
                    help="password for master database",
                ),
                cli.Arg(
                    "--delete-db",
                    action="store_true",
                    help="remove current master database",
                ),
                cli.BoolOptArg(
                    "--gpu",
                    "--no-gpu",
                    dest="gpu",
                    default=("darwin" not in sys.platform),
                    true_help="enable GPU support for agent",
                    false_help="disable GPU support for agent",
                ),
                cli.Arg(
                    "--no-autorestart",
                    help="disable container auto-restart (recommended for local development)",
                    action="store_true",
                ),
                cli.Arg(
                    "--auto-work-dir",
                    type=pathlib.Path,
                    default=None,
                    help="the default work dir, used for interactive jobs",
                ),
                cli.Arg(
                    "--enterprise-edition",
                    action="store_true",
                    help="Deploy the enterprise edition of Determined",
                ),
            ],
        ),
        cli.Cmd(
            "cluster-down",
            handle_cluster_down,
            "Stop a Determined cluster",
            [
                cli.Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                cli.Arg(
                    "--delete-db",
                    action="store_true",
                    help="remove current master database",
                ),
            ],
        ),
        cli.Cmd(
            "master-up",
            handle_master_up,
            "Start a Determined master",
            [
                cli.Group(
                    cli.Arg(
                        "--master-config-path",
                        type=pathlib.Path,
                        default=None,
                        help="path to master configuration",
                    ),
                    cli.Arg(
                        "--storage-host-path",
                        type=pathlib.Path,
                        default=None,
                        help="Storage location for cluster data (e.g. checkpoints)",
                    ),
                ),
                cli.Arg(
                    "--initial-user-password",
                    type=str,
                    default=None,
                    help="Initial password for admin/determined users",
                ),
                cli.Arg(
                    "--master-port",
                    type=int,
                    default=cluster_utils.MASTER_PORT_DEFAULT,
                    help="port to expose master on",
                ),
                cli.Arg(
                    "--master-name",
                    type=str,
                    default=None,
                    help="name for the master instance",
                ),
                cli.Arg(
                    "--det-version",
                    type=str,
                    help="version or commit to use",
                ),
                cli.Arg(
                    "--db-password",
                    type=str,
                    default="postgres",
                    help="password for master database",
                ),
                cli.Arg(
                    "--delete-db",
                    action="store_true",
                    help="remove current master database",
                ),
                cli.Arg(
                    "--no-autorestart",
                    help="disable container auto-restart (recommended for local development)",
                    action="store_true",
                ),
                cli.Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                cli.Arg(
                    "--image-repo-prefix",
                    type=str,
                    default="determinedai",
                    help="prefix for the master image",
                ),
                cli.Arg(
                    "--auto-work-dir",
                    type=pathlib.Path,
                    default=None,
                    help="the default work dir, used for interactive jobs",
                ),
                cli.Arg(
                    "--enterprise-edition",
                    action="store_true",
                    help="Deploy the enterprise edition of Determined",
                ),
            ],
        ),
        cli.Cmd(
            "master-down",
            handle_master_down,
            "Stop a Determined master",
            [
                cli.Arg(
                    "--master-name",
                    type=str,
                    default=None,
                    help="name for the master instance",
                ),
                cli.Arg(
                    "--delete-db",
                    action="store_true",
                    help="remove current master database",
                ),
                cli.Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
            ],
        ),
        cli.Cmd(
            "logs",
            handle_logs,
            "Show the logs of a Determined cluster",
            [
                cli.Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                cli.Arg("--no-follow", help="disable following logs", action="store_true"),
            ],
        ),
        cli.Cmd(
            "agent-up",
            handle_agent_up,
            "Start a Determined agent",
            [
                cli.Arg("master_host", type=str, help="master hostname"),
                cli.Arg(
                    "--master-port",
                    type=int,
                    default=cluster_utils.MASTER_PORT_DEFAULT,
                    help="master port",
                ),
                cli.Arg(
                    "--agent-config-path",
                    type=pathlib.Path,
                    default=None,
                    help="path to agent configuration",
                ),
                cli.Arg(
                    "--det-version",
                    type=str,
                    help="version or commit to use",
                ),
                cli.Arg(
                    "--agent-name",
                    type=str,
                    default=cluster_utils.AGENT_NAME_DEFAULT,
                    help="agent name",
                ),
                cli.Arg(
                    "--agent-resource-pool", type=str, default=None, help="agent resource pool"
                ),
                cli.BoolOptArg(
                    "--gpu",
                    "--no-gpu",
                    dest="gpu",
                    default=("darwin" not in sys.platform),
                    true_help="enable GPU support for agent",
                    false_help="disable GPU support for agent",
                ),
                cli.Arg(
                    "--no-autorestart",
                    help="disable container auto-restart (recommended for local development)",
                    action="store_true",
                ),
                cli.Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
                cli.Arg(
                    "--image-repo-prefix",
                    type=str,
                    default="determinedai",
                    help="prefix for the master image",
                ),
                cli.Arg(
                    "--enterprise-edition",
                    action="store_true",
                    help="Deploy the enterprise edition of Determined",
                ),
            ],
        ),
        cli.Cmd(
            "agent-down",
            handle_agent_down,
            "Stop a Determined agent",
            [
                cli.Arg(
                    "--agent-name",
                    type=str,
                    default=cluster_utils.AGENT_NAME_DEFAULT,
                    help="agent name",
                ),
                cli.Arg("--all", help="stop all running agents", action="store_true"),
                cli.Arg(
                    "--cluster-name",
                    type=str,
                    default="determined",
                    help="name for the cluster resources",
                ),
            ],
        ),
    ],
)
