import json
import subprocess
from typing import Any, List

import pytest

from tests import config as conf

from .test_users import get_random_string, ADMIN_CREDENTIALS, create_test_user, logged_in_user


def det_cmd(cmd: List[str], **kwargs: Any) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["det", "-m", conf.make_master_url()] + cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        **kwargs,
    )


def det_cmd_json(cmd: List[str]) -> Any:
    res = det_cmd(cmd, check=True)
    return json.loads(res.stdout)


def det_cmd_expect_error(cmd: List[str], expected: str) -> None:
    res = det_cmd(cmd)
    assert res.returncode != 0
    assert expected in res.stderr.decode()


@pytest.mark.e2e_cpu
#@pytest.mark.parametrize("add_users", [[], ["admin", "determined"]])
def test_rbac_permission_assignment() -> None:
    test_user_creds = create_test_user(ADMIN_CREDENTIALS)
    
    # User has no permissions.
    with logged_in_user(test_user_creds):
        assert "no permissions" in det_cmd(["rbac", "my-permissions"], check=True).stdout.decode()
        json_out = det_cmd_json(["rbac", "my-permissions", "--json"])
        assert json_out["roles"] == []
        assert json_out["assignments"] == []

    group_name = get_random_string()        
    with logged_in_user(ADMIN_CREDENTIALS):
        # Assign user to role directly.
        det_cmd(["rbac", "assign-role", "WorkspaceCreator", "--username-to-assign",
                 test_user_creds.username], check=True)
        det_cmd(["rbac", "assign-role", "Viewer", "--username-to-assign",
                 test_user_creds.username, "--workspace-name", "Uncategorized"], check=True)

        # Assign user to a group with roles.
        det_cmd(["user-group", "create", group_name, "--add-user",
                 test_user_creds.username], check=True)
        det_cmd(["rbac", "assign-role", "WorkspaceCreator", "--group-name-to-assign",
                 group_name], check=True)        
        det_cmd(["rbac", "assign-role", "Editor", "--group-name-to-assign",
                 group_name], check=True)
        det_cmd(["rbac", "assign-role", "Editor", "--group-name-to-assign",
                 group_name, "--workspace-name", "Uncategorized"], check=True)
        
    # User has those roles assigned.
    with logged_in_user(test_user_creds):
        assert "no permissions" not in det_cmd(["rbac", "my-permissions"], check=True).stdout.decode()
        json_out = det_cmd_json(["rbac", "my-permissions", "--json"])        
        assert len(json_out["roles"]) == 3
        assert len(json_out["assignments"]) == 3
        
        creator = [role for role in json_out["roles"] if role["name"] == "WorkspaceCreator"]
        assert len(creator) == 1
        creator_assignment = [a for a in json_out["assignments"] if a["roleId"] == creator[0]["roleId"]]
        assert creator_assignment[0]["scopeWorkspaceIds"] == []
        assert creator_assignment[0]["isGlobal"]

        viewer = [role for role in json_out["roles"] if role["name"] == "Viewer"]
        assert len(viewer) == 1        
        viewer_assignment = [a for a in json_out["assignments"] if a["roleId"] == viewer[0]["roleId"]]
        assert viewer_assignment[0]["scopeWorkspaceIds"] == [1]
        assert not viewer_assignment[0]["isGlobal"]

        editor = [role for role in json_out["roles"] if role["name"] == "Editor"]
        assert len(editor) == 1        
        editor_assignment = [a for a in json_out["assignments"] if a["roleId"] == editor[0]["roleId"]]
        assert editor_assignment[0]["scopeWorkspaceIds"] == [1]
        assert editor_assignment[0]["isGlobal"]

    # Remove from the group.
    with logged_in_user(ADMIN_CREDENTIALS):
        det_cmd(["user-group", "remove-user", group_name, test_user_creds.username], check=True)

    # User doesn't have any group roles assigned.
    with logged_in_user(test_user_creds):
        assert "no permissions" not in det_cmd(["rbac", "my-permissions"], check=True).stdout.decode()
        json_out = det_cmd_json(["rbac", "my-permissions", "--json"])
        
        assert len(json_out["roles"]) == 2
        assert len(json_out["assignments"]) == 2
        assert len([role for role in json_out["roles"] if role["name"] == "Editor"]) == 0

    # Remove user assignments.
    with logged_in_user(ADMIN_CREDENTIALS):
        # Assign user to role directly.
        det_cmd(["rbac", "unassign-role", "WorkspaceCreator", "--username-to-assign",
                 test_user_creds.username], check=True)
        det_cmd(["rbac", "unassign-role", "Viewer", "--username-to-assign",
                 test_user_creds.username, "--workspace-name", "Uncategorized"], check=True)    
    
    # User has no permissions.
    with logged_in_user(test_user_creds):
        assert "no permissions" in det_cmd(["rbac", "my-permissions"], check=True).stdout.decode()
        json_out = det_cmd_json(["rbac", "my-permissions", "--json"])
        assert json_out["roles"] == []
        assert json_out["assignments"] == []

        

