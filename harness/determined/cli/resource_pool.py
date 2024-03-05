import argparse
from typing import Any, List

from determined import cli
from determined.cli import render
from determined.common.api import bindings


def add_binding(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    body = bindings.v1BindRPToWorkspaceRequest(
        resourcePoolName=args.pool_name, workspaceNames=args.workspace_names
    )
    bindings.post_BindRPToWorkspace(sess, body=body, resourcePoolName=args.pool_name)

    print(
        f'added bindings between the resource pool "{args.pool_name}" '
        f"and the following workspaces: {args.workspace_names}"
    )
    return


def remove_binding(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    body = bindings.v1UnbindRPFromWorkspaceRequest(
        resourcePoolName=args.pool_name,
        workspaceNames=args.workspace_names,
    )
    bindings.delete_UnbindRPFromWorkspace(sess, body=body, resourcePoolName=args.pool_name)

    print(
        f'removed bindings between the resource pool "{args.pool_name}" '
        f"and the following workspaces: {args.workspace_names}"
    )
    return


def replace_bindings(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    body = bindings.v1OverwriteRPWorkspaceBindingsRequest(
        resourcePoolName=args.pool_name,
        workspaceNames=args.workspace_names,
    )
    bindings.put_OverwriteRPWorkspaceBindings(sess, body=body, resourcePoolName=args.pool_name)

    print(
        f'replaced bindings of the resource pool "{args.pool_name}" '
        f"with those to the following workspaces: {args.workspace_names}"
    )
    return


def list_workspaces(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    resp = bindings.get_ListWorkspacesBoundToRP(sess, resourcePoolName=args.pool_name)
    workspace_names = ""

    if resp.workspaceIds:
        workspace_names = ", ".join(
            [
                workspace.name
                for workspace in bindings.get_GetWorkspaces(sess).workspaces
                if workspace.id in set(resp.workspaceIds)
            ]
        )

    render.tabulate_or_csv(
        headers=["resource pool", "workspaces"],
        values=[[args.pool_name, workspace_names]],
        as_csv=False,
    )
    return


args_description = [
    cli.Cmd(
        "resource-pool rp",
        None,
        "manage resource pools",
        [
            cli.Cmd(
                "bindings",
                None,
                "manage resource pool bindings",
                [
                    cli.Cmd(
                        "add",
                        add_binding,
                        "add a resource-pool-to-workspace binding",
                        [
                            cli.Arg(
                                "pool_name", type=str, help="name of the resource pool to bind"
                            ),
                            cli.Arg(
                                "workspace_names",
                                nargs=argparse.ONE_OR_MORE,
                                type=str,
                                default=None,
                                help="the workspace to bind to",
                            ),
                        ],
                    ),
                    cli.Cmd(
                        "remove",
                        remove_binding,
                        "remove a resource-pool-to-workspace binding",
                        [
                            cli.Arg(
                                "pool_name", type=str, help="name of the resource pool to unbind"
                            ),
                            cli.Arg(
                                "workspace_names",
                                nargs=argparse.ONE_OR_MORE,
                                type=str,
                                default=None,
                                help="the workspace to unbind from",
                            ),
                        ],
                    ),
                    cli.Cmd(
                        "replace",
                        replace_bindings,
                        "replace all existing resource-pool-to-workspace bindings",
                        [
                            cli.Arg(
                                "pool_name", type=str, help="name of the resource pool to bind"
                            ),
                            cli.Arg(
                                "workspace_names",
                                nargs=argparse.ONE_OR_MORE,
                                type=str,
                                default=None,
                                help="the workspaces to bind to",
                            ),
                        ],
                    ),
                    cli.Cmd(
                        "list-workspaces",
                        list_workspaces,
                        "list all workspaces bound to the pool",
                        [
                            cli.Arg("pool_name", type=str, help="name of the resource pool"),
                        ],
                    ),
                ],
            ),
        ],
    )
]  # type: List[Any]
