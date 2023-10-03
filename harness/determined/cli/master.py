from argparse import Namespace
from typing import Any, List, Optional

import determined.cli.render
from determined import cli
from determined.cli.errors import CliError
from determined.common import util
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def config(args: Namespace) -> None:
    resp = bindings.get_GetMasterConfig(cli.setup_session(args)).config
    if args.json:
        determined.cli.render.print_json(resp)
    else:
        print(yaml.safe_dump(resp, default_flow_style=False))


@authentication.required
def set_master_config(args: Namespace) -> None:
    if args.__dict__["log.level"] and args.__dict__["log.color"]:
        if args.__dict__["log.color"].lower() == "on":
            master_config = bindings.v1Config(
                log=bindings.v1LogConfig(
                    level=parseLogLevel(args.__dict__["log.level"]), color=True
                )
            )
        elif args.__dict__["log.color"].lower() == "off":
            master_config = bindings.v1Config(
                log=bindings.v1LogConfig(
                    level=parseLogLevel(args.__dict__["log.level"]), color=False
                )
            )
        else:
            raise CliError(
                "Error: Invalid value for --color provided: "
                + args.__dict__["log.color"]
                + ". This optional parameter only accepts 'on' and 'off' values."
            )
        field_mask = ["log"]
    elif args.__dict__["log.level"]:
        master_config = bindings.v1Config(
            log=bindings.v1LogConfig(level=parseLogLevel(args.__dict__["log.level"]))
        )
        field_mask = ["log.level"]
    elif args.__dict__["log.color"]:
        if args.__dict__["log.color"].lower() == "on":
            master_config = bindings.v1Config(log=bindings.v1LogConfig(color=True))
        elif args.__dict__["log.color"].lower() == "off":
            master_config = bindings.v1Config(log=bindings.v1LogConfig(color=False))
        else:
            raise CliError(
                "Error: Invalid value for --color provided: "
                + args.__dict__["log.color"]
                + ". This optional parameter only accepts 'on' and 'off' values."
            )
        field_mask = ["log.color"]
    else:
        raise CliError(
            "Error: No config values provided, at least one value required to set master config. "
            + "Currently, --log.level and --log.color are the supported set config attributes."
        )
    req = bindings.v1PatchMasterConfigRequest(
        config=master_config, fieldMask=bindings.protobufFieldMask(paths=field_mask)
    )
    resp = bindings.patch_PatchMasterConfig(cli.setup_session(args), body=req).config

    if args.json:
        determined.cli.render.print_json(resp)
    else:
        print(util.yaml_safe_dump(resp, default_flow_style=False))


def get_master(args: Namespace) -> None:
    resp = bindings.get_GetMaster(cli.setup_session(args))
    if args.json:
        determined.cli.render.print_json(resp.to_json())
    else:
        print(util.yaml_safe_dump(resp.to_json(), default_flow_style=False))


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


def parseLogLevel(level: str) -> bindings.v1LogLevel:
    if level.upper() == "INFO":
        return bindings.v1LogLevel.INFO
    elif level.upper() == "TRACE":
        return bindings.v1LogLevel.TRACE
    elif level.upper() == "DEBUG":
        return bindings.v1LogLevel.DEBUG
    elif level.upper() == "WARNING" or level.upper() == "WARN":
        return bindings.v1LogLevel.WARNING
    elif level.upper() == "ERROR":
        return bindings.v1LogLevel.ERROR
    elif level.upper() == "CRITICAL":
        return bindings.v1LogLevel.CRITICAL
    else:
        raise CliError(
            "Error: Invalid Log level provided. "
            + "Acceptable levels are: INFO, TRACE, DEBUG, WARNING, ERROR and CRITICAL."
        )


# fmt: off

args_description = [
    Cmd("master", None, "manage master", [
        Cmd(
            "config",
            None,
            "manage master config",
            [
                Cmd(
                    "show",
                    config,
                    "show master config",
                    [
                        Group(cli.output_format_args["json"],
                              cli.output_format_args["yaml"])
                    ],
                    is_default=True,
                ),
                Cmd(
                    "set",
                    set_master_config,
                    "set master config",
                    [
                        Arg("--log.level", type=str, default=None, required=False,
                            help="set log level in the master config"),
                        Arg("--log.color", type=str, default=None, required=False,
                            help="set log color in the master config"),
                        Group(cli.output_format_args["json"],
                              cli.output_format_args["yaml"])
                    ]
                ),
            ]
        ),
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
