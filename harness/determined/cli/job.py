import json
from argparse import Namespace
from typing import Any, List

import yaml

from determined.cli.session import setup_session
from determined.common.api import authentication
from determined.common.api.b import get_GetJobs
from determined.common.declarative_argparse import Arg, Cmd


@authentication.required
def ls(args: Namespace) -> None:
    response = get_GetJobs(
        setup_session(args),
        resourcePool=args.resource_pool,
        pagination_limit=args.limit,
        pagination_offset=args.offset,
        orderBy=args.order_by,
    )
    if args.output == "yaml":
        print(yaml.safe_dump(response.to_json(), default_flow_style=False))
    elif args.output == "json":
        print(json.dumps(response.to_json(), indent=4, default=str))
    elif ["csv", "table"].count(args.output) > 0:
        raise NotImplementedError(f"Output not implemented, adopt a cat to unlock: {args.output}")
    else:
        raise ValueError(f"Bad output format: {args.output}")


pagination_args = [
    Arg(
        "--offset",
        type=int,
        default=0,
        help="Offset the returned set.",
    ),
    Arg(
        "--limit",
        type=int,
        default=50,
        help="Limit the returned set.",
    ),
    Arg(
        "--order-by",
        type=str,
        help="Whether to sort the results in descending or ascending order",
    ),
]

args_description = [
    Cmd(
        "j|ob",
        None,
        "manage job",
        [
            Cmd(
                "list",
                ls,
                "list jobs",
                [
                    Arg(
                        "-o",
                        "--output",
                        type=str,
                        default="yaml",
                        help="Output format, one of json|yaml",
                    ),
                    Arg("-rp", "--resource-pool", type=str, default="default", help=""),
                    *pagination_args,
                ],
            ),
        ],
    )
]  # type: List[Any]
