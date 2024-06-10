import argparse
import time
from typing import Any, Dict, List, Sequence, Tuple

from determined import cli
from determined.cli import render, workspace
from determined.common import api
from determined.common.api import bindings, errors


def render_experiments(
    args: argparse.Namespace, experiments: Sequence[bindings.v1Experiment]
) -> None:
    def format_experiment(e: bindings.v1Experiment) -> List[Any]:
        result = [
            e.id,
            e.username,
            e.name,
            e.forkedFrom,
            e.state,
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
        project.key,
        project.name,
        project.description,
        project.numExperiments,
        project.numActiveExperiments,
    ]
    PROJECT_HEADERS = ["ID", "Key", "Name", "Description", "# Experiments", "# Active Experiments"]
    render.tabulate_or_csv(PROJECT_HEADERS, [values], False)


def project_by_name(
    sess: api.Session, workspace_name: str, project_name: str
) -> Tuple[bindings.v1Workspace, bindings.v1Project]:
    w = api.workspace_by_name(sess, workspace_name)
    p = bindings.get_GetWorkspaceProjects(sess, id=w.id, name=project_name).projects
    if len(p) == 0:
        raise api.not_found_errs("project", project_name, sess)
    return (w, p[0])


def list_project_experiments(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    kwargs: Dict[str, Any] = {
        "projectId": p.id,
        "orderBy": bindings.v1OrderBy[args.order_by.upper()],
        "sortBy": bindings.v1GetExperimentsRequestSortBy[args.sort_by.upper()],
    }
    if not args.all:
        kwargs["users"] = [sess.username]
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
        render.print_json([e.to_json() for e in all_experiments])
    else:
        render_experiments(args, all_experiments)


def create_project(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    w = api.workspace_by_name(sess, args.workspace_name)
    content = bindings.v1PostProjectRequest(
        name=args.name, description=args.description, workspaceId=w.id, key=args.key
    )
    p = bindings.post_PostProject(sess, body=content, workspaceId=w.id).project
    if args.json:
        render.print_json(p.to_json())
    else:
        render_project(p)


def describe_project(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    if args.json:
        render.print_json(p.to_json())
    else:
        render_project(p)
        print("\nAssociated Experiments")
        vars(args)["order_by"] = "asc"
        vars(args)["sort_by"] = "id"
        list_project_experiments(args)


def delete_project(args: argparse.Namespace) -> None:
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
                time.sleep(2)
                try:
                    p = bindings.get_GetProject(sess, id=p.id).project
                    if p.state == bindings.v1WorkspaceState.DELETE_FAILED:
                        raise errors.DeleteFailedException(p.errorMessage)
                    elif p.state == bindings.v1WorkspaceState.DELETING:
                        print(f"Remaining experiment count: {p.numExperiments}")
                except errors.NotFoundException:
                    print("Project deleted successfully.")
                    break
    else:
        print("Aborting project deletion.")


def edit_project(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    updated = bindings.v1PatchProject(
        name=args.new_name, description=args.description, key=args.key
    )
    new_p = bindings.patch_PatchProject(sess, body=updated, id=p.id).project

    if args.json:
        render.print_json(new_p.to_json())
    else:
        render_project(new_p)


def archive_project(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    bindings.post_ArchiveProject(sess, id=p.id)
    print(f"Successfully archived project {args.project_name}.")


def unarchive_project(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project_by_name(sess, args.workspace_name, args.project_name)
    bindings.post_UnarchiveProject(sess, id=p.id)
    print(f"Successfully un-archived project {args.project_name}.")


args_description = [
    cli.Cmd(
        "p|roject",
        None,
        "manage projects",
        [
            cli.Cmd(
                "list ls",
                workspace.list_workspace_projects,
                "list the projects associated with a workspace",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg(
                        "--sort-by",
                        type=str,
                        choices=["id", "name"],
                        default="id",
                        help="sort workspaces by the given field",
                    ),
                    cli.Arg(
                        "--order-by",
                        type=str,
                        choices=["asc", "desc"],
                        default="asc",
                        help="order workspaces in either ascending or descending order",
                    ),
                    *workspace.pagination_args,
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "list-experiments",
                list_project_experiments,
                "list the experiments associated with a project",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg("project_name", type=str, help="name of the project"),
                    cli.Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        default=False,
                        help="show all experiments (including archived and other users')",
                    ),
                    cli.Arg(
                        "--sort-by",
                        type=str,
                        choices=["id", "name"],
                        default="id",
                        help="sort workspaces by the given field",
                    ),
                    cli.Arg(
                        "--order-by",
                        type=str,
                        choices=["asc", "desc"],
                        default="asc",
                        help="order workspaces in either ascending or descending order",
                    ),
                    *workspace.pagination_args,
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "create",
                create_project,
                "create project",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg("name", type=str, help="name of the project"),
                    cli.Arg("--description", type=str, help="description of the project"),
                    cli.Arg("--key", type=str, help="key of the project"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "delete",
                delete_project,
                "delete project",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg("project_name", type=str, help="name of the project"),
                    cli.Arg(
                        "--yes",
                        action="store_true",
                        default=False,
                        help="automatically answer yes to prompts",
                    ),
                ],
            ),
            cli.Cmd(
                "archive",
                archive_project,
                "archive project",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg("project_name", type=str, help="name of the project"),
                ],
            ),
            cli.Cmd(
                "unarchive",
                unarchive_project,
                "unarchive project",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg("project_name", type=str, help="name of the project"),
                ],
            ),
            cli.Cmd(
                "describe",
                describe_project,
                "describe project",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg("project_name", type=str, help="name of the project"),
                    cli.Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        default=False,
                        help="show all experiments (including archived and other users')",
                    ),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "edit",
                edit_project,
                "edit project",
                [
                    cli.Arg("workspace_name", type=str, help="current name of the workspace"),
                    cli.Arg("project_name", type=str, help="name of the project"),
                    cli.Arg("--new_name", type=str, help="new name of the project"),
                    cli.Arg("--description", type=str, help="description of the project"),
                    cli.Arg("--key", type=str, help="key of the project"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
        ],
    )
]  # type: List[Any]
