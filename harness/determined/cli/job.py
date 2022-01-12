import json
from argparse import Namespace
from datetime import datetime
from typing import Any, List

import pytz
import yaml

from determined.cli import render
from determined.cli.session import setup_session
from determined.cli.util import format_args, pagination_args
from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd, Group


@authentication.required
def ls(args: Namespace) -> None:
    is_priority = True
    config = api.get(args.master, "config").json()
    try:
        for pool in config["resource_pools"]:
            if (
                pool["pool_name"] == args.resource_pool
                and pool["scheduler"]["type"] == "fair_share"
            ):
                is_priority = False
    except (KeyError, TypeError):
        try:
            if config["resource_manager"]["scheduler"]["type"] == "fair_share":
                is_priority = False
        except KeyError:
            pass

    response = bindings.get_GetJobs(
        setup_session(args),
        resourcePool=args.resource_pool,
        pagination_limit=args.limit,
        pagination_offset=args.offset,
        orderBy=bindings.v1OrderBy.ORDER_BY_ASC
        if not args.reverse
        else bindings.v1OrderBy.ORDER_BY_DESC,
    )
    if args.yaml:
        print(yaml.safe_dump(response.to_json(), default_flow_style=False))
    elif args.json:
        print(json.dumps(response.to_json(), indent=4, default=str))
    else:
        headers = [
            "#",
            "ID",
            "Type",
            "Job Name",
            "Priority" if is_priority else "Weight",
            "Submitted",
            "Slots (acquired/needed)",
            "Status",
            "User",
        ]
        values = [
            [
                j.summary.jobsAhead
                if j.summary is not None and j.summary.jobsAhead > -1
                else "N/A",
                j.jobId,
                j.type.value,
                j.name,
                j.priority if is_priority else j.weight,
                pytz.utc.localize(
                    datetime.strptime(j.submissionTime.split(".")[0], "%Y-%m-%dT%H:%M:%S")
                ),
                f"{j.allocatedSlots}/{j.requestedSlots}",
                j.summary.state.value if j.summary is not None else "N/A",
                j.username,
            ]
            for j in response.jobs
        ]
        render.tabulate_or_csv(headers, values, as_csv=args.csv)


@authentication.required
def update(args: Namespace) -> None:
    update = bindings.v1QueueControl(jobId=args.job_id, priority=args.priority, weight=args.weight)
    bindings.post_UpdateJobQueue(
        setup_session(args), body=bindings.v1UpdateJobQueueRequest([update])
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
                        "--resource-pool",
                        type=str,
                        help="The target resource pool, if any.",
                    ),
                    *pagination_args,
                    Group(
                        format_args["json"],
                        format_args["yaml"],
                        format_args["table"],
                        format_args["csv"],
                    ),
                ],
            ),
            Cmd(
                "u|pdate",
                update,
                "update job",
                [
                    Arg(
                        "job_id",
                        type=str,
                        help="The target job ID",
                    ),
                    Group(
                        Arg(
                            "-p",
                            "--priority",
                            type=int,
                            help="The new priority. Exclusive to priority scheduler.",
                        ),
                        Arg(
                            "-w",
                            "--weight",
                            type=float,
                            help="The new weight. Exclusive to fair_share scheduler.",
                        ),
                    ),
                ],
            ),
        ],
    ),
]  # type: List[Any]
