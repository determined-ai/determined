import argparse
import contextlib
import getpass
import os
import platform
import shutil
import subprocess
import sys
import tempfile
from functools import partial
from pathlib import Path
from typing import IO, Any, ContextManager, Dict, Iterator, List, Tuple, Union, cast

import appdirs
from termcolor import colored

from determined import cli
from determined.cli import ntsc, render, task
from determined.common import api
from determined.common.api import authentication, bindings, certs
from determined.common.declarative_argparse import Arg, ArgsDescription, Cmd, Group


@authentication.required
def start_shell(args: argparse.Namespace) -> None:
    data = {}
    if args.passphrase:
        data["passphrase"] = getpass.getpass("Enter new passphrase: ")
    config = ntsc.parse_config(args.config_file, None, args.config, args.volume)
    workspace_id = cli.workspace.get_workspace_id_from_args(args)

    resp = ntsc.launch_command(
        args.master,
        "api/v1/shells",
        config,
        args.template,
        context_path=args.context,
        includes=args.include,
        data=data,
        workspace_id=workspace_id,
    )["shell"]

    sid = resp["id"]

    if args.detach:
        print(sid)
        return

    render.report_job_launched("shell", sid)

    session = cli.setup_session(args)

    shell = bindings.get_GetShell(session, shellId=sid).shell
    _open_shell(
        session,
        args.master,
        shell.to_json(),
        args.ssh_opts,
        retain_keys_and_print=args.show_ssh_command,
        print_only=False,
    )


@authentication.required
def open_shell(args: argparse.Namespace) -> None:
    shell_id = cast(str, ntsc.expand_uuid_prefixes(args))

    shell = api.get(args.master, f"api/v1/shells/{shell_id}").json()["shell"]
    _open_shell(
        cli.setup_session(args),
        args.master,
        shell,
        args.ssh_opts,
        retain_keys_and_print=args.show_ssh_command,
        print_only=False,
    )


@authentication.required
def show_ssh_command(args: argparse.Namespace) -> None:
    if platform.system() == "Linux" and "WSL" in os.uname().release:
        cli.warn(
            "WSL remote-ssh integration is not supported in VSCode, which "
            "uses Windows openssh. For Windows VSCode integration, rerun this "
            "command in a Windows shell. For PyCharm users, configure the Pycharm "
            "ssh command to target the WSL ssh command."
        )
    shell_id = ntsc.expand_uuid_prefixes(args)
    shell = api.get(args.master, f"api/v1/shells/{shell_id}").json()["shell"]
    _open_shell(
        cli.setup_session(args),
        args.master,
        shell,
        args.ssh_opts,
        retain_keys_and_print=True,
        print_only=True,
    )


def show_ssh_cmd_legacy(args: argparse.Namespace) -> None:
    cli.warn(
        "DEPRECATION WARNING: show_ssh_command is being deprecated in favor" "of show-ssh-command"
    )
    show_ssh_command(args)


def _prepare_key(retention_dir: Union[Path, None]) -> Tuple[ContextManager[IO], str]:
    if retention_dir:
        key_path = retention_dir / "key"
        keyfile = key_path.open("w")

        if platform.system() == "Windows":
            # On Windows, chmod only affects the read-only flag on the file. To emulate the
            # actual functionality of chmod, an external library is used for Windows systems.
            import oschmod

            oschmod.set_mode(str(key_path), "600")
        else:
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
                    print(colored(f"failed to cleanup {path}: {e}", "yellow"), file=sys.stderr)

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
    sess: api.Session,
    master: str,
    shell: Dict[str, Any],
    additional_opts: List[str],
    retain_keys_and_print: bool,
    print_only: bool,
) -> None:
    cli.wait_ntsc_ready(sess, api.NTSC_Kind.shell, shell["id"])

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
        if cert_bundle_path is False:
            proxy_cmd += " --cert-file noverify"
        elif isinstance(cert_bundle_path, str):
            proxy_cmd += f' --cert-file "{cert_bundle_path}"'
        elif cert_bundle_path is not None:
            raise RuntimeError(
                f"unexpected cert_bundle_path ({cert_bundle_path}) "
                f"of type ({type(cert_bundle_path).__name__})"
            )

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
            print(colored(subprocess.list2cmdline(cmd), "green"))
            if print_only:
                return

        subprocess.run(cmd)

        print(colored(f"To reconnect, run: det shell open {shell['id']}", "green"))


