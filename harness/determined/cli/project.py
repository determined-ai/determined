import json
from argparse import Namespace
from typing import Any, List, Tuple

from determined.cli.session import setup_session
from determined.common.api import authentication, bindings
from determined.common.api.bindings import v1Project, v1Workspace
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.experimental import session

from workspace import get_workspace_by_name, list_workspace_projects

from . import render

PROJECT_HEADERS = ["ID", "Name", "# Experiments", "# Active Experiments"]
WORKSPACE_HEADERS = ["ID", "Name"]


def render_project(project: v1Project) -> None:
    values = [
        project.id,
        project.name,
        project.numExperiments,
        project.numActiveExperiments,
    ]
    render.tabulate_or_csv(PROJECT_HEADERS, [values], False)


def project_by_name(sess: session.Session, workspace_name: str, project_name: str) -> Tuple[int, int]:
    w = get_workspace_by_name(sess, workspace_name)
    p = bindings.get_GetWorkspaceProjects(sess, name=project_name).projects
    if len(p) == 0:
        raise Exception(f'No project found on this workspace with name: "{project_name}"')
    return (w.id, p[0].id)


@authentication.required
def list_projects(args: Namespace) -> None:
    list_workspace_projects(args)


@authentication.required
def list_workspace_projects(args: Namespace) -> None:
    sess = setup_session(args)
    w = workspace_by_name(sess, args.workspace_name)

    orderArg = bindings.v1OrderBy[f"ORDER_BY_{args.order_by.upper()}"]
    sortArg = bindings.v1GetWorkspaceProjectsRequestSortBy[f"SORT_BY_{args.sort_by.upper()}"]
    projects = bindings.get_GetWorkspaceProjects(sess, id=w.id, orderBy=orderArg, sortBy=sortArg).projects
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
                p.numExperiments,
                p.numActiveExperiments,
            ]
            for p in projects
        ]
        render.tabulate_or_csv(PROJECT_HEADERS, values, False)


@authentication.required
def delete_workspace(args: Namespace) -> None:
    if args.yes or render.yes_or_no(
        "Deleting a workspace will result in the unrecoverable \n"
        "deletion of all associated projects. For a recoverable \n"
        "alternative, see the 'archive' command. Do you still \n"
        "wish to proceed?"
    ):
        sess = setup_session(args)
        w = workspace_by_name(sess, args.workspace_name)
        bindings.delete_DeleteWorkspace(sess, id=w.id)
        print(f"Successfully deleted workspace {args.workspace_name}.")
    else:
        print("Aborting workspace deletion.")


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
        "p|roject",
        None,
        "manage projects",
        [
            Cmd(
                "list",
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
        ],
    )
]  # type: List[Any]
