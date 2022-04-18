import json
from argparse import Namespace
from typing import Any, List

from determined.cli.session import setup_session
from determined.common.api import authentication, bindings
from determined.common.api.bindings import v1Workspace
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.experimental import session

from . import render

PROJECT_HEADERS = ["ID", "Name", "Description", "# Experiments", "# Active Experiments"]
WORKSPACE_HEADERS = ["ID", "Name"]


def render_workspace(workspace: v1Workspace) -> None:
    values = [
        workspace.id,
        workspace.name,
    ]
    render.tabulate_or_csv(WORKSPACE_HEADERS, [values], False)


def workspace_by_name(sess: session.Session, name: str) -> v1Workspace:
    w = bindings.get_GetWorkspaces(sess, name=name).workspaces
    if len(w) == 0:
        raise Exception(f'No workspace found with name: "{name}"')
    return w[0]


@authentication.required
def list_workspaces(args: Namespace) -> None:
    # limit, name, offset, users
    orderArg = bindings.v1OrderBy[f"ORDER_BY_{args.order_by.upper()}"]
    sortArg = bindings.v1GetWorkspacesRequestSortBy[f"SORT_BY_{args.sort_by.upper()}"]
    workspaces = bindings.get_GetWorkspaces(
        setup_session(args), orderBy=orderArg, sortBy=sortArg
    ).workspaces
    if args.json:
        print(json.dumps([w.to_json() for w in workspaces], indent=2))
    else:
        values = [
            [
                w.id,
                w.name,
            ]
            for w in workspaces
        ]
        render.tabulate_or_csv(WORKSPACE_HEADERS, values, False)


@authentication.required
def list_workspace_projects(args: Namespace) -> None:
    sess = setup_session(args)
    w = workspace_by_name(sess, args.workspace_name)

    orderArg = bindings.v1OrderBy[f"ORDER_BY_{args.order_by.upper()}"]
    sortArg = bindings.v1GetWorkspaceProjectsRequestSortBy[f"SORT_BY_{args.sort_by.upper()}"]
    projects = bindings.get_GetWorkspaceProjects(
        sess, id=w.id, orderBy=orderArg, sortBy=sortArg
    ).projects
    if args.json:
        print(json.dumps([p.to_json() for p in projects], indent=2))
    else:
        values = [
            [
                p.id,
                p.name,
                p.description,
                p.numExperiments,
                p.numActiveExperiments,
            ]
            for p in projects
        ]
        render.tabulate_or_csv(PROJECT_HEADERS, values, False)


@authentication.required
def create_workspace(args: Namespace) -> None:
    content = bindings.v1PostWorkspaceRequest(args.name)
    w = bindings.post_PostWorkspace(setup_session(args), body=content).workspace

    if args.json:
        print(json.dumps(w.to_json(), indent=2))
    else:
        render_workspace(w)


@authentication.required
def describe_workspace(args: Namespace) -> None:
    sess = setup_session(args)
    w = workspace_by_name(sess, args.workspace_name)
    if args.json:
        print(json.dumps(w.to_json(), indent=2))
    else:
        render_workspace(w)
        print("\nAssociated Projects")
        projects = bindings.get_GetWorkspaceProjects(sess, id=w.id).projects
        values = [
            [
                p.id,
                p.name,
                p.description,
                p.numExperiments,
                p.numActiveExperiments,
            ]
            for p in projects
        ]
        render.tabulate_or_csv(PROJECT_HEADERS, values, False)


@authentication.required
def delete_workspace(args: Namespace) -> None:
    if args.yes or render.yes_or_no(
        'Deleting workspace "' + args.workspace_name + '" will result \n'
        "in the unrecoverable deletion of all associated projects. For a \n"
        "recoverable alternative, see the 'archive' command. Do you still \n"
        "wish to proceed?"
    ):
        sess = setup_session(args)
        w = workspace_by_name(sess, args.workspace_name)
        bindings.delete_DeleteWorkspace(sess, id=w.id)
        print(f"Successfully deleted workspace {args.workspace_name}.")
    else:
        print("Aborting workspace deletion.")


@authentication.required
def archive_workspace(args: Namespace) -> None:
    sess = setup_session(args)
    current = workspace_by_name(sess, args.workspace_name)
    bindings.post_ArchiveWorkspace(sess, id=current.id)
    print(f"Successfully archived workspace {args.workspace_name}.")


@authentication.required
def unarchive_workspace(args: Namespace) -> None:
    sess = setup_session(args)
    current = workspace_by_name(sess, args.workspace_name)
    bindings.post_UnarchiveWorkspace(sess, id=current.id)
    print(f"Successfully un-archived workspace {args.workspace_name}.")


@authentication.required
def edit_workspace(args: Namespace) -> None:
    sess = setup_session(args)
    current = workspace_by_name(sess, args.workspace_name)
    updated = bindings.v1PatchWorkspace(name=args.new_name)
    w = bindings.patch_PatchWorkspace(sess, body=updated, id=current.id).workspace

    if args.json:
        print(json.dumps(w.to_json(), indent=2))
    else:
        render_workspace(w)


args_description = [
    Cmd(
        "w|orkspace",
        None,
        "manage workspaces",
        [
            Cmd(
                "list",
                list_workspaces,
                "list all workspaces",
                [
                    Arg(
                        "--sort-by",
                        type=str,
                        choices=["id", "name"],
                        default="id",
                        help="sort workspaces by the given field",
                    ),
                    Arg(
                        "--order-by",
                        type=str,
                        choices=["asc", "desc"],
                        default="asc",
                        help="order workspaces in either ascending or descending order",
                    ),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
                is_default=True,
            ),
            Cmd(
                "list-projects",
                list_workspace_projects,
                "list the projects associated with a workspace",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg(
                        "--sort-by",
                        type=str,
                        choices=["id", "name"],
                        default="id",
                        help="sort workspaces by the given field",
                    ),
                    Arg(
                        "--order-by",
                        type=str,
                        choices=["asc", "desc"],
                        default="asc",
                        help="order workspaces in either ascending or descending order",
                    ),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "create",
                create_workspace,
                "create workspace",
                [
                    Arg("name", type=str, help="unique name of the workspace"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "delete",
                delete_workspace,
                "delete workspace",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg(
                        "--yes",
                        action="store_true",
                        default=False,
                        help="automatically answer yes to prompts",
                    ),
                ],
            ),
            Cmd(
                "describe",
                describe_workspace,
                "describe workspace",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "edit",
                edit_workspace,
                "edit workspace",
                [
                    Arg("workspace_name", type=str, help="current name of the workspace"),
                    Arg("new_name", type=str, help="new name of the workspace"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "archive",
                archive_workspace,
                "archive workspace",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                ],
            ),
            Cmd(
                "unarchive",
                unarchive_workspace,
                "unarchive workspace",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                ],
            ),
        ],
    )
]  # type: List[Any]
