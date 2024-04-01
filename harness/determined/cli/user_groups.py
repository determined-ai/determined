import argparse
import collections
from typing import Any, List

from determined import cli
from determined.cli import render
from determined.common import api
from determined.common.api import bindings

v1UserHeaders = collections.namedtuple(
    "v1UserHeaders",
    ["id", "username", "displayName", "admin", "active", "agentUserGroup", "modifiedAt"],
)

v1GroupHeaders = collections.namedtuple(
    "v1GroupHeaders",
    ["groupId", "name", "numMembers"],  # numMembers
)

rbac_flag_disabled_message = (
    "User groups commands require the Determined Enterprise Edition "
    + "and the Master Configuration option security.authz.rbac_ui_enabled."
)


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def create_group(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    add_users = api.usernames_to_user_ids(sess, args.add_user)
    body = bindings.v1CreateGroupRequest(name=args.group_name, addUsers=add_users)
    resp = bindings.post_CreateGroup(sess, body=body)
    group = resp.group

    print(f"user group with name {group.name} and ID {group.groupId} created")
    if group.users:
        print(f"{', '.join([g.username for g in group.users])} was added to the group")


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def list_groups(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    user_id = None
    if args.groups_user_belongs_to:
        user_id = api.usernames_to_user_ids(sess, [args.groups_user_belongs_to])[0]

    body = bindings.v1GetGroupsRequest(offset=args.offset, limit=args.limit, userId=user_id)
    resp = bindings.post_GetGroups(sess, body=body)
    if args.json:
        render.print_json(resp.to_json())
    else:
        if resp.groups is None:
            resp.groups = []
        group_list = []
        for g in resp.groups:
            group = g.group.to_json()
            group["numMembers"] = g.numMembers
            group_list.append(group)

        render.render_objects(
            v1GroupHeaders, [render.unmarshal(v1GroupHeaders, g) for g in group_list]
        )


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def describe_group(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    group_id = api.group_name_to_group_id(sess, args.group_name)
    resp = bindings.get_GetGroup(sess, groupId=group_id)
    group_details = resp.group

    if args.json:
        render.print_json(group_details.to_json())
    else:
        print(f"group ID {group_details.groupId} group name {group_details.name} with users added")
        if group_details.users is None:
            group_details.users = []
        render.render_objects(
            v1UserHeaders,
            [render.unmarshal(v1UserHeaders, u.to_json()) for u in group_details.users],
        )


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def add_user_to_group(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    usernames = args.usernames.split(",")
    group_id = api.group_name_to_group_id(sess, args.group_name)
    user_ids = api.usernames_to_user_ids(sess, usernames)

    body = bindings.v1UpdateGroupRequest(groupId=group_id, addUsers=user_ids)
    resp = bindings.put_UpdateGroup(sess, groupId=group_id, body=body)

    print(f"user group with ID {resp.group.groupId} name {resp.group.name}")
    for user_id, username in zip(user_ids, usernames):
        print(f"user added to group with username {username} and ID {user_id}")


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def remove_user_from_group(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    usernames = args.usernames.split(",")
    group_id = api.group_name_to_group_id(sess, args.group_name)
    user_ids = api.usernames_to_user_ids(sess, usernames)

    body = bindings.v1UpdateGroupRequest(groupId=group_id, removeUsers=user_ids)
    resp = bindings.put_UpdateGroup(sess, groupId=group_id, body=body)

    print(f"user group with ID {resp.group.groupId} name {resp.group.name}")
    for user_id, username in zip(user_ids, usernames):
        print(f"user removed from the group with username {username} and ID {user_id}")


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def change_group_name(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    group_id = api.group_name_to_group_id(sess, args.old_group_name)
    body = bindings.v1UpdateGroupRequest(groupId=group_id, name=args.new_group_name)
    resp = bindings.put_UpdateGroup(sess, groupId=group_id, body=body)
    g = resp.group

    print(f"user group with ID {g.groupId} name changed from {args.old_group_name} to {g.name}")


@cli.require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def delete_group(args: argparse.Namespace) -> None:
    if args.yes or render.yes_or_no(
        "Deleting a group will result in an unrecoverable \n"
        "deletion of the group along with all the membership  \n"
        "information of the group. Do you still wish to proceed? \n"
    ):
        sess = cli.setup_session(args)
        group_id = api.group_name_to_group_id(sess, args.group_name)
        bindings.delete_DeleteGroup(sess, groupId=group_id)
        print(f"user group with name {args.group_name} and ID {group_id} deleted")
    else:
        print("Skipping group deletion.")


args_description = [
    cli.Cmd(
        "user-group",
        None,
        "manage user groups",
        [
            cli.Cmd(
                "create",
                create_group,
                "create a user group",
                [
                    cli.Arg("group_name", help="name of user group to be created"),
                    cli.Arg(
                        "--add-user",
                        action="append",
                        default=[],
                        help="usernames to add to group upon creation. "
                        + "This can be specified multiple times to add multiple users.",
                    ),
                ],
            ),
            cli.Cmd(
                "delete",
                delete_group,
                "delete a user group",
                [
                    cli.Arg("group_name", help="name of user group to be deleted"),
                    cli.Arg(
                        "--yes", action="store_true", help="skip prompt asking for confirmation"
                    ),
                ],
            ),
            cli.Cmd(
                "list ls",
                list_groups,
                "list user groups",
                [
                    *cli.make_pagination_args(),
                    cli.Arg("--groups-user-belongs-to", help="list groups that the username is in"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
                is_default=True,
            ),
            cli.Cmd(
                "describe",
                describe_group,
                "describe a user group",
                [
                    cli.Arg("group_name", help="name of user group to describe"),
                    cli.Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            cli.Cmd(
                "add-user",
                add_user_to_group,
                "add users to a group",
                [
                    cli.Arg("group_name", help="name of user group to add users to"),
                    cli.Arg("usernames", help="a comma seperated list of usernames"),
                ],
            ),
            cli.Cmd(
                "remove-user",
                remove_user_from_group,
                "remove user from a group",
                [
                    cli.Arg("group_name", help="name of user group to remove users from"),
                    cli.Arg("usernames", help="a comma seperated list of usernames"),
                ],
            ),
            cli.Cmd(
                "change-name",
                change_group_name,
                "change name of a user group",
                [
                    cli.Arg("old_group_name", help="name of user group to be updated"),
                    cli.Arg("new_group_name", help="name of user group to change to"),
                ],
            ),
        ],
    )
]  # type: List[Any]
