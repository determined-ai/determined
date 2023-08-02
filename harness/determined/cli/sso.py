import sys
import webbrowser
from argparse import Namespace
from getpass import getpass
from http.server import BaseHTTPRequestHandler, HTTPServer
from typing import Any, Callable, List
from urllib.parse import parse_qs, urlparse

from determined.common import api
from determined.common.api import authentication
from determined.common.declarative_argparse import Arg, Cmd
from determined.errors import EnterpriseOnlyError

CLI_REDIRECT_PORT = 49176


def handle_token(master_url: str, token: str) -> None:
    tmp_auth = {"Cookie": "auth={token}".format(token=token)}
    me = api.get(master_url, "/users/me", headers=tmp_auth, authenticated=False).json()

    token_store = authentication.TokenStore(master_url)
    token_store.set_token(me["username"], token)
    token_store.set_active(me["username"])

    print("Authenticated as {}.".format(me["username"]))


def make_handler(master_url: str, close_cb: Callable[[int], None]) -> Any:
    class TokenAcceptHandler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            try:
                """Serve a GET request."""
                token = parse_qs(urlparse(self.path).query)["token"][0]
                handle_token(master_url, token)

                self.send_response(200)
                self.send_header("Content-type", "text/html")
                self.end_headers()
                self.wfile.write(b"You can close this window now.")
                close_cb(0)
            except Exception as e:
                print("Error authenticating: {}.".format(e))
                close_cb(1)

        def log_message(self, format: Any, *args: List[Any]) -> None:  # noqa: A002
            # Silence server logging.
            return

    return TokenAcceptHandler


def sso(parsed_args: Namespace) -> None:
    master_info = api.get(parsed_args.master, "info", authenticated=False).json()
    try:
        sso_providers = master_info["sso_providers"]
    except KeyError:
        raise EnterpriseOnlyError("No SSO providers data")
    if not sso_providers:
        print("No SSO providers found.")
        return
    elif not parsed_args.provider:
        if len(sso_providers) > 1:
            print("Provider must be specified when multiple are available.")
            return
        matched_provider = sso_providers[0]
    else:
        matching_providers = [
            p for p in sso_providers if p["name"].lower() == parsed_args.provider.lower()
        ]
        if not matching_providers:
            ps = ", ".join(p["name"].lower() for p in sso_providers)
            print("Provider {} unsupported. (Providers found: {})".format(parsed_args.provider, ps))
            return
        elif len(matching_providers) > 1:
            print("Multiple SSO providers found with name {}.".format(parsed_args.provider))
            return
        matched_provider = matching_providers[0]

    sso_url = matched_provider["sso_url"] + "?relayState=cli"

    if not parsed_args.headless:
        if webbrowser.open(sso_url):
            print(
                "Your browser should open and prompt you to sign on;"
                " if it did not, please visit {}".format(sso_url)
            )
            print("Killing this process before signing on will cancel authentication.")
            with HTTPServer(
                ("localhost", CLI_REDIRECT_PORT),
                make_handler(parsed_args.master, lambda code: sys.exit(code)),
            ) as httpd:
                return httpd.serve_forever()

        print("Failed to open Web Browser. Falling back to --headless CLI mode.")

    example_url = f"Example: 'http://localhost:{CLI_REDIRECT_PORT}/?token=v2.public.[long_str]'"

    print(
        f"Please open this URL in your browser: '{sso_url}'\n"
        "After authenticating, copy/paste the localhost URL "
        f"from your browser into the prompt.\n{example_url}"
    )
    token = None
    while not token:
        user_input_url = getpass(prompt="\n(hidden) localhost URL? ")
        try:
            token = parse_qs(urlparse(user_input_url).query)["token"][0]
            handle_token(parsed_args.master, token)
        except (KeyError, IndexError):
            print(f"Could not extract token from localhost URL. {example_url}")


def list_providers(parsed_args: Namespace) -> None:
    master_info = api.get(parsed_args.master, "info", authenticated=False).json()

    try:
        sso_providers = master_info["sso_providers"]
    except KeyError:
        raise EnterpriseOnlyError("No SSO providers data")

    if len(sso_providers) == 0:
        print("No SSO providers found.")
        return

    print("Available providers: " + ", ".join(provider["name"] for provider in sso_providers) + ".")


# fmt: off

args_description = [
    Cmd("auth", None, "manage auth", [
        Cmd("login", sso, "sign on with an auth provider", [
            Arg("-p", "--provider", type=str,
                help="auth provider to use (not needed if the Determined master only supports"
                " one provider)"),
            Arg("--headless", action="store_true", help="force headless cli auth")
        ]),
        Cmd("list-providers", list_providers, "lists the available auth providers", []),
    ])
]  # type: List[Any]

# fmt: on
