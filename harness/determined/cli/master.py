from argparse import Namespace
from typing import Any, List, Optional

import determined.cli.render
from determined import cli
from determined.cli.errors import CliError
from determined.common import yaml
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def config(args: Namespace) -> None:
    if args.log:
        if args.level and args.color:
            if args.color.lower() == "on":
                master_config = bindings.v1Config(
                    log=bindings.v1LogConfig(level=args.level, color=True)
                )
            elif args.color.lower() == "off":
                master_config = bindings.v1Config(
                    log=bindings.v1LogConfig(level=args.level, color=False)
                )
            else:
                raise CliError(
                    "Error: Invalid value for --color provided: "
                    + args.color
                    + ". This optional parameter only accepts 'on' and 'off' values."
                )
            field_mask = ["log"]
        elif args.level:
            master_config = bindings.v1Config(log=bindings.v1LogConfig(level=args.level))
            field_mask = ["log.level"]
        elif args.color:
            if args.color.lower() == "on":
                master_config = bindings.v1Config(log=bindings.v1LogConfig(color=True))
            elif args.color.lower() == "off":
                master_config = bindings.v1Config(log=bindings.v1LogConfig(color=False))
            else:
                raise CliError(
                    "Error: Invalid value for --color provided: "
                    + args.color
                    + ". This optional parameter only accepts 'on' and 'off' values."
                )
            field_mask = ["log.color"]
        else:
            raise CliError(
                "Error: No log level or log color provided. Either log level or log"
                + "  color is required to make changes to the master config log setting."
            )
        req = bindings.v1PatchMasterConfigRequest(
            config=master_config, fieldMask=bindings.protobufFieldMask(paths=field_mask)
        )
        resp = bindings.patch_PatchMasterConfig(cli.setup_session(args), body=req).config
    elif not args.log and (args.level or args.color):
        raise CliError(
            "Error: Invalid command: --level and/or --color used without --log. Please try"
            + " again using 'det master config --log --level <level> --color <on/off>'."
        )
    else:
        resp = bindings.get_GetMasterConfig(cli.setup_session(args)).config
    if args.json:
        determined.cli.render.print_json(resp)
    else:
        print(yaml.safe_dump(resp, default_flow_style=False))


def get_master(args: Namespace) -> None:
    resp = bindings.get_GetMaster(cli.setup_session(args))
    if args.json:
        determined.cli.render.print_json(resp.to_json())
    else:
        print(yaml.safe_dump(resp.to_json(), default_flow_style=False))


def format_log_entry(log: bindings.v1LogEntry) -> str:
    """Format v1LogEntry for printing."""
    log_level = log.level if log.level else ""
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
        Cmd("config", config, "fetch or patch master config", [
            Arg("--log", action="store_true",
                help="patch log in master config"),
            Arg("--level", type=str, default=None, required=False,
                help="set log level in the master config"),
            Arg("--color", type=str, default=None, required=False,
                help="set log color in the master config"),
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
