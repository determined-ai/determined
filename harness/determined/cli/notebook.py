from argparse import ONE_OR_MORE, FileType, Namespace
from pathlib import Path
from typing import Any, List

from termcolor import colored

from determined.cli import command, render
from determined.common import api
from determined.common.api import authentication
from determined.common.check import check_eq
from determined.common.declarative_argparse import Arg, Cmd

from .command import (
    CONFIG_DESC,
    CONTEXT_DESC,
    VOLUME_DESC,
    launch_command,
    parse_config,
    render_event_stream,
)


@authentication.required
def start_notebook(args: Namespace) -> None:
    config = parse_config(args.config_file, None, args.config, args.volume)

    resp = launch_command(
        args.master,
        "api/v1/notebooks",
        config,
        args.template,
        context_path=args.context,
        preview=args.preview,
    )

    if args.preview:
        print(render.format_object_as_yaml(resp["config"]))
        return

    obj = resp["notebook"]

    if args.detach:
        print(obj["id"])
        return

    with api.ws(args.master, "notebooks/{}/events".format(obj["id"])) as ws:
        for msg in ws:
            if msg["service_ready_event"] and not args.no_browser:
                url = api.browser_open(args.master, obj["serviceAddress"])
                print(colored("Jupyter Notebook is running at: {}".format(url), "green"))
            render_event_stream(msg)


@authentication.required
def open_notebook(args: Namespace) -> None:
    resp = api.get(args.master, "api/v1/notebooks/{}".format(args.notebook_id)).json()["notebook"]
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
                Arg("id", type=str, help="notebook ID"),
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
        Cmd("logs", command.tail_logs, "fetch notebook logs", [
            Arg("notebook_id", help="notebook ID"),
            Arg("-f", "--follow", action="store_true",
                help="follow the logs of a notebook, similar to tail -f"),
            Arg("--tail", type=int, default=200,
                help="number of lines to show, counting from the end "
                     "of the log")
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
