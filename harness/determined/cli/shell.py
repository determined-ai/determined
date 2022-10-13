import contextlib
import getpass
import logging
import os
import shutil
import subprocess
import sys
import tempfile
from argparse import ONE_OR_MORE, FileType, Namespace
from functools import partial
from pathlib import Path
from typing import IO, Any, ContextManager, Dict, Iterator, List, Tuple, Union

import appdirs
from termcolor import colored

from determined import cli
from determined.cli import command, task
from determined.common import api
from determined.common.api import authentication, certs
from determined.common.check import check_eq
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def start_shell(args: Namespace) -> None:
    data = {}
    if args.passphrase:
        data["passphrase"] = getpass.getpass("Enter new passphrase: ")
    config = command.parse_config(args.config_file, None, args.config, args.volume)
    resp = command.launch_command(
        args.master,
        "api/v1/shells",
        config,
        args.template,
        context_path=args.context,
        includes=args.include,
        data=data,
    )["shell"]

    if args.detach:
        print(resp["id"])
        return

    ready = False
    with api.ws(args.master, f"shells/{resp['id']}/events") as ws:
        for msg in ws:
            if msg["service_ready_event"]:
                ready = True
                break
            command.render_event_stream(msg)
    if ready:
        shell = api.get(args.master, f"api/v1/shells/{resp['id']}").json()["shell"]
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
    shell_id = command.expand_uuid_prefixes(args)
    shell = api.get(args.master, f"api/v1/shells/{shell_id}").json()["shell"]
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
    shell_id = command.expand_uuid_prefixes(args)
    shell = api.get(args.master, f"api/v1/shells/{shell_id}").json()["shell"]
    check_eq(shell["state"], "STATE_RUNNING", "Shell must be in a running state")
    _open_shell(args.master, shell, args.ssh_opts, retain_keys_and_print=True, print_only=True)


def _prepare_key(retention_dir: Union[Path, None]) -> Tuple[ContextManager[IO], str]:
    if retention_dir:
        key_path = retention_dir / "key"
        keyfile = key_path.open("w")
        key_path.chmod(0o600)

        return keyfile, str(key_path)

    else:

        # Avoid using tempfile.NamedTemporaryFile, which does not produce a file that can be opened
        # by name on Windows, which prevents the ssh process from reading it.
        fd, path = tempfile.mkstemp(text=True)
        f = open(fd, "w")

        @contextlib.contextmanager
        def file_closer() -> Iterator[IO]:
            try:
                yield f
            finally:
                f.close()
                try:
                    os.remove(path)
                except Exception as e:
                    logging.warning(f"failed to cleanup {path}: {e}")

        return file_closer(), path


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

    f, keypath = _prepare_key(cache_dir)
    with f as keyfile:
        keyfile.write(shell["privateKey"])
        keyfile.flush()

        # Use determined.cli.tunnel as a portable script for using the HTTP CONNECT mechanism,
        # similar to `nc -X CONNECT -x ...` but without any dependency on external binaries.
        proxy_cmd = f"{sys.executable} -m determined.cli.tunnel {master} %h"

        cert_bundle_path = _prepare_cert_bundle(cache_dir)
        if cert_bundle_path is not None:
            assert isinstance(cert_bundle_path, str), cert_bundle_path
            proxy_cmd += f' --cert-file "{cert_bundle_path}"'

        cert = certs.cli_cert
        assert cert is not None, "cli_cert was not configured"
        if cert.name:
            proxy_cmd += f' --cert-name "{cert.name}"'

        username = shell["agentUserGroup"]["user"] or "root"

        unixy_keypath = str(keypath)
        if sys.platform == "win32":
            # Convert the backslashes of the -i argument to ssh to forwardslashes.  This is
            # important because when passing the output of ssh_show_command to VSCode, VSCode would
            # put backslashes in .ssh/config, which would not be handled correctly by ssh.  When
            # invoking ssh directly, it behaves the same whether -i has backslashes or not.
            unixy_keypath = unixy_keypath.replace("\\", "/")

        cmd = [
            "ssh",
            "-o",
            f"ProxyCommand={proxy_cmd}",
            "-o",
            "StrictHostKeyChecking=no",
            "-tt",
            "-o",
            "IdentitiesOnly=yes",
            "-i",
            unixy_keypath,
            f"{username}@{shell['id']}",
            *additional_opts,
        ]

        if retain_keys_and_print:
            print(colored(subprocess.list2cmdline(cmd), "yellow"))
            if print_only:
                return

        subprocess.run(cmd)

        print(colored(f"To reconnect, run: det shell open {shell['id']}", "green"))


# fmt: off

args_description = [
    Cmd("shell", None, "manage shells", [
        Cmd("list ls", partial(command.list_tasks), "list shells", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all shells (including other users')"),
            Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
        ], is_default=True),
        Cmd("config", partial(command.config),
            "display shell config", [
                Arg("shell_id", type=str, help="shell ID"),
        ]),
        Cmd("start", start_shell, "start a new shell", [
            Arg("ssh_opts", nargs="*", help="additional SSH options when connecting to the shell"),
            Arg("--config-file", default=None, type=FileType("r"),
                help="command config file (.yaml)"),
            Arg("-v", "--volume", action="append", default=[],
                help=command.VOLUME_DESC),
            Arg("-c", "--context", default=None, type=Path, help=command.CONTEXT_DESC),
            Arg(
                "-i",
                "--include",
                default=[],
                action="append",
                type=Path,
                help=command.INCLUDE_DESC
            ),
            Arg("--config", action="append", default=[], help=command.CONFIG_DESC),
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
        Cmd("logs", partial(task.logs),
            "fetch shell logs", [
            Arg("task_id", help="shell ID", metavar="shell_id"),
            *task.common_log_options
        ]),
        Cmd("kill", partial(command.kill), "kill a shell", [
            Arg("shell_id", help="shell ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
        Cmd("set", None, "set shell attributes", [
            Cmd("priority", partial(command.set_priority), "set shell priority", [
                Arg("shell_id", help="shell ID"),
                Arg("priority", type=int, help="priority"),
            ]),
        ]),
    ])
]  # type: List[Any]

# fmt: on
