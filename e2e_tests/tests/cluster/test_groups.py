import json
import subprocess
from typing import Any, List

import pytest

from tests import config as conf

from .test_users import get_random_string


def det_cmd(cmd: List[str]) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["det", "-m", conf.make_master_url()] + cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )


def det_cmd_json(cmd: List[str]) -> Any:
    res = det_cmd(cmd)
    assert res.returncode == 0
    return json.loads(res.stdout)


def det_cmd_expect_error(cmd: List[str], expected: str) -> None:
    res = det_cmd(cmd)
    assert res.returncode != 0
    assert expected in res.stderr.decode()


@pytest.mark.e2e_cpu
@pytest.mark.parametrize("add_users", [[], ["admin", "determined"]])
def test_group_creation(add_users: List[str]) -> None:
    group_name = get_random_string()
    create_group_cmd = ["user-group", "create", group_name]
    for add_user in add_users:
        create_group_cmd += ["--add-user", add_user]

    assert det_cmd(create_group_cmd).returncode == 0

    # Can view through list.
    group_list = det_cmd_json(["user-group", "list", "--json"])
    assert sum([group["name"] == group_name for group in group_list["groups"]]) == 1

    # Can describe properly.
    group_desc = det_cmd_json(["user-group", "describe", group_name, "--json"])
    assert group_desc["name"] == group_name
    for add_user in add_users:
        assert sum([u["username"] == add_user for u in group_desc["users"]]) == 1

    # Can delete.
    assert det_cmd(["user-group", "delete", group_name]).returncode == 0
    det_cmd_expect_error(["user-group", "describe", group_name], "not find")


@pytest.mark.e2e_cpu
def test_group_updates() -> None:
    group_name = get_random_string()
    assert det_cmd(["user-group", "create", group_name]).returncode == 0

    # Adds admin and determined to our group then remove determined.
    assert det_cmd(["user-group", "add-user", group_name, "admin,determined"]).returncode == 0
    assert det_cmd(["user-group", "remove-user", group_name, "determined"]).returncode == 0

    group_desc = det_cmd_json(["user-group", "describe", group_name, "--json"])
    assert group_desc["name"] == group_name
    assert len(group_desc["users"]) == 1
    assert group_desc["users"][0]["username"] == "admin"

    # Rename our group.
    new_group_name = get_random_string()
    assert det_cmd(["user-group", "change-name", group_name, new_group_name]).returncode == 0

    # Old name is gone.
    det_cmd_expect_error(["user-group", "describe", group_name, "--json"], "not find")

    # New name is here.
    group_desc = det_cmd_json(["user-group", "describe", new_group_name, "--json"])
    assert group_desc["name"] == new_group_name
    assert len(group_desc["users"]) == 1
    assert group_desc["users"][0]["username"] == "admin"


@pytest.mark.parametrize("offset", [0, 2])
@pytest.mark.parametrize("limit", [0, 2, 10])
@pytest.mark.e2e_cpu
def test_group_list_pagination(offset: int, limit: int) -> None:
    offset = 3
    limit = 5

    # Ensure we have at minimum n groups.
    n = 5
    group_list = det_cmd_json(["user-group", "list", "--json"])["groups"]
    needed_groups = max(n - len(group_list), 0)
    for _ in range(needed_groups):
        assert det_cmd(["user-group", "create", get_random_string()]).returncode == 0

    # Get baseline group list to compare pagination to.
    group_list = det_cmd_json(["user-group", "list", "--json"])["groups"]
    expected = group_list[offset : offset + limit]

    paged_group_list = det_cmd_json(
        ["user-group", "list", "--json", "--offset", f"{offset}", "--limit", f"{limit}"]
    )
    assert expected == paged_group_list["groups"]


@pytest.mark.e2e_cpu
def test_group_errors() -> None:
    fake_group = get_random_string()
    group_name = get_random_string()
    assert det_cmd(["user-group", "create", group_name]).returncode == 0

    # Creating group with same name.
    det_cmd_expect_error(["user-group", "create", group_name], "Duplicate")

    # Adding non existent users to groups.
    fake_user = get_random_string()
    det_cmd_expect_error(["user-group", "create", fake_group, "--add-user", fake_user], "not find")
    det_cmd_expect_error(["user-group", "add-user", group_name, fake_user], "not find")

    # Removing a non existent user from group.
    det_cmd_expect_error(["user-group", "remove-user", group_name, fake_user], "not find")

    # Removing a user not in a group.
    det_cmd_expect_error(["user-group", "remove-user", group_name, "admin"], "NotFound")

    # Describing a non existent group.
    det_cmd_expect_error(["user-group", "describe", get_random_string()], "not find")
