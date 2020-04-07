import getpass
from argparse import Namespace
from collections import namedtuple
from functools import wraps
from typing import Any, Callable, Dict, List, Optional

from requests import Response
from termcolor import colored

import determined_common.api.authentication as auth
from determined_common import api
from determined_common.api import gql

from . import render
from .declarative_argparse import Arg, Cmd

FullUser = namedtuple(
    "FullUser",
    ["username", "admin", "active", "agent_uid", "agent_gid", "agent_user", "agent_group"],
)


def authentication_optional(func: Callable[[Namespace], Any]) -> Callable[[Namespace], Any]:
    @wraps(func)
    def f(namespace: Namespace) -> Any:
        v = vars(namespace)
        try:
            auth.initialize_session(namespace.master, v.get("user"), try_reauth=False)
        except api.errors.UnauthenticatedException:
            pass

        return func(namespace)

    return f


def authentication_required(func: Callable[[Namespace], Any]) -> Callable[..., Any]:
    @wraps(func)
    def f(namespace: Namespace) -> Any:
        v = vars(namespace)
        auth.initialize_session(namespace.master, v.get("user"), try_reauth=True)
        return func(namespace)

    return f


def update_user(
    username: str,
    master_address: str,
    active: Optional[bool] = None,
    password: Optional[str] = None,
    agent_user_group: Optional[Dict[str, Any]] = None,
) -> Response:
    if active is None and password is None and agent_user_group is None:
        raise Exception("Internal error (must supply at least one kwarg to update_user).")

    request = {}  # type: Dict[str, Any]
    if active is not None:
        request["active"] = active

    if password is not None:
        request["password"] = password

    if agent_user_group is not None:
        request["agent_user_group"] = agent_user_group

    return api.patch(master_address, "users/{}".format(username), body=request)


@authentication_required
def list_users(parsed_args: Namespace) -> None:
    q = api.GraphQLQuery(parsed_args.master)
    users = q.op.users(order_by=[gql.users_order_by(id=gql.order_by.asc)])
    users.id()
    users.username()
    users.active()
    users.admin()

    groups = users.agent_user_group()
    groups.uid()
    groups.gid()
    groups.user_()
    groups.group_()
    resp = q.send()

    def user_to_dict(u: gql.users) -> Dict[str, Any]:
        a = u.agent_user_group.__to_json_value__()
        return {
            **u.__to_json_value__(),
            "agent_uid": a.get("gid"),
            "agent_gid": a.get("uid"),
            "agent_user": a.get("user_"),
            "agent_group": a.get("group_"),
        }

    render.render_dicts(FullUser, [user_to_dict(u) for u in resp.users])


@authentication_required
def activate_user(parsed_args: Namespace) -> None:
    update_user(parsed_args.username, parsed_args.master, active=True)


@authentication_required
def deactivate_user(parsed_args: Namespace) -> None:
    update_user(parsed_args.username, parsed_args.master, active=False)


def log_in_user(parsed_args: Namespace) -> None:
    if parsed_args.username is None:
        username = input("Username: ")
    else:
        username = parsed_args.username

    message = "Password for user '{}': ".format(username)

    # In order to not send clear-text passwords, we hash the password.
    password = api.salt_and_hash(getpass.getpass(message))

    auth_inst = api.Authentication.instance()

    auth.do_login(parsed_args.master, auth_inst, username=username, password=password)
    auth_inst.token_store.set_active(username, True)


@authentication_optional
def log_out_user(parsed_args: Namespace) -> None:
    auth_inst = api.Authentication.instance()
    if auth_inst.session is None:
        return

    try:
        api.post(
            parsed_args.master,
            "logout",
            headers={"Authorization": "Bearer {}".format(auth_inst.get_session_token())},
            authenticated=False,
        )
    except api.errors.APIException as e:
        if e.status_code != 401:
            raise e

    auth_inst.token_store.drop_user(auth_inst.get_session_user())


