import json
from argparse import ONE_OR_MORE, Namespace
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
from determined.common.experimental import Session


@authentication.required
def ls(args: Namespace) -> None:
    config = api.get(args.master, "config").json()
    is_priority = check_is_priority(config, args.resource_pool)
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

        def computed_job_name(job: bindings.v1Job) -> str:
            if job.type == bindings.determinedjobv1Type.TYPE_EXPERIMENT:
                return f"{job.name} ({job.entityId})"
            else:
                return job.name

        values = [
            [
                j.summary.jobsAhead
                if j.summary is not None and j.summary.jobsAhead > -1
                else "N/A",
                j.jobId,
                j.type.value,
                computed_job_name(j),
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
    update = bindings.v1QueueControl(
        jobId=args.job_id,
        priority=args.priority,
        weight=args.weight,
        resourcePool=args.resource_pool,
        behindOf=args.behind_of,
        aheadOf=args.ahead_of,
    )
    bindings.post_UpdateJobQueue(
        setup_session(args), body=bindings.v1UpdateJobQueueRequest([update])
    )


@authentication.required
def process_updates(args: Namespace) -> None:
    session = setup_session(args)
    for arg in args.operation:
        inputs = validate_operation_args(arg)
        _single_update(session=session, **inputs)


def _single_update(
    job_id: str,
    session: Session,
    priority: str = None,
    weight: str = None,
    resource_pool: str = None,
    behind_of: str = None,
    ahead_of: str = None,
) -> None:
    update = bindings.v1QueueControl(
        jobId=job_id,
        priority=priority,
        weight=int(weight) if weight else None,
        resourcePool=resource_pool,
        behindOf=behind_of,
        aheadOf=ahead_of,
    )
    bindings.post_UpdateJobQueue(session, body=bindings.v1UpdateJobQueueRequest([update]))
    return


def is_priority_rm(config: dict) -> bool:
    try:
        if config["resource_manager"]["scheduler"]["type"] != "priority":
            return False
    except KeyError:
        pass
    return True


def check_is_priority(config: dict, resource_pool: str) -> bool:
    try:
        for pool in config["resource_pools"]:
            if pool["pool_name"] == resource_pool and pool["scheduler"]["type"] != "priority":
                return False
        return is_priority_rm(config)

    except (KeyError, TypeError):
        return is_priority_rm(config)
    return True


def validate_operation_args(operation: str) -> dict:
    valid_cmds = ("priority", "weight", "resource_pool", "ahead_of", "behind_of")
    replacements = {
        "resource-pool": "resource_pool",
        "ahead-of": "ahead_of",
        "behind-of": "behind_of",
    }
    args = {}
    values = operation.split(".")
    if len(values) != 2:
        raise ValueError(
            f"Job {values[0]} and its operation have an invalid format. "
            f"Please ensure the update is formatted as <jobID>.<operation>=<value>."
        )
    args["job_id"] = values[0]
    operation = values[1].split("=")
    if len(operation) != 2:
        raise ValueError(
            f"The operation for job {values[0]} has invalid format. "
            f"Please ensure the operation is formatted as <operation>=<value>."
        )

    if operation[0] not in valid_cmds:
        raise ValueError(
            f"Invalid operation {operation[0]} specified for job {values[0]}. "
            f"Supported commands include: {valid_cmds}."
        )

    args[replacements.get(operation[0], operation[0])] = operation[1]

    return args


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
                        "-p",
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
                    Arg("job_id", type=str, help="The target job ID"),
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
                        Arg(
                            "--resource-pool",
                            type=str,
                            help="The target resource pool to move the job to.",
                        ),
                        Arg(
                            "--ahead-of",
                            type=str,
                            help="The job ID of the job to be put ahead of in the queue.",
                        ),
                        Arg(
                            "--behind-of",
                            type=str,
                            help="The job ID of the job to be put behind in the queue.",
                        ),
                    ),
                ],
            ),
            Cmd(
                "update-batch",
                process_updates,
                "batch update jobs",
                [
                    Arg(
                        "operation",
                        nargs=ONE_OR_MORE,
                        type=str,
                        help="The target job ID(s) and target operation(s), formatted as "
                        "<jobID>.<operation>=<value>. Operations include priority, weight, "
                        "resource-pool, ahead-of, and behind-of.",
                    )
                ],
            ),
        ],
    ),
]  # type: List[Any]
