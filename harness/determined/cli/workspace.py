import argparse
import json
import time
from typing import Any, Dict, List, Optional, Sequence

from determined import cli
from determined.cli import render, user
from determined.common import api, util
from determined.common.api import bindings, errors
from determined.common.experimental import workspace

PROJECT_HEADERS = ["ID", "Name", "Description", "# Experiments", "# Active Experiments"]
WORKSPACE_HEADERS = [
    "ID",
    "Name",
    "# Projects",
    "Agent Uid",
    "Agent Gid",
    "Agent User",
    "Agent Group",
    "Default Compute Pool",
    "Default Aux Pool",
]

workspace_arg: cli.Arg = cli.Arg("-w", "--workspace-name", type=str, help="workspace name")


def get_workspace_id_from_args(args: argparse.Namespace) -> Optional[int]:
    sess = cli.setup_session(args)
    workspace_id = None
    if args.workspace_name:
        workspace = api.workspace_by_name(sess, args.workspace_name)
        if workspace.archived:
            raise argparse.ArgumentError(None, f'Workspace "{args.workspace_name}" is archived.')
        workspace_id = workspace.id
    return workspace_id


def get_workspace_names(session: api.Session) -> Dict[int, str]:
    """Get a mapping of workspace IDs to workspace names."""
    resp = bindings.get_GetWorkspaces(session)
    mapping = {}
    for w in resp.workspaces:
        assert w.id not in mapping, "workspace ids are assumed to be unique."
        mapping[w.id] = w.name
    return mapping


def render_workspaces(
    workspaces: Sequence[bindings.v1Workspace], from_list_api: bool = False
) -> None:
    values = []
    for w in workspaces:
        value = [
            w.id,
            w.name,
            w.numProjects,
            w.agentUserGroup.agentUid if w.agentUserGroup else None,
            w.agentUserGroup.agentGid if w.agentUserGroup else None,
            w.agentUserGroup.agentUser if w.agentUserGroup else None,
            w.agentUserGroup.agentGroup if w.agentUserGroup else None,
            w.defaultComputePool if w.defaultComputePool else None,
            w.defaultAuxPool if w.defaultAuxPool else None,
        ]
        if not from_list_api:
            value.append(w.checkpointStorageConfig)
        values.append(value)

    headers = WORKSPACE_HEADERS
    if not from_list_api:
        headers = WORKSPACE_HEADERS + ["Checkpoint Storage Config"]
    render.tabulate_or_csv(headers, values, False)


