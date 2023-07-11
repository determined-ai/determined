import contextlib
from typing import Any, Callable, Dict, Generator, List, NamedTuple, Optional, Tuple

import pytest

from determined import cli
from determined.cli.user_groups import group_name_to_group_id, usernames_to_user_ids
from determined.common import api
from determined.common.api import authentication, bindings, errors
from tests import api_utils, utils
from tests.api_utils import configure_token_store, create_test_user, determined_test_session
from tests.cluster.test_workspace_org import setup_workspaces

from .test_groups import det_cmd, det_cmd_expect_error, det_cmd_json
from .test_users import ADMIN_CREDENTIALS, get_random_string, logged_in_user


def roles_not_implemented() -> bool:
    return "Unimplemented" in det_cmd(["rbac", "my-permissions"]).stderr.decode()


def rbac_disabled() -> bool:
    if roles_not_implemented():
        return True
    try:
        return not bindings.get_GetMaster(determined_test_session()).rbacEnabled
    except (errors.APIException, errors.MasterNotFoundException):
        return True


def strict_q_control_disabled() -> bool:
    if roles_not_implemented() or rbac_disabled():
        return True
    try:
        return not bindings.get_GetMaster(determined_test_session()).strictJobQueueControl
    except (errors.APIException, errors.MasterNotFoundException):
        return True


PermCase = NamedTuple(
    "PermCase", [("cred", api.authentication.Credentials), ("raises", Optional[Any])]
)


def run_permission_tests(
    action: Callable[[authentication.Credentials], None], cases: List[PermCase]
) -> None:
    for cred, raises in cases:
        if raises is None:
            action(cred)
        else:
            with pytest.raises(raises):
                action(cred)


def create_users_with_gloabl_roles(user_roles: List[List[str]]) -> List[authentication.Credentials]:
    """
    Set up users with the provided global role assignments.
    user_roles: list of roles to assign to each user, one entry per user.
    """
    user_creds: List[authentication.Credentials] = []
    with logged_in_user(ADMIN_CREDENTIALS):
        for roles in user_roles:
            user = bindings.v1User(username=api_utils.get_random_string(), admin=False, active=True)
            creds = api_utils.create_test_user(True, user=user)
            for role in roles:
                cli.rbac.assign_role(
                    utils.CliArgsMock(
                        username_to_assign=creds.username,
                        role_name=role,
                    )
                )
            user_creds.append(creds)
    return user_creds


@pytest.fixture(scope="session")
def cluster_admin_creds() -> authentication.Credentials:
    [creds] = create_users_with_gloabl_roles([["ClusterAdmin"]])
    return creds


@contextlib.contextmanager
def create_workspaces_with_users(
    assignments_list: List[List[Tuple[int, List[str]]]]
) -> Generator[
    Tuple[List[bindings.v1Workspace], Dict[int, authentication.Credentials]], None, None
]:
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
    configure_token_store(ADMIN_CREDENTIALS)
    rid_to_creds: Dict[int, authentication.Credentials] = {}
    with setup_workspaces(count=len(assignments_list)) as workspaces:
        for workspace, user_list in zip(workspaces, assignments_list):
            for rid, roles in user_list:
                if rid not in rid_to_creds:
                    rid_to_creds[rid] = create_test_user()
                for role in roles:
                    cli.rbac.assign_role(
                        utils.CliArgsMock(
                            username_to_assign=rid_to_creds[rid].username,
                            workspace_name=workspace.name,
                            role_name=role,
                        )
                    )
        yield workspaces, rid_to_creds


@pytest.mark.e2e_cpu
@pytest.mark.skipif(roles_not_implemented(), reason="ee is required for this test")
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
    with create_workspaces_with_users(perm_assigments) as (workspaces, rid_to_creds):
        assert len(rid_to_creds) == 2
        assert len(workspaces) == 2