args_description: ArgsDescription = [
    Cmd(
        "shell",
        None,
        "manage shells",
        [
            Cmd(
                "list ls",
                partial(ntsc.list_tasks),
                "list shells",
                ntsc.ls_sort_args
                + [
                    Arg("-q", "--quiet", action="store_true", help="only display the IDs"),
                    Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        help="show all shells (including other users')",
                    ),
                    cli.workspace.workspace_arg,
                    Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
                ],
                is_default=True,
            ),
            Cmd(
                "config",
                partial(ntsc.config),
                "display shell config",
                [
                    Arg("shell_id", type=str, help="shell ID"),
                ],
            ),
            Cmd(
                "start",
                start_shell,
                "start a new shell",
                [
                    Arg(
                        "ssh_opts",
                        nargs="*",
                        help="additional SSH options when connecting to the shell",
                    ),
                    Arg(
                        "--config-file",
                        default=None,
                        type=argparse.FileType("r"),
                        help="command config file (.yaml)",
                    ),
                    cli.workspace.workspace_arg,
                    Arg("-v", "--volume", action="append", default=[], help=ntsc.VOLUME_DESC),
                    Arg("-c", "--context", default=None, type=Path, help=ntsc.CONTEXT_DESC),
                    Arg(
                        "-i",
                        "--include",
                        default=[],
                        action="append",
                        type=Path,
                        help=ntsc.INCLUDE_DESC,
                    ),
                    Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                    Arg(
                        "-p",
                        "--passphrase",
                        action="store_true",
                        help="passphrase to encrypt the shell private key",
                    ),
                    Arg(
                        "--template",
                        type=str,
                        help="name of template to apply to the shell configuration",
                    ),
                    Arg(
                        "-d",
                        "--detach",
                        action="store_true",
                        help="run in the background and print the ID",
                    ),
                    Arg(
                        "--show-ssh-command",
                        action="store_true",
                        help="show ssh command (e.g. for use in IDE) when starting the shell",
                    ),
                ],
            ),
            Cmd(
                "open",
                open_shell,
                "open an existing shell",
                [
                    Arg("shell_id", help="shell ID"),
                    Arg(
                        "ssh_opts",
                        nargs="*",
                        help="additional SSH options when connecting to the shell",
                    ),
                    Arg(
                        "--show-ssh-command",
                        action="store_true",
                        help="show ssh command (e.g. for use in IDE) when starting the shell",
                    ),
                ],
            ),
            Cmd(
                "show_ssh_command",
                show_ssh_cmd_legacy,
                argparse.SUPPRESS,
                [
                    Arg("shell_id", help="shell ID"),
                    Arg(
                        "ssh_opts",
                        nargs="*",
                        help="additional SSH options when connecting to the shell",
                    ),
                ],
            ),
            Cmd(
                "show-ssh-command",
                show_ssh_command,
                "print the ssh command",
                [
                    Arg("shell_id", help="shell ID"),
                    Arg(
                        "ssh_opts",
                        nargs="*",
                        help="additional SSH options when connecting to the shell",
                    ),
                ],
            ),
            Cmd(
                "logs",
                partial(task.logs),
                "fetch shell logs",
                [Arg("task_id", help="shell ID", metavar="shell_id"), *task.common_log_options],
            ),
            Cmd(
                "kill",
                partial(ntsc.kill),
                "kill a shell",
                [
                    Arg("shell_id", help="shell ID", nargs=argparse.ONE_OR_MORE),
                    Arg("-f", "--force", action="store_true", help="ignore errors"),
                ],
            ),
            Cmd(
                "set",
                None,
                "set shell attributes",
                [
                    Cmd(
                        "priority",
                        partial(ntsc.set_priority),
                        "set shell priority",
                        [
                            Arg("shell_id", help="shell ID"),
                            Arg("priority", type=int, help="priority"),
                        ],
                    ),
                ],
            ),
        ],
    )
]
