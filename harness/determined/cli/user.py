import argparse
import collections
import getpass
import json
from typing import Any, List, Sequence

from determined import cli
from determined.cli import errors, render
from determined.common import api, util
from determined.common.api import authentication, bindings
from determined.experimental import client

TOKEN_HEADERS = [
    "ID",
    "User ID",
    "Description",
    "Created At",
    "Expires At",
    "Revoked",
    "Token Type",
]

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


def render_token_info(token_info: Sequence[bindings.v1TokenInfo]) -> None:
    values = [
        [t.id, t.userId, t.description, t.createdAt, t.expiry, t.revoked, t.tokenType]
        for t in token_info
    ]
    render.tabulate_or_csv(TOKEN_HEADERS, values, False)


def describe_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    filter_data = json.dumps({"Token_Ids": args.token_id})
    try:
        resp = bindings.get_GetAccessTokens(session=sess, filter=filter_data)
        render_token_info(resp.tokenInfo)
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except Exception as e:
        raise errors.CliError(f"Error fetching tokens: {e}")


def list_tokens(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    filter_data = {
        **({"Username": args.username} if args.username else {}),
        **({"only_Active": args.only_active} if args.only_active else {}),
    }
    try:
        filter_json = json.dumps(filter_data)
        resp = bindings.get_GetAccessTokens(sess, filter=filter_json)
        if args.json or args.yaml:
            json_data = [t.to_json() for t in resp.tokenInfo]
            if args.json:
                render.print_json(json_data)
            else:
                print(util.yaml_safe_dump(json_data, default_flow_style=False))
        else:
            render_token_info(resp.tokenInfo)
    except Exception as e:
        print(f"Error fetching tokens: {e}")


def revoke_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        request = bindings.v1PatchAccessTokenRequest(
            tokenId=args.token_id, description=None, setRevoked=True
        )
        resp = bindings.patch_PatchAccessToken(sess, body=request, tokenId=args.token_id)
        print(json.dumps(resp.to_json(), indent=2))
    except api.errors.NotFoundException:
        raise errors.CliError("Token not found")
    print(f"Successfully updated token with ID: {args.token_id}.")


def create_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    username = args.username or sess.username
    user_id = bindings.get_GetUserByUsername(session=sess, username=username).user.id

    request = None
    request = bindings.v1PostAccessTokenRequest(
        lifespan=args.expiration_duration, userId=user_id, description=args.description
    )
    resp = bindings.post_PostAccessToken(sess, userId=user_id, body=request).to_json()

    output_string = None
    if args.yaml:
        output_string = util.yaml_safe_dump(resp, default_flow_style=False)
    elif args.json:
        output_string = json.dumps(resp, indent=2)
    else:
        output_string = f'{resp["token"]}\n{resp["tokenId"]}'

    if args.output_file:
        with open(args.output_file, "w") as file:
            file.write(output_string)
    else:
        print(output_string)


def edit_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        if args.description and args.token_id:
            request = bindings.v1PatchAccessTokenRequest(
                tokenId=args.token_id, description=args.description, setRevoked=False
            )
            resp = bindings.patch_PatchAccessToken(sess, body=request, tokenId=args.token_id)
        print(json.dumps(resp.to_json(), indent=2))
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except api.errors.NotFoundException:
        raise errors.CliError("Token not found")
    print(f"Successfully updated token with ID: {args.token_id}.")


def login_with_token(args: argparse.Namespace) -> None:
    unauth_session = api.UnauthSession(master=args.master, cert=cli.cert)
    auth_headers = {"Authorization": f"Bearer {args.token}"}
    user_data = unauth_session.get("/api/v1/me", headers=auth_headers).json()
    if "user" in user_data and "username" in user_data.get("user"):
        username = user_data.get("user").get("username")

    token_store = authentication.TokenStore(args.master)
    token_store.set_token(username, args.token)
    token_store.set_active(username)
    print(f"Authenticated as {username}.")


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
            cli.Arg("username", nargs="?", default=None, help="name of user to log in as"),
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
        cli.Cmd("token", None, "manage access tokens", [
            cli.Cmd("describe", describe_token, "describe token info", [
                cli.Arg("token_id", type=int, nargs=argparse.ONE_OR_MORE, default=None,
                        help="token id(s) specifying access tokens to describe"),
                cli.Group(
                    cli.output_format_args["json"],
                    cli.output_format_args["yaml"],
                ),
            ]),
            cli.Cmd("list ls", list_tokens, "list all active access tokens", [
                cli.Arg("username", type=str, nargs=argparse.OPTIONAL,
                        help="list token for the given username", default=None),
                cli.Arg("--only-active", action="store_true", default=None,
                        help="list only the active tokens"),
                cli.Group(
                    cli.output_format_args["json"],
                    cli.output_format_args["yaml"],
                ),
            ]),
            cli.Cmd("revoke", revoke_token, "revoke token", [
                cli.Arg("token_id", help="revoke given access token id"),
            ]),
            cli.Cmd("create", create_token, "create token", [
                cli.Arg("username", type=str, nargs=argparse.OPTIONAL,
                        help="name of user to create token", default=None),
                cli.Arg("--expiration-duration", "-e", type=str,
                        help="give expiry duration like 2h or 5m or 10s"),
                cli.Arg("--description", "-d", type=str, default=None,
                        help="description of new token"),
                cli.Arg("--output-file", "-o", type=str, help="write token to a file"),
                cli.Group(
                    cli.output_format_args["json"],
                    cli.output_format_args["yaml"],
                ),
            ]),
            cli.Cmd("edit", edit_token, "edit token info", [
                cli.Arg("token_id", help="edit given access token"),
                cli.Arg("--description", "-d", type=str, default=None,
                        help="description of token to edit"),
                cli.Group(
                    cli.output_format_args["json"],
                    cli.output_format_args["yaml"],
                ),
            ]),
            cli.Cmd("login", login_with_token, "log in with token", [
                cli.Arg("token", help="token to use for authentication", default=None),
            ]),
        ]),
    ])
]  # type: List[Any]

# fmt: on
