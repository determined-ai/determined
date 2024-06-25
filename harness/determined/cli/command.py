import argparse
import functools
import pathlib

import termcolor

from determined import cli
from determined.cli import ntsc, render, task, workspace
from determined.common import api
from determined.common.api import bindings


def run_command(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    config = ntsc.parse_config(args.config_file, args.entrypoint, args.config, args.volume)
    workspace_id = workspace.get_workspace_id_from_args(args)
    resp = ntsc.launch_command(
        sess,
        "api/v1/commands",
        config,
        args.template,
        context_path=args.context,
        includes=args.include,
        workspace_id=workspace_id,
    )
    cmd = bindings.v1LaunchCommandResponse.from_json(resp).command

    if args.detach:
        print(cmd.id)
        return

    render.report_job_launched("command", cmd.id, cmd.description)

    try:
        logs = api.task_logs(sess, cmd.id, follow=True)
        api.pprint_logs(logs)
    finally:
        print(
            termcolor.colored(
                "Task log stream ended. To reopen log stream, run: "
                "det task logs -f {}".format(cmd.id),
                "green",
            )
        )


args_description: cli.ArgsDescription = [
    cli.Cmd(
        "command cmd",
        None,
        "manage commands",
        [
            cli.Cmd(
                "list ls",
                functools.partial(ntsc.list_tasks),
                "list commands",
                ntsc.ls_sort_args
                + [
                    cli.Arg("-q", "--quiet", action="store_true", help="only display the IDs"),
                    workspace.workspace_arg,
                    cli.Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        help="show all commands (including other users')",
                    ),
                    cli.Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
                ],
                is_default=True,
            ),
            cli.Cmd(
                "describe",
                functools.partial(ntsc.describe),
                "display command metadata",
                [
                    cli.Arg("command_id", type=str, help="command ID"),
                    cli.Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
                ],
            ),
            cli.Cmd(
                "config",
                functools.partial(ntsc.config),
                "display command config",
                [
                    cli.Arg("command_id", type=str, help="command ID"),
                ],
            ),
            cli.Cmd(
                "run",
                run_command,
                "create command",
                [
                    cli.Arg(
                        "entrypoint",
                        type=str,
                        nargs=argparse.REMAINDER,
                        help="entrypoint command and arguments to execute",
                    ),
                    cli.Arg(
                        "--config-file",
                        default=None,
                        type=argparse.FileType("r"),
                        help="command config file (.yaml)",
                    ),
                    cli.Arg("-v", "--volume", action="append", default=[], help=ntsc.VOLUME_DESC),
                    cli.Arg(
                        "-c", "--context", default=None, type=pathlib.Path, help=ntsc.CONTEXT_DESC
                    ),
                    cli.Arg(
                        "-i",
                        "--include",
                        default=[],
                        action="append",
                        type=pathlib.Path,
                        help=ntsc.INCLUDE_DESC,
                    ),
                    workspace.workspace_arg,
                    cli.Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                    cli.Arg(
                        "--template",
                        type=str,
                        help="name of template to apply to the command configuration",
                    ),
                    cli.Arg(
                        "-d",
                        "--detach",
                        action="store_true",
                        help="run in the background and print the ID",
                    ),
                ],
            ),
            cli.Cmd(
                "logs",
                functools.partial(task.logs),
                "fetch command logs",
                [
                    cli.Arg("task_id", help="command ID", metavar="command_id"),
                    *task.common_log_options,
                ],
            ),
            cli.Cmd(
                "kill",
                functools.partial(ntsc.kill),
                "forcibly terminate a command",
                [
                    cli.Arg("command_id", help="command ID", nargs=argparse.ONE_OR_MORE),
                    cli.Arg("-f", "--force", action="store_true", help="ignore errors"),
                ],
            ),
            cli.Cmd(
                "set",
                None,
                "set command attributes",
                [
                    cli.Cmd(
                        "priority",
                        functools.partial(ntsc.set_priority),
                        "set command priority",
                        [
                            cli.Arg("command_id", help="command ID"),
                            cli.Arg("priority", type=int, help="priority"),
                        ],
                    ),
                ],
            ),
        ],
    )
]
