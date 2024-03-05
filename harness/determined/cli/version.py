import argparse
import sys
from typing import Any, Dict, List

import requests
import termcolor
from packaging import version

import determined as det
from determined import cli
from determined.cli import render
from determined.common import api


def get_version(sess: api.BaseSession) -> Dict[str, Any]:
    client_info = {"version": det.__version__}

    master_info = {"cluster_id": "", "master_id": "", "version": ""}

    try:
        master_info = sess.get("info").json()
        # Most connection errors mean that the master is unreachable, which this function handles.
        # An SSL error, however, means it was reachable but something went wrong, so let that error
        # propagate out.
    except requests.exceptions.SSLError:
        raise
    except api.errors.MasterNotFoundException:
        pass

    return {"client": client_info, "master": master_info, "master_address": sess.master}


def check_version(sess: api.BaseSession, args: argparse.Namespace) -> None:
    info = get_version(sess)

    master_version = info["master"]["version"]
    client_version = info["client"]["version"]
    if not master_version:
        print(
            termcolor.colored(
                "Master not found at {}. "
                "Hint: Remember to set the DET_MASTER environment variable "
                "to the correct Determined master IP and port or use the '-m' flag.".format(
                    args.master
                ),
                "yellow",
            ),
            file=sys.stderr,
        )
    elif version.Version(client_version) < version.Version(master_version):
        print(
            termcolor.colored(
                "CLI version {} is less than master version {}. "
                "Consider upgrading the CLI.".format(client_version, master_version),
                "yellow",
            ),
            file=sys.stderr,
        )
    elif version.Version(client_version) > version.Version(master_version):
        print(
            termcolor.colored(
                "Master version {} is less than CLI version {}. "
                "Consider upgrading the master.".format(master_version, client_version),
                "yellow",
            ),
            file=sys.stderr,
        )


def describe_version(args: argparse.Namespace) -> None:
    sess = cli.unauth_session(args)
    info = get_version(sess)

    print(render.format_object_as_yaml(info))


args_description = [
    cli.Cmd("version", describe_version, "show version information", [])
]  # type: List[Any]
