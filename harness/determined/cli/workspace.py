import json
from argparse import Namespace
from typing import Any, List

from determined.cli.session import setup_session
from determined.common.api import authentication, bindings
from determined.common.api.bindings import v1Project, v1Workspace
from determined.common.declarative_argparse import Arg, Cmd

from . import render

PROJECT_HEADERS = ["ID", "Name", "# Experiments", "# Active Experiments"]
WORKSPACE_HEADERS = ["ID", "Name"]


def render_workspace(workspace: v1Workspace) -> None:
    values = [
        workspace.id,
        workspace.name,
    ]
    render.tabulate_or_csv(WORKSPACE_HEADERS, [values], False)


def render_project(project: v1Project) -> None:
    values = [
        project.id,
        project.name,
        project.numExperiments,
        project.numActiveExperiments,
    ]
    render.tabulate_or_csv(PROJECT_HEADERS, [values], False)


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
    orderArg = bindings.v1OrderBy[f"ORDER_BY_{args.order_by.upper()}"]
    sortArg = bindings.v1GetWorkspaceProjectsRequestSortBy[f"SORT_BY_{args.sort_by.upper()}"]
    projects = bindings.get_GetWorkspaceProjects(
        setup_session(args), id=args.id, orderBy=orderArg, sortBy=sortArg
    ).projects
    if args.json:
        print(json.dumps([p.to_json() for p in projects], indent=2))
    else:
        values = [
            [
                p.id,
                p.name,
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
    w = bindings.get_GetWorkspace(setup_session(args), id=args.id).workspace
    if args.json:
        print(json.dumps(w.to_json(), indent=2))
    else:
        render_workspace(w)
        print("\nAssociated Projects")
        args.order_by = "asc"
        args.sort_by = "id"
        list_workspace_projects(args)


@authentication.required
def delete_workspace(args: Namespace) -> None:
    if args.yes or render.yes_or_no(
        "Deleting a workspace will result in the unrecoverable \n"
        "deletion of all associated projects. For a recoverable \n"
        "alternative, see the 'archive' command. Do you still \n"
        "wish to proceed?"
    ):
        bindings.delete_DeleteWorkspace(setup_session(args), id=args.id)
        print("Successfully deleted workspace {}.".format(args.id))
    else:
        print("Aborting workspace deletion.")


@authentication.required
def edit_workspace(args: Namespace) -> None:
    content = bindings.v1PatchWorkspace(name=args.name)
    w = bindings.patch_PatchWorkspace(setup_session(args), body=content, id=args.id).workspace

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
                    Arg("id", type=int, help="unique id of the workspace"),
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
                    Arg("id", type=int, help="unique ID of the workspace"),
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
                    Arg("id", type=int, help="unique ID of the workspace"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "edit",
                edit_workspace,
                "edit workspace",
                [
                    Arg("id", type=str, help="unique ID of the workspace"),
                    Arg("name", type=str, help="new name of the workspace"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
        ],
    )
]  # type: List[Any]