@pytest.mark.e2e_cpu
@pytest.mark.skipif(roles_not_implemented(), reason="ee is required for this test")
def test_rbac_permission_assignment() -> None:
    api_utils.configure_token_store(ADMIN_CREDENTIALS)
    test_user_creds = api_utils.create_test_user()

    # User has no permissions.
    with logged_in_user(test_user_creds):
        assert "no permissions" in det_cmd(["rbac", "my-permissions"], check=True).stdout.decode()
        json_out = det_cmd_json(["rbac", "my-permissions", "--json"])
        assert json_out["roles"] == []
        assert json_out["assignments"] == []

    group_name = get_random_string()
    with logged_in_user(ADMIN_CREDENTIALS):
        # Assign user to role directly.
        det_cmd(
            [
                "rbac",
                "assign-role",
                "WorkspaceCreator",
                "--username-to-assign",
                test_user_creds.username,
            ],
            check=True,
        )
        det_cmd(
            [
                "rbac",
                "assign-role",
                "Viewer",
                "--username-to-assign",
                test_user_creds.username,
                "--workspace-name",
                "Uncategorized",
            ],
            check=True,
        )

        # Assign user to a group with roles.
        det_cmd(
            ["user-group", "create", group_name, "--add-user", test_user_creds.username], check=True
        )
        det_cmd(
            ["rbac", "assign-role", "WorkspaceCreator", "--group-name-to-assign", group_name],
            check=True,
        )
        det_cmd(["rbac", "assign-role", "Editor", "--group-name-to-assign", group_name], check=True)
        det_cmd(
            [
                "rbac",
                "assign-role",
                "Editor",
                "--group-name-to-assign",
                group_name,
                "--workspace-name",
                "Uncategorized",
            ],
            check=True,
        )

    # User has those roles assigned.
    with logged_in_user(test_user_creds):
        assert (
            "no permissions" not in det_cmd(["rbac", "my-permissions"], check=True).stdout.decode()
        )
        json_out = det_cmd_json(["rbac", "my-permissions", "--json"])
        assert len(json_out["roles"]) == 3
        assert len(json_out["assignments"]) == 3

        creator = [role for role in json_out["roles"] if role["name"] == "WorkspaceCreator"]
        assert len(creator) == 1
        creator_assignment = [
            a for a in json_out["assignments"] if a["roleId"] == creator[0]["roleId"]
        ]
        assert creator_assignment[0]["scopeWorkspaceIds"] == []
        assert creator_assignment[0]["scopeCluster"]

        viewer = [role for role in json_out["roles"] if role["name"] == "Viewer"]
        assert len(viewer) == 1
        viewer_assignment = [
            a for a in json_out["assignments"] if a["roleId"] == viewer[0]["roleId"]
        ]
        assert viewer_assignment[0]["scopeWorkspaceIds"] == [1]
        assert not viewer_assignment[0]["scopeCluster"]

        editor = [role for role in json_out["roles"] if role["name"] == "Editor"]
        assert len(editor) == 1
        editor_assignment = [
            a for a in json_out["assignments"] if a["roleId"] == editor[0]["roleId"]
        ]
        assert editor_assignment[0]["scopeWorkspaceIds"] == [1]
        assert editor_assignment[0]["scopeCluster"]

    # Remove from the group.
    with logged_in_user(ADMIN_CREDENTIALS):
        det_cmd(["user-group", "remove-user", group_name, test_user_creds.username], check=True)

    # User doesn't have any group roles assigned.
    with logged_in_user(test_user_creds):
        assert (
            "no permissions" not in det_cmd(["rbac", "my-permissions"], check=True).stdout.decode()
        )
        json_out = det_cmd_json(["rbac", "my-permissions", "--json"])

        assert len(json_out["roles"]) == 2
        assert len(json_out["assignments"]) == 2
        assert len([role for role in json_out["roles"] if role["name"] == "Editor"]) == 0

    # Remove user assignments.
    with logged_in_user(ADMIN_CREDENTIALS):
        # Assign user to role directly.
        det_cmd(
            [
                "rbac",
                "unassign-role",
                "WorkspaceCreator",
                "--username-to-assign",
                test_user_creds.username,
            ],
            check=True,
        )
        det_cmd(
            [
                "rbac",
                "unassign-role",
                "Viewer",
                "--username-to-assign",
                test_user_creds.username,
                "--workspace-name",
                "Uncategorized",
            ],
            check=True,
        )

    # User has no permissions.
    with logged_in_user(test_user_creds):
        assert "no permissions" in det_cmd(["rbac", "my-permissions"], check=True).stdout.decode()
        json_out = det_cmd_json(["rbac", "my-permissions", "--json"])
        assert json_out["roles"] == []
        assert json_out["assignments"] == []


