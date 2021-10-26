from argparse import Namespace
import json
from typing import Any, List

import yaml

from determined.common.api.fapi import client, set_host
from determined.common.api import authentication
from determined.common.api.fastapi_client.api.experiments_api import SyncExperimentsApi
from determined.common.declarative_argparse import Arg, Cmd

experiments_api = SyncExperimentsApi(client)  # type: ignore

@authentication.required
@set_host
def list(args: Namespace) -> None:
    response = experiments_api.determined_get_experiments(limit=5)
    # print(response.experiments[0].name)
    # print(json.dumps(response, cls=MyEncoder))
    # print(json.dumps(response.to_jsonble()))
    jsonable_e_list = [e.to_jsonble() for e in response.experiments]
    if args.output == "yaml":
        print(yaml.safe_dump(jsonable_e_list, default_flow_style=False))
    elif args.output == "json":
        print(json.dumps(jsonable_e_list, indent=4, default=str))
    elif ["csv", "table"].count(args.output) > 0:
        # render.tabulate_or_csv # TODO maybe add support for csv or tabular format. ref exp list
        raise NotImplementedError(f"Output not implemented, adopt a cat to unlock: {args.output}")
    else:
        raise ValueError(f"Bad output format: {args.output}")

@authentication.required
@set_host
def archive(args: Namespace) -> None:
    experiments_api.determined_archive_experiment(args.experiment_id)
    print("Archived experiment {}".format(args.experiment_id))

@authentication.required
@set_host
def unarchive(args: Namespace) -> None:
    experiments_api.determined_unarchive_experiment(args.experiment_id)
    print("Archived experiment {}".format(args.experiment_id))

def experiment_id_arg(help: str) -> Arg:  # noqa: A002
    return Arg("experiment_id", type=int, help=help)

args_description = [
    Cmd(
        "x|periment",
        None,
        "manage experiment",
        [
            Cmd(
                "list",
                list,
                "list experiments",
                [
                    Arg(
                        "-o",
                        "--output",
                        type=str,
                        default="yaml",
                        help="Output format, one of json|yaml",
                    ),
                    Arg("-rp", "--resource-pool", type=str, default="default", help=""),
                ],
            ),
            Cmd(
                "archive",
                archive,
                "archive experiment",
                [experiment_id_arg("experiment ID to archive")],
            ),
            Cmd(
                "unarchive",
                unarchive,
                "unarchive experiment",
                [experiment_id_arg("experiment ID to unarchive")],
            ),
        ],
    )
]  # type: List[Any]
