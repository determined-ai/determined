import json
from argparse import Namespace
from typing import Any, List

import yaml

from determined.common.api.fapi import auth_required, experiments_api, to_json
from determined.common.declarative_argparse import Arg, Cmd

# from determined.common.api.fastapi_client.models import V1LoginRequest

# import determined.common.api.swagger as swg
# import determined.common.api.swagger_client.models as models
# @swg.auth_required
# def list(_: Namespace) -> None:
#     # type info shows for some language servers but not all since the generated code
#     # is using legacy docstring types
#     api_response = swg.experiment_api.determined_get_experiments(resource_pools=["default"])
#     print(api_response.experiments)


@auth_required
def list(args: Namespace) -> None:
    response = experiments_api.determined_get_experiments(limit=5)
    experiments_json = to_json(response.experiments)
    if args.output == "yaml":
        print(yaml.safe_dump(experiments_json, default_flow_style=False))
    elif args.output == "json":
        print(json.dumps(experiments_json, indent=4, default=str))
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
