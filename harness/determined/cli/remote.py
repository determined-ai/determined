from argparse import ONE_OR_MORE, REMAINDER, FileType, Namespace
from functools import partial
from pathlib import Path
from typing import Any, List

from determined import cli
from determined.cli import command, task
from determined.common import api
from determined.common.api import authentication
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def run_command(args: Namespace) -> None:
    config = command.parse_config(args.config_file, args.entrypoint, args.config, args.volume)
    resp = command.launch_command(
        args.master,
        "api/v1/commands",
        config,
        args.template,
        context_path=args.context,
        includes=args.include,
    )["command"]

    if args.detach:
        print(resp["id"])
        return

    logs = api.task_logs(cli.setup_session(args), resp["id"], follow=True)
    api.pprint_task_logs(resp["id"], logs)


# fmt: off

args_description = [
    Cmd("command cmd", None, "manage commands", [
        Cmd("list ls", partial(command.list_tasks), "list commands", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all commands (including other users')"),
            Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
        ], is_default=True),
        Cmd("config", partial(command.config),
            "display command config", [
                Arg("command_id", type=str, help="command ID"),
        ]),
        Cmd("run", run_command, "create command", [
            Arg("entrypoint", type=str, nargs=REMAINDER,
                help="entrypoint command and arguments to execute"),
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
            Arg("--template", type=str,
                help="name of template to apply to the command configuration"),
            Arg("-d", "--detach", action="store_true",
                help="run in the background and print the ID")
        ]),
        Cmd("logs", partial(task.logs), "fetch command logs", [
            Arg("task_id", help="command ID", metavar="command_id"),
            *task.common_log_options,
        ]),
        Cmd("kill", partial(command.kill), "forcibly terminate a command", [
            Arg("command_id", help="command ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
        Cmd("set", None, "set command attributes", [
            Cmd("priority", partial(command.set_priority), "set command priority", [
                Arg("command_id", help="command ID"),
                Arg("priority", type=int, help="priority"),
            ]),
        ]),
    ])
]  # type: List[Any]

# fmt: on
