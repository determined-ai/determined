import argparse
from argparse import Namespace
from datetime import datetime
from typing import Any, List, Optional

from determined import cli
from determined.cli import render
from determined.common import util
from determined.common.api import bindings
from determined.common.declarative_argparse import Arg, Cmd, Group


def show_config(args: Namespace) -> None:
    sess = cli.setup_session(args)
    resp = bindings.get_GetMasterConfig(sess).config
    if args.json:
        render.print_json(resp)
    else:
        print(util.yaml_safe_dump(resp, default_flow_style=False))


def set_master_config(args: Namespace) -> None:
    sess = cli.setup_session(args)
    log_config = bindings.v1LogConfig()
    field_masks = []
    if "log_color" in args:
        log_config.color = True if args.log_color == "on" else False
        field_masks.append("log.color")
    if "log_level" in args:
        log_config.level = bindings.v1LogLevel[args.log_level]
        field_masks.append("log.level")

    if len(field_masks) == 0:
        raise cli.errors.CliError(
            "Please provide at least one argument to set master config. "
            + "Currently, the supported fields are --log.level and --log.color."
        )

    master_config = bindings.v1Config(log=log_config)
    req = bindings.v1PatchMasterConfigRequest(
        config=master_config, fieldMask=bindings.protobufFieldMask(paths=field_masks)
    )
    bindings.patch_PatchMasterConfig(sess, body=req)
    cli.warn(
        "This will only make ephermeral changes to the master config, "
        + "that will be lost if the user restarts the cluster."
    )
    print("Successfully made changes to the master config.")


def get_master(args: Namespace) -> None:
    sess = cli.setup_session(args)
    resp = bindings.get_GetMaster(sess)
    if args.json:
        render.print_json(resp.to_json())
    else:
        print(util.yaml_safe_dump(resp.to_json(), default_flow_style=False))


def format_log_entry(log: bindings.v1LogEntry) -> str:
    """Format v1LogEntry for printing."""
    log_level = log.level if log.level else ""
    return f"{log.timestamp} [{log_level}]: {log.message}"


def logs(args: Namespace) -> None:
    sess = cli.setup_session(args)
    offset: Optional[int] = None
    if args.tail:
        offset = -args.tail
    responses = bindings.get_MasterLogs(sess, follow=args.follow, offset=offset)
    for response in responses:
        print(format_log_entry(response.logEntry))


def set_cluster_message(args: Namespace) -> None:
    sess = cli.setup_session(args)

    if args.message is None:
        raise ValueError("Provide a message using the -m flag.")
    body = bindings.v1SetClusterMessageRequest(
        startTime=args.start, endTime=args.end, message=args.message
    )
    bindings.put_SetClusterMessage(sess, body=body)


def clear_cluster_message(args: Namespace) -> None:
    sess = cli.setup_session(args)
    bindings.delete_DeleteClusterMessage(sess)

# TODO: use the GetClusterMessage endpoint so future-scheduled messages are visible to admins
def get_cluster_message(args: Namespace) -> None:
    sess = cli.setup_session(args)

    resp = bindings.get_GetMaster(sess)
    message = resp.to_json()['clusterMessage']

    if args.json:
        render.print_json(message)
    else:
        print(util.yaml_safe_dump(message, default_flow_style=False))


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
                    show_config,
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
                        Arg("--log.level", type=str, default=argparse.SUPPRESS, required=False,
                            help="set log level in the master config", dest="log_level",
                            choices=[lvl.name for lvl in bindings.v1LogLevel
                                     if lvl != bindings.v1LogLevel.UNSPECIFIED]),
                        Arg("--log.color", type=str, default=argparse.SUPPRESS, required=False,
                            help="set log color in the master config", dest="log_color",
                            choices=["on", "off"])
                    ]
                ),
                Group(cli.output_format_args["json"],
                      cli.output_format_args["yaml"]),
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
        # TODO: display the message each time someone uses the CLI, same as version mismatch
        Cmd("display-message", None, "set or clear cluster-wide message", [
            Cmd("set", set_cluster_message, "create or edit the displayed cluster-wide message", [
                Arg("-s", "--start", default=datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ"),
                    help="Timestamp to start displaying message (RFC 3339 format), "
                    + "e.g. '2021-10-26T23:17:12Z'; default is now."),
                Group(
                    Arg("-e", "--end", default=None,
                        help="Timestamp to end displaying message (RFC 3339 format), "
                        + "e.g. '2021-10-26T23:17:12Z'; default is indefinite."),
                    Arg("-d", "--duration", default=None,
                        help="How long the message should last; mutually exclusive with --end and should"
                        + "be formatted as a Go duration string e.g. 24h, 2w, 5d"),
                ),
                Arg("-m", "--message", default=None,
                    help="Text of the message to display to users"),
            ]),
            Cmd("clear", clear_cluster_message, "clear cluster-wide message", [
                Arg("-c", "--clear", action="store_true", default=False,
                    help="Clear all cluster-wide message"),
            ]),
            Cmd("get", get_cluster_message, "get cluster-wide message", [
                Group(cli.output_format_args["json"], cli.output_format_args["yaml"])
            ]),
        ])
    ])
]  # type: List[Any]

# fmt: on
