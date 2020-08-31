from argparse import ONE_OR_MORE, FileType, Namespace
from pathlib import Path
from typing import Any, Dict, List

from termcolor import colored

from determined_common import api
from determined_common.api.authentication import authentication_required
from determined_common.check import check_eq

from . import render
from .command import (
    CONFIG_DESC,
    CONTEXT_DESC,
    VOLUME_DESC,
    Command,
    CommandDescription,
    describe_command,
    launch_command,
    parse_config,
    render_event_stream,
)
from .declarative_argparse import Arg, Cmd


@authentication_required
def start_notebook(args: Namespace) -> None:
    config = parse_config(args.config_file, None, args.config, args.volume)

    resp = launch_command(
        args.master,
        "notebooks",
        config,
        args.template,
        context_path=args.context,
    )

    if args.detach:
        print(resp["id"])
        return

    with api.ws(args.master, "notebooks/{}/events".format(resp["id"])) as ws:
        for msg in ws:
            if msg["service_ready_event"] and not args.no_browser:
                url = api.open(args.master, resp["service_address"])
                print(colored("Jupyter Notebook is running at: {}".format(url), "green"))
            render_event_stream(msg)


@authentication_required
def open_notebook(args: Namespace) -> None:
    resp = api.get(args.master, "notebooks/{}".format(args.notebook_id)).json()
    notebook = render.unmarshal(Command, resp)
    check_eq(notebook.state, "RUNNING", "Notebook must be in a running state")
    api.open(args.master, resp["service_address"])


@authentication_required
def tail_notebook_logs(args: Namespace) -> None:
    url = "notebooks/{}/events?follow={}&tail={}".format(args.notebook_id, args.follow, args.tail)
    with api.ws(args.master, url) as ws:
        for msg in ws:
            render_event_stream(msg)


@authentication_required
def list_notebooks(args: Namespace) -> None:
    if args.all:
        params = {}  # type: Dict[str, Any]
    else:
        params = {"user": api.Authentication.instance().get_session_user()}
    commands = [
        render.unmarshal(Command, command)
        for command in api.get(args.master, "notebooks", params=params).json().values()
    ]

    if args.quiet:
        for command in commands:
            print(command.id)
        return

    render.render_objects(CommandDescription, [describe_command(command) for command in commands])


@authentication_required
def kill_notebook(args: Namespace) -> None:
    for i, nid in enumerate(args.notebook_id):
        try:
            api.delete(args.master, "notebooks/{}".format(nid))
            print(colored("Killed notebook {}".format(nid), "green"))
        except api.errors.APIException as e:
            if not args.force:
                for ignored in args.notebook_id[i + 1 :]:
                    print("Cowardly not killing {}".format(ignored))
                raise e
            print(colored("Skipping: {} ({})".format(e, type(e).__name__), "red"))


@authentication_required
def notebook_config(args: Namespace) -> None:
    res_json = api.get(args.master, "notebooks/{}".format(args.id)).json()
    print(render.format_object_as_yaml(res_json["config"]))


# fmt: off

args_description = [
    Cmd("notebook", None, "manage notebooks", [
        Cmd("list ls", list_notebooks, "list notebooks", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all notebooks (including other users')")
        ], is_default=True),
        Cmd("config", notebook_config,
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
        ]),
        Cmd("open", open_notebook, "open an existing notebook", [
            Arg("notebook_id", help="notebook ID")
        ]),
        Cmd("logs", tail_notebook_logs, "fetch notebook logs", [
            Arg("notebook_id", help="notebook ID"),
            Arg("-f", "--follow", action="store_true",
                help="follow the logs of a notebook, similar to tail -f"),
            Arg("--tail", type=int, default=200,
                help="number of lines to show, counting from the end "
                     "of the log")
        ]),
        Cmd("kill", kill_notebook, "kill a notebook", [
            Arg("notebook_id", help="notebook ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
    ])
]  # type: List[Any]

# fmt: on