@pytest.mark.e2e_cpu
def test_rbac_permission_assignment_errors() -> None:
    # Specifying args incorrectly.
    det_cmd_expect_error(["rbac", "assign-role", "Viewer"], "must provide exactly one of")
    det_cmd_expect_error(["rbac", "unassign-role", "Viewer"], "must provide exactly one of")
    det_cmd_expect_error(["rbac", "assign-role", "Viewer",
                          "--username-to-assign", "u", "--group-name-to-assign", "g"],
                         "must provide exactly one of")
    det_cmd_expect_error(["rbac", "unassign-role", "Viewer",
                          "--username-to-assign", "u", "--group-name-to-assign", "g"],
                         "must provide exactly one of")
    
    # Non existent role.
    det_cmd_expect_error(["rbac", "assign-role", "fakeRoleNameThatDoesntExist",
                          "--username-to-assign", "admin"], "could not find role name")    
    det_cmd_expect_error(["rbac", "unassign-role", "fakeRoleNameThatDoesntExist",
                          "--username-to-assign", "admin"], "could not find role name")
    
    # Non existent user
    det_cmd_expect_error(["rbac", "assign-role", "Viewer",
                          "--username-to-assign", "fakeUserNotExist"], "could not find user")    
    det_cmd_expect_error(["rbac", "unassign-role", "Viewer",
                          "--username-to-assign", "fakeUserNotExist"], "could not find user")    
    
    # Non existent group.
    det_cmd_expect_error(["rbac", "assign-role", "Viewer", "--group-name-to-assign",
                          "fakeGroupNotExist"], "could not find user group")    
    det_cmd_expect_error(["rbac", "unassign-role", "Viewer", "--group-name-to-assign",
                          "fakeGroupNotExist"], "could not find user group")    
    
    # Non existent workspace
    det_cmd_expect_error(["rbac", "assign-role", "Viewer", "--workspace-name", "fakeWorkspace",
                          "--username-to-assign", "admin"], "not find a workspace")    
    det_cmd_expect_error(["rbac", "unassign-role", "Viewer", "--workspace-name", "fakeWorkspace",
                          "--username-to-assign", "admin"], "not find a workspace")

    test_user_creds = create_test_user(ADMIN_CREDENTIALS)
    group_name = get_random_string()        
    with logged_in_user(ADMIN_CREDENTIALS):                
        det_cmd(["user-group", "create", group_name], check=True)
        det_cmd(["rbac", "assign-role", "Viewer", "--group-name-to-assign",
                 group_name], check=True)                
        det_cmd(["rbac", "assign-role", "Viewer", "--username-to-assign",
                 test_user_creds.username], check=True)
        
        # Unassigned role group doesn't have.
        det_cmd_expect_error(["rbac", "unassign-role", "Editor", "--group-name-to-assign",
                              group_name], "not found")        
        det_cmd_expect_error(["rbac", "unassign-role", "Viewer", "--group-name-to-assign",
                              group_name, "--workspace-name", "Uncategorized"], "not found")                
        
        # Unassigned role user doesn't have.
        det_cmd_expect_error(["rbac", "unassign-role", "Editor", "--username-to-assign",
                              test_user_creds.username], "not found")        
        det_cmd_expect_error(["rbac", "unassign-role", "Viewer", "--username-to-assign",
                              test_user_creds.username, "--workspace-name", "Uncategorized"],
                             "not found")                
        
    
@pytest.mark.e2e_cpu
def test_rbac_list_roles() -> None:
    pass

@pytest.mark.e2e_cpu
def test_rbac_describe_role() -> None:
    pass

        

        
# assign_role (x)
# unassign_role (x)
# my_permissions (x) (technically don't check no json output since it is too hard...)
        
# list_roles
# list_users_roles
# list_groups_roles

# describe_role 