def list_workspaces(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    orderArg = bindings.v1OrderBy[args.order_by.upper()]
    sortArg = bindings.v1GetWorkspacesRequestSortBy[args.sort_by.upper()]
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
        render.print_json([w.to_json() for w in all_workspaces])
    else:
        render_workspaces(all_workspaces, from_list_api=True)


def list_workspace_projects(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    all_projects = workspace.Workspace(
        session=sess, workspace_name=args.workspace_name
    ).list_projects()

    sort_key = args.sort_by
    sort_order = args.order_by
    offset = args.offset or 0  # No passed offset is interpreted as a 0 offset
    limit = args.limit

    # TODO: Remove typechecking suppression when mypy is upgraded to 1.4.0
    all_projects.sort(
        key=lambda p: getattr(p, sort_key),  # type: ignore
        reverse=sort_order == "desc",
    )
    projects = all_projects[offset : offset + limit]

    if args.json:
        render.print_json([render.project_to_json(p) for p in projects])
    else:
        values = [
            [
                p.id,
                p.name,
                p.description,
                p.n_experiments,
                p.n_active_experiments,
            ]
            for p in projects
        ]
        render.tabulate_or_csv(PROJECT_HEADERS, values, False)


def list_pools(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    w = api.workspace_by_name(sess, args.workspace_name)
    resp = bindings.get_ListRPsBoundToWorkspace(sess, workspaceId=w.id)
    pools_str = ""
    if resp.resourcePools:
        pools_str = ", ".join(resp.resourcePools)

    render.tabulate_or_csv(
        headers=["Workspace", "Available Resource Pools"],
        values=[[args.workspace_name, pools_str]],
        as_csv=False,
    )


def _parse_agent_user_group_args(args: argparse.Namespace) -> Optional[bindings.v1AgentUserGroup]:
    if args.agent_uid or args.agent_gid or args.agent_user or args.agent_group:
        return bindings.v1AgentUserGroup(
            agentUid=args.agent_uid,
            agentGid=args.agent_gid,
            agentUser=args.agent_user,
            agentGroup=args.agent_group,
        )
    return None


def _parse_checkpoint_storage_args(args: argparse.Namespace) -> Any:
    if (args.checkpoint_storage_config is not None) and (
        args.checkpoint_storage_config_file is not None
    ):
        raise api.errors.BadRequestException(
            "can only provide --checkpoint_storage_config or --checkpoint_storage_config_file"
        )
    checkpoint_storage = args.checkpoint_storage_config_file
    if args.checkpoint_storage_config is not None:
        checkpoint_storage = json.loads(args.checkpoint_storage_config)
    return checkpoint_storage


def create_workspace(args: argparse.Namespace) -> None:
    agent_user_group = _parse_agent_user_group_args(args)
    checkpoint_storage = _parse_checkpoint_storage_args(args)

    sess = cli.setup_session(args)
    content = bindings.v1PostWorkspaceRequest(
        name=args.name,
        agentUserGroup=agent_user_group,
        checkpointStorageConfig=checkpoint_storage,
        defaultComputePool=args.default_compute_pool,
        defaultAuxPool=args.default_aux_pool,
    )
    w = bindings.post_PostWorkspace(sess, body=content).workspace

    if args.json:
        render.print_json(w.to_json())
    else:
        render_workspaces([w])


def describe_workspace(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    w = api.workspace_by_name(sess, args.workspace_name)
    if args.json:
        render.print_json(w.to_json())
    else:
        render_workspaces([w])


def delete_workspace(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    w = api.workspace_by_name(sess, args.workspace_name)
    if args.yes or render.yes_or_no(
        'Deleting workspace "' + args.workspace_name + '" will result \n'
        "in the unrecoverable deletion of all associated projects, experiments,\n"
        "Notebooks, shells, commands, Tensorboards, and Templates.\n"
        "For a recoverable alternative, see the 'archive' command. Do you still \n"
        "wish to proceed?"
    ):
        resp = bindings.delete_DeleteWorkspace(sess, id=w.id)
        if resp.completed:
            print(f"Successfully deleted workspace {args.workspace_name}.")
        else:
            print(f"Started deletion of workspace {args.workspace_name}...")
            while True:
                time.sleep(2)
                try:
                    w = bindings.get_GetWorkspace(sess, id=w.id).workspace
                    if w.state == bindings.v1WorkspaceState.DELETE_FAILED:
                        raise errors.DeleteFailedException(w.errorMessage)
                    elif w.state == bindings.v1WorkspaceState.DELETING:
                        print(f"Remaining project count: {w.numProjects}")
                except errors.NotFoundException:
                    print("Workspace deleted successfully.")
                    break
    else:
        print("Aborting workspace deletion.")


def archive_workspace(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    current = api.workspace_by_name(sess, args.workspace_name)
    bindings.post_ArchiveWorkspace(sess, id=current.id)
    print(f"Successfully archived workspace {args.workspace_name}.")


def unarchive_workspace(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    current = api.workspace_by_name(sess, args.workspace_name)
    bindings.post_UnarchiveWorkspace(sess, id=current.id)
    print(f"Successfully un-archived workspace {args.workspace_name}.")


def edit_workspace(args: argparse.Namespace) -> None:
    checkpoint_storage = _parse_checkpoint_storage_args(args)

    sess = cli.setup_session(args)
    current = api.workspace_by_name(sess, args.workspace_name)
    agent_user_group = _parse_agent_user_group_args(args)
    updated = bindings.v1PatchWorkspace(
        name=args.name,
        agentUserGroup=agent_user_group,
        checkpointStorageConfig=checkpoint_storage,
        defaultComputeResourcePool=args.default_compute_pool,
        defaultAuxResourcePool=args.default_aux_pool,
    )
    w = bindings.patch_PatchWorkspace(sess, body=updated, id=current.id).workspace

    if args.json:
        render.print_json(w.to_json())
    else:
        render_workspaces([w])


def yaml_file_arg(val: str) -> Any:
    with open(val) as f:
        return util.safe_load_yaml_with_exceptions(f)


CHECKPOINT_STORAGE_WORKSPACE_ARGS = [
    cli.Arg(
        "--checkpoint-storage-config",
        type=str,
        help="Storage config (JSON-formatted string). To remove storage config use '{}'",
    ),
    cli.Arg(
        "--checkpoint-storage-config-file",
        type=yaml_file_arg,
        help="Storage config (path to YAML or JSON formatted file)",
    ),
]

DEFAULT_POOL_ARGS = [
    cli.Arg(
        "--default-compute-pool",
        type=str,
        help="name of the pool to set as the default compute pool",
    ),
    cli.Arg(
        "--default-aux-pool",
        type=str,
        help="name of the pool to set as the default auxiliary pool",
    ),
]


# do not use util.py's pagination_args because behavior here is
# to hide pagination and unify all pages of experiments into one output
pagination_args = [
    cli.Arg(
        "--limit",
        type=int,
        default=200,
        help="Maximum items per page of results",
    ),
    cli.Arg(
        "--offset",
        type=int,
        default=None,
        help="Number of items to skip before starting page of results",
    ),
]
args_description = [
    cli.Cmd(
        "w|orkspace",
        None,
        "manage workspaces",
        [
            cli.Cmd(
                "list ls",
                list_workspaces,
                "list all workspaces",
                [
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
                    *pagination_args,
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
                is_default=True,
            ),
            cli.Cmd(
                "list-projects",
                list_workspace_projects,
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
                    *pagination_args,
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "list-pools",
                list_pools,
                "list the resource pools available to a workspace",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                ],
            ),
            cli.Cmd(
                "create",
                create_workspace,
                "create workspace",
                [
                    cli.Arg("name", type=str, help="unique name of the workspace"),
                    *user.AGENT_USER_GROUP_ARGS,
                    *CHECKPOINT_STORAGE_WORKSPACE_ARGS,
                    *DEFAULT_POOL_ARGS,
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "delete",
                delete_workspace,
                "delete workspace",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg(
                        "--yes",
                        action="store_true",
                        default=False,
                        help="automatically answer yes to prompts",
                    ),
                ],
            ),
            cli.Cmd(
                "describe",
                describe_workspace,
                "describe workspace",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "edit",
                edit_workspace,
                "edit workspace",
                [
                    cli.Arg("workspace_name", type=str, help="current name of the workspace"),
                    cli.Arg("--name", type=str, help="new name of the workspace"),
                    *user.AGENT_USER_GROUP_ARGS,
                    *CHECKPOINT_STORAGE_WORKSPACE_ARGS,
                    *DEFAULT_POOL_ARGS,
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "archive",
                archive_workspace,
                "archive workspace",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                ],
            ),
            cli.Cmd(
                "unarchive",
                unarchive_workspace,
                "unarchive workspace",
                [
                    cli.Arg("workspace_name", type=str, help="name of the workspace"),
                ],
            ),
        ],
    )
]  # type: List[Any]
