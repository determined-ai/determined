import getpass
from argparse import Namespace
from collections import namedtuple
from typing import Any, List

from termcolor import colored

from determined.cli import login_sdk_client
from determined.common import api
from determined.common.api import authentication, certs
from determined.common.declarative_argparse import Arg, Cmd
from determined.experimental import client

from . import render

FullUser = namedtuple(
    "FullUser",
    ["username", "admin", "active", "agent_uid", "agent_gid", "agent_user", "agent_group"],
)


@login_sdk_client
def list_users(args: Namespace) -> None:
    render.render_objects(FullUser, client.list_users())


@login_sdk_client
def activate_user(parsed_args: Namespace) -> None:
    user_obj = client.get_user_by_name(parsed_args.username)
    user_obj.activate()


@login_sdk_client
def deactivate_user(parsed_args: Namespace) -> None:
    user_obj = client.get_user_by_name(parsed_args.username)
    user_obj.deactivate()


def log_in_user(parsed_args: Namespace) -> None:
    if parsed_args.username is None:
        username = input("Username: ")
    else:
        username = parsed_args.username

    message = "Password for user '{}': ".format(username)
    password = getpass.getpass(message)

    token_store = authentication.TokenStore(parsed_args.master)
    token = authentication.do_login(parsed_args.master, username, password, certs.cli_cert)
    token_store.set_token(username, token)
    token_store.set_active(username)


def log_out_user(parsed_args: Namespace) -> None:
    # Log out of the user specified by the command line, or the active user.
    authentication.logout(parsed_args.master, parsed_args.user, certs.cli_cert)


@login_sdk_client
def rename(parsed_args: Namespace) -> None:
    user_obj = client.get_user_by_name(parsed_args.target_user)
    user_obj.rename(new_username=parsed_args.new_username)


@login_sdk_client
def change_password(parsed_args: Namespace) -> None:
    if parsed_args.target_user:
        username = parsed_args.target_user
    elif parsed_args.user:
        username = parsed_args.user
    else:
        username = authentication.must_cli_auth().get_session_user()

    if not username:
        # The default user should have been set by now by autologin.
        print(colored("Please log in as an admin or user to change passwords", "red"))
        return

    password = getpass.getpass("New password for user '{}': ".format(username))
    check_password = getpass.getpass("Confirm password: ")

    if password != check_password:
        print(colored("Passwords do not match", "red"))
        return

    user_obj = client.get_user_by_name(username)
    user_obj.change_password(new_password=password)

    # If the target user's password isn't being changed by another user, reauthenticate after
    # password change so that the user doesn't have to do so manually.
    if parsed_args.target_user is None:
        token_store = authentication.TokenStore(parsed_args.master)
        token = authentication.do_login(parsed_args.master, username, password, certs.cli_cert)
        token_store.set_token(username, token)
        token_store.set_active(username)


@login_sdk_client
def link_with_agent_user(parsed_args: Namespace) -> None:
    if parsed_args.agent_uid is None:
        raise api.errors.BadRequestException("agent-uid argument required")
    elif parsed_args.agent_user is None:
        raise api.errors.BadRequestException("agent-user argument required")
    elif parsed_args.agent_gid is None:
        raise api.errors.BadRequestException("agent-gid argument required")
    elif parsed_args.agent_group is None:
        raise api.errors.BadRequestException("agent-group argument required")

    user_obj = client.get_user_by_name(parsed_args.det_username)
    user_obj.link_with_agent(
        agent_gid=parsed_args.agent_gid,
        agent_group=parsed_args.agent_group,
        agent_uid=parsed_args.agent_uid,
        agent_user=parsed_args.agent_user,
    )


@login_sdk_client
def create_user(parsed_args: Namespace) -> None:
    username = parsed_args.username
    admin = bool(parsed_args.admin)
    client.create_user(username=username, admin=admin)


@login_sdk_client
def whoami(parsed_args: Namespace) -> None:
    user = client.whoami()
    print("You are logged in as user '{}'".format(user.username))


AGENT_USER_GROUP_ARGS = [
    Arg("--agent-uid", type=int, help="UID on the agent to run tasks as"),
    Arg("--agent-user", help="user on the agent to run tasks as"),
    Arg("--agent-gid", type=int, help="GID on agent to run tasks as"),
    Arg("--agent-group", help="group on the agent to run tasks as"),
]

# fmt: off

args_description = [
    Cmd("u|ser", None, "manage users", [
        Cmd("list ls", list_users, "list users", [], is_default=True),
        Cmd("login", log_in_user, "log in user", [
            Arg("username", nargs="?", default=None, help="name of user to log in as")
        ]),
        Cmd("rename", rename, "change username for user", [
            Arg("target_user", default=None, help="name of user whose username should be changed"),
            Arg("new_username", default=None, help="new username for target_user"),
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
            *AGENT_USER_GROUP_ARGS,
        ]),
        Cmd("whoami", whoami, "print the active user", [])
    ])
]  # type: List[Any]

# fmt: on