@authentication_required
def change_password(parsed_args: Namespace) -> None:
    auth_inst = api.Authentication.instance()

    if parsed_args.target_user:
        username = parsed_args.target_user
    elif parsed_args.user:
        username = parsed_args.user
    else:
        username = auth_inst.get_session_user()

    if not username:
        # The default user should have been set by now by autologin.
        print(colored("Please log in as an admin or user to change passwords", "red"))
        return

    # If the target user's password isn't being changed by another user, reauthenticate after
    # password change so that the user doesn't have to do so manually.
    reauthenticate = parsed_args.target_user is None

    password = getpass.getpass("New password for user '{}': ".format(username))
    check_password = getpass.getpass("Confirm password: ")

    if password != check_password:
        print(colored("Passwords do not match", "red"))
        return

    # Hash the password to avoid sending it in cleartext.
    password = api.salt_and_hash(password)

    update_user(username, parsed_args.master, password=password)

    if reauthenticate:
        set_active = auth_inst.is_user_active(username)
        auth_inst = api.Authentication.instance()

        auth.do_login(parsed_args.master, auth_inst, username, password)
        auth_inst.token_store.set_active(username, set_active)


@authentication_required
def link_with_agent_user(parsed_args: Namespace) -> None:
    if parsed_args.agent_uid is None:
        raise api.errors.BadRequestException("agent-uid argument required")
    elif parsed_args.agent_user is None:
        raise api.errors.BadRequestException("agent-user argument required")
    elif parsed_args.agent_gid is None:
        raise api.errors.BadRequestException("agent-gid argument required")
    elif parsed_args.agent_group is None:
        raise api.errors.BadRequestException("agent-group argument required")

    agent_user_group = {
        "uid": parsed_args.agent_uid,
        "user": parsed_args.agent_user,
        "gid": parsed_args.agent_gid,
        "group": parsed_args.agent_group,
    }

    update_user(parsed_args.det_username, parsed_args.master, agent_user_group=agent_user_group)


@authentication_required
def create_user(parsed_args: Namespace) -> None:
    username = parsed_args.username
    admin = bool(parsed_args.admin)

    request = {"username": username, "admin": admin, "active": True}
    api.post(parsed_args.master, "users", body=request)


@authentication_required
def whoami(parsed_args: Namespace) -> None:
    response = api.get(parsed_args.master, "users/me")
    user = response.json()

    print("You are logged in as user '{}'".format(user["username"]))


# fmt: off

args_description = [
    Cmd("u|ser", None, "manage users", [
        Cmd("list", list_users, "list users", [], is_default=True),
        Cmd("login", log_in_user, "log in user", [
            Arg("username", nargs="?", default=None, help="name of user to log in as")
        ]),
        Cmd("change-password", change_password, "change password for user", [
            Arg("target_user", nargs="?", default=None, help="name of user to change password of")
        ]),
        Cmd("logout", log_out_user, "log out user", []),
        Cmd("activate", activate_user, "activate user", [
            Arg("username", help="name of user to activate")
        ]),
        Cmd("deactivate", deactivate_user, "deactivate user", [
            Arg("username", help="name of user to deactivate")
        ]),
        Cmd("create", create_user, "create user", [
            Arg("username", help="name of new user"),
            Arg("--admin", action="store_true", help="give new user admin rights"),
        ]),
        Cmd("link-with-agent-user", link_with_agent_user, "link a user with UID/GID on agent", [
            Arg("det_username", help="name of Determined user to link"),
            Arg("--agent-uid", type=int, help="UID on the agent to run tasks as"),
            Arg("--agent-user", help="user on the agent to run tasks as"),
            Arg("--agent-gid", type=int, help="GID on agent to run tasks as"),
            Arg("--agent-group", help="group on the agent to run tasks as"),
        ]),
        Cmd("whoami", whoami, "print the active user", [])
    ])
]  # type: List[Any]

# fmt: on
