import argparse
import functools
import pathlib
import typing
import webbrowser

import termcolor

from determined import cli
from determined.cli import ntsc, render, task
from determined.common import api, check, context
from determined.common.api import bindings


def start_tensorboard(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    if not (args.trial_ids or args.experiment_ids):
        raise argparse.ArgumentError(None, "Either experiment_ids or trial_ids must be specified.")

    config = ntsc.parse_config(args.config_file, None, args.config, [])

    workspace_id = cli.workspace.get_workspace_id_from_args(args)

    body = bindings.v1LaunchTensorboardRequest(
        config=config,
        trialIds=args.trial_ids,
        experimentIds=args.experiment_ids,
        files=context.read_v1_context(args.context, args.include),
        workspaceId=workspace_id,
    )

    resp = bindings.post_LaunchTensorboard(sess, body=body)
    tsb = resp.tensorboard

    if args.detach:
        print(resp.tensorboard.id)
        return

    render.report_job_launched("tensorboard", tsb.id, tsb.description)

    if resp.warnings:
        cli.print_launch_warnings(resp.warnings)
    currentSlotsExceeded = (resp.warnings is not None) and (
        bindings.v1LaunchWarning.CURRENT_SLOTS_EXCEEDED in resp.warnings
    )
    cli.wait_ntsc_ready(sess, api.NTSC_Kind.tensorboard, tsb.id)

    assert tsb.serviceAddress is not None, "missing tensorboard serviceAddress"
    tb_path = ntsc.make_interactive_task_url(
        task_id=tsb.id,
        service_address=tsb.serviceAddress,
        description=tsb.description,
        resource_pool=tsb.resourcePool,
        task_type="tensorboard",
        currentSlotsExceeded=currentSlotsExceeded,
    )
    url = f"{args.master}/{tb_path}"
    if not args.no_browser:
        webbrowser.open(url)
    print(termcolor.colored(f"Tensorboard is running at: {url}", "green"))


def open_tensorboard(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    tensorboard_id = typing.cast(str, ntsc.expand_uuid_prefixes(sess, args))

    task = bindings.get_GetTask(sess, taskId=tensorboard_id).task
    check.check_none(task.endTime, "Tensorboard has ended")

    tsb = bindings.get_GetTensorboard(sess, tensorboardId=tensorboard_id).tensorboard
    assert tsb.serviceAddress is not None, "missing tensorboard serviceAddress"
    tb_path = ntsc.make_interactive_task_url(
        task_id=tsb.id,
        service_address=tsb.serviceAddress,
        description=tsb.description,
        resource_pool=tsb.resourcePool,
        task_type="tensorboard",
        currentSlotsExceeded=False,
    )
    webbrowser.open(f"{args.master}/{tb_path}")


args_description: cli.ArgsDescription = [
    cli.Cmd(
        "tensorboard",
        None,
        "manage TensorBoard instances",
        [
            cli.Cmd(
                "list ls",
                functools.partial(ntsc.list_tasks),
                "list TensorBoard instances",
                ntsc.ls_sort_args
                + [
                    cli.Arg("-q", "--quiet", action="store_true", help="only display the IDs"),
                    cli.Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        help="show all TensorBoards (including other users')",
                    ),
                    cli.workspace.workspace_arg,
                    cli.Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
                ],
                is_default=True,
            ),
            cli.Cmd(
                "start",
                start_tensorboard,
                "start new TensorBoard instance",
                [
                    cli.Arg(
                        "experiment_ids",
                        type=int,
                        nargs="*",
                        help="experiment IDs to load into TensorBoard. At most 100 trials from "
                        "the specified experiment will be loaded into TensorBoard. If the "
                        "experiment has more trials, the 100 best-performing trials will "
                        "be used.",
                    ),
                    cli.Arg(
                        "-t",
                        "--trial-ids",
                        nargs=argparse.ONE_OR_MORE,
                        type=int,
                        help="trial IDs to load into TensorBoard; at most 100 trials are "
                        "allowed per TensorBoard instance",
                    ),
                    cli.workspace.workspace_arg,
                    cli.Arg(
                        "--config-file",
                        default=None,
                        type=argparse.FileType("r"),
                        help="command config file (.yaml)",
                    ),
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
                    cli.Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                    cli.Arg(
                        "--no-browser",
                        action="store_true",
                        help="don't open TensorBoard in a browser after startup",
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
                "config",
                functools.partial(ntsc.config),
                "display TensorBoard config",
                [cli.Arg("tensorboard_id", type=str, help="TensorBoard ID")],
            ),
            cli.Cmd(
                "open",
                open_tensorboard,
                "open existing TensorBoard instance",
                [cli.Arg("tensorboard_id", help="TensorBoard ID")],
            ),
            cli.Cmd(
                "logs",
                functools.partial(task.logs),
                "fetch TensorBoard instance logs",
                [
                    cli.Arg("task_id", help="TensorBoard ID", metavar="tensorboard_id"),
                    *task.common_log_options,
                ],
            ),
            cli.Cmd(
                "kill",
                functools.partial(ntsc.kill),
                "kill TensorBoard instance",
                [
                    cli.Arg("tensorboard_id", help="TensorBoard ID", nargs=argparse.ONE_OR_MORE),
                    cli.Arg("-f", "--force", action="store_true", help="ignore errors"),
                ],
            ),
            cli.Cmd(
                "set",
                None,
                "set TensorBoard attributes",
                [
                    cli.Cmd(
                        "priority",
                        functools.partial(ntsc.set_priority),
                        "set TensorBoard priority",
                        [
                            cli.Arg("tensorboard_id", help="TensorBoard ID"),
                            cli.Arg("priority", type=int, help="priority"),
                        ],
                    ),
                ],
            ),
        ],
    )
]
