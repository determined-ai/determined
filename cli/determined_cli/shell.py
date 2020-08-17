import getpass
import subprocess
import tempfile
from argparse import ONE_OR_MORE, FileType, Namespace
from pathlib import Path
from typing import Any, Dict, List

from termcolor import colored

from determined_common import api
from determined_common.api import request
from determined_common.api.authentication import authentication_required
from determined_common.check import check_eq, check_len

from . import render
from .command import (
    CONFIG_DESC,
    CONTEXT_DESC,
    VOLUME_DESC,
    Command,
    CommandDescription,
    describe_command,
    launch_command,
    parse_config,
    render_event_stream,
)
from .declarative_argparse import Arg, Cmd


@authentication_required
def start_shell(args: Namespace) -> None:
    data = {}
    if args.passphrase:
        data["passphrase"] = getpass.getpass("Enter new passphrase: ")
    config = parse_config(args.config_file, None, args.config, args.volume)
    resp = launch_command(
        args.master, "shells", config, args.template, context_path=args.context, data=data,
    )

    if args.detach:
        print(resp["id"])
        return

    command = None
    with api.ws(args.master, "shells/{}/events".format(resp["id"])) as ws:
        for msg in ws:
            if msg["service_ready_event"]:
                command = render.unmarshal(Command, msg["snapshot"])
                break
            render_event_stream(msg)
    if command:
        _open_shell(args.master, command, args.ssh_opts)


@authentication_required
def open_shell(args: Namespace) -> None:
    shell = render.unmarshal(
        Command, api.get(args.master, "shells/{}".format(args.shell_id)).json()
    )
    check_eq(shell.state, "RUNNING", "Shell must be in a running state")
    _open_shell(args.master, shell, args.ssh_opts)


def _open_shell(master: str, shell: Command, additional_opts: List[str]) -> None:
    LOOPBACK_ADDRESS = "[::1]"
    with tempfile.NamedTemporaryFile("w") as fp:
        fp.write(shell.misc["privateKey"])
        fp.flush()
        check_len(shell.addresses, 1, "Cannot find address for shell")
        host, port = shell.addresses[0]["host_ip"], shell.addresses[0]["host_port"]
        if host == LOOPBACK_ADDRESS:
            host = "localhost"

        # Use determined_cli.tunnel as a portable script for using the HTTP CONNECT mechanism,
        # similar to `nc -X CONNECT -x ...` but without any dependency on external binaries.
        proxy_cmd = "python -m determined_cli.tunnel {} %h".format(master)
        if request.get_master_cert_bundle():
            proxy_cmd += ' "{}"'.format(request.get_master_cert_bundle())

        username = shell.agent_user_group["user"] or "root"

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
            "{}@{}".format(username, shell.id),
            *additional_opts,
        ]

        subprocess.run(cmd)

        print(colored("To reconnect, run: det shell open {}".format(shell.id), "green"))


@authentication_required
def tail_shell_logs(args: Namespace) -> None:
    url = "shells/{}/events?follow={}&tail={}".format(args.shell_id, args.follow, args.tail)
    with api.ws(args.master, url) as ws:
        for msg in ws:
            render_event_stream(msg)


@authentication_required
def list_shells(args: Namespace) -> None:
    if args.all:
        params = {}  # type: Dict[str, Any]
    else:
        params = {"user": api.Authentication.instance().get_session_user()}
    commands = [
        render.unmarshal(Command, command)
        for command in api.get(args.master, "shells", params=params).json().values()
    ]

    if args.quiet:
        for command in commands:
            print(command.id)
        return

    render.render_objects(CommandDescription, [describe_command(command) for command in commands])


@authentication_required
def kill_shell(args: Namespace) -> None:
    for i, nid in enumerate(args.shell_id):
        try:
            api.delete(args.master, "shells/{}".format(nid))
            print(colored("Killed shell {}".format(nid), "green"))
        except api.errors.APIException as e:
            if not args.force:
                for ignored in args.shell_id[i + 1 :]:
                    print("Cowardly not killing {}".format(ignored))
                raise e
            print(colored("Skipping: {} ({})".format(e, type(e).__name__), "red"))


@authentication_required
def shell_config(args: Namespace) -> None:
    res_json = api.get(args.master, "shells/{}".format(args.id)).json()
    print(render.format_object_as_yaml(res_json["config"]))


# fmt: off

args_description = [
    Cmd("shell", None, "manage shells", [
        Cmd("list", list_shells, "list shells", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all shells (including other users')")
        ], is_default=True),
        Cmd("config", shell_config,
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
        Cmd("logs", tail_shell_logs, "fetch shell logs", [
            Arg("shell_id", help="shell ID"),
            Arg("-f", "--follow", action="store_true",
                help="follow the logs of a shell, similar to tail -f"),
            Arg("--tail", type=int, default=200,
                help="number of lines to show, counting from the end "
                     "of the log")
        ]),
        Cmd("kill", kill_shell, "kill a shell", [
            Arg("shell_id", help="shell ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
    ])
]  # type: List[Any]

# fmt: on
