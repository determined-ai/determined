import urllib.parse
from argparse import ONE_OR_MORE, REMAINDER, FileType, Namespace
from pathlib import Path
from typing import Any, Dict, List

from termcolor import colored

from determined_common import api
from determined_common.api.authentication import authentication_required

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
def run_command(args: Namespace) -> None:
    config = parse_config(args.config_file, args.entrypoint, args.config, args.volume)
    resp = launch_command(
        args.master,
        "commands",
        config,
        args.template,
        context_path=args.context,
    )

    if args.detach:
        print(resp["id"])
        return

    url = "commands/{}/events".format(resp["id"])

    with api.ws(args.master, url) as ws:
        for msg in ws:
            render_event_stream(msg)


@authentication_required
def list_commands(args: Namespace) -> None:
    if args.all:
        params = {}  # type: Dict[str, Any]
    else:
        params = {"user": api.Authentication.instance().get_session_user()}
    commands = [
        render.unmarshal(Command, command)
        for command in api.get(args.master, path="commands", params=params).json().values()
    ]

    if args.quiet:
        for command in commands:
            print(command.id)
        return

    render.render_objects(CommandDescription, [describe_command(command) for command in commands])


@authentication_required
def tail_command_logs(args: Namespace) -> None:
    token = api.Authentication.instance().get_session_token()
    params = {"follow": args.follow, "tail": args.tail, "_auth": token}

    url = "commands/{}/events?{}".format(args.command_id, urllib.parse.urlencode(params))

    with api.ws(args.master, url) as ws:
        for msg in ws:
            render_event_stream(msg)


@authentication_required
def kill_command(args: Namespace) -> None:
    for i, cid in enumerate(args.command_id):
        try:
            api.delete(args.master, "commands/{}".format(cid))
            print(colored("Killed command {}".format(cid), "green"))
        except api.errors.APIException as e:
            if not args.force:
                for ignored in args.command_id[i + 1 :]:
                    print("Cowardly not killing {}".format(ignored))
                raise e
            print(colored("Skipping: {} ({})".format(e, type(e).__name__), "red"))


@authentication_required
def command_config(args: Namespace) -> None:
    res_json = api.get(args.master, "commands/{}".format(args.id)).json()
    print(render.format_object_as_yaml(res_json["config"]))


# fmt: off

args_description = [
    Cmd("command cmd", None, "manage commands", [
        Cmd("list ls", list_commands, "list commands", [
            Arg("-q", "--quiet", action="store_true",
                help="only display the IDs"),
            Arg("--all", "-a", action="store_true",
                help="show all commands (including other users')"),
        ], is_default=True),
        Cmd("config", command_config,
            "display command config", [
                Arg("id", type=str, help="command ID"),
            ]),
        Cmd("run", run_command, "create command", [
            Arg("entrypoint", type=str, nargs=REMAINDER,
                help="entrypoint command and arguments to execute"),
            Arg("--config-file", default=None, type=FileType("r"),
                help="command config file (.yaml)"),
            Arg("-v", "--volume", action="append", default=[],
                help=VOLUME_DESC),
            Arg("-c", "--context", default=None, type=Path, help=CONTEXT_DESC),
            Arg("--config", action="append", default=[], help=CONFIG_DESC),
            Arg("--template", type=str,
                help="name of template to apply to the command configuration"),
            Arg("-d", "--detach", action="store_true",
                help="run in the background and print the ID")
        ]),
        Cmd("logs", tail_command_logs, "fetch command logs", [
            Arg("command_id", help="command ID"),
            Arg("-f", "--follow", action="store_true",
                help="follow the logs of a command, similar to tail -f"),
            Arg("--tail", type=int, default=200,
                help="number of lines to show, counting from the end "
                     "of the log")
        ]),
        Cmd("kill", kill_command, "forcibly terminate a command", [
            Arg("command_id", help="command ID", nargs=ONE_OR_MORE),
            Arg("-f", "--force", action="store_true", help="ignore errors"),
        ]),
    ])
]  # type: List[Any]

# fmt: on
