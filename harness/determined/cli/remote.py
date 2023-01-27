from argparse import ONE_OR_MORE, REMAINDER, FileType, Namespace
from functools import partial
from pathlib import Path
from typing import Any, List

from determined import cli
from determined.cli import command, task
from determined.common import api, context
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def run_command(args: Namespace) -> None:
    config = command.parse_config(args.config_file, args.entrypoint, args.config, args.volume)
    workspace_id = cli.workspace.get_workspace_id_from_args(args)
    files = context.read_v1_context(args.context, args.include)
    body = bindings.v1LaunchCommandRequest(
        config=config,
        files=files,
        templateName=args.template,
        workspaceId=workspace_id,
    )
    resp = bindings.post_LaunchCommand(cli.setup_session(args), body=body)
    command_id = resp.command.id

    if args.detach:
        print(command_id)
        return

    if resp.warnings:
        cli.print_warnings(resp.warnings)

    logs = api.task_logs(cli.setup_session(args), command_id, follow=True)
    api.pprint_task_logs(command_id, logs)


# fmt: off

args_description = [
    Cmd("command cmd", None, "manage commands", [
        Cmd("list ls", partial(command.list_tasks), "list commands", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            cli.workspace.workspace_arg,
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
            cli.workspace.workspace_arg,
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
