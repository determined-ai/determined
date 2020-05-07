import sys
import webbrowser
from argparse import Namespace
from typing import Any, List

from determined_common import api

from .declarative_argparse import Arg, Cmd


def sso(parsed_args: Namespace) -> None:
    if not parsed_args.provider:
        print("Provider must be specified.")
        sys.exit(1)

    master_info = api.get(parsed_args.master, "info", authenticated=False).json()

    sso_providers = master_info["sso_providers"]
    if not sso_providers:
        print("No SSO providers found.")
        return

    requested_providers = [
        p for p in sso_providers if p["name"].lower() == parsed_args.provider.lower()
    ]

    if not requested_providers:
        print(
            "Provider {} unsupported. (Providers found: {})".format(
                parsed_args.provider, ", ".join(sso_providers)
            )
        )
        return

    if len(requested_providers) > 1:
        print("Multiple SSO providers found with name {}.".format(parsed_args.provider))
        return

    requested_provider = requested_providers[0]
    sso_url = requested_provider["sso_url"] + "?relayState=cli%3Dtrue"

    webbrowser.open(sso_url)
    print(
        "Your browser should open and prompt you to sign on; if it did not, please visit {}".format(
            sso_url
        )
    )
    raw_token = input("Enter the token you receive here: ")
    token = raw_token.strip()

    tmp_auth = {"Cookie": "auth={token}".format(token=token)}
    me = api.get(parsed_args.master, "/users/me", headers=tmp_auth, authenticated=False).json()

    token_store = api.Authentication.instance().token_store
    token_store.set_token(me["username"], token)
    token_store.set_active(me["username"], True)

    print(f"Authenticated as {me['username']}.")


def list_providers(parsed_args: Namespace) -> None:
    master_info = api.get(parsed_args.master, "info", authenticated=False).json()

    sso_providers = master_info["sso_providers"]
    if len(sso_providers) == 0:
        print("No SSO providers found.")
        return

    print("Available providers: " + ", ".join(provider["name"] for provider in sso_providers) + ".")


# fmt: off

args_description = [
    Cmd("auth", None, "manage auth", [
        Cmd("login", sso, "sign on with an auth provider", [
            Arg("provider", default=None, type=str, help="auth provider to use")
        ]),
        Cmd("list-providers", list_providers, "lists the available auth providers", []),
    ])
]  # type: List[Any]

# fmt: on
