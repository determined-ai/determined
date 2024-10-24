import argparse
import json
from typing import Any, List

from determined import cli
from determined.cli import errors, render
from determined.common import api, util
from determined.common.api import authentication
from determined.common.experimental import token
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


def render_token_info(token_info: List[token.AccessToken]) -> None:
    values = [
        [t.id, t.user_id, t.description, t.created_at, t.expiry, t.revoked, t.token_type]
        for t in token_info
    ]
    render.tabulate_or_csv(TOKEN_HEADERS, values, False)


def describe_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        d = client.Determined._from_session(sess)
        token_info = d.describe_tokens(args.token_id)

        if args.json or args.yaml:
            json_data = [t.to_json() for t in token_info]
            print(json_data)
            if args.json:
                render.print_json(json_data)
            else:
                print(util.yaml_safe_dump(json_data, default_flow_style=False))
        else:
            render_token_info(token_info)
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except Exception as e:
        raise errors.CliError(f"Error fetching tokens: {e}")


def list_tokens(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        username = args.username if args.username else None
        show_inactive = True if args.show_inactive else False
        d = client.Determined._from_session(sess)
        token_info = d.list_tokens(username, show_inactive)

        if args.json or args.yaml:
            json_data = [t.to_json() for t in token_info]
            if args.json:
                render.print_json(json_data)
            else:
                print(util.yaml_safe_dump(json_data, default_flow_style=False))
        else:
            render_token_info(token_info)
    except Exception as e:
        raise errors.CliError(f"Error fetching tokens: {e}")


def revoke_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        d = client.Determined._from_session(sess)
        print(args.token_id)
        token_info_list = d.describe_token(args.token_id)
        # Only one token will be returned, use the first one
        token_info = token_info_list[0]
        token_info.revoke_token()
        render_token_info([token_info])
        print(f"Successfully revoked token {args.token_id}.")
    except api.errors.NotFoundException:
        raise errors.CliError("Token not found")


def create_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        username = args.username or sess.username
        d = client.Determined._from_session(sess)
        user_obj = d.get_user_by_name(username)
        if user_obj is None or user_obj.user_id is None:
            raise errors.CliError(f"User '{username}' not found or does not have an ID")

        # convert days into hours Go duration format
        expiration_in_hours = None
        if args.expiration_days:
            expiration_in_hours = str(24 * args.expiration_days) + "h"

        token_info = d.create_token(user_obj.user_id, expiration_in_hours, args.description)

        output_string = None
        if args.yaml:
            output_string = util.yaml_safe_dump(token_info.to_json(), default_flow_style=False)
        elif args.json:
            output_string = json.dumps(token_info.to_json(), indent=2)
        else:
            output_string = f"TokenID: {token_info.tokenId}\nAccess-Token: {token_info.token}"

        print(output_string)
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except api.errors.NotFoundException as e:
        raise errors.CliError(f"Caught NotFoundException: {str(e)}")


def edit_token(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        if args.token_id:
            d = client.Determined._from_session(sess)
            token_info_list = d.describe_token(args.token_id)
            # Only one token will be returned, use the first one
            token_info = token_info_list[0]
            if args.description:
                token_info.edit_token(args.description)
                if args.json or args.yaml:
                    json_data = token_info.to_json()
                    print(json_data)
                    if args.json:
                        render.print_json(json_data)
                    else:
                        print(util.yaml_safe_dump(json_data, default_flow_style=False))
                else:
                    render_token_info([token_info])
                print(f"Successfully updated token with ID: {args.token_id}.")
            else:
                raise errors.CliError(
                    f"Please provide a description for token ID '{args.token_id}'."
                )
    except api.errors.APIException as e:
        raise errors.CliError(f"Caught APIException: {str(e)}")
    except api.errors.NotFoundException:
        raise errors.CliError("Token not found")


def login_with_token(args: argparse.Namespace) -> None:
    try:
        sess = authentication.login_with_token(
            master_address=args.master, token=args.token, cert=cli.cert
        )
        print(f"Authenticated as {sess.username}.")
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
