import getpass
import shutil
import subprocess
import sys
import tempfile
from argparse import ONE_OR_MORE, FileType, Namespace
from pathlib import Path
from typing import IO, Any, Dict, List, Union

import appdirs
from termcolor import colored

from determined.cli import command
from determined.common import api
from determined.common.api import authentication, certs
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


@authentication.required
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
        _open_shell(
            args.master,
            shell,
            args.ssh_opts,
            retain_keys_and_print=args.show_ssh_command,
            print_only=False,
        )


@authentication.required
def open_shell(args: Namespace) -> None:
    shell = api.get(args.master, "api/v1/shells/{}".format(args.shell_id)).json()["shell"]
    check_eq(shell["state"], "STATE_RUNNING", "Shell must be in a running state")
    _open_shell(
        args.master,
        shell,
        args.ssh_opts,
        retain_keys_and_print=args.show_ssh_command,
        print_only=False,
    )


@authentication.required
def show_ssh_command(args: Namespace) -> None:
    shell = api.get(args.master, "api/v1/shells/{}".format(args.shell_id)).json()["shell"]
    check_eq(shell["state"], "STATE_RUNNING", "Shell must be in a running state")
    _open_shell(args.master, shell, args.ssh_opts, retain_keys_and_print=True, print_only=True)


def _prepare_key(retention_dir: Union[Path, None]) -> IO:
    if retention_dir:
        retention_dir = retention_dir

        key_path = retention_dir / "key"
        keyfile = key_path.open("w")
        key_path.chmod(0o600)

        return keyfile
    else:
        return tempfile.NamedTemporaryFile("w")


def _prepare_cert_bundle(retention_dir: Union[Path, None]) -> Union[str, bool, None]:
    cert = certs.cli_cert
    assert cert is not None, "cli_cert was not configured"
    if retention_dir and isinstance(cert.bundle, str):
        retained_cert_bundle_path = retention_dir / "cert_bundle"
        shutil.copy2(str(cert.bundle), retained_cert_bundle_path)
        return str(retained_cert_bundle_path)
    return cert.bundle


def _open_shell(
    master: str,
    shell: Dict[str, Any],
    additional_opts: List[str],
    retain_keys_and_print: bool,
    print_only: bool,
) -> None:
    cache_dir = None
    if retain_keys_and_print:
        cache_dir = Path(appdirs.user_cache_dir("determined")) / "shell" / shell["id"]
        if not cache_dir.exists():
            cache_dir.mkdir(parents=True)

    with _prepare_key(cache_dir) as keyfile:
        keyfile.write(shell["privateKey"])
        keyfile.flush()

        check_len(shell["addresses"], 1, "Cannot find address for shell")
        _, port = shell["addresses"][0]["host_ip"], shell["addresses"][0]["host_port"]

        # Use determined.cli.tunnel as a portable script for using the HTTP CONNECT mechanism,
        # similar to `nc -X CONNECT -x ...` but without any dependency on external binaries.
        python = sys.executable
        proxy_cmd = "{} -m determined.cli.tunnel {} %h".format(python, master)

        cert_bundle_path = _prepare_cert_bundle(cache_dir)
        if cert_bundle_path is not None:
            proxy_cmd += ' --cert-file "{}"'.format(cert_bundle_path)

        cert = certs.cli_cert
        assert cert is not None, "cli_cert was not configured"
        if cert.name:
            proxy_cmd += ' --cert-name "{}"'.format(cert.name)

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
            str(keyfile.name),
            "-p",
            str(port),
            "{}@{}".format(username, shell["id"]),
            *additional_opts,
        ]

        if retain_keys_and_print:
            print(colored(subprocess.list2cmdline(cmd), "yellow"))
            if print_only:
                return

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
            Arg("--show-ssh-command", action="store_true",
                help="show ssh command (e.g. for use in IDE) when starting the shell"),
        ]),
        Cmd("open", open_shell, "open an existing shell", [
            Arg("shell_id", help="shell ID"),
            Arg("ssh_opts", nargs="*", help="additional SSH options when connecting to the shell"),
            Arg("--show-ssh-command", action="store_true",
                help="show ssh command (e.g. for use in IDE) when starting the shell"),
        ]),
        Cmd("show_ssh_command", show_ssh_command, "print the ssh command", [
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
