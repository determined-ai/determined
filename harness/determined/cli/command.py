from argparse import ONE_OR_MORE, REMAINDER, FileType, Namespace
from functools import partial
from pathlib import Path

from termcolor import colored

from determined import cli
from determined.cli import ntsc, render, task, workspace
from determined.common import api
from determined.common.api import authentication
from determined.common.declarative_argparse import Arg, ArgsDescription, Cmd, Group


@authentication.required
def run_command(args: Namespace) -> None:
    config = ntsc.parse_config(args.config_file, args.entrypoint, args.config, args.volume)
    workspace_id = workspace.get_workspace_id_from_args(args)
    resp = ntsc.launch_command(
        args.master,
        "api/v1/commands",
        config,
        args.template,
        context_path=args.context,
        includes=args.include,
        workspace_id=workspace_id,
    )["command"]

    if args.detach:
        print(resp["id"])
        return

    render.report_job_launched("command", resp["id"])

    try:
        logs = api.task_logs(cli.setup_session(args), resp["id"], follow=True)
        api.pprint_logs(logs)
    finally:
        print(
            colored(
                "Task log stream ended. To reopen log stream, run: "
                "det task logs -f {}".format(resp["id"]),
                "green",
            )
        )


args_description: ArgsDescription = [
    Cmd(
        "command cmd",
        None,
        "manage commands",
        [
            Cmd(
                "list ls",
                partial(ntsc.list_tasks),
                "list commands",
                ntsc.ls_sort_args
                + [
                    Arg("-q", "--quiet", action="store_true", help="only display the IDs"),
                    workspace.workspace_arg,
                    Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        help="show all commands (including other users')",
                    ),
                    Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
                ],
                is_default=True,
            ),
            Cmd(
                "config",
                partial(ntsc.config),
                "display command config",
                [
                    Arg("command_id", type=str, help="command ID"),
                ],
            ),
            Cmd(
                "run",
                run_command,
                "create command",
                [
                    Arg(
                        "entrypoint",
                        type=str,
                        nargs=REMAINDER,
                        help="entrypoint command and arguments to execute",
                    ),
                    Arg(
                        "--config-file",
                        default=None,
                        type=FileType("r"),
                        help="command config file (.yaml)",
                    ),
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
                    workspace.workspace_arg,
                    Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                    Arg(
                        "--template",
                        type=str,
                        help="name of template to apply to the command configuration",
                    ),
                    Arg(
                        "-d",
                        "--detach",
                        action="store_true",
                        help="run in the background and print the ID",
                    ),
                ],
            ),
            Cmd(
                "logs",
                partial(task.logs),
                "fetch command logs",
                [
                    Arg("task_id", help="command ID", metavar="command_id"),
                    *task.common_log_options,
                ],
            ),
            Cmd(
                "kill",
                partial(ntsc.kill),
                "forcibly terminate a command",
                [
                    Arg("command_id", help="command ID", nargs=ONE_OR_MORE),
                    Arg("-f", "--force", action="store_true", help="ignore errors"),
                ],
            ),
            Cmd(
                "set",
                None,
                "set command attributes",
                [
                    Cmd(
                        "priority",
                        partial(ntsc.set_priority),
                        "set command priority",
                        [
                            Arg("command_id", help="command ID"),
                            Arg("priority", type=int, help="priority"),
                        ],
                    ),
                ],
            ),
        ],
    )
]
