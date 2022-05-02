from argparse import ONE_OR_MORE, FileType, Namespace
from pathlib import Path
from typing import Any, List

from termcolor import colored

from determined.cli import command, render, task
from determined.cli.session import setup_session
from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.check import check_eq
from determined.common.context import Context
from determined.common.declarative_argparse import Arg, Cmd

from .command import CONFIG_DESC, CONTEXT_DESC, VOLUME_DESC, parse_config, render_event_stream


@authentication.required
def start_notebook(args: Namespace) -> None:
    config = parse_config(args.config_file, None, args.config, args.volume)

    files = None
    if args.context is not None:
        context = Context.from_local(args.context)
        files = [
            bindings.v1File(
                content=e.content.decode("utf-8"),
                gid=e.gid,
                mode=e.mode,
                mtime=e.mtime,
                path=e.path,
                type=e.type,
                uid=e.uid,
            )
            for e in context.entries
        ]
    body = bindings.v1LaunchNotebookRequest(config, files=files, preview=False)
    resp = bindings.post_LaunchNotebook(setup_session(args), body=body)

    if args.preview:
        print(render.format_object_as_yaml(resp.config))
        return

    nb = resp.notebook

    if args.detach:
        print(nb.id)
        return

    with api.ws(args.master, "notebooks/{}/events".format(nb.id)) as ws:
        for msg in ws:
            if msg["service_ready_event"] and nb.serviceAddress and not args.no_browser:
                url = api.browser_open(args.master, nb.serviceAddress)
                print(colored("Jupyter Notebook is running at: {}".format(url), "green"))
            render_event_stream(msg)


@authentication.required
def open_notebook(args: Namespace) -> None:
    notebook_id = command.expand_uuid_prefixes(args)
    resp = api.get(args.master, "api/v1/notebooks/{}".format(notebook_id)).json()["notebook"]
    check_eq(resp["state"], "STATE_RUNNING", "Notebook must be in a running state")
    api.browser_open(args.master, resp["serviceAddress"])


# fmt: off

args_description = [
    Cmd("notebook", None, "manage notebooks", [
        Cmd("list ls", command.list_tasks, "list notebooks", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all notebooks (including other users')")
        ], is_default=True),
        Cmd("config", command.config,
            "display notebook config", [
                Arg("notebook_id", type=str, help="notebook ID"),
            ]),
        Cmd("start", start_notebook, "start a new notebook", [
            Arg("--config-file", default=None, type=FileType("r"),
                help="command config file (.yaml)"),
            Arg("-v", "--volume", action="append", default=[],
                help=VOLUME_DESC),
            Arg("-c", "--context", default=None, type=Path, help=CONTEXT_DESC),
            Arg("--config", action="append", default=[], help=CONFIG_DESC),
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
        Cmd("logs", lambda *args, **kwargs: task.logs(*args, **kwargs), "fetch notebook logs", [
            Arg("task_id", help="notebook ID", metavar="notebook_id"),
            *task.common_log_options
        ]),
        Cmd("kill", command.kill, "kill a notebook", [
            Arg("notebook_id", help="notebook ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
        Cmd("set", None, "set notebook attributes", [
            Cmd("priority", command.set_priority, "set notebook priority", [
                Arg("notebook_id", help="notebook ID"),
                Arg("priority", type=int, help="priority"),
            ]),
        ]),
    ])
]  # type: List[Any]

# fmt: on
