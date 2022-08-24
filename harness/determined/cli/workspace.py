import json
from argparse import Namespace
from time import sleep
from typing import Any, List, Sequence

from determined import cli
from determined.common import api
from determined.common.api import authentication, bindings, errors
from determined.common.declarative_argparse import Arg, Cmd

from . import render

PROJECT_HEADERS = ["ID", "Name", "Description", "# Experiments", "# Active Experiments"]
WORKSPACE_HEADERS = ["ID", "Name", "# Projects"]


def render_workspaces(workspaces: Sequence[bindings.v1Workspace]) -> None:
    values = [
        [
            w.id,
            w.name,
            w.numProjects,
        ]
        for w in workspaces
    ]
    render.tabulate_or_csv(WORKSPACE_HEADERS, values, False)


def workspace_by_name(sess: api.Session, name: str) -> bindings.v1Workspace:
    w = bindings.get_GetWorkspaces(sess, name=name).workspaces
    if len(w) == 0:
        raise errors.EmptyResultException(f'Did not find a workspace with name "{name}".')
    return w[0]


@authentication.required
def list_workspaces(args: Namespace) -> None:
    sess = cli.setup_session(args)
    orderArg = bindings.v1OrderBy[f"ORDER_BY_{args.order_by.upper()}"]
    sortArg = bindings.v1GetWorkspacesRequestSortBy[f"SORT_BY_{args.sort_by.upper()}"]
    internal_offset = args.offset or 0
    all_workspaces: List[bindings.v1Workspace] = []
    while True:
        workspaces = bindings.get_GetWorkspaces(
            sess,
            limit=args.limit,
            offset=internal_offset,
            orderBy=orderArg,
            sortBy=sortArg,
        ).workspaces
        all_workspaces += workspaces
        internal_offset += len(workspaces)
        if args.offset or len(workspaces) < args.limit:
            break

    if args.json:
        print(json.dumps([w.to_json() for w in all_workspaces], indent=2))
    else:
        render_workspaces(all_workspaces)


@authentication.required
def list_workspace_projects(args: Namespace) -> None:
    sess = cli.setup_session(args)
    w = workspace_by_name(sess, args.workspace_name)
    orderArg = bindings.v1OrderBy[f"ORDER_BY_{args.order_by.upper()}"]
    sortArg = bindings.v1GetWorkspaceProjectsRequestSortBy[f"SORT_BY_{args.sort_by.upper()}"]
    internal_offset = args.offset if ("offset" in args and args.offset) else 0
    limit = args.limit if "limit" in args else 200
    all_projects: List[bindings.v1Project] = []

    while True:
        projects = bindings.get_GetWorkspaceProjects(
            sess,
            id=w.id,
            limit=limit,
            offset=internal_offset,
            orderBy=orderArg,
            sortBy=sortArg,
        ).projects
        all_projects += projects
        internal_offset += len(projects)
        if ("offset" in args and args.offset) or len(projects) < limit:
            break

    if args.json:
        print(json.dumps([p.to_json() for p in all_projects], indent=2))
    else:
        values = [
            [
                p.id,
                p.name,
                p.description,
                p.numExperiments,
                p.numActiveExperiments,
            ]
            for p in all_projects
        ]
        render.tabulate_or_csv(PROJECT_HEADERS, values, False)


@authentication.required
def create_workspace(args: Namespace) -> None:
    content = bindings.v1PostWorkspaceRequest(name=args.name)
    w = bindings.post_PostWorkspace(cli.setup_session(args), body=content).workspace

    if args.json:
        print(json.dumps(w.to_json(), indent=2))
    else:
        render_workspaces([w])


@authentication.required
def describe_workspace(args: Namespace) -> None:
    sess = cli.setup_session(args)
    w = workspace_by_name(sess, args.workspace_name)
    if args.json:
        print(json.dumps(w.to_json(), indent=2))
    else:
        render_workspaces([w])
        print("\nAssociated Projects")
        vars(args)["order_by"] = "asc"
        vars(args)["sort_by"] = "id"
        list_workspace_projects(args)


@authentication.required
def delete_workspace(args: Namespace) -> None:
    sess = cli.setup_session(args)
    w = workspace_by_name(sess, args.workspace_name)
    if args.yes or render.yes_or_no(
        'Deleting workspace "' + args.workspace_name + '" will result \n'
        "in the unrecoverable deletion of all associated projects and experiments.\n"
        "For a recoverable alternative, see the 'archive' command. Do you still \n"
        "wish to proceed?"
    ):
        resp = bindings.delete_DeleteWorkspace(sess, id=w.id)
        if resp.completed:
            print(f"Successfully deleted workspace {args.workspace_name}.")
        else:
            print(f"Started deletion of workspace {args.workspace_name}...")
            while True:
                sleep(2)
                try:
                    w = bindings.get_GetWorkspace(sess, id=w.id).workspace
                    if w.state == bindings.v1WorkspaceState.WORKSPACE_STATE_DELETE_FAILED:
                        raise errors.DeleteFailedException(w.errorMessage)
                    elif w.state == bindings.v1WorkspaceState.WORKSPACE_STATE_DELETING:
                        print(f"Remaining project count: {w.numProjects}")
                except errors.NotFoundException:
                    print("Workspace deleted successfully.")
                    break
    else:
        print("Aborting workspace deletion.")


@authentication.required
def archive_workspace(args: Namespace) -> None:
    sess = cli.setup_session(args)
    current = workspace_by_name(sess, args.workspace_name)
    bindings.post_ArchiveWorkspace(sess, id=current.id)
    print(f"Successfully archived workspace {args.workspace_name}.")


@authentication.required
def unarchive_workspace(args: Namespace) -> None:
    sess = cli.setup_session(args)
    current = workspace_by_name(sess, args.workspace_name)
    bindings.post_UnarchiveWorkspace(sess, id=current.id)
    print(f"Successfully un-archived workspace {args.workspace_name}.")


@authentication.required
def edit_workspace(args: Namespace) -> None:
    sess = cli.setup_session(args)
    current = workspace_by_name(sess, args.workspace_name)
    updated = bindings.v1PatchWorkspace(name=args.new_name)
    w = bindings.patch_PatchWorkspace(sess, body=updated, id=current.id).workspace

    if args.json:
        print(json.dumps(w.to_json(), indent=2))
    else:
        render_workspaces([w])


# do not use util.py's pagination_args because behavior here is
# to hide pagination and unify all pages of experiments into one output
pagination_args = [
    Arg(
        "--limit",
        type=int,
        default=200,
        help="Maximum items per page of results",
    ),
    Arg(
        "--offset",
        type=int,
        default=None,
        help="Number of items to skip before starting page of results",
    ),
]
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
                    *pagination_args,
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
                    *pagination_args,
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