@pytest.mark.e2e_cpu
@pytest.mark.skipif(roles_not_implemented(), reason="ee is required for this test")
def test_rbac_permission_assignment_errors() -> None:
    # Specifying args incorrectly.
    det_cmd_expect_error(["rbac", "assign-role", "Viewer"], "must provide exactly one of")
    det_cmd_expect_error(["rbac", "unassign-role", "Viewer"], "must provide exactly one of")
    det_cmd_expect_error(
        [
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
    det_cmd_expect_error(
        [
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
    det_cmd_expect_error(
        ["rbac", "assign-role", "fakeRoleNameThatDoesntExist", "--username-to-assign", "admin"],
        "could not find role name",
    )
    det_cmd_expect_error(
        ["rbac", "unassign-role", "fakeRoleNameThatDoesntExist", "--username-to-assign", "admin"],
        "could not find role name",
    )

    # Non existent user
    det_cmd_expect_error(
        ["rbac", "assign-role", "Viewer", "--username-to-assign", "fakeUserNotExist"],
        "could not find user",
    )
    det_cmd_expect_error(
        ["rbac", "unassign-role", "Viewer", "--username-to-assign", "fakeUserNotExist"],
        "could not find user",
    )

    # Non existent group.
    det_cmd_expect_error(
        ["rbac", "assign-role", "Viewer", "--group-name-to-assign", "fakeGroupNotExist"],
        "could not find user group",
    )
    det_cmd_expect_error(
        ["rbac", "unassign-role", "Viewer", "--group-name-to-assign", "fakeGroupNotExist"],
        "could not find user group",
    )

    # Non existent workspace
    det_cmd_expect_error(
        [
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
    det_cmd_expect_error(
        [
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

    api_utils.configure_token_store(ADMIN_CREDENTIALS)
    test_user_creds = api_utils.create_test_user()
    group_name = get_random_string()
    with logged_in_user(ADMIN_CREDENTIALS):
        det_cmd(["user-group", "create", group_name], check=True)
        det_cmd(["rbac", "assign-role", "Viewer", "--group-name-to-assign", group_name], check=True)
        det_cmd(
            ["rbac", "assign-role", "Viewer", "--username-to-assign", test_user_creds.username],
            check=True,
        )

        # Assign a role multiple times.
        det_cmd_expect_error(
            ["rbac", "assign-role", "Viewer", "--group-name-to-assign", group_name],
            "row already exists",
        )

        # Unassigned role group doesn't have.
        det_cmd_expect_error(
            ["rbac", "unassign-role", "Editor", "--group-name-to-assign", group_name], "Not Found"
        )
        det_cmd_expect_error(
            [
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
        det_cmd_expect_error(
            ["rbac", "unassign-role", "Editor", "--username-to-assign", test_user_creds.username],
            "Not Found",
        )
        det_cmd_expect_error(
            [
                "rbac",
                "unassign-role",
                "Viewer",
                "--username-to-assign",
                test_user_creds.username,
                "--workspace-name",
                "Uncategorized",
            ],
            "Not Found",
        )


@pytest.mark.e2e_cpu
@pytest.mark.skipif(roles_not_implemented(), reason="ee is required for this test")
def test_rbac_list_roles() -> None:
    with logged_in_user(ADMIN_CREDENTIALS):
        det_cmd(["rbac", "list-roles"], check=True)
        all_roles = det_cmd_json(["rbac", "list-roles", "--json"])["roles"]

        # Test list-roles excluding global roles properly.
        non_excluded_roles = det_cmd_json(
            ["rbac", "list-roles", "--exclude-global-roles", "--json"]
        )["roles"]
        non_excluded_role_ids = {r["roleId"] for r in non_excluded_roles}
        for role in all_roles:
            is_excluded = role["roleId"] not in non_excluded_role_ids
            is_global = any(not p["scopeTypeMask"]["workspace"] for p in role["permissions"])
            assert is_excluded == is_global

        # Test list-roles pagination.
        json_out = det_cmd_json(["rbac", "list-roles", "--limit=2", "--json"])
        assert len(json_out["roles"]) == 2
        assert json_out["pagination"]["limit"] == 2
        assert json_out["pagination"]["total"] == len(all_roles)
        assert json_out["pagination"]["offset"] == 0

        json_out = det_cmd_json(["rbac", "list-roles", "--offset=1", "--limit=199", "--json"])
        assert len(json_out["roles"]) == len(all_roles) - 1
        assert json_out["pagination"]["limit"] == 199
        assert json_out["pagination"]["total"] == len(all_roles)
        assert json_out["pagination"]["offset"] == 1

        # Set up group/user to test with.
        api_utils.configure_token_store(ADMIN_CREDENTIALS)
        test_user_creds = api_utils.create_test_user()
        group_name = get_random_string()
        det_cmd(
            ["user-group", "create", group_name, "--add-user", test_user_creds.username], check=True
        )

        # No roles should be returned since no assignmnets have happened.
        list_user_roles = ["rbac", "list-users-roles", test_user_creds.username]
        list_group_roles = ["rbac", "list-groups-roles", group_name]

        assert det_cmd_json(list_user_roles + ["--json"])["roles"] == []
        assert (
            "user has no role assignments" in det_cmd(list_user_roles, check=True).stdout.decode()
        )

        assert det_cmd_json(list_group_roles + ["--json"])["roles"] == []
        assert (
            "group has no role assignments" in det_cmd(list_group_roles, check=True).stdout.decode()
        )

        # Assign roles.
        det_cmd(
            ["rbac", "assign-role", "Viewer", "--username-to-assign", test_user_creds.username],
            check=True,
        )
        det_cmd(
            [
                "rbac",
                "assign-role",
                "Viewer",
                "--username-to-assign",
                test_user_creds.username,
                "--workspace-name",
                "Uncategorized",
            ],
            check=True,
        )

        det_cmd(["rbac", "assign-role", "Editor", "--group-name-to-assign", group_name], check=True)
        det_cmd(
            [
                "rbac",
                "assign-role",
                "Editor",
                "--group-name-to-assign",
                group_name,
                "--workspace-name",
                "Uncategorized",
            ],
            check=True,
        )

        # Test list-users-roles.
        det_cmd(list_user_roles, check=True)
        json_out = det_cmd_json(list_user_roles + ["--json"])
        assert len(json_out["roles"]) == 2
        json_out["roles"].sort(key=lambda x: -1 if x["role"]["name"] == "Viewer" else 1)
        assert json_out["roles"][0]["role"]["name"] == "Viewer"

        assert len(json_out["roles"][0]["groupRoleAssignments"]) == 0
        workspace_ids = [
            a["roleAssignment"]["scopeWorkspaceId"]
            for a in json_out["roles"][0]["userRoleAssignments"]
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
        det_cmd(list_group_roles, check=True)
        json_out = det_cmd_json(list_group_roles + ["--json"])
        assert len(json_out["roles"]) == 1
        assert len(json_out["assignments"]) == 1
        assert json_out["roles"][0]["name"] == "Editor"
        assert json_out["assignments"][0]["roleId"] == json_out["roles"][0]["roleId"]
        assert json_out["assignments"][0]["scopeWorkspaceIds"] == [1]
        assert json_out["assignments"][0]["scopeCluster"]


@pytest.mark.e2e_cpu
@pytest.mark.skipif(roles_not_implemented(), reason="ee is required for this test")
def test_rbac_describe_role() -> None:
    with logged_in_user(ADMIN_CREDENTIALS):
        # Role doesn't exist.
        det_cmd_expect_error(
            ["rbac", "describe-role", "roleDoesntExist"], "could not find role name"
        )

        # Role is assigned to our group and user.
        api_utils.configure_token_store(ADMIN_CREDENTIALS)
        test_user_creds = api_utils.create_test_user()
        group_name = get_random_string()

        det_cmd(["user-group", "create", group_name], check=True)
        det_cmd(["rbac", "assign-role", "Viewer", "--group-name-to-assign", group_name], check=True)
        det_cmd(
            [
                "rbac",
                "assign-role",
                "Viewer",
                "--group-name-to-assign",
                group_name,
                "--workspace-name",
                "Uncategorized",
            ],
            check=True,
        )

        sess = api_utils.determined_test_session()
        user_id = usernames_to_user_ids(sess, [test_user_creds.username])[0]
        group_id = group_name_to_group_id(sess, group_name)

        det_cmd(
            ["rbac", "assign-role", "Viewer", "--username-to-assign", test_user_creds.username],
            check=True,
        )
        det_cmd(
            [
                "rbac",
                "assign-role",
                "Viewer",
                "--username-to-assign",
                test_user_creds.username,
                "--workspace-name",
                "Uncategorized",
            ],
            check=True,
        )

        # No errors printing non-json output.
        det_cmd(["rbac", "describe-role", "Viewer"], check=True)

        # Output is returned correctly.
        json_out = det_cmd_json(["rbac", "describe-role", "Viewer", "--json"])
        assert json_out["role"]["name"] == "Viewer"

        group_assign = [a for a in json_out["groupRoleAssignments"] if a["groupId"] == group_id]
        assert len(group_assign) == 2
        group_assign.sort(
            key=lambda x: -1 if x["roleAssignment"]["scopeWorkspaceId"] is None else 1
        )
        assert group_assign[0]["roleAssignment"]["scopeWorkspaceId"] is None
        assert group_assign[1]["roleAssignment"]["scopeWorkspaceId"] == 1

        user_assign = [a for a in json_out["userRoleAssignments"] if a["userId"] == user_id]
        assert len(user_assign) == 2
        user_assign.sort(key=lambda x: -1 if x["roleAssignment"]["scopeWorkspaceId"] is None else 1)
        assert user_assign[0]["roleAssignment"]["scopeWorkspaceId"] is None
        assert user_assign[1]["roleAssignment"]["scopeWorkspaceId"] == 1


@pytest.mark.e2e_cpu
@pytest.mark.skipif(roles_not_implemented(), reason="ee is required for this test")
def test_group_access() -> None:
    # create relevant workspace and project, with group having access
    group_name = get_random_string()
    workspace_name = get_random_string()
    project_name = get_random_string()
    with logged_in_user(ADMIN_CREDENTIALS):
        det_cmd(["workspace", "create", workspace_name], check=True)
        det_cmd(["project", "create", workspace_name, project_name], check=True)
        det_cmd(["user-group", "create", group_name], check=True)
        det_cmd(
            ["rbac", "assign-role", "WorkspaceAdmin", "-w", workspace_name, "-g", group_name],
            check=True,
        )

    # create test user which cannot access workspace
    creds1 = api_utils.create_test_user(True)
    with logged_in_user(creds1):
        det_cmd_expect_error(
            ["workspace", "describe", workspace_name], "Did not find a workspace with name"
        )

    # add user to group
    with logged_in_user(ADMIN_CREDENTIALS):
        det_cmd(["user-group", "add-user", group_name, creds1.username], check=True)

    # with user now in group, access possible
    with logged_in_user(creds1):
        det_cmd(["workspace", "describe", workspace_name], check=True)
        # test code from https://github.com/determined-ai/determined/pull/6503
        det_cmd(["project", "list-experiments", workspace_name, project_name], check=True)
