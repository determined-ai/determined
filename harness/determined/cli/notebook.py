from argparse import ONE_OR_MORE, FileType, Namespace
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
def start_notebook(args: Namespace) -> None:
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
    resp = bindings.post_LaunchNotebook(cli.setup_session(args), body=body)

    if args.preview:
        print(render.format_object_as_yaml(resp.config))
        return

    nb = resp.notebook

    if args.detach:
        print(nb.id)
        return

    render.report_job_launched("notebook", resp.notebook.id)

    if resp.warnings:
        cli.print_launch_warnings(resp.warnings)
    currentSlotsExceeded = (resp.warnings is not None) and (
        bindings.v1LaunchWarning.CURRENT_SLOTS_EXCEEDED in resp.warnings
    )

    cli.wait_ntsc_ready(cli.setup_session(args), api.NTSC_Kind.notebook, nb.id)

    assert nb.serviceAddress is not None, "missing tensorboard serviceAddress"
    nb_path = request.make_interactive_task_url(
        task_id=nb.id,
        service_address=nb.serviceAddress,
        description=nb.description,
        resource_pool=nb.resourcePool,
        task_type="jupyter-lab",
        currentSlotsExceeded=currentSlotsExceeded,
    )
    url = api.make_url(args.master, nb_path)
    if not args.no_browser:
        api.browser_open(args.master, nb_path)
    print(colored("Jupyter Notebook is running at: {}".format(url), "green"))


@authentication.required
def open_notebook(args: Namespace) -> None:
    notebook_id = cast(str, ntsc.expand_uuid_prefixes(args))

    sess = cli.setup_session(args)
    task = bindings.get_GetTask(sess, taskId=notebook_id).task
    check_none(task.endTime, "Notebook has ended")

    nb = bindings.get_GetNotebook(sess, notebookId=notebook_id).notebook
    assert nb.serviceAddress is not None, "missing tensorboard serviceAddress"

    api.browser_open(
        args.master,
        request.make_interactive_task_url(
            task_id=nb.id,
            service_address=nb.serviceAddress,
            description=nb.description,
            resource_pool=nb.resourcePool,
            task_type="jupyter-lab",
            currentSlotsExceeded=False,
        ),
    )


args_description: ArgsDescription = [
    Cmd(
        "notebook",
        None,
        "manage notebooks",
        [
            Cmd(
                "list ls",
                partial(ntsc.list_tasks),
                "list notebooks",
                ntsc.ls_sort_args
                + [
                    Arg("-q", "--quiet", action="store_true", help="only display the IDs"),
                    Arg(
                        "--all",
                        "-a",
                        action="store_true",
                        help="show all notebooks (including other users')",
                    ),
                    cli.workspace.workspace_arg,
                    Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
                ],
                is_default=True,
            ),
            Cmd(
                "config",
                partial(ntsc.config),
                "display notebook config",
                [
                    Arg("notebook_id", type=str, help="notebook ID"),
                ],
            ),
            Cmd(
                "start",
                start_notebook,
                "start a new notebook",
                [
                    Arg(
                        "--config-file",
                        default=None,
                        type=FileType("r"),
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
                        "--template",
                        type=str,
                        help="name of template to apply to the notebook configuration",
                    ),
                    Arg(
                        "--no-browser",
                        action="store_true",
                        help="don't open the notebook in a browser after startup",
                    ),
                    Arg(
                        "-d",
                        "--detach",
                        action="store_true",
                        help="run in the background and print the ID",
                    ),
                    Arg(
                        "--preview", action="store_true", help="preview the notebook configuration"
                    ),
                ],
            ),
            Cmd(
                "open",
                open_notebook,
                "open an existing notebook",
                [Arg("notebook_id", help="notebook ID")],
            ),
            Cmd(
                "logs",
                partial(task.logs),
                "fetch notebook logs",
                [
                    Arg("task_id", help="notebook ID", metavar="notebook_id"),
                    *task.common_log_options,
                ],
            ),
            Cmd(
                "kill",
                partial(ntsc.kill),
                "kill a notebook",
                [
                    Arg("notebook_id", help="notebook ID", nargs=ONE_OR_MORE),
                    Arg("-f", "--force", action="store_true", help="ignore errors"),
                ],
            ),
            Cmd(
                "set",
                None,
                "set notebook attributes",
                [
                    Cmd(
                        "priority",
                        partial(ntsc.set_priority),
                        "set notebook priority",
                        [
                            Arg("notebook_id", help="notebook ID"),
                            Arg("priority", type=int, help="priority"),
                        ],
                    ),
                ],
            ),
        ],
    )
]
