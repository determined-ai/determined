import argparse
import collections
import getpass
from typing import Any, List

from determined import cli
from determined.cli import errors, render
from determined.common import api
from determined.common.api import authentication, bindings
from determined.experimental import client

FullUser = collections.namedtuple(
    "FullUser",
    [
        "user_id",
        "username",
        "display_name",
        "admin",
        "active",
        "remote",
        "agent_uid",
        "agent_gid",
        "agent_user",
        "agent_group",
    ],
)
FullUserNoAdmin = collections.namedtuple(
    "FullUserNoAdmin",
    [
        "user_id",
        "username",
        "display_name",
        "active",
        "remote",
        "agent_uid",
        "agent_gid",
        "agent_user",
        "agent_group",
    ],
)


def list_users(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    resp = bindings.get_GetMaster(sess)
    users_list = d.list_users(active=None if args.all else True)
    renderer = FullUser  # type: Any
    if resp.to_json().get("rbacEnabled"):
        renderer = FullUserNoAdmin
    render.render_objects(renderer, users_list)


def activate_user(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    user_obj = d.get_user_by_name(args.username)
    user_obj.activate()


def deactivate_user(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    user_obj = d.get_user_by_name(args.username)
    user_obj.deactivate()


def log_in_user(args: argparse.Namespace) -> None:
    if args.username is None:
        username = input("Username: ")
    else:
        username = args.username

    message = "Password for user '{}': ".format(username)
    password = getpass.getpass(message)

    token_store = authentication.TokenStore(args.master)
    try:
        sess = authentication.login(args.master, username, password, cli.cert)
    except api.errors.UnauthenticatedException:
        raise api.errors.InvalidCredentialsException()

    try:
        authentication.check_password_complexity(password)
    except ValueError as e:
        authentication.warn_about_complexity(e)

    token_store.set_token(sess.username, sess.token)
    token_store.set_active(sess.username)


def log_out_user(args: argparse.Namespace) -> None:
    token_store = authentication.TokenStore(args.master)
    if args.all:
        authentication.logout_all(args.master, cli.cert)
        token_store.clear_active()
    else:
        # Log out of the user specified by the command line, or the active user.
        logged_out_user = authentication.logout(args.master, args.user, cli.cert)
        if logged_out_user and token_store.get_active_user() == logged_out_user:
            token_store.clear_active()


def rename(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    user_obj = d.get_user_by_name(args.target_user)
    user_obj.rename(new_username=args.new_username)


def change_password(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    if args.target_user:
        username = args.target_user
    elif args.user:
        username = args.user
    else:
        username = d.get_session_username()

    if not username:
        # The default user should have been set by now by autologin.
        raise errors.CliError("Please log in as an admin or user to change passwords")

    password = getpass.getpass("New password for user '{}': ".format(username))
    check_password = getpass.getpass("Confirm password: ")

    if password != check_password:
        raise errors.CliError("Passwords do not match")

    user_obj = d.get_user_by_name(username)
    user_obj.change_password(new_password=password)

    # If the target user's password isn't being changed by another user, reauthenticate after
    # password change so that the user doesn't have to do so manually.
    if args.target_user is None:
        token_store = authentication.TokenStore(args.master)
        sess = authentication.login(args.master, username, password, cli.cert)
        token_store.set_token(sess.username, sess.token)
        token_store.set_active(sess.username)


def link_with_agent_user(args: argparse.Namespace) -> None:
    if args.agent_uid is None:
        raise api.errors.BadRequestException("agent-uid argument required")
    elif args.agent_user is None:
        raise api.errors.BadRequestException("agent-user argument required")
    elif args.agent_gid is None:
        raise api.errors.BadRequestException("agent-gid argument required")
    elif args.agent_group is None:
        raise api.errors.BadRequestException("agent-group argument required")

    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    user_obj = d.get_user_by_name(args.det_username)
    user_obj.link_with_agent(
        agent_gid=args.agent_gid,
        agent_group=args.agent_group,
        agent_uid=args.agent_uid,
        agent_user=args.agent_user,
    )


def create_user(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    username = args.username
    admin = bool(args.admin)
    remote = bool(args.remote)
    password = args.password

    if not remote and not password:
        password = getpass.getpass("Password for user '{}': ".format(username))
        check_password = getpass.getpass("Confirm password: ")
        if password != check_password:
            raise errors.CliError("Passwords do not match")

    d.create_user(username=username, admin=admin, password=password, remote=remote)


def whoami(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    user = d.whoami()
    print("You are logged in as user '{}'".format(user.username))


def edit(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    user_obj = d.get_user_by_name(args.target_user)
    changes = []
    patch_user = bindings.v1PatchUser()
    if args.display_name is not None:
        patch_user.displayName = args.display_name
        changes.append("Display Name")

    if args.remote is not None:
        patch_user.remote = args.remote
        changes.append("Remote")

    if args.activate is not None:
        patch_user.active = args.activate
        changes.append("Active")

    if args.username is not None:
        patch_user.username = args.username
        changes.append("Username")

    if args.admin is not None:
        patch_user.admin = args.admin
        changes.append("Admin")

    if len(changes) > 0:
        bindings.patch_PatchUser(sess, body=patch_user, userId=user_obj.user_id)
        print("Changes made to the following fields: " + ", ".join(changes))
    else:
        raise errors.CliError("No field provided. Use 'det user edit -h' for usage.")


AGENT_USER_GROUP_ARGS = [
    cli.Arg("--agent-uid", type=int, help="UID on the agent to run tasks as"),
    cli.Arg("--agent-user", help="user on the agent to run tasks as"),
    cli.Arg("--agent-gid", type=int, help="GID on agent to run tasks as"),
    cli.Arg("--agent-group", help="group on the agent to run tasks as"),
]

# fmt: off

args_description = [
    cli.Cmd("u|ser", None, "manage users", [
        cli.Cmd("list ls", list_users, "list users", [
            cli.Arg(
                "--all",
                "-a",
                action="store_true",
                help="List all active and inactive users.",
            ),
        ], is_default=True),
        cli.Cmd("login", log_in_user, "log in user", [
            cli.Arg("username", nargs="?", default=None, help="name of user to log in as")
        ]),
        cli.Cmd("rename", rename, "change username for user", [
            cli.Arg(
                "target_user", default=None, help="name of user whose username should be changed"
            ),
            cli.Arg("new_username", default=None, help="new username for target_user"),
        ], deprecation_message="Please use 'det user edit <target_user> --username <username>'"),
        cli.Cmd("change-password", change_password, "change password for user", [
            cli.Arg(
                "target_user", nargs="?", default=None, help="name of user to change password of"
            )
        ]),
        cli.Cmd("logout", log_out_user, "log out user", [
            cli.Arg(
                "--all",
                "-a",
                action="store_true",
                help="log out of all cached sessions for the current master",
            ),
        ]),
        cli.Cmd("activate", activate_user, "activate user", [
            cli.Arg("username", help="name of user to activate")
        ], deprecation_message="Please use 'det user edit <target_user> --activate'"),
        cli.Cmd("deactivate", deactivate_user, "deactivate user", [
            cli.Arg("username", help="name of user to deactivate")
        ], deprecation_message="Please use 'det user edit <target_user> --deactivate'"),
        cli.Cmd("create", create_user, "create user", [
            cli.Arg("username", help="name of new user"),
            cli.Arg("--admin", action="store_true", help="give new user admin rights"),
            cli.Arg("--password", help="password of new user"),
            cli.Arg(
                "--remote",
                action="store_true",
                help="disallow using passwords, user must use the configured external IdP",
            ),
        ]),
        cli.Cmd("link-with-agent-user", link_with_agent_user, "link a user with UID/GID on agent", [
            cli.Arg("det_username", help="name of Determined user to link"),
            *AGENT_USER_GROUP_ARGS,
        ]),
        cli.Cmd("whoami", whoami, "print the active user", []),
        cli.Cmd("edit", edit, "edit user fields", [
            cli.Arg(
                "target_user",
                default=None,
                help="name of user that should be edited"
            ),
            cli.Arg("--display-name", default=None, help="new display name for target_user"),
            cli.Arg("--username", default=None, help="new username for target_user"),
            cli.Arg(
                "--remote",
                dest="remote",
                type=cli.string_to_bool,
                metavar="(true|false)",
                default=None,
                help="set user as remote",
            ),
            cli.Arg(
                "--active",
                dest="activate",
                type=cli.string_to_bool,
                metavar="(true|false)",
                default=None,
                help="set user as active/inactive",
            ),
            cli.Arg(
                "--admin",
                dest="admin",
                type=cli.string_to_bool,
                metavar="(true|false)",
                default=None,
                help="grant/remove user admin permissions",
            ),
        ]),
    ])
]  # type: List[Any]

# fmt: on
