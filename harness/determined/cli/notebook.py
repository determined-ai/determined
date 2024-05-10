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


def start_notebook(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    config = ntsc.parse_config(args.config_file, None, args.config, args.volume)

    files = context.read_v1_context(args.context, args.include)

    workspace_id = cli.workspace.get_workspace_id_from_args(args)

    body = bindings.v1LaunchNotebookRequest(
        config=config,
        files=files,
        preview=args.preview,
        templateName=args.template,
        workspaceId=workspace_id,
    )
    resp = bindings.post_LaunchNotebook(sess, body=body)

    if args.preview:
        print(render.format_object_as_yaml(resp.config))
        return

    nb = resp.notebook

    if args.detach:
        print(nb.id)
        return

    render.report_job_launched("notebook", resp.notebook.id, nb.description)

    if resp.warnings:
        cli.print_launch_warnings(resp.warnings)
    currentSlotsExceeded = (resp.warnings is not None) and (
        bindings.v1LaunchWarning.CURRENT_SLOTS_EXCEEDED in resp.warnings
    )

    cli.wait_ntsc_ready(sess, api.NTSC_Kind.notebook, nb.id)

    assert nb.serviceAddress is not None, "missing Jupyter serviceAddress"
    nb_path = ntsc.make_interactive_task_url(
        task_id=nb.id,
        service_address=nb.serviceAddress,
        description=nb.description,
        resource_pool=nb.resourcePool,
        task_type="jupyter-lab",
        currentSlotsExceeded=currentSlotsExceeded,
    )
    url = f"{args.master}/{nb_path}"
    if not args.no_browser:
        webbrowser.open(url)
    print(termcolor.colored(f"Jupyter Notebook is running at: {url}", "green"))
    print(
        termcolor.colored(
            f"Connect to remote Jupyter server: " f"{args.master}{nb.serviceAddress}", "blue"
        )
    )


def open_notebook(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    notebook_id = typing.cast(str, ntsc.expand_uuid_prefixes(sess, args))

    task_obj = bindings.get_GetTask(sess, taskId=notebook_id).task
    check.check_none(task_obj.endTime, "Notebook has ended")

    nb = bindings.get_GetNotebook(sess, notebookId=notebook_id).notebook
    assert nb.serviceAddress is not None, "missing tensorboard serviceAddress"

    nb_path = ntsc.make_interactive_task_url(
        task_id=nb.id,
        service_address=nb.serviceAddress,
        description=nb.description,
        resource_pool=nb.resourcePool,
        task_type="jupyter-lab",
        currentSlotsExceeded=False,
    )

    webbrowser.open(f"{args.master}/{nb_path}")


args_description: cli.ArgsDescription = [
    cli.Cmd(
        "notebook",
        None,
        "manage notebooks",
        [
            cli.Cmd(
                "list ls",
                functools.partial(ntsc.list_tasks),
                "list notebooks",
                ntsc.ls_sort_args
                + [
                    cli.Arg("-q", "--quiet", action="store_true", help="only display the IDs"),
                    cli.Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        help="show all notebooks (including other users')",
                    ),
                    cli.workspace.workspace_arg,
                    cli.Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
                ],
                is_default=True,
            ),
            cli.Cmd(
                "config",
                functools.partial(ntsc.config),
                "display notebook config",
                [
                    cli.Arg("notebook_id", type=str, help="notebook ID"),
                ],
            ),
            cli.Cmd(
                "start",
                start_notebook,
                "start a new notebook",
                [
                    cli.Arg(
                        "--config-file",
                        default=None,
                        type=argparse.FileType("r"),
                        help="command config file (.yaml)",
                    ),
                    cli.workspace.workspace_arg,
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
                    cli.Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                    cli.Arg(
                        "--template",
                        type=str,
                        help="name of template to apply to the notebook configuration",
                    ),
                    cli.Arg(
                        "--no-browser",
                        action="store_true",
                        help="don't open the notebook in a browser after startup",
                    ),
                    cli.Arg(
                        "-d",
                        "--detach",
                        action="store_true",
                        help="run in the background and print the ID",
                    ),
                    cli.Arg(
                        "--preview", action="store_true", help="preview the notebook configuration"
                    ),
                ],
            ),
            cli.Cmd(
                "open",
                open_notebook,
                "open an existing notebook",
                [cli.Arg("notebook_id", help="notebook ID")],
            ),
            cli.Cmd(
                "logs",
                functools.partial(task.logs),
                "fetch notebook logs",
                [
                    cli.Arg("task_id", help="notebook ID", metavar="notebook_id"),
                    *task.common_log_options,
                ],
            ),
            cli.Cmd(
                "kill",
                functools.partial(ntsc.kill),
                "kill a notebook",
                [
                    cli.Arg("notebook_id", help="notebook ID", nargs=argparse.ONE_OR_MORE),
                    cli.Arg("-f", "--force", action="store_true", help="ignore errors"),
                ],
            ),
            cli.Cmd(
                "set",
                None,
                "set notebook attributes",
                [
                    cli.Cmd(
                        "priority",
                        functools.partial(ntsc.set_priority),
                        "set notebook priority",
                        [
                            cli.Arg("notebook_id", help="notebook ID"),
                            cli.Arg("priority", type=int, help="priority"),
                        ],
                    ),
                ],
            ),
        ],
    )
]
