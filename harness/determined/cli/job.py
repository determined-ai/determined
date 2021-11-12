import json
from argparse import Namespace
from typing import Any, List

import yaml

from determined.common.api.b import get_Determined_GetJobs
from determined.cli.session import setup_session
from determined.common.declarative_argparse import Arg, Cmd

def list(args: Namespace) -> None:
    response = get_Determined_GetJobs(setup_session(args), resourcePool=args.resource_pool)
    if response.jobs is None:  # TODO remove once proto annotations are inplace.
        return
    if args.output == "yaml":
        print(yaml.safe_dump(response.to_json(), default_flow_style=False))
    elif args.output == "json":
        print(json.dumps(response.to_json(), indent=4, default=str))
    elif ["csv", "table"].count(args.output) > 0:
        raise NotImplementedError(f"Output not implemented, adopt a cat to unlock: {args.output}")
    else:
        raise ValueError(f"Bad output format: {args.output}")


args_description = [
    Cmd(
        "j|ob",
        None,
        "manage job",
        [
            Cmd(
                "list",
                list,
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
                ],
            ),
        ],
    )
]  # type: List[Any]
