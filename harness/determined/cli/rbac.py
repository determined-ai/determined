import argparse
import collections
from typing import Any, Dict, List, Set, Tuple

from determined import cli
from determined.cli import render
from determined.common import api
from determined.common.api import bindings

rbac_flag_disabled_message = (
    "RBAC commands require the Determined Enterprise Edition "
    + "and the Master Configuration option security.authz.rbac_ui_enabled."
)

v1PermissionHeaders = collections.namedtuple(
    "v1PermissionHeaders",
    ["id", "name", "scopeTypeMask"],
)

roleAssignmentHeaders = collections.namedtuple(
    "roleAssignmentHeaders",
    [
        "roleName",
        "roleID",
        "assignedDirectlyToUser",
        "assignedToGroupName",
        "assignedToGroupID",
        "scopeCluster",
        "scopeWorkspaceName",
        "scopeWorkspaceID",
    ],
)

workspaceAssignedToHeaders = collections.namedtuple(
    "workspaceAssignedToHeaders",
    ["assignedGlobally", "workspaceID", "workspaceName"],
)

groupAssignmentHeaders = collections.namedtuple(
    "groupAssignmentHeaders",
    ["groupID", "groupName", "workspaceID", "workspaceName", "assignedGlobally"],
)


userAssignmentHeaders = collections.namedtuple(
    "userAssignmentHeaders",
    ["userID", "username", "workspaceID", "workspaceName", "assignedGlobally"],
)


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def my_permissions(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    resp = bindings.get_GetPermissionsSummary(sess)
    if args.json:
        render.print_json(resp.to_json())
        return

    role_id_to_permissions: Dict[int, Set[bindings.v1Permission]] = {}
    for r in resp.roles:
        if r.permissions is not None and r.roleId is not None:
            role_id_to_permissions[r.roleId] = set(r.permissions)

    scope_id_to_permissions: Dict[int, Set[bindings.v1Permission]] = {}
    for a in resp.assignments:
        if a.roleId is None:
            raise api.errors.BadResponseException("expected roleId to be provided")

        if a.scopeCluster:
            if 0 not in scope_id_to_permissions:
                scope_id_to_permissions[0] = set()
            scope_id_to_permissions[0].update(role_id_to_permissions[a.roleId])

        if a.scopeWorkspaceIds is None:
            a.scopeWorkspaceIds = []
        for wid in a.scopeWorkspaceIds:
            if wid not in scope_id_to_permissions:
                scope_id_to_permissions[wid] = set()
            scope_id_to_permissions[wid].update(role_id_to_permissions[a.roleId])

    if len(scope_id_to_permissions) == 0:
        print("no permissions assigned")
        return
    for wid, perms in scope_id_to_permissions.items():
        if wid == 0:
            print("global permissions assigned")
        else:
            workspace_name = bindings.get_GetWorkspace(sess, id=wid).workspace.name
            print(f"permissions assigned over workspace '{workspace_name}' with ID '{wid}'")

        perms_to_render = []
        perms_added: Set[bindings.v1PermissionType] = set()
        for p in perms:
            if p.id not in perms_added:
                perms_added.add(p.id)
                perms_to_render.append(render.unmarshal(v1PermissionHeaders, p.to_json()))
        render.render_objects(v1PermissionHeaders, perms_to_render)
        print()


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def list_roles(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    req = bindings.v1SearchRolesAssignableToScopeRequest(
        limit=args.limit,
        offset=args.offset,
        workspaceId=1 if args.exclude_global_roles else None,
    )
    resp = bindings.post_SearchRolesAssignableToScope(sess, body=req)
    if args.json:
        render.print_json(resp.to_json())
        return

    if resp.roles is None or len(resp.roles) == 0:
        print("no roles found")
        return
    for r in resp.roles:
        print(f"role '{r.name}' with ID {r.roleId} with permissions")
        if r.permissions is None:
            print("role has no permissions assigned")
            continue

        render.render_objects(
            v1PermissionHeaders,
            [render.unmarshal(v1PermissionHeaders, p.to_json()) for p in r.permissions],
        )
        print()


def role_with_assignment_to_dict(
    session: api.Session,
    r: bindings.v1RoleWithAssignments,
    assignment: bindings.v1RoleAssignment,
) -> Dict[str, Any]:
    scope_cluster = assignment.scopeCluster
    workspace_id = assignment.scopeWorkspaceId
    workspace_name = None
    if workspace_id is not None:
        workspace_name = bindings.get_GetWorkspace(session, id=workspace_id).workspace.name

    if not r.role:  # This should not happen.
        return {}
    return {
        "roleName": r.role.name,
        "roleID": r.role.roleId,
        "assignedDirectlyToUser": False,
        "assignedToGroupID": None,
        "assignedToGroupName": None,
        "scopeCluster": scope_cluster,
        "scopeWorkspaceID": workspace_id,
        "scopeWorkspaceName": workspace_name,
    }


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def list_users_roles(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    user_id = api.usernames_to_user_ids(sess, [args.username])[0]
    resp = bindings.get_GetRolesAssignedToUser(sess, userId=user_id)
    if args.json:
        render.print_json(resp.to_json())
        return

    if resp.roles is None or len(resp.roles) == 0:
        print("user has no role assignments")
        return

    output = []
    for r in resp.roles:
        if r.userRoleAssignments is not None:
            for u in r.userRoleAssignments:
                o = role_with_assignment_to_dict(sess, r, u.roleAssignment)
                o["assignedDirectlyToUser"] = True
                output.append(o)
        if r.groupRoleAssignments is not None:
            for g in r.groupRoleAssignments:
                o = role_with_assignment_to_dict(sess, r, g.roleAssignment)
                o["assignedToGroupID"] = g.groupId
                o["assignedToGroupName"] = bindings.get_GetGroup(sess, groupId=g.groupId).group.name
                output.append(o)

    render.render_objects(
        roleAssignmentHeaders,
        [render.unmarshal(roleAssignmentHeaders, o) for o in output],
    )


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def list_groups_roles(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    group_id = api.group_name_to_group_id(sess, args.group_name)
    resp = bindings.get_GetRolesAssignedToGroup(sess, groupId=group_id)
    if args.json:
        render.print_json(resp.to_json())
        return

    if resp.roles is None or len(resp.roles) == 0:
        print("group has no role assignments")
        return

    for i, r in enumerate(resp.roles):
        workspaces = []  # type: List[Dict[str, Any]]
        if resp.assignments[i].scopeCluster:
            workspaces.append(
                {"workspaceID": None, "workspaceName": None, "assignedGlobally": True}
            )

        workspace_ids = resp.assignments[i].scopeWorkspaceIds or []
        for wid in workspace_ids:
            workspace_name = bindings.get_GetWorkspace(sess, id=wid).workspace.name
            workspaces.append(
                {
                    "workspaceID": wid,
                    "workspaceName": workspace_name,
                    "assignedGlobally": False,
                }
            )

        print(f"role '{r.name}' with ID {r.roleId} assigned")
        render.render_objects(
            workspaceAssignedToHeaders,
            [render.unmarshal(workspaceAssignedToHeaders, w) for w in workspaces],
        )
        print()


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def describe_role(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    role_id = api.role_name_to_role_id(sess, args.role_name)
    req = bindings.v1GetRolesByIDRequest(roleIds=[role_id])
    resp = bindings.post_GetRolesByID(sess, body=req)
    if args.json:
        render.print_json(resp.roles[0].to_json() if resp.roles else None)
        return

    if resp.roles is None or len(resp.roles) != 1:
        raise api.errors.BadRequestException(f"could not find role name {args.role_name}")

    role = resp.roles[0].role
    if role is None:
        raise api.errors.BadResponseException("expected role to be provided")

    print(f"role '{role.name}' with ID {role.roleId} with permissions")
    if role.permissions is None:
        print("role has no permissions assigned")
    else:
        render.render_objects(
            v1PermissionHeaders,
            [render.unmarshal(v1PermissionHeaders, p.to_json()) for p in role.permissions],
        )
        print()

    group_assignments = resp.roles[0].groupRoleAssignments
    if group_assignments is None or len(group_assignments) == 0:
        print("role is not assigned to any group")
    else:
        print("role is assigned to groups")
        output = []
        for group_assignment in group_assignments:
            workspace_id = group_assignment.roleAssignment.scopeWorkspaceId
            workspace_name = None
            group_name = bindings.get_GetGroup(sess, groupId=group_assignment.groupId).group.name
            if workspace_id is not None:
                workspace_name = bindings.get_GetWorkspace(sess, id=workspace_id).workspace.name

            output.append(
                {
                    "groupID": group_assignment.groupId,
                    "groupName": group_name,
                    "workspaceID": workspace_id,
                    "workspaceName": workspace_name,
                    "assignedGlobally": workspace_id is None,
                }
            )
        render.render_objects(
            groupAssignmentHeaders, [render.unmarshal(groupAssignmentHeaders, o) for o in output]
        )
        print()

    user_assignments = resp.roles[0].userRoleAssignments
    if user_assignments is None or len(user_assignments) == 0:
        print("role is not assigned to any users")
    else:
        print("role is assigned to users")
        output = []
        for user_assignment in user_assignments:
            workspace_id = user_assignment.roleAssignment.scopeWorkspaceId
            workspace_name = None
            username = bindings.get_GetUser(sess, userId=user_assignment.userId).user.username
            if workspace_id is not None:
                workspace_name = bindings.get_GetWorkspace(sess, id=workspace_id).workspace.name

            output.append(
                {
                    "userID": user_assignment.userId,
                    "username": username,
                    "workspaceID": workspace_id,
                    "workspaceName": workspace_name,
                    "assignedGlobally": workspace_id is None,
                }
            )

        render.render_objects(
            userAssignmentHeaders, [render.unmarshal(userAssignmentHeaders, o) for o in output]
        )
        print()


def make_assign_req(
    session: api.Session,
    args: argparse.Namespace,
) -> Tuple[List[bindings.v1UserRoleAssignment], List[bindings.v1GroupRoleAssignment]]:
    """
    A helper for assign_role and unassign_role, which take the same command line flags.
    """
    if args.username_to_assign:
        user_assign = api.create_user_assignment_request(
            session,
            user=args.username_to_assign,
            role=args.role_name,
            workspace=args.workspace_name,
        )
    else:
        user_assign = []

    if args.group_name_to_assign:
        group_assign = api.create_group_assignment_request(
            session,
            group=args.group_name_to_assign,
            role=args.role_name,
            workspace=args.workspace_name,
        )
    else:
        group_assign = []

    return user_assign, group_assign


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def assign_role(args: argparse.Namespace) -> None:
    # Valid CLI usage is enforced before even creating a session.
    if (args.username_to_assign is None) == (args.group_name_to_assign is None):
        raise api.errors.BadRequestException(
            "must provide exactly one of --username-to-assign or --group-name-to-assign"
        )

    sess = cli.setup_session(args)
    user_assign, group_assign = make_assign_req(sess, args)
    req = bindings.v1AssignRolesRequest(
        userRoleAssignments=user_assign, groupRoleAssignments=group_assign
    )
    bindings.post_AssignRoles(sess, body=req)

    scope = " globally"
    if args.workspace_name:
        scope = f" to workspace {args.workspace_name}"
    if len(user_assign) > 0:
        role_id = user_assign[0].roleAssignment.role.roleId
        print(
            f"assigned role '{args.role_name}' with ID {role_id} "
            + f"to user '{args.username_to_assign}' with ID {user_assign[0].userId}{scope}"
        )
    else:
        role_id = group_assign[0].roleAssignment.role.roleId
        print(
            f"assigned role '{args.role_name}' with ID {role_id} "
            + f"to group '{args.group_name_to_assign}' with ID {group_assign[0].groupId}{scope}"
        )


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def unassign_role(args: argparse.Namespace) -> None:
    # Valid CLI usage is enforced before even creating a session.
    if (args.username_to_assign is None) == (args.group_name_to_assign is None):
        raise api.errors.BadRequestException(
            "must provide exactly one of --username-to-assign or --group-name-to-assign"
        )

    sess = cli.setup_session(args)
    user_assign, group_assign = make_assign_req(sess, args)
    req = bindings.v1RemoveAssignmentsRequest(
        userRoleAssignments=user_assign, groupRoleAssignments=group_assign
    )
    bindings.post_RemoveAssignments(sess, body=req)

    scope = " globally"
    if args.workspace_name:
        scope = f" to workspace {args.workspace_name}"
    if len(user_assign) > 0:
        print(
            f"removed role '{args.role_name}' with ID {user_assign[0].roleAssignment.role.roleId} "
            f"from user '{args.username_to_assign}' with ID {user_assign[0].userId}{scope}"
        )
    else:
        print(
            f"removed role '{args.role_name}' with ID {group_assign[0].roleAssignment.role.roleId} "
            f"from group '{args.group_name_to_assign}' with ID {group_assign[0].groupId}{scope}"
        )


args_description = [
    cli.Cmd(
        "rbac",
        None,
        "manage roles based access controls",
        [
            cli.Cmd(
                "my-permissions",
                my_permissions,
                "list permissions the current user has",
                [
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "list-roles",
                list_roles,
                "list roles",
                [
                    cli.Arg(
                        "--exclude-global-roles",
                        action="store_true",
                        help="Ignore roles with global permissions",
                    ),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                    *cli.make_pagination_args(),
                ],
                is_default=True,
            ),
            cli.Cmd(
                "list-users-roles",
                list_users_roles,
                "list user's roles",
                [
                    cli.Arg("username", help="username of user to list role's assigned to"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "list-groups-roles",
                list_groups_roles,
                "list group's roles",
                [
                    cli.Arg(
                        "group_name", help="name of the group for which to list assigned roles"
                    ),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "describe-role",
                describe_role,
                "describe a role",
                [
                    cli.Arg("role_name", help="name of role to describe"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "assign-role",
                assign_role,
                "assign a role to a user or group",
                [
                    cli.Arg("role_name", help="name of role to assign"),
                    cli.Arg(
                        "-w",
                        "--workspace-name",
                        default=None,
                        help="name of the workspace the role is assigned to",
                    ),
                    cli.Arg(
                        "-u",
                        "--username-to-assign",
                        default=None,
                        help="username to assign the role to",
                    ),
                    cli.Arg(
                        "-g",
                        "--group-name-to-assign",
                        default=None,
                        help="name of the group the role is assigned to",
                    ),
                ],
            ),
            cli.Cmd(
                "unassign-role",
                unassign_role,
                "unassign a role from a user or group",
                [
                    cli.Arg("role_name", help="name of role to unassign"),
                    cli.Arg(
                        "-w",
                        "--workspace-name",
                        default=None,
                        help="name of the workspace the role is unassigned from",
                    ),
                    cli.Arg(
                        "-u",
                        "--username-to-assign",
                        default=None,
                        help="username the role is unassigned from",
                    ),
                    cli.Arg(
                        "-g",
                        "--group-name-to-assign",
                        default=None,
                        help="name of the group the role is unassigned from",
                    ),
                ],
            ),
        ],
    )
]  # type: List[Any]
