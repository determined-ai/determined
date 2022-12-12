import json
from argparse import Namespace
from collections import namedtuple
from typing import Any, Dict, List, Set, Tuple

from determined.cli import (
    default_pagination_args,
    render,
    require_feature_flag,
    setup_session,
    user_groups,
    workspace,
)
from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.experimental import session

rbac_flag_disabled_message = (
    "RBAC commands require the Determined Enterprise Edition "
    + "and the Master Configuration option security.authz.rbac_ui_enabled."
)

v1PermissionHeaders = namedtuple(
    "v1PermissionHeaders",
    ["id", "name", "scopeTypeMask"],
)

roleAssignmentHeaders = namedtuple(
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

workspaceAssignedToHeaders = namedtuple(
    "workspaceAssignedToHeaders",
    ["assignedGlobally", "workspaceID", "workspaceName"],
)

groupAssignmentHeaders = namedtuple(
    "groupAssignmentHeaders",
    ["groupID", "groupName", "workspaceID", "workspaceName", "assignedGlobally"],
)


userAssignmentHeaders = namedtuple(
    "userAssignmentHeaders",
    ["userID", "username", "workspaceID", "workspaceName", "assignedGlobally"],
)


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def my_permissions(args: Namespace) -> None:
    session = setup_session(args)
    resp = bindings.get_GetPermissionsSummary(session)
    if args.json:
        print(json.dumps(resp.to_json(), indent=2))
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
            workspace_name = bindings.get_GetWorkspace(session, id=wid).workspace.name
            print(f"permissions assigned over workspace '{workspace_name}' with ID '{wid}'")

        render.render_objects(
            v1PermissionHeaders, [render.unmarshal(v1PermissionHeaders, p.to_json()) for p in perms]
        )
        print()


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def list_roles(args: Namespace) -> None:
    req = bindings.v1SearchRolesAssignableToScopeRequest(
        limit=args.limit,
        offset=args.offset,
        workspaceId=1 if args.exclude_global_roles else None,
    )
    resp = bindings.post_SearchRolesAssignableToScope(setup_session(args), body=req)
    if args.json:
        print(json.dumps(resp.to_json(), indent=2))
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
    session: session.Session,
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


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def list_users_roles(args: Namespace) -> None:
    session = setup_session(args)
    user_id = user_groups.usernames_to_user_ids(session, [args.username])[0]
    resp = bindings.get_GetRolesAssignedToUser(session, userId=user_id)
    if args.json:
        print(json.dumps(resp.to_json(), indent=2))
        return

    if resp.roles is None or len(resp.roles) == 0:
        print("user has no role assignments")
        return

    output = []
    for r in resp.roles:
        if r.userRoleAssignments is not None:
            for u in r.userRoleAssignments:
                o = role_with_assignment_to_dict(session, r, u.roleAssignment)
                o["assignedDirectlyToUser"] = True
                output.append(o)
        if r.groupRoleAssignments is not None:
            for g in r.groupRoleAssignments:
                o = role_with_assignment_to_dict(session, r, g.roleAssignment)
                o["assignedToGroupID"] = g.groupId
                o["assignedToGroupName"] = bindings.get_GetGroup(
                    session, groupId=g.groupId
                ).group.name
                output.append(o)

    render.render_objects(
        roleAssignmentHeaders,
        [render.unmarshal(roleAssignmentHeaders, o) for o in output],
    )


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def list_groups_roles(args: Namespace) -> None:
    session = setup_session(args)
    group_id = user_groups.group_name_to_group_id(session, args.group_name)
    resp = bindings.get_GetRolesAssignedToGroup(session, groupId=group_id)
    if args.json:
        print(json.dumps(resp.to_json(), indent=2))
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
            workspace_name = bindings.get_GetWorkspace(session, id=wid).workspace.name
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


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def describe_role(args: Namespace) -> None:
    session = setup_session(args)
    role_id = role_name_to_role_id(session, args.role_name)
    req = bindings.v1GetRolesByIDRequest(roleIds=[role_id])
    resp = bindings.post_GetRolesByID(session, body=req)
    if args.json:
        print(json.dumps(resp.roles[0].to_json() if resp.roles else None, indent=2))
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
            group_name = bindings.get_GetGroup(session, groupId=group_assignment.groupId).group.name
            if workspace_id is not None:
                workspace_name = bindings.get_GetWorkspace(session, id=workspace_id).workspace.name

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
            username = bindings.get_GetUser(session, userId=user_assignment.userId).user.username
            if workspace_id is not None:
                workspace_name = bindings.get_GetWorkspace(session, id=workspace_id).workspace.name

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


def create_assignment_request(
    session: session.Session, args: Namespace
) -> Tuple[List[bindings.v1UserRoleAssignment], List[bindings.v1GroupRoleAssignment]]:
    if (args.username_to_assign is None) == (args.group_name_to_assign is None):
        raise api.errors.BadRequestException(
            "must provide exactly one of --username-to-assign or --group-name-to-assign"
        )

    role = bindings.v1Role(roleId=role_name_to_role_id(session, args.role_name))

    workspace_id = None
    if args.workspace_name is not None:
        workspace_id = workspace.workspace_by_name(session, args.workspace_name).id
    role_assign = bindings.v1RoleAssignment(role=role, scopeWorkspaceId=workspace_id)

    if args.username_to_assign is not None:
        user_id = user_groups.usernames_to_user_ids(session, [args.username_to_assign])[0]
        return [bindings.v1UserRoleAssignment(userId=user_id, roleAssignment=role_assign)], []

    group_id = user_groups.group_name_to_group_id(session, args.group_name_to_assign)
    return [], [bindings.v1GroupRoleAssignment(groupId=group_id, roleAssignment=role_assign)]


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def assign_role(args: Namespace) -> None:
    session = setup_session(args)
    user_assign, group_assign = create_assignment_request(session, args)
    req = bindings.v1AssignRolesRequest(
        userRoleAssignments=user_assign, groupRoleAssignments=group_assign
    )
    bindings.post_AssignRoles(session, body=req)

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


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def unassign_role(args: Namespace) -> None:
    session = setup_session(args)
    user_assign, group_assign = create_assignment_request(session, args)
    req = bindings.v1RemoveAssignmentsRequest(
        userRoleAssignments=user_assign, groupRoleAssignments=group_assign
    )
    bindings.post_RemoveAssignments(session, body=req)

    scope = " globally"
    if args.workspace_name:
        scope = f" to workspace {args.workspace_name}"
    if len(user_assign) > 0:
        print(
            f"removed role '{args.role_name}' with ID {user_assign[0].roleAssignment.role.roleId} "
            + f"from user '{args.username_to_assign}' with ID {user_assign[0].userId}{scope}"
        )
    else:
        print(
            f"removed role '{args.role_name}' with ID {group_assign[0].roleAssignment.role.roleId} "
            + f"from group '{args.group_name_to_assign}' with ID {group_assign[0].groupId}{scope}"
        )


def role_name_to_role_id(session: session.Session, role_name: str) -> int:
    req = bindings.v1ListRolesRequest(limit=499, offset=0)
    resp = bindings.post_ListRoles(session=session, body=req)
    for r in resp.roles:
        if r.name == role_name and r.roleId is not None:
            return r.roleId
    raise api.errors.BadRequestException(f"could not find role name {role_name}")


args_description = [
    Cmd(
        "rbac",
        None,
        "manage roles based access controls",
        [
            Cmd(
                "my-permissions",
                my_permissions,
                "list permissions the current user has",
                [
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "list-roles",
                list_roles,
                "list roles",
                [
                    Arg(
                        "--exclude-global-roles",
                        action="store_true",
                        help="Ignore roles with global permissions",
                    ),
                    Arg("--json", action="store_true", help="print as JSON"),
                    *default_pagination_args,
                ],
                is_default=True,
            ),
            Cmd(
                "list-users-roles",
                list_users_roles,
                "list user's roles",
                [
                    Arg("username", help="username of user to list role's assigned to"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "list-groups-roles",
                list_groups_roles,
                "list group's roles",
                [
                    Arg("group_name", help="name of the group for which to list assigned roles"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "describe-role",
                describe_role,
                "describe a role",
                [
                    Arg("role_name", help="name of role to describe"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "assign-role",
                assign_role,
                "assign a role to a user or group",
                [
                    Arg("role_name", help="name of role to assign"),
                    Arg(
                        "-w",
                        "--workspace-name",
                        default=None,
                        help="name of the workspace the role is assigned to",
                    ),
                    Arg(
                        "-u",
                        "--username-to-assign",
                        default=None,
                        help="username to assign the role to",
                    ),
                    Arg(
                        "-g",
                        "--group-name-to-assign",
                        default=None,
                        help="name of the group the role is assigned to",
                    ),
                ],
            ),
            Cmd(
                "unassign-role",
                unassign_role,
                "unassign a role from a user or group",
                [
                    Arg("role_name", help="name of role to unassign"),
                    Arg(
                        "-w",
                        "--workspace-name",
                        default=None,
                        help="name of the workspace the role is unassigned from",
                    ),
                    Arg(
                        "-u",
                        "--username-to-assign",
                        default=None,
                        help="username the role is unassigned from",
                    ),
                    Arg(
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
