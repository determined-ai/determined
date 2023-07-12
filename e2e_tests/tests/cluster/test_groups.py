import json
import subprocess
from typing import Any, List

import pytest

from tests import config as conf

from .test_users import ADMIN_CREDENTIALS, get_random_string, logged_in_user


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
@pytest.mark.parametrize("add_users", [[], ["admin", "determined"]])
def test_group_creation(add_users: List[str]) -> None:
    with logged_in_user(ADMIN_CREDENTIALS):
        group_name = get_random_string()
        create_group_cmd = ["user-group", "create", group_name]
        for add_user in add_users:
            create_group_cmd += ["--add-user", add_user]
        det_cmd(create_group_cmd, check=True)

        # Can view through list.
        group_list = det_cmd_json(["user-group", "list", "--json"])
        assert (
            len([group for group in group_list["groups"] if group["group"]["name"] == group_name])
            == 1
        )

        # Can view through list with userID filter.
        for add_user in add_users:
            group_list = det_cmd_json(
                ["user-group", "list", "--json", "--groups-user-belongs-to", add_user]
            )
            assert (
                len(
                    [
                        group
                        for group in group_list["groups"]
                        if group["group"]["name"] == group_name
                    ]
                )
                == 1
            )

        # Can describe properly.
        group_desc = det_cmd_json(["user-group", "describe", group_name, "--json"])
        assert group_desc["name"] == group_name
        for add_user in add_users:
            assert len([u for u in group_desc["users"] if u["username"] == add_user]) == 1

        # Can delete.
        det_cmd(["user-group", "delete", group_name, "--yes"], check=True)
        det_cmd_expect_error(["user-group", "describe", group_name], "not find")


@pytest.mark.e2e_cpu
def test_group_updates() -> None:
    with logged_in_user(ADMIN_CREDENTIALS):
        group_name = get_random_string()
        det_cmd(["user-group", "create", group_name], check=True)

        # Adds admin and determined to our group then remove determined.
        det_cmd(["user-group", "add-user", group_name, "admin,determined"], check=True)
        det_cmd(["user-group", "remove-user", group_name, "determined"], check=True)

        group_desc = det_cmd_json(["user-group", "describe", group_name, "--json"])
        assert group_desc["name"] == group_name
        assert len(group_desc["users"]) == 1
        assert group_desc["users"][0]["username"] == "admin"

        # Rename our group.
        new_group_name = get_random_string()
        det_cmd(["user-group", "change-name", group_name, new_group_name], check=True)

        # Old name is gone.
        det_cmd_expect_error(["user-group", "describe", group_name, "--json"], "not find")

        # New name is here.
        group_desc = det_cmd_json(["user-group", "describe", new_group_name, "--json"])
        assert group_desc["name"] == new_group_name
        assert len(group_desc["users"]) == 1
        assert group_desc["users"][0]["username"] == "admin"


@pytest.mark.parametrize("offset", [0, 2])
@pytest.mark.parametrize("limit", [1, 3])
@pytest.mark.e2e_cpu
def test_group_list_pagination(offset: int, limit: int) -> None:
    # Ensure we have at minimum n groups.
    n = 5
    group_list = det_cmd_json(["user-group", "list", "--json"])["groups"]
    needed_groups = max(n - len(group_list), 0)

    with logged_in_user(ADMIN_CREDENTIALS):
        for _ in range(needed_groups):
            det_cmd(["user-group", "create", get_random_string()], check=True)

    # Get baseline group list to compare pagination to.
    group_list = det_cmd_json(["user-group", "list", "--json"])["groups"]
    expected = group_list[offset : offset + limit]

    paged_group_list = det_cmd_json(
        ["user-group", "list", "--json", "--offset", f"{offset}", "--limit", f"{limit}"]
    )
    assert expected == paged_group_list["groups"]


@pytest.mark.e2e_cpu
def test_group_errors() -> None:
    with logged_in_user(ADMIN_CREDENTIALS):
        fake_group = get_random_string()
        group_name = get_random_string()
        det_cmd(["user-group", "create", group_name], check=True)

        # Creating group with same name.
        det_cmd_expect_error(["user-group", "create", group_name], "already exists")

        # Adding non existent users to groups.
        fake_user = get_random_string()
        det_cmd_expect_error(
            ["user-group", "create", fake_group, "--add-user", fake_user], "not find"
        )
        det_cmd_expect_error(["user-group", "add-user", group_name, fake_user], "not find")

        # Removing a non existent user from group.
        det_cmd_expect_error(["user-group", "remove-user", group_name, fake_user], "not find")

        # Removing a user not in a group.
        det_cmd_expect_error(["user-group", "remove-user", group_name, "admin"], "Not Found")

        # Describing a non existent group.
        det_cmd_expect_error(["user-group", "describe", get_random_string()], "not find")
