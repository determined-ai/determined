import json
from argparse import Namespace
from typing import Any, List, Optional

from determined import cli
from determined.common import api, yaml
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def config(args: Namespace) -> None:
    response = api.get(args.master, "config")
    if args.json:
        print(json.dumps(response.json(), indent=4))
    else:
        print(yaml.safe_dump(response.json(), default_flow_style=False))


def get_master(args: Namespace) -> None:
    resp = bindings.get_GetMaster(cli.setup_session(args))
    if args.json:
        print(json.dumps(resp.to_json(), indent=4))
    else:
        print(yaml.safe_dump(resp.to_json(), default_flow_style=False))


def format_log_entry(log: bindings.v1LogEntry) -> str:
    """Format v1LogEntry for printing."""
    log_level = str(log.level.value)[len("LOG_LEVEL_") :] if log.level else ""
    return f"{log.timestamp} [{log_level}]: {log.message}"


@authentication.required
def logs(args: Namespace) -> None:
    offset: Optional[int] = None
    if args.tail:
        offset = -args.tail
    responses = bindings.get_MasterLogs(cli.setup_session(args), follow=args.follow, offset=offset)
    for response in responses:
        print(format_log_entry(response.logEntry))


# fmt: off

args_description = [
    Cmd("master", None, "manage master", [
        Cmd("config", config, "fetch master config", [
            Group(cli.output_format_args["json"], cli.output_format_args["yaml"])
        ]),
        Cmd("info", get_master, "fetch master info", [
            Group(cli.output_format_args["json"], cli.output_format_args["yaml"])
        ]),
        Cmd("logs", logs, "fetch master logs", [
            Arg("-f", "--follow", action="store_true",
                help="follow the logs of master, similar to tail -f"),
            Arg("--tail", type=int,
                help="number of lines to show, counting from the end "
                "of the log (default is all)")
        ]),
    ])
]  # type: List[Any]

# fmt: on
