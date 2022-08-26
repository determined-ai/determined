import json
import time
from argparse import Namespace
from typing import Any, List

from requests import Response

from determined import cli
from determined.common import api, yaml
from determined.common.api import authentication, bindings
from determined.common.check import check_gt
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


@authentication.required
def logs(args: Namespace) -> None:
    def process_response(response: Response, latest_log_id: int) -> int:
        for log in response.json():
            check_gt(log["id"], latest_log_id)
            latest_log_id = log["id"]
            print("{} [{}]: {}".format(log["time"], log["level"], log["message"]))
        return latest_log_id

    params = {}
    if args.tail:
        params["tail"] = args.tail

    response = api.get(args.master, "logs", params=params)
    latest_log_id = process_response(response, -1)

    # "Follow" mode is implemented as a loop in the CLI. We assume that
    # newer log messages have a numerically larger ID than older log
    # messages, so we keep track of the max ID seen so far.
    if args.follow:
        while True:
            try:
                # Poll for new logs every 100 ms.
                time.sleep(0.1)

                # The `tail` parameter only makes sense the first time we
                # fetch logs.
                response = api.get(
                    args.master, "logs", params={"greater_than_id": str(latest_log_id)}
                )
                latest_log_id = process_response(response, latest_log_id)
            except KeyboardInterrupt:
                break


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
