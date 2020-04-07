import argparse
from typing import Any, Dict, List

import requests
import termcolor
from packaging import version

import determined_cli
from determined_common import api

from . import render
from .declarative_argparse import Cmd


def get_version(host: str) -> Dict[str, Any]:
    client_info = {"version": determined_cli.__version__}

    master_info = {"cluster_id": "", "master_id": "", "version": ""}

    try:
        master_info = api.get(host, "info", authenticated=False).json()
        # Most connection errors mean that the master is unreachable, which this function handles.
        # An SSL error, however, means it was reachable but something went wrong, so let that error
        # propagate out.
    except requests.exceptions.SSLError:
        raise
    except api.errors.MasterNotFoundException:
        pass

    return {"client": client_info, "master": master_info, "master_address": host}


def check_version(parsed_args: argparse.Namespace) -> None:
    info = get_version(parsed_args.master)

    master_version = info["master"]["version"]
    client_version = info["client"]["version"]
    if not master_version:
        print(
            termcolor.colored(
                "Master not found at {}. "
                "Hint: Remember to set the DET_MASTER environment variable "
                "to the correct Determined master IP or use the '-m' flag.".format(
                    parsed_args.master
                ),
                "yellow",
            )
        )
    elif version.Version(client_version) < version.Version(master_version):
        print(
            termcolor.colored(
                "CLI version {} is less than master version {}. "
                "Consider upgrading the CLI.".format(client_version, master_version),
                "yellow",
            )
        )
    elif version.Version(client_version) > version.Version(master_version):
        print(
            termcolor.colored(
                "Master version {} is less than CLI version {}. "
                "Consider upgrading the master.".format(master_version, client_version),
                "yellow",
            )
        )


def describe_version(parsed_args: argparse.Namespace) -> None:
    info = get_version(parsed_args.master)

    print(render.format_object_as_yaml(info))


args_description = [
    Cmd("version", describe_version, "show version information", [])
]  # type: List[Any]
