import getpass
import subprocess
import sys
import tempfile
from argparse import ONE_OR_MORE, FileType, Namespace
from pathlib import Path
from typing import Any, Dict, List

from termcolor import colored

from determined.cli import command
from determined.common import api
from determined.common.api import request
from determined.common.api.authentication import authentication_required
from determined.common.check import check_eq, check_len
from determined.common.declarative_argparse import Arg, Cmd

from .command import (
    CONFIG_DESC,
    CONTEXT_DESC,
    VOLUME_DESC,
    launch_command,
    parse_config,
    render_event_stream,
)


@authentication_required
def start_shell(args: Namespace) -> None:
    data = {}
    if args.passphrase:
        data["passphrase"] = getpass.getpass("Enter new passphrase: ")
    config = parse_config(args.config_file, None, args.config, args.volume)
    resp = launch_command(
        args.master,
        "api/v1/shells",
        config,
        args.template,
        context_path=args.context,
        data=data,
    )["shell"]

    if args.detach:
        print(resp["id"])
        return

    ready = False
    with api.ws(args.master, "shells/{}/events".format(resp["id"])) as ws:
        for msg in ws:
            if msg["service_ready_event"]:
                ready = True
                break
            render_event_stream(msg)
    if ready:
        shell = api.get(args.master, "api/v1/shells/{}".format(resp["id"])).json()["shell"]
        check_eq(shell["state"], "STATE_RUNNING", "Shell must be in a running state")
        _open_shell(args.master, shell, args.ssh_opts)


@authentication_required
def open_shell(args: Namespace) -> None:
    shell = api.get(args.master, "api/v1/shells/{}".format(args.shell_id)).json()["shell"]
    check_eq(shell["state"], "STATE_RUNNING", "Shell must be in a running state")
    _open_shell(args.master, shell, args.ssh_opts)


def _open_shell(master: str, shell: Dict[str, Any], additional_opts: List[str]) -> None:
    with tempfile.NamedTemporaryFile("w") as fp:
        fp.write(shell["privateKey"])
        fp.flush()
        check_len(shell["addresses"], 1, "Cannot find address for shell")
        _, port = shell["addresses"][0]["host_ip"], shell["addresses"][0]["host_port"]

        # Use determined.cli.tunnel as a portable script for using the HTTP CONNECT mechanism,
        # similar to `nc -X CONNECT -x ...` but without any dependency on external binaries.
        python = sys.executable
        proxy_cmd = "{} -m determined.cli.tunnel {} %h".format(python, master)
        if request.get_master_cert_bundle() is not None:
            proxy_cmd += ' --cert-file "{}"'.format(request.get_master_cert_bundle())
        if request.get_master_cert_name():
            proxy_cmd += ' --cert-name "{}"'.format(request.get_master_cert_name())

        username = shell["agentUserGroup"]["user"] or "root"

        cmd = [
            "ssh",
            "-o",
            "ProxyCommand={}".format(proxy_cmd),
            "-o",
            "StrictHostKeyChecking=no",
            "-tt",
            "-o",
            "IdentitiesOnly=yes",
            "-i",
            str(fp.name),
            "-p",
            str(port),
            "{}@{}".format(username, shell["id"]),
            *additional_opts,
        ]

        subprocess.run(cmd)

        print(colored("To reconnect, run: det shell open {}".format(shell["id"]), "green"))


# fmt: off

args_description = [
    Cmd("shell", None, "manage shells", [
        Cmd("list", command.list, "list shells", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all shells (including other users')")
        ], is_default=True),
        Cmd("config", command.config,
            "display shell config", [
                Arg("id", type=str, help="shell ID"),
            ]),
        Cmd("start", start_shell, "start a new shell", [
            Arg("ssh_opts", nargs="*", help="additional SSH options when connecting to the shell"),
            Arg("--config-file", default=None, type=FileType("r"),
                help="command config file (.yaml)"),
            Arg("-v", "--volume", action="append", default=[],
                help=VOLUME_DESC),
            Arg("-c", "--context", default=None, type=Path, help=CONTEXT_DESC),
            Arg("--config", action="append", default=[], help=CONFIG_DESC),
            Arg("-p", "--passphrase", action="store_true",
                help="passphrase to encrypt the shell private key"),
            Arg("--template", type=str,
                help="name of template to apply to the shell configuration"),
            Arg("-d", "--detach", action="store_true",
                help="run in the background and print the ID"),
        ]),
        Cmd("open", open_shell, "open an existing shell", [
            Arg("shell_id", help="shell ID"),
            Arg("ssh_opts", nargs="*", help="additional SSH options when connecting to the shell"),
        ]),
        Cmd("logs", command.tail_logs, "fetch shell logs", [
            Arg("shell_id", help="shell ID"),
            Arg("-f", "--follow", action="store_true",
                help="follow the logs of a shell, similar to tail -f"),
            Arg("--tail", type=int, default=200,
                help="number of lines to show, counting from the end "
                     "of the log")
        ]),
        Cmd("kill", command.kill, "kill a shell", [
            Arg("shell_id", help="shell ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
    ])
]  # type: List[Any]

# fmt: on
