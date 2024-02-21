import contextlib
from typing import Any, Dict, Generator, List, NamedTuple, Optional, Tuple

import pytest

from determined.common import api
from determined.common.api import bindings
from tests import api_utils, detproc
from tests.cluster import test_workspace_org

PermCase = NamedTuple("PermCase", [("sess", api.Session), ("raises", Optional[Any])])


@contextlib.contextmanager
def create_workspaces_with_users(
    assignments_list: List[List[Tuple[int, List[str]]]]
) -> Generator[Tuple[List[bindings.v1Workspace], Dict[int, api.Session]], None, None]:
    """
    Set up workspaces and users with the provided role assignments.
    For example the following sets up 2 workspaces and 2 users referenced
    with the integer ids 1 and 2. User 1 has the roles Editor and Viewer on
    workspace 1 and the role Viewer on workspace 2. User 2 has the role Viewer
    on workspace 1 and no roles on workspace 2.
    perm_assigments = [
        [
            (1, ["Editor", "Viewer"]),
            (2, ["Viewer"]),
        ],
        [
            (1, ["Viewer"]),
        ]
    ]
    """
    sess = api_utils.admin_session()
    rid_to_sess: Dict[int, api.Session] = {}
    with test_workspace_org.setup_workspaces(count=len(assignments_list)) as workspaces:
        for workspace, user_list in zip(workspaces, assignments_list):
            for rid, roles in user_list:
                if rid not in rid_to_sess:
                    rid_to_sess[rid], _ = api_utils.create_test_user()
                for role in roles:
                    api_utils.assign_user_role(
                        session=sess,
                        user=rid_to_sess[rid].username,
                        role=role,
                        workspace=workspace.name,
                    )
        yield workspaces, rid_to_sess


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_user_role_setup() -> None:
    perm_assigments = [
        [
            (1, ["Editor", "Viewer"]),
            (2, ["Viewer"]),
        ],
        [
            (1, ["Viewer"]),
        ],
    ]
    with create_workspaces_with_users(perm_assigments) as (workspaces, rid_to_sess):
        assert len(rid_to_sess) == 2
        assert len(workspaces) == 2


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_permission_assignment() -> None:
    admin = api_utils.admin_session()
    sess, _ = api_utils.create_test_user()

    # User has no permissions.
    assert "no permissions" in detproc.check_output(sess, ["det", "rbac", "my-permissions"])
    json_out = detproc.check_json(sess, ["rbac", "my-permissions", "--json"])
    assert json_out["roles"] == []
    assert json_out["assignments"] == []

    group_name = api_utils.get_random_string()
    # Assign user to role directly.
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "WorkspaceCreator",
            "--username-to-assign",
            sess.username,
        ],
    )
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "Viewer",
            "--username-to-assign",
            sess.username,
            "--workspace-name",
            "Uncategorized",
        ],
    )

    # Assign user to a group with roles.
    detproc.check_call(
        admin, ["det", "user-group", "create", group_name, "--add-user", sess.username]
    )
    detproc.check_call(
        admin,
        ["det", "rbac", "assign-role", "WorkspaceCreator", "--group-name-to-assign", group_name],
    )
    detproc.check_call(
        admin, ["det", "rbac", "assign-role", "Editor", "--group-name-to-assign", group_name]
    )
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "Editor",
            "--group-name-to-assign",
            group_name,
            "--workspace-name",
            "Uncategorized",
        ],
    )

    # User has those roles assigned.
    assert "no permissions" not in detproc.check_output(sess, ["det", "rbac", "my-permissions"])
    json_out = detproc.check_json(sess, ["det", "rbac", "my-permissions", "--json"])
    assert len(json_out["roles"]) == 3
    assert len(json_out["assignments"]) == 3

    creator = [role for role in json_out["roles"] if role["name"] == "WorkspaceCreator"]
    assert len(creator) == 1
    creator_assignment = [a for a in json_out["assignments"] if a["roleId"] == creator[0]["roleId"]]
    assert creator_assignment[0]["scopeWorkspaceIds"] == []
    assert creator_assignment[0]["scopeCluster"]

    viewer = [role for role in json_out["roles"] if role["name"] == "Viewer"]
    assert len(viewer) == 1
    viewer_assignment = [a for a in json_out["assignments"] if a["roleId"] == viewer[0]["roleId"]]
    assert viewer_assignment[0]["scopeWorkspaceIds"] == [1]
    assert not viewer_assignment[0]["scopeCluster"]

    editor = [role for role in json_out["roles"] if role["name"] == "Editor"]
    assert len(editor) == 1
    editor_assignment = [a for a in json_out["assignments"] if a["roleId"] == editor[0]["roleId"]]
    assert editor_assignment[0]["scopeWorkspaceIds"] == [1]
    assert editor_assignment[0]["scopeCluster"]

    # Remove from the group.
    detproc.check_call(admin, ["det", "user-group", "remove-user", group_name, sess.username])

    # User doesn't have any group roles assigned.
    assert "no permissions" not in detproc.check_output(sess, ["det", "rbac", "my-permissions"])
    json_out = detproc.check_json(sess, ["det", "rbac", "my-permissions", "--json"])

    assert len(json_out["roles"]) == 2
    assert len(json_out["assignments"]) == 2
    assert len([role for role in json_out["roles"] if role["name"] == "Editor"]) == 0

    # Remove user assignments.
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "unassign-role",
            "WorkspaceCreator",
            "--username-to-assign",
            sess.username,
        ],
    )
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "unassign-role",
            "Viewer",
            "--username-to-assign",
            sess.username,
            "--workspace-name",
            "Uncategorized",
        ],
    )

    # User has no permissions.
    assert "no permissions" in detproc.check_output(sess, ["det", "rbac", "my-permissions"])
    json_out = detproc.check_json(sess, ["det", "rbac", "my-permissions", "--json"])
    assert json_out["roles"] == []
    assert json_out["assignments"] == []


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_permission_assignment_errors() -> None:
    admin = api_utils.admin_session()

    # Specifying args incorrectly.
    detproc.check_error(
        admin, ["det", "rbac", "assign-role", "Viewer"], "must provide exactly one of"
    )
    detproc.check_error(
        admin, ["det", "rbac", "unassign-role", "Viewer"], "must provide exactly one of"
    )
    detproc.check_error(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "Viewer",
            "--username-to-assign",
            "u",
            "--group-name-to-assign",
            "g",
        ],
        "must provide exactly one of",
    )
    detproc.check_error(
        admin,
        [
            "det",
            "rbac",
            "unassign-role",
            "Viewer",
            "--username-to-assign",
            "u",
            "--group-name-to-assign",
            "g",
        ],
        "must provide exactly one of",
    )

    # Non existent role.
    detproc.check_error(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "fakeRoleNameThatDoesntExist",
            "--username-to-assign",
            "admin",
        ],
        "could not find role name",
    )
    detproc.check_error(
        admin,
        [
            "det",
            "rbac",
            "unassign-role",
            "fakeRoleNameThatDoesntExist",
            "--username-to-assign",
            "admin",
        ],
        "could not find role name",
    )

    # Non existent user
    detproc.check_error(
        admin,
        ["det", "rbac", "assign-role", "Viewer", "--username-to-assign", "fakeUserNotExist"],
        "could not find user",
    )
    detproc.check_error(
        admin,
        ["det", "rbac", "unassign-role", "Viewer", "--username-to-assign", "fakeUserNotExist"],
        "could not find user",
    )

    # Non existent group.
    detproc.check_error(
        admin,
        ["det", "rbac", "assign-role", "Viewer", "--group-name-to-assign", "fakeGroupNotExist"],
        "could not find user group",
    )
    detproc.check_error(
        admin,
        ["det", "rbac", "unassign-role", "Viewer", "--group-name-to-assign", "fakeGroupNotExist"],
        "could not find user group",
    )

    # Non existent workspace
    detproc.check_error(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "Viewer",
            "--workspace-name",
            "fakeWorkspace",
            "--username-to-assign",
            "admin",
        ],
        "not found",
    )
    detproc.check_error(
        admin,
        [
            "det",
            "rbac",
            "unassign-role",
            "Viewer",
            "--workspace-name",
            "fakeWorkspace",
            "--username-to-assign",
            "admin",
        ],
        "not found",
    )

    sess, _ = api_utils.create_test_user()
    group_name = api_utils.get_random_string()
    detproc.check_call(admin, ["det", "user-group", "create", group_name])
    detproc.check_call(
        admin,
        ["det", "rbac", "assign-role", "Viewer", "--group-name-to-assign", group_name],
    )
    detproc.check_call(
        admin,
        ["det", "rbac", "assign-role", "Viewer", "--username-to-assign", sess.username],
    )

    # Assign a role multiple times.
    detproc.check_error(
        admin,
        ["rbac", "assign-role", "Viewer", "--group-name-to-assign", group_name],
        "row already exists",
    )

    # Unassigned role group doesn't have.
    detproc.check_error(
        admin,
        ["det", "rbac", "unassign-role", "Editor", "--group-name-to-assign", group_name],
        "Not Found",
    )
    detproc.check_error(
        admin,
        [
            "det",
            "rbac",
            "unassign-role",
            "Viewer",
            "--group-name-to-assign",
            group_name,
            "--workspace-name",
            "Uncategorized",
        ],
        "Not Found",
    )

    # Unassigned role user doesn't have.
    detproc.check_error(
        admin,
        ["det", "rbac", "unassign-role", "Editor", "--username-to-assign", sess.username],
        "Not Found",
    )
    detproc.check_error(
        admin,
        [
            "det",
            "rbac",
            "unassign-role",
            "Viewer",
            "--username-to-assign",
            sess.username,
            "--workspace-name",
            "Uncategorized",
        ],
        "Not Found",
    )


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_list_roles() -> None:
    admin = api_utils.admin_session()
    detproc.check_call(admin, ["det", "rbac", "list-roles"])
    all_roles = detproc.check_json(admin, ["det", "rbac", "list-roles", "--json"])["roles"]

    # Test list-roles excluding global roles properly.
    non_excluded_roles = detproc.check_json(
        admin, ["det", "rbac", "list-roles", "--exclude-global-roles", "--json"]
    )["roles"]
    non_excluded_role_ids = {r["roleId"] for r in non_excluded_roles}
    for role in all_roles:
        is_excluded = role["roleId"] not in non_excluded_role_ids
        is_global = any(not p["scopeTypeMask"]["workspace"] for p in role["permissions"])
        assert is_excluded == is_global

    # Test list-roles pagination.
    json_out = detproc.check_json(admin, ["det", "rbac", "list-roles", "--limit=2", "--json"])
    assert len(json_out["roles"]) == 2
    assert json_out["pagination"]["limit"] == 2
    assert json_out["pagination"]["total"] == len(all_roles)
    assert json_out["pagination"]["offset"] == 0

    json_out = detproc.check_json(
        admin, ["det", "rbac", "list-roles", "--offset=1", "--limit=199", "--json"]
    )
    assert len(json_out["roles"]) == len(all_roles) - 1
    assert json_out["pagination"]["limit"] == 199
    assert json_out["pagination"]["total"] == len(all_roles)
    assert json_out["pagination"]["offset"] == 1

    # Set up group/user to test with.
    sess, _ = api_utils.create_test_user()
    group_name = api_utils.get_random_string()
    detproc.check_call(
        admin, ["det", "user-group", "create", group_name, "--add-user", sess.username]
    )

    # No roles should be returned since no assignmnets have happened.
    list_user_roles = ["det", "rbac", "list-users-roles", sess.username]
    list_group_roles = ["det", "rbac", "list-groups-roles", group_name]

    assert detproc.check_json(admin, list_user_roles + ["--json"])["roles"] == []
    assert "user has no role assignments" in detproc.check_output(admin, list_user_roles)

    assert detproc.check_json(admin, list_group_roles + ["--json"])["roles"] == []
    assert "group has no role assignments" in detproc.check_output(admin, list_group_roles)

    # Assign roles.
    detproc.check_call(
        admin,
        ["det", "rbac", "assign-role", "Viewer", "--username-to-assign", sess.username],
    )
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "Viewer",
            "--username-to-assign",
            sess.username,
            "--workspace-name",
            "Uncategorized",
        ],
    )

    detproc.check_call(
        admin, ["det", "rbac", "assign-role", "Editor", "--group-name-to-assign", group_name]
    )
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "Editor",
            "--group-name-to-assign",
            group_name,
            "--workspace-name",
            "Uncategorized",
        ],
    )

    # Test list-users-roles.
    detproc.check_call(admin, list_user_roles)
    json_out = detproc.check_json(admin, list_user_roles + ["--json"])
    assert len(json_out["roles"]) == 2
    json_out["roles"].sort(key=lambda x: -1 if x["role"]["name"] == "Viewer" else 1)
    assert json_out["roles"][0]["role"]["name"] == "Viewer"

    assert len(json_out["roles"][0]["groupRoleAssignments"]) == 0
    workspace_ids = [
        a["roleAssignment"]["scopeWorkspaceId"] for a in json_out["roles"][0]["userRoleAssignments"]
    ]
    assert len(workspace_ids) == 2
    assert 1 in workspace_ids
    assert None in workspace_ids

    assert json_out["roles"][1]["role"]["name"] == "Editor"
    assert len(json_out["roles"][1]["groupRoleAssignments"]) == 2
    workspace_ids = [
        a["roleAssignment"]["scopeWorkspaceId"]
        for a in json_out["roles"][1]["groupRoleAssignments"]
    ]
    assert len(workspace_ids) == 2
    assert len(json_out["roles"][1]["userRoleAssignments"]) == 0

    # Test list-groups-roles.
    detproc.check_call(admin, list_group_roles)
    json_out = detproc.check_json(admin, list_group_roles + ["--json"])
    assert len(json_out["roles"]) == 1
    assert len(json_out["assignments"]) == 1
    assert json_out["roles"][0]["name"] == "Editor"
    assert json_out["assignments"][0]["roleId"] == json_out["roles"][0]["roleId"]
    assert json_out["assignments"][0]["scopeWorkspaceIds"] == [1]
    assert json_out["assignments"][0]["scopeCluster"]


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_rbac_describe_role() -> None:
    admin = api_utils.admin_session()
    # Role doesn't exist.
    detproc.check_error(
        admin, ["det", "rbac", "describe-role", "roleDoesntExist"], "could not find role name"
    )

    # Role is assigned to our group and user.
    sess, _ = api_utils.create_test_user()
    group_name = api_utils.get_random_string()

    detproc.check_call(admin, ["det", "user-group", "create", group_name])
    detproc.check_call(
        admin, ["det", "rbac", "assign-role", "Viewer", "--group-name-to-assign", group_name]
    )
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "Viewer",
            "--group-name-to-assign",
            group_name,
            "--workspace-name",
            "Uncategorized",
        ],
    )

    user_id = api.usernames_to_user_ids(admin, [sess.username])[0]
    group_id = api.group_name_to_group_id(admin, group_name)

    detproc.check_call(
        admin,
        ["det", "rbac", "assign-role", "Viewer", "--username-to-assign", sess.username],
    )
    detproc.check_call(
        admin,
        [
            "det",
            "rbac",
            "assign-role",
            "Viewer",
            "--username-to-assign",
            sess.username,
            "--workspace-name",
            "Uncategorized",
        ],
    )

    # No errors printing non-json output.
    detproc.check_call(admin, ["det", "rbac", "describe-role", "Viewer"])

    # Output is returned correctly.
    json_out = detproc.check_json(admin, ["det", "rbac", "describe-role", "Viewer", "--json"])
    assert json_out["role"]["name"] == "Viewer"

    group_assign = [a for a in json_out["groupRoleAssignments"] if a["groupId"] == group_id]
    assert len(group_assign) == 2
    group_assign.sort(key=lambda x: -1 if x["roleAssignment"]["scopeWorkspaceId"] is None else 1)
    assert group_assign[0]["roleAssignment"]["scopeWorkspaceId"] is None
    assert group_assign[1]["roleAssignment"]["scopeWorkspaceId"] == 1

    user_assign = [a for a in json_out["userRoleAssignments"] if a["userId"] == user_id]
    assert len(user_assign) == 2
    user_assign.sort(key=lambda x: -1 if x["roleAssignment"]["scopeWorkspaceId"] is None else 1)
    assert user_assign[0]["roleAssignment"]["scopeWorkspaceId"] is None
    assert user_assign[1]["roleAssignment"]["scopeWorkspaceId"] == 1


@pytest.mark.e2e_cpu_rbac
@api_utils.skipif_rbac_not_enabled()
def test_group_access() -> None:
    admin = api_utils.admin_session()
    # create relevant workspace and project, with group having access
    group_name = api_utils.get_random_string()
    workspace_name = api_utils.get_random_string()
    detproc.check_call(admin, ["det", "workspace", "create", workspace_name])
    detproc.check_call(admin, ["det", "user-group", "create", group_name])
    detproc.check_call(
        admin,
        ["det", "rbac", "assign-role", "WorkspaceAdmin", "-w", workspace_name, "-g", group_name],
    )

    # create test user which cannot access workspace
    sess, _ = api_utils.create_test_user()
    detproc.check_error(
        sess, ["det", "workspace", "describe", workspace_name], "Failed to describe workspace"
    )

    # add user to group
    detproc.check_call(admin, ["det", "user-group", "add-user", group_name, sess.username])

    # with user now in group, access possible
    detproc.check_call(sess, ["det", "workspace", "describe", workspace_name])
