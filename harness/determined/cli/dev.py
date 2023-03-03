import argparse
import json
import shlex
import shutil
import subprocess
import sys
from argparse import Namespace
from typing import Any, List

from termcolor import colored

from determined.common.api import authentication
from determined.common.api.request import make_url
from determined.common.declarative_argparse import Arg, Cmd


@authentication.required
def token(_: Namespace) -> None:
    token = authentication.must_cli_auth().get_session_token()
    print(token)


@authentication.required
def curl(args: Namespace) -> None:
    assert authentication.cli_auth is not None
    if shutil.which("curl") is None:
        print(colored("curl is not installed on this machine", "red"))
        sys.exit(1)

    cmd: List[str] = [
        "curl",
        make_url(args.master, args.path),
        "-H",
        f"Authorization: Bearer {authentication.cli_auth.get_session_token()}",
        "-s",
    ]
    if args.curl_args:
        cmd += args.curl_args

    if args.x:
        if hasattr(shlex, "join"):  # added in py 3.8
            print(shlex.join(cmd))  # type: ignore
        else:
            print(" ".join(shlex.quote(arg) for arg in cmd))
    output = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    if output.stderr:
        print(output.stderr.decode("utf8"), file=sys.stderr)

    out = output.stdout.decode("utf8")
    try:
        json_resp = json.loads(out)
        if shutil.which("jq") is not None:
            subprocess.run(["jq", "."], input=out, text=True)
        else:
            print(json.dumps(json_resp, indent=4))
    except json.decoder.JSONDecodeError:
        print(out)

    sys.exit(output.returncode)


args_description = [
    Cmd(
        "dev",
        None,
        argparse.SUPPRESS,
        [
            Cmd("auth-token", token, "print the active user's auth token", []),
            Cmd(
                "curl",
                curl,
                "invoke curl",
                [
                    Arg(
                        "-x", help="display the curl command that will be run", action="store_true"
                    ),
                    Arg("path", help="path to curl (e.g. /api/v1/experiments?x=z)"),
                    Arg("curl_args", nargs=argparse.REMAINDER, help="curl arguments"),
                ],
            ),
        ],
    ),
]  # type: List[Any]
