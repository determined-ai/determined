import json
from argparse import Namespace
from collections import namedtuple
from typing import Any, Dict, List, Optional

from determined.cli import default_pagination_args, render, require_feature_flag, setup_session
from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.experimental import session

v1UserHeaders = namedtuple(
    "v1UserHeaders",
    ["id", "username", "displayName", "admin", "active", "agentUserGroup", "modifiedAt"],
)

v1GroupHeaders = namedtuple(
    "v1GroupHeaders",
    ["groupId", "name", "numMembers"],  # numMembers
)

rbac_flag_disabled_message = (
    "User groups commands require the Determined Enterprise Edition "
    + "and the Master Configuration option security.authz.rbac_ui_enabled."
)


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def create_group(args: Namespace) -> None:
    session = setup_session(args)
    add_users = usernames_to_user_ids(session, args.add_user)
    body = bindings.v1CreateGroupRequest(name=args.group_name, addUsers=add_users)
    resp = bindings.post_CreateGroup(session, body=body)
    group = resp.group

    print(f"user group with name {group.name} and ID {group.groupId} created")
    if group.users:
        print(f"{', '.join([g.username for g in group.users])} was added to the group")


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def list_groups(args: Namespace) -> None:
    sess = setup_session(args)
    user_id = None
    if args.groups_user_belongs_to:
        user_id = usernames_to_user_ids(sess, [args.groups_user_belongs_to])[0]

    body = bindings.v1GetGroupsRequest(offset=args.offset, limit=args.limit, userId=user_id)
    resp = bindings.post_GetGroups(sess, body=body)
    if args.json:
        print(json.dumps(resp.to_json(), indent=2))
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


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def describe_group(args: Namespace) -> None:
    session = setup_session(args)
    group_id = group_name_to_group_id(session, args.group_name)
    resp = bindings.get_GetGroup(session, groupId=group_id)
    group_details = resp.group

    if args.json:
        print(json.dumps(group_details.to_json(), indent=2))
    else:
        print(f"group ID {group_details.groupId} group name {group_details.name} with users added")
        if group_details.users is None:
            group_details.users = []
        render.render_objects(
            v1UserHeaders,
            [render.unmarshal(v1UserHeaders, u.to_json()) for u in group_details.users],
        )


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def add_user_to_group(args: Namespace) -> None:
    session = setup_session(args)
    usernames = args.usernames.split(",")
    group_id = group_name_to_group_id(session, args.group_name)
    user_ids = usernames_to_user_ids(session, usernames)

    body = bindings.v1UpdateGroupRequest(groupId=group_id, addUsers=user_ids)
    resp = bindings.put_UpdateGroup(session, groupId=group_id, body=body)

    print(f"user group with ID {resp.group.groupId} name {resp.group.name}")
    for user_id, username in zip(user_ids, usernames):
        print(f"user added to group with username {username} and ID {user_id}")


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def remove_user_from_group(args: Namespace) -> None:
    session = setup_session(args)
    usernames = args.usernames.split(",")
    group_id = group_name_to_group_id(session, args.group_name)
    user_ids = usernames_to_user_ids(session, usernames)

    body = bindings.v1UpdateGroupRequest(groupId=group_id, removeUsers=user_ids)
    resp = bindings.put_UpdateGroup(setup_session(args), groupId=group_id, body=body)

    print(f"user group with ID {resp.group.groupId} name {resp.group.name}")
    for user_id, username in zip(user_ids, usernames):
        print(f"user removed from the group with username {username} and ID {user_id}")


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def change_group_name(args: Namespace) -> None:
    session = setup_session(args)
    group_id = group_name_to_group_id(session, args.old_group_name)
    body = bindings.v1UpdateGroupRequest(groupId=group_id, name=args.new_group_name)
    resp = bindings.put_UpdateGroup(session, groupId=group_id, body=body)
    g = resp.group

    print(f"user group with ID {g.groupId} name changed from {args.old_group_name} to {g.name}")


@authentication.required
@require_feature_flag("rbacEnabled", rbac_flag_disabled_message)
def delete_group(args: Namespace) -> None:
    if args.yes or render.yes_or_no(
        "Deleting a group will result in an unrecoverable \n"
        "deletion of the group along with all the membership  \n"
        "information of the group. Do you still wish to proceed? \n"
    ):
        session = setup_session(args)
        group_id = group_name_to_group_id(session, args.group_name)
        bindings.delete_DeleteGroup(session, groupId=group_id)
        print(f"user group with name {args.group_name} and ID {group_id} deleted")
    else:
        print("Skipping group deletion.")


def usernames_to_user_ids(session: session.Session, usernames: List[str]) -> List[int]:
    usernames_to_ids: Dict[str, Optional[int]] = {u: None for u in usernames}
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


def group_name_to_group_id(session: session.Session, group_name: str) -> int:
    body = bindings.v1GetGroupsRequest(name=group_name, limit=1, offset=0)
    resp = bindings.post_GetGroups(session, body=body)
    groups = resp.groups
    if groups is None or len(groups) != 1 or groups[0].group.groupId is None:
        raise api.errors.BadRequestException(f"could not find user group name {group_name}")
    return groups[0].group.groupId


args_description = [
    Cmd(
        "user-group",
        None,
        "manage user groups",
        [
            Cmd(
                "create",
                create_group,
                "create a user group",
                [
                    Arg("group_name", help="name of user group to be created"),
                    Arg(
                        "--add-user",
                        action="append",
                        default=[],
                        help="usernames to add to group upon creation. "
                        + "This can be specified multiple times to add multiple users.",
                    ),
                ],
            ),
            Cmd(
                "delete",
                delete_group,
                "delete a user group",
                [
                    Arg("group_name", help="name of user group to be deleted"),
                    Arg("--yes", action="store_true", help="skip prompt asking for confirmation"),
                ],
            ),
            Cmd(
                "list ls",
                list_groups,
                "list user groups",
                [
                    *default_pagination_args,
                    Arg("--groups-user-belongs-to", help="list groups that the username is in"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
                is_default=True,
            ),
            Cmd(
                "describe",
                describe_group,
                "describe a user group",
                [
                    Arg("group_name", help="name of user group to describe"),
                    Arg("--json", action="store_true", help="print as JSON"),
                ],
            ),
            Cmd(
                "add-user",
                add_user_to_group,
                "add users to a group",
                [
                    Arg("group_name", help="name of user group to add users to"),
                    Arg("usernames", help="a comma seperated list of usernames"),
                ],
            ),
            Cmd(
                "remove-user",
                remove_user_from_group,
                "remove user from a group",
                [
                    Arg("group_name", help="name of user group to remove users from"),
                    Arg("usernames", help="a comma seperated list of usernames"),
                ],
            ),
            Cmd(
                "change-name",
                change_group_name,
                "change name of a user group",
                [
                    Arg("old_group_name", help="name of user group to be updated"),
                    Arg("new_group_name", help="name of user group to change to"),
                ],
            ),
        ],
    )
]  # type: List[Any]
