from argparse import ONE_OR_MORE, FileType, Namespace
from functools import partial
from pathlib import Path
from typing import Any, List

from termcolor import colored

from determined import cli
from determined.cli import command, render, task
from determined.common import api, context
from determined.common.api import authentication, bindings, request
from determined.common.check import check_eq
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def start_notebook(args: Namespace) -> None:
    config = command.parse_config(args.config_file, None, args.config, args.volume)

    files = context.read_v1_context(args.context, args.include)
    body = bindings.v1LaunchNotebookRequest(
        config=config, files=files, preview=args.preview, templateName=args.template
    )
    resp = bindings.post_LaunchNotebook(cli.setup_session(args), body=body)

    if args.preview:
        print(render.format_object_as_yaml(resp.config))
        return

    nb = resp.notebook

    if args.detach:
        print(nb.id)
        return

    if resp.warnings:
        cli.print_warnings(resp.warnings)
    currentSlotsExceeded = (resp.warnings is not None) and (
        bindings.v1LaunchWarning.LAUNCH_WARNING_CURRENT_SLOTS_EXCEEDED in resp.warnings
    )

    with api.ws(args.master, "notebooks/{}/events".format(nb.id)) as ws:
        for msg in ws:
            if msg["service_ready_event"] and nb.serviceAddress and not args.no_browser:
                url = api.browser_open(
                    args.master,
                    request.make_interactive_task_url(
                        task_id=nb.id,
                        service_address=nb.serviceAddress,
                        description=nb.description,
                        resource_pool=nb.resourcePool,
                        task_type="notebook",
                        currentSlotsExceeded=currentSlotsExceeded,
                    ),
                )
                print(colored("Jupyter Notebook is running at: {}".format(url), "green"))
            command.render_event_stream(msg)


@authentication.required
def open_notebook(args: Namespace) -> None:
    notebook_id = command.expand_uuid_prefixes(args)
    resp = api.get(args.master, "api/v1/notebooks/{}".format(notebook_id)).json()["notebook"]
    check_eq(resp["state"], "STATE_RUNNING", "Notebook must be in a running state")

    api.browser_open(
        args.master,
        request.make_interactive_task_url(
            task_id=resp["id"],
            service_address=resp["serviceAddress"],
            description=resp["description"],
            resource_pool=resp["resourcePool"],
            task_type="notebook",
            currentSlotsExceeded=False,
        ),
    )


# fmt: off

args_description = [
    Cmd("notebook", None, "manage notebooks", [
        Cmd("list ls", partial(command.list_tasks), "list notebooks", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all notebooks (including other users')"),
            Group(cli.output_format_args["json"], cli.output_format_args["csv"]),
        ], is_default=True),
        Cmd("config", partial(command.config),
            "display notebook config", [
                Arg("notebook_id", type=str, help="notebook ID"),
        ]),
        Cmd("start", start_notebook, "start a new notebook", [
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
                help="name of template to apply to the notebook configuration"),
            Arg("--no-browser", action="store_true",
                help="don't open the notebook in a browser after startup"),
            Arg("-d", "--detach", action="store_true",
                help="run in the background and print the ID"),
            Arg("--preview", action="store_true",
                help="preview the notebook configuration"),
        ]),
        Cmd("open", open_notebook, "open an existing notebook", [
            Arg("notebook_id", help="notebook ID")
        ]),
        Cmd("logs", partial(task.logs), "fetch notebook logs", [
            Arg("task_id", help="notebook ID", metavar="notebook_id"),
            *task.common_log_options
        ]),
        Cmd("kill", partial(command.kill), "kill a notebook", [
            Arg("notebook_id", help="notebook ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
        Cmd("set", None, "set notebook attributes", [
            Cmd("priority", partial(command.set_priority), "set notebook priority", [
                Arg("notebook_id", help="notebook ID"),
                Arg("priority", type=int, help="priority"),
            ]),
        ]),
    ])
]  # type: List[Any]

# fmt: on
