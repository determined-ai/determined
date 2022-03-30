import json
from argparse import Namespace
from typing import Any, List

from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.api.bindings import v1Project, v1Workspace
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.experimental import Determined, session

from . import render


def d_session(args: Namespace) -> session.Session:
    d = Determined(args.master, None)
    return d._session


def render_workspace(workspace: v1Workspace) -> None:
    table = [
        ["ID", workspace.id],
        ["Name", workspace.name],
    ]
    headers, values = zip(*table)  # type: ignore
    render.tabulate_or_csv(headers, [values], False)


def render_project(project: v1Project) -> None:
    table = [
        ["ID", project.id],
        ["Name", project.name],
        ["# Experiments", project.num_experiments],
        ["# Active Experiments", project.num_active_experiments],
    ]
    headers, values = zip(*table)  # type: ignore
    render.tabulate_or_csv(headers, [values], False)


def list_workspaces(args: Namespace) -> None:
    sess = d_session(args)
    # limit, name, offset, users
    orderArg = bindings.v1OrderBy[f"ORDER_BY_{args.order_by.upper()}"]
    sortArg = bindings.v1GetWorkspacesRequestSortBy[f"SORT_BY_{args.sort_by.upper()}"]
    workspaces = bindings.get_GetWorkspaces(sess, orderBy=orderArg, sortBy=sortArg).workspaces
    if args.json:
        print(json.dumps([w.to_json() for w in workspaces], indent=2))
    else:
        headers = ["ID", "Name"]
        values = [
            [
                w.id,
                w.name,
            ]
            for w in workspaces
        ]
        render.tabulate_or_csv(headers, values, False)


def create_workspace(args: Namespace) -> None:
    sess = d_session(args)
    content = bindings.v1PostWorkspaceRequest(args.name)
    w = bindings.post_PostWorkspace(sess, body=content).workspace

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
                "create",
                create_workspace,
                "create workspace",
                [
                    Arg("name", type=str, help="unique name of the workspace"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
        ],
    )
]  # type: List[Any]
