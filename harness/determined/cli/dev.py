import shutil
import subprocess
import sys
from argparse import Namespace

from termcolor import colored

from determined.common.api import authentication
from determined.common.api.request import make_url
from determined.common.declarative_argparse import Arg, Cmd


@authentication.required
def token(_: Namespace) -> None:
    assert authentication.cli_auth is not None
    token = authentication.cli_auth.get_session_token()
    print(token)


@authentication.required
def curl(args: Namespace) -> None:
    assert authentication.cli_auth is not None
    if shutil.which("curl") is None:
        print(colored("curl is not installed on this machine", "red"))
        sys.exit(1)
    cmd = [
        "curl",
        make_url(args.master, args.path),
        "-H",
        f"'Authorization: Bearer {authentication.cli_auth.get_session_token()}'",
        "-s",
        args.curl_args or "",
    ]

    if shutil.which("jq") is not None:
        cmd.append("| jq .")

    output = subprocess.run(" ".join(cmd), shell=True)
    sys.exit(output.returncode)


args_description = [
    Cmd(
        "dev",
        None,
        "dev utilities",
        [
            Cmd("auth-token", token, "print the active user's auth token", []),
            Cmd(
                "curl",
                curl,
                "invoke curl",
                [
                    Arg("path", help="path to curl (e.g. /api/v1/experiments?x=z)"),
                    Arg("curl_args", nargs="?"),
                ],
            ),
        ],
    ),
]
