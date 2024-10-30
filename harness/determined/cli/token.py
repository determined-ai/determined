import argparse
import json
from typing import Any, List, Sequence

from determined import cli
from determined.cli import errors, render
from determined.common import api, util
from determined.common.api import authentication, bindings

TOKEN_HEADERS = [
    "ID",
    "User ID",
    "Description",
    "Created At",
    "Expires At",
    "Revoked",
    "Token Type",
]


def render_token_info(token_info: Sequence[bindings.v1TokenInfo]) -> None:
    values = [
        [t.id, t.userId, t.description, t.createdAt, t.expiry, t.revoked, t.tokenType]
        for t in token_info
    ]
    render.tabulate_or_csv(TOKEN_HEADERS, values, False)


def describe_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        resp = bindings.get_GetAccessTokens(session=sess, tokenIds=args.token_id)
        if args.json or args.yaml:
            json_data = [t.to_json() for t in resp.tokenInfo]
            if args.json:
                render.print_json(json_data)
            else:
                print(util.yaml_safe_dump(json_data, default_flow_style=False))
        else:
            render_token_info(resp.tokenInfo)
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except Exception as e:
        raise errors.CliError(f"Error fetching tokens: {e}")


def list_tokens(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        username = args.username if args.username else None
        show_inactive = True if args.show_inactive else False
        resp = bindings.get_GetAccessTokens(sess, username=username, showInactive=show_inactive)
        if args.json or args.yaml:
            json_data = [t.to_json() for t in resp.tokenInfo]
            if args.json:
                render.print_json(json_data)
            else:
                print(util.yaml_safe_dump(json_data, default_flow_style=False))
        else:
            render_token_info(resp.tokenInfo)
    except Exception as e:
        raise errors.CliError(f"Error fetching tokens: {e}")


def revoke_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        request = bindings.v1PatchAccessTokenRequest(
            tokenId=args.token_id, description=None, setRevoked=True
        )
        resp = bindings.patch_PatchAccessToken(sess, body=request, tokenId=args.token_id)
        print(json.dumps(resp.to_json(), indent=2))
        print(f"Successfully revoked token {args.token_id}.")
    except api.errors.NotFoundException:
        raise errors.CliError("Token not found")


def create_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        username = args.username or sess.username
        user = bindings.get_GetUserByUsername(session=sess, username=username).user

        if user is None or user.id is None:
            raise errors.CliError(f"User '{username}' not found or does not have an ID")

        # convert days into hours Go duration format
        expiration_in_hours = None
        if args.expiration_days is not None:
            expiration_in_hours = (
                "-1" if args.expiration_days == -1 else f"{24 * args.expiration_days}h"
            )

        request = bindings.v1PostAccessTokenRequest(
            userId=user.id, lifespan=expiration_in_hours, description=args.description
        )
        resp = bindings.post_PostAccessToken(sess, body=request).to_json()

        output_string = None
        if args.yaml:
            output_string = util.yaml_safe_dump(resp, default_flow_style=False)
        elif args.json:
            output_string = json.dumps(resp, indent=2)
        else:
            output_string = f'TokenID: {resp["tokenId"]}\nAccess-Token: {resp["token"]}'

        print(output_string)
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except api.errors.NotFoundException as e:
        raise errors.CliError(f"Caught NotFoundException: {str(e)}")


def edit_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        if args.token_id:
            request = bindings.v1PatchAccessTokenRequest(
                tokenId=args.token_id,
                description=args.description if args.description else None,
                setRevoked=False,
            )
            resp = bindings.patch_PatchAccessToken(sess, body=request, tokenId=args.token_id)
            print(json.dumps(resp.to_json(), indent=2))
            print(f"Successfully updated token with ID: {args.token_id}.")
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except api.errors.NotFoundException:
        raise errors.CliError("Token not found")


def login_with_token(args: argparse.Namespace) -> None:
    try:
        unauth_session = api.UnauthSession(master=args.master, cert=cli.cert)
        auth_headers = {"Authorization": f"Bearer {args.token}"}
        user_data = unauth_session.get("/api/v1/me", headers=auth_headers).json()
        username = user_data.get("user").get("username")

        token_store = authentication.TokenStore(args.master)
        token_store.set_token(username, args.token)
        token_store.set_active(username)
        print(f"Authenticated as {username}.")
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except api.errors.UnauthenticatedException as e:
        raise errors.CliError(f"Caught UnauthenticatedException: {str(e)}")
    except api.errors.NotFoundException as e:
        raise errors.CliError(f"Caught NotFoundException: {str(e)}")


# fmt: off

args_description = [
    cli.Cmd("token tkn", None, "manage access tokens", [
        cli.Cmd("describe", describe_token, "describe token info", [
            cli.Arg("token_id", type=int, nargs=argparse.ONE_OR_MORE, default=None,
                    help="token id(s) specifying access tokens to describe"),
            cli.Group(
                cli.output_format_args["json"],
                cli.output_format_args["yaml"],
            ),
        ]),
        cli.Cmd("list ls", list_tokens, "list access tokens accessible to users", [
            cli.Arg("username", type=str, nargs=argparse.OPTIONAL,
                    help="list access tokens for the given username", default=None),
            cli.Arg("--show-inactive", action="store_true", default=None,
                    help="list all access tokens accessible to the current user"),
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
            cli.Arg("--expiration-days", "-e", type=int,
                    help="specify the token expiration in days. '-e 2' sets it to 2 days."),
            cli.Arg("--description", "-d", type=str, default=None,
                    help="description of new token"),
            cli.Group(
                cli.output_format_args["json"],
                cli.output_format_args["yaml"],
            ),
        ]),
        cli.Cmd("edit", edit_token, "edit token info", [
            cli.Arg("token_id", help="edit given access token info"),
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
    ])
]  # type: List[Any]

# fmt: on
