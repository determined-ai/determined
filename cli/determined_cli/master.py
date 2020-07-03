import json
import time
from argparse import Namespace
from typing import Any, List

from requests import Response

from determined_common import api
from determined_common.api.authentication import authentication_required
from determined_common.check import check_gt

from .declarative_argparse import Arg, Cmd


@authentication_required
def config(args: Namespace) -> None:
    response = api.get(args.master, "config")
    print(json.dumps(response.json(), indent=4))


@authentication_required
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
    Cmd("m|aster", None, "manage master", [
        Cmd("config", config, "fetch master config as JSON", []),
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
