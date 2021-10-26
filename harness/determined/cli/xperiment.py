from argparse import Namespace
import json
from typing import Any, List

import yaml

from determined.common.api.fapi import MyEncoder, auth_required, client
from determined.common.api.fastapi_client.api.experiments_api import SyncExperimentsApi
# from determined.common.api.fapi_helper import to_json
from determined.common.declarative_argparse import Arg, Cmd

experiments_api = SyncExperimentsApi(client)  # type: ignore

@auth_required
def list(args: Namespace) -> None:
    response = experiments_api.determined_get_experiments(limit=5)
    # print(response.experiments[0].name)
    # print(json.dumps(response, cls=MyEncoder))
    # print(json.dumps(response.to_jsonble()))
    # print(response.to_jsonble())
    # experiments_json = to_json(response.experiments)
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
        ],
    )
]  # type: List[Any]
