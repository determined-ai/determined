from typing import List

import pytest

from tests import api_utils, detproc


@pytest.mark.e2e_cpu_rbac
@pytest.mark.parametrize("add_users", [[], ["admin", "determined"]])
def test_group_creation(add_users: List[str]) -> None:
    admin = api_utils.admin_session()
    group_name = api_utils.get_random_string()
    create_group_cmd = ["det", "user-group", "create", group_name]
    for add_user in add_users:
        create_group_cmd += ["--add-user", add_user]
    detproc.check_call(admin, create_group_cmd)

    # Can view through list.
    group_list = detproc.check_json(admin, ["det", "user-group", "list", "--json"])
    assert (
        len([group for group in group_list["groups"] if group["group"]["name"] == group_name]) == 1
    )

    # Can view through list with userID filter.
    for add_user in add_users:
        group_list = detproc.check_json(
            admin, ["det", "user-group", "list", "--json", "--groups-user-belongs-to", add_user]
        )
        assert (
            len([group for group in group_list["groups"] if group["group"]["name"] == group_name])
            == 1
        )

    # Can describe properly.
    group_desc = detproc.check_json(admin, ["det", "user-group", "describe", group_name, "--json"])
    assert group_desc["name"] == group_name
    for add_user in add_users:
        assert len([u for u in group_desc["users"] if u["username"] == add_user]) == 1

    # Can delete.
    detproc.check_call(admin, ["det", "user-group", "delete", group_name, "--yes"])
    detproc.check_error(admin, ["det", "user-group", "describe", group_name], "not find")


@pytest.mark.e2e_cpu_rbac
def test_group_updates() -> None:
    admin = api_utils.admin_session()
    group_name = api_utils.get_random_string()
    detproc.check_call(admin, ["det", "user-group", "create", group_name])

    # Adds admin and determined to our group then remove determined.
    detproc.check_call(admin, ["det", "user-group", "add-user", group_name, "admin,determined"])
    detproc.check_call(admin, ["det", "user-group", "remove-user", group_name, "determined"])

    group_desc = detproc.check_json(admin, ["det", "user-group", "describe", group_name, "--json"])
    assert group_desc["name"] == group_name
    assert len(group_desc["users"]) == 1
    assert group_desc["users"][0]["username"] == "admin"

    # Rename our group.
    new_group_name = api_utils.get_random_string()
    detproc.check_call(admin, ["det", "user-group", "change-name", group_name, new_group_name])

    # Old name is gone.
    detproc.check_error(admin, ["det", "user-group", "describe", group_name, "--json"], "not find")

    # New name is here.
    group_desc = detproc.check_json(
        admin, ["det", "user-group", "describe", new_group_name, "--json"]
    )
    assert group_desc["name"] == new_group_name
    assert len(group_desc["users"]) == 1
    assert group_desc["users"][0]["username"] == "admin"


@pytest.mark.parametrize("offset", [0, 2])
@pytest.mark.parametrize("limit", [1, 3])
@pytest.mark.e2e_cpu_rbac
def test_group_list_pagination(offset: int, limit: int) -> None:
    admin = api_utils.admin_session()
    # Ensure we have at minimum n groups.
    n = 5
    group_list = detproc.check_json(admin, ["det", "user-group", "list", "--json"])["groups"]
    needed_groups = max(n - len(group_list), 0)

    for _ in range(needed_groups):
        detproc.check_call(admin, ["det", "user-group", "create", api_utils.get_random_string()])

    # Get baseline group list to compare pagination to.
    group_list = detproc.check_json(admin, ["det", "user-group", "list", "--json"])["groups"]
    expected = group_list[offset : offset + limit]

    paged_group_list = detproc.check_json(
        admin,
        ["det", "user-group", "list", "--json", "--offset", f"{offset}", "--limit", f"{limit}"],
    )
    assert expected == paged_group_list["groups"]


@pytest.mark.e2e_cpu_rbac
def test_group_errors() -> None:
    admin = api_utils.admin_session()
    fake_group = api_utils.get_random_string()
    group_name = api_utils.get_random_string()
    detproc.check_output(admin, ["det", "user-group", "create", group_name])

    # Creating group with same name.
    detproc.check_error(admin, ["det", "user-group", "create", group_name], "already exists")

    # Adding non existent users to groups.
    fake_user = api_utils.get_random_string()
    detproc.check_error(
        admin,
        ["det", "user-group", "create", fake_group, "--add-user", fake_user],
        "not find",
    )
    detproc.check_error(admin, ["det", "user-group", "add-user", group_name, fake_user], "not find")

    # Removing a non existent user from group.
    detproc.check_error(
        admin, ["det", "user-group", "remove-user", group_name, fake_user], "not find"
    )

    # Removing a user not in a group.
    detproc.check_error(
        admin, ["det", "user-group", "remove-user", group_name, "admin"], "not found"
    )

    # Describing a non existent group.
    detproc.check_error(
        admin, ["det", "user-group", "describe", api_utils.get_random_string()], "not find"
    )
