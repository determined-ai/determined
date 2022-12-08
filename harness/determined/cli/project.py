import json
from argparse import Namespace
from time import sleep
from typing import Any, Dict, List, Sequence, Tuple

from determined import cli
from determined.common import api
from determined.common.api import authentication, bindings, errors
from determined.common.declarative_argparse import Arg, Cmd

from . import render
from .workspace import list_workspace_projects, pagination_args, workspace_by_name


def render_experiments(args: Namespace, experiments: Sequence[bindings.v1Experiment]) -> None:
    def format_experiment(e: bindings.v1Experiment) -> List[Any]:
        result = [
            e.id,
            e.username,
            e.name,
            e.forkedFrom,
            e.state.value.replace("STATE_", ""),
            render.format_percent(e.progress),
            render.format_time(e.startTime),
            render.format_time(e.endTime),
            e.resourcePool,
        ]
        if args.all:
            result.append(e.archived)
        return result

    headers = [
        "ID",
        "Owner",
        "Name",
        "Parent ID",
        "State",
        "Progress",
        "Started",
        "Ended",
        "Resource Pool",
    ]
    if args.all:
        headers.append("Archived")
    values = [format_experiment(e) for e in experiments]
    render.tabulate_or_csv(headers, values, False)


def render_project(project: bindings.v1Project) -> None:
    values = [
        project.id,
        project.name,
        project.description,
        project.numExperiments,
        project.numActiveExperiments,
    ]
    PROJECT_HEADERS = ["ID", "Name", "Description", "# Experiments", "# Active Experiments"]
    render.tabulate_or_csv(PROJECT_HEADERS, [values], False)


def project_by_name(
    sess: api.Session, workspace_name: str, project_name: str
) -> Tuple[bindings.v1Workspace, bindings.v1Project]:
    w = workspace_by_name(sess, workspace_name)
    p = bindings.get_GetWorkspaceProjects(sess, id=w.id, name=project_name).projects
    if len(p) == 0:
        raise errors.EmptyResultException(
            f'Did not find a project with name "{project_name}"'
            f' in workspace "{workspace_name}".'
        )
    return (w, p[0])


@authentication.required
def list_project_experiments(args: Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    kwargs: Dict[str, Any] = {
        "projectId": p.id,
        "orderBy": bindings.v1OrderBy[f"ORDER_BY_{args.order_by.upper()}"],
        "sortBy": bindings.v1GetExperimentsRequestSortBy[f"SORT_BY_{args.sort_by.upper()}"],
    }
    if not args.all:
        kwargs["users"] = [authentication.must_cli_auth().get_session_user()]
        kwargs["archived"] = "false"

    all_experiments: List[bindings.v1Experiment] = []
    internal_offset = args.offset if ("offset" in args and args.offset) else 0
    limit = args.limit if "limit" in args else 200
    while True:
        experiments = bindings.get_GetExperiments(
            sess, limit=limit, offset=internal_offset, **kwargs
        ).experiments
        all_experiments += experiments
        internal_offset += len(experiments)
        if ("offset" in args and args.offset) or len(experiments) < limit:
            break

    if args.json:
        print(json.dumps([e.to_json() for e in all_experiments], indent=2))
    else:
        render_experiments(args, all_experiments)


@authentication.required
def create_project(args: Namespace) -> None:
    sess = cli.setup_session(args)
    w = workspace_by_name(sess, args.workspace_name)
    content = bindings.v1PostProjectRequest(
        name=args.name, description=args.description, workspaceId=w.id
    )
    p = bindings.post_PostProject(sess, body=content, workspaceId=w.id).project
    if args.json:
        print(json.dumps(p.to_json(), indent=2))
    else:
        render_project(p)


@authentication.required
def describe_project(args: Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    if args.json:
        print(json.dumps(p.to_json(), indent=2))
    else:
        render_project(p)
        print("\nAssociated Experiments")
        vars(args)["order_by"] = "asc"
        vars(args)["sort_by"] = "id"
        list_project_experiments(args)


@authentication.required
def delete_project(args: Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    if args.yes or render.yes_or_no(
        'Deleting project "' + args.project_name + '" will result in the \n'
        "unrecoverable deletion of this project and all of its experiments and notes.\n"
        "For a recoverable alternative, see the 'archive' command. Do you still \n"
        "wish to proceed?"
    ):
        resp = bindings.delete_DeleteProject(sess, id=p.id)
        if resp.completed:
            print(f"Successfully deleted project {args.project_name}.")
        else:
            print(f"Started deletion of project {args.project_name}...")
            while True:
                sleep(2)
                try:
                    p = bindings.get_GetProject(sess, id=p.id).project
                    if p.state == bindings.v1WorkspaceState.WORKSPACE_STATE_DELETE_FAILED:
                        raise errors.DeleteFailedException(p.errorMessage)
                    elif p.state == bindings.v1WorkspaceState.WORKSPACE_STATE_DELETING:
                        print(f"Remaining experiment count: {p.numExperiments}")
                except errors.NotFoundException:
                    print("Project deleted successfully.")
                    break
    else:
        print("Aborting project deletion.")


@authentication.required
def edit_project(args: Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    updated = bindings.v1PatchProject(name=args.new_name, description=args.description)
    new_p = bindings.patch_PatchProject(sess, body=updated, id=p.id).project

    if args.json:
        print(json.dumps(new_p.to_json(), indent=2))
    else:
        render_project(new_p)


@authentication.required
def archive_project(args: Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    bindings.post_ArchiveProject(sess, id=p.id)
    print(f"Successfully archived project {args.project_name}.")


@authentication.required
def unarchive_project(args: Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    bindings.post_UnarchiveProject(sess, id=p.id)
    print(f"Successfully un-archived project {args.project_name}.")


args_description = [
    Cmd(
        "p|roject",
        None,
        "manage projects",
        [
            Cmd(
                "list ls",
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
                "list-experiments",
                list_project_experiments,
                "list the experiments associated with a project",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg("project_name", type=str, help="name of the project"),
                    Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        default=False,
                        help="show all experiments (including archived and other users')",
                    ),
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
                create_project,
                "create project",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg("name", type=str, help="name of the project"),
                    Arg("--description", type=str, help="description of the project"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "delete",
                delete_project,
                "delete project",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg("project_name", type=str, help="name of the project"),
                    Arg(
                        "--yes",
                        action="store_true",
                        default=False,
                        help="automatically answer yes to prompts",
                    ),
                ],
            ),
            Cmd(
                "archive",
                archive_project,
                "archive project",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg("project_name", type=str, help="name of the project"),
                ],
            ),
            Cmd(
                "unarchive",
                unarchive_project,
                "unarchive project",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg("project_name", type=str, help="name of the project"),
                ],
            ),
            Cmd(
                "describe",
                describe_project,
                "describe project",
                [
                    Arg("workspace_name", type=str, help="name of the workspace"),
                    Arg("project_name", type=str, help="name of the project"),
                    Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        default=False,
                        help="show all experiments (including archived and other users')",
                    ),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "edit",
                edit_project,
                "edit project",
                [
                    Arg("workspace_name", type=str, help="current name of the workspace"),
                    Arg("project_name", type=str, help="name of the project"),
                    Arg("--new_name", type=str, help="new name of the project"),
                    Arg("--description", type=str, help="description of the project"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
        ],
    )
]  # type: List[Any]
