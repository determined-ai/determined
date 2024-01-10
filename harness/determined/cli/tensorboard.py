from argparse import ONE_OR_MORE, ArgumentError, FileType, Namespace
from functools import partial
from pathlib import Path
from typing import cast

from termcolor import colored

from determined import cli
from determined.cli import ntsc, render, task
from determined.common import api, context
from determined.common.api import authentication, bindings, request
from determined.common.check import check_none
from determined.common.declarative_argparse import Arg, ArgsDescription, Cmd, Group


@authentication.required
def start_tensorboard(args: Namespace) -> None:
    if not (args.trial_ids or args.experiment_ids):
        raise ArgumentError(None, "Either experiment_ids or trial_ids must be specified.")

    config = ntsc.parse_config(args.config_file, None, args.config, [])

    workspace_id = cli.workspace.get_workspace_id_from_args(args)

    body = bindings.v1LaunchTensorboardRequest(
        config=config,
        trialIds=args.trial_ids,
        experimentIds=args.experiment_ids,
        files=context.read_v1_context(args.context, args.include),
        workspaceId=workspace_id,
    )

    resp = bindings.post_LaunchTensorboard(cli.setup_session(args), body=body)
    tsb = resp.tensorboard

    if args.detach:
        print(resp.tensorboard.id)
        return

    render.report_job_launched("tensorboard", tsb.id)

    if resp.warnings:
        cli.print_launch_warnings(resp.warnings)
    currentSlotsExceeded = (resp.warnings is not None) and (
        bindings.v1LaunchWarning.CURRENT_SLOTS_EXCEEDED in resp.warnings
    )
    cli.wait_ntsc_ready(cli.setup_session(args), api.NTSC_Kind.tensorboard, tsb.id)

    assert tsb.serviceAddress is not None, "missing tensorboard serviceAddress"
    nb_path = request.make_interactive_task_url(
        task_id=tsb.id,
        service_address=tsb.serviceAddress,
        description=tsb.description,
        resource_pool=tsb.resourcePool,
        task_type="tensorboard",
        currentSlotsExceeded=currentSlotsExceeded,
    )
    url = api.make_url(args.master, nb_path)
    if not args.no_browser:
        api.browser_open(args.master, nb_path)
    print(colored("Tensorboard is running at: {}".format(url), "green"))


@authentication.required
def open_tensorboard(args: Namespace) -> None:
    tensorboard_id = cast(str, ntsc.expand_uuid_prefixes(args))

    sess = cli.setup_session(args)
    task = bindings.get_GetTask(sess, taskId=tensorboard_id).task
    check_none(task.endTime, "Tensorboard has ended")

    tsb = bindings.get_GetTensorboard(sess, tensorboardId=tensorboard_id).tensorboard
    assert tsb.serviceAddress is not None, "missing tensorboard serviceAddress"

    api.browser_open(
        args.master,
        request.make_interactive_task_url(
            task_id=tsb.id,
            service_address=tsb.serviceAddress,
            description=tsb.description,
            resource_pool=tsb.resourcePool,
            task_type="tensorboard",
            currentSlotsExceeded=False,
        ),
    )


args_description: ArgsDescription = [
    Cmd(
        "tensorboard",
        None,
        "manage TensorBoard instances",
        [
            Cmd(
                "list ls",
                partial(ntsc.list_tasks),
                "list TensorBoard instances",
                ntsc.ls_sort_args
                + [
                    Arg("-q", "--quiet", action="store_true", help="only display the IDs"),
                    Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        help="show all TensorBoards (including other users')",
                    ),
                    cli.workspace.workspace_arg,
                    Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
                ],
                is_default=True,
            ),
            Cmd(
                "start",
                start_tensorboard,
                "start new TensorBoard instance",
                [
                    Arg(
                        "experiment_ids",
                        type=int,
                        nargs="*",
                        help="experiment IDs to load into TensorBoard. At most 100 trials from "
                        "the specified experiment will be loaded into TensorBoard. If the "
                        "experiment has more trials, the 100 best-performing trials will "
                        "be used.",
                    ),
                    Arg(
                        "-t",
                        "--trial-ids",
                        nargs=ONE_OR_MORE,
                        type=int,
                        help="trial IDs to load into TensorBoard; at most 100 trials are "
                        "allowed per TensorBoard instance",
                    ),
                    cli.workspace.workspace_arg,
                    Arg(
                        "--config-file",
                        default=None,
                        type=FileType("r"),
                        help="command config file (.yaml)",
                    ),
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
                        "--no-browser",
                        action="store_true",
                        help="don't open TensorBoard in a browser after startup",
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
                "config",
                partial(ntsc.config),
                "display TensorBoard config",
                [Arg("tensorboard_id", type=str, help="TensorBoard ID")],
            ),
            Cmd(
                "open",
                open_tensorboard,
                "open existing TensorBoard instance",
                [Arg("tensorboard_id", help="TensorBoard ID")],
            ),
            Cmd(
                "logs",
                partial(task.logs),
                "fetch TensorBoard instance logs",
                [
                    Arg("task_id", help="TensorBoard ID", metavar="tensorboard_id"),
                    *task.common_log_options,
                ],
            ),
            Cmd(
                "kill",
                partial(ntsc.kill),
                "kill TensorBoard instance",
                [
                    Arg("tensorboard_id", help="TensorBoard ID", nargs=ONE_OR_MORE),
                    Arg("-f", "--force", action="store_true", help="ignore errors"),
                ],
            ),
            Cmd(
                "set",
                None,
                "set TensorBoard attributes",
                [
                    Cmd(
                        "priority",
                        partial(ntsc.set_priority),
                        "set TensorBoard priority",
                        [
                            Arg("tensorboard_id", help="TensorBoard ID"),
                            Arg("priority", type=int, help="priority"),
                        ],
                    ),
                ],
            ),
        ],
    )
]
