import json
from argparse import Namespace
from typing import Any, List

import yaml

from determined.cli import render
from determined.cli.session import setup_session
from determined.common.api import authentication
from determined.common.api.b import get_GetJobs
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def ls(args: Namespace) -> None:
    response = get_GetJobs(
        setup_session(args),
        resourcePool=args.resource_pool,
        pagination_limit=args.limit,
        pagination_offset=args.offset,
        orderBy="ORDER_BY_DESC" if not args.reverse else "ORDER_BY_ASC",
    )
    if args.yaml:
        print(yaml.safe_dump(response.to_json(), default_flow_style=False))
    elif args.json:
        print(json.dumps(response.to_json(), indent=4, default=str))
    elif args.table or args.csv:
        headers = [
            "Jobs Ahead",
            "ID",
            "Entity ID",
            "Status",
            "Type",
            "Slots Acquired",
            "Slots Requested",
            "Name",
            "User",
            "Submission Time",
        ]
        values = [
            [
                j.summary.jobsAhead
                if j.summary is not None and j.summary.jobsAhead > -1
                else "N/A",
                j.jobId,
                j.entityId,
                j.summary.state if j.summary is not None else "N/A",
                j.type,
                j.allocatedSlots,
                j.requestedSlots,
                j.name,
                j.username,
                j.submissionTime,
            ]
            for j in response.jobs
        ]
        render.tabulate_or_csv(headers, values, as_csv=args.csv)
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
        "--reverse",
        default=False,
        action="store_true",
        help="Reverse the requested order of results.",
    ),
]


output_format = Group(
    Arg("--csv", action="store_true", help="Print as CSV format."),
    Arg("--json", action="store_true", help="Print as JSON format."),
    Arg("--yaml", action="store_true", help="Print in YAML fromat."),
    Arg("--table", action="store_true", default=True, help="Print in a tabular format."),
)

args_description = [
    Cmd(
        "j|ob",
        None,
        "manage job",
        [
            Cmd(
                "list ls",
                ls,
                "list jobs",
                [
                    Arg(
                        "-rp", "--resource-pool", type=str, help="The target resource pool, if any."
                    ),
                    *pagination_args,
                    output_format,
                ],
            ),
        ],
    )
]  # type: List[Any]
