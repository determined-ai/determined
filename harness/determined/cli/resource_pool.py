from argparse import ONE_OR_MORE, Namespace
from typing import List

from determined.cli import render, setup_session
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd


def get_workspaces_string(workspaces: List[str]) -> str:
    workspaces_str = ""
    for workspace in workspaces[:-1]:
        workspaces_str += workspace + ", "

    if workspaces_str != "":
        workspaces_str += "and "
    workspaces_str += workspaces[-1]
    return workspaces_str


@authentication.required
def add_binding(args: Namespace) -> None:
    body = bindings.v1BindRPToWorkspaceRequest(
        resourcePoolName=args.pool_name, workspaceNames=args.workspace_names
    )
    bindings.post_BindRPToWorkspace(setup_session(args), body=body, resourcePoolName=args.pool_name)

    workspaces_string = get_workspaces_string(args.workspace_names)

    print(
        f'added bindings between the resource pool "{args.pool_name}" '
        f"and the following workspaces: {workspaces_string}"
    )
    return


@authentication.required
def remove_binding(args: Namespace) -> None:
    body = bindings.v1UnbindRPFromWorkspaceRequest(
        resourcePoolName=args.pool_name,
        workspaceNames=args.workspace_names,
    )
    bindings.delete_UnbindRPFromWorkspace(
        setup_session(args), body=body, resourcePoolName=args.pool_name
    )

    workspaces_string = get_workspaces_string(args.workspace_names)
    print(
        f'removed bindings between the resource pool "{args.pool_name}'
        f"and the following workspaces: {workspaces_string}"
    )
    return


@authentication.required
def replace_bindings(args: Namespace) -> None:
    body = bindings.v1OverwriteRPWorkspaceBindingsRequest(
        resourcePoolName=args.pool_name,
        workspaceNames=args.workspace_names,
    )
    bindings.put_OverwriteRPWorkspaceBindings(
        setup_session(args), body=body, resourcePoolName=args.pool_name
    )

    workspaces_string = get_workspaces_string(args.workspace_names)
    print(
        f'replaced bindings of the resource pool "{args.pool_name}'
        f"with those to the following workspaces: {workspaces_string}"
    )
    return


@authentication.required
def list_workspaces(args: Namespace) -> None:
    resp = bindings.get_ListWorkspacesBoundToRP(
        setup_session(args), resourcePoolName=args.pool_name
    )
    if resp.workspaceIds is None or len(resp.workspaceIds) == 0:
        print("resource pool has no assignments")
        return

    print("ids are:", resp.workspaceIds)  # TODO: convert to names

    workspaces_string = ""
    for workspace in resp.workspaceIds[:-1]:
        workspaces_string += str(workspace) + ", "
    workspaces_string += str(resp.workspaceIds[-1])

    headers = ["resource pool", "workspaces"]
    values = [[args.pool_name, workspaces_string]]
    render.tabulate_or_csv(headers, values, False)
    return


args_description = [
    Cmd(
        "resource-pool rp",
        None,
        "manage resource pools",
        [
            Cmd(
                "bindings",
                None,
                "manage resource pool bindings",
                [
                    Cmd(
                        "add",
                        add_binding,
                        "add a resource-pool-to-workspace binding",
                        [
                            Arg("pool_name", type=str, help="name of the resource pool to bind"),
                            Arg(
                                "workspace_names",
                                nargs=ONE_OR_MORE,
                                type=str,
                                default=None,
                                help="the workspace to bind to",
                            ),
                        ],
                    ),
                    Cmd(
                        "remove",
                        remove_binding,
                        "remove a resource-pool-to-workspace binding",
                        [
                            Arg("pool_name", type=str, help="name of the resource pool to unbind"),
                            Arg(
                                "workspace_names",
                                nargs=ONE_OR_MORE,
                                type=str,
                                default=None,
                                help="the workspace to unbind from",
                            ),
                        ],
                    ),
                    Cmd(
                        "replace",
                        replace_bindings,
                        "replace all existing resource-pool-to-workspace bindings",
                        [
                            Arg("pool_name", type=str, help="name of the resource pool to bind"),
                            Arg(
                                "workspace_names",
                                nargs=ONE_OR_MORE,
                                type=str,
                                default=None,
                                help="the workspaces to bind to",
                            ),
                        ],
                    ),
                    Cmd(
                        "list-workspaces",
                        list_workspaces,
                        "list all workspaces bound to the pool",
                        [
                            Arg("pool_name", type=str, help="name of the resource pool"),
                        ],
                        is_default=True,
                    ),
                ],
            ),
        ],
    )
]
