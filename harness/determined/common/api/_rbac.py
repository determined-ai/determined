from typing import Dict, List, Optional

from determined.common import api
from determined.common.api import bindings


def role_name_to_role_id(session: api.Session, role_name: str) -> int:
    # No need to use read-paginated since the number of roles is fixed and small.
    req = bindings.v1ListRolesRequest(limit=500, offset=0)
    resp = bindings.post_ListRoles(session=session, body=req)
    for r in resp.roles:
        if r.name == role_name and r.roleId is not None:
            return r.roleId
    raise api.errors.BadRequestException(f"could not find role name {role_name}")


def create_user_assignment_request(
    session: api.Session, user: str, role: str, workspace: Optional[str] = None
) -> List[bindings.v1UserRoleAssignment]:
    role_obj = bindings.v1Role(roleId=role_name_to_role_id(session, role))
    workspace_id = None
    if workspace is not None:
        workspace_id = workspace_by_name(session, workspace).id
    role_assign = bindings.v1RoleAssignment(role=role_obj, scopeWorkspaceId=workspace_id)
    user_id = usernames_to_user_ids(session, [user])[0]
    return [bindings.v1UserRoleAssignment(userId=user_id, roleAssignment=role_assign)]


def create_group_assignment_request(
    session: api.Session, group: str, role: str, workspace: Optional[str] = None
) -> List[bindings.v1GroupRoleAssignment]:
    role_obj = bindings.v1Role(roleId=role_name_to_role_id(session, role))
    workspace_id = None
    if workspace is not None:
        workspace_id = workspace_by_name(session, workspace).id
    role_assign = bindings.v1RoleAssignment(role=role_obj, scopeWorkspaceId=workspace_id)
    group_id = group_name_to_group_id(session, group)
    return [bindings.v1GroupRoleAssignment(groupId=group_id, roleAssignment=role_assign)]


def usernames_to_user_ids(session: api.Session, usernames: List[str]) -> List[int]:
    usernames_to_ids: Dict[str, Optional[int]] = dict.fromkeys(usernames, None)
    users = bindings.get_GetUsers(session).users or []
    for user in users:
        if user.username in usernames_to_ids:
            usernames_to_ids[user.username] = user.id

    missing_users = []
    user_ids = []
    for username, user_id in usernames_to_ids.items():
        if user_id is None:
            missing_users.append(username)
        else:
            user_ids.append(user_id)

    if missing_users:
        raise api.errors.BadRequestException(
            f"could not find users for usernames {', '.join(missing_users)}"
        )
    return user_ids


def group_name_to_group_id(session: api.Session, group_name: str) -> int:
    body = bindings.v1GetGroupsRequest(name=group_name, limit=1, offset=0)
    resp = bindings.post_GetGroups(session, body=body)
    groups = resp.groups
    if groups is None or len(groups) != 1 or groups[0].group.groupId is None:
        raise api.errors.BadRequestException(f"could not find user group name {group_name}")
    return groups[0].group.groupId


def workspace_by_name(session: api.Session, name: str) -> bindings.v1Workspace:
    assert name, "workspace name cannot be empty"
    w = bindings.get_GetWorkspaces(session, nameCaseSensitive=name).workspaces
    assert len(w) <= 1, "workspace name is assumed to be unique."
    if len(w) == 0:
        raise not_found_errs("workspace", name, session)
    return bindings.get_GetWorkspace(session, id=w[0].id).workspace


def not_found_errs(
    category: str, name: str, session: api.Session
) -> api.errors.BadRequestException:
    """Construct a NotFoundException from passed strings.

    This function creates NotFoundExceptions with the same syntax as the NotFoundErrs from
    api/errors.go so that messages from both layers look the same. If RBAC is enabled, the
    exceptions underlying the 404s that this function constructs might be the result of a
    permissions error. In that event, this function appends a suggestion to the constructed
    exception string that the user to check permissions.
    """
    resp = bindings.get_GetMaster(session)
    msg = f"{category} '{name}' not found"
    if not resp.to_json().get("rbacEnabled"):
        return api.errors.NotFoundException(msg)
    return api.errors.NotFoundException(msg + ", please check your permissions.")
