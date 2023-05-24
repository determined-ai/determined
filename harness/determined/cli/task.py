from argparse import Namespace
from functools import partial
from typing import Any, Dict, List, Union, cast

from termcolor import colored

from determined import cli
from determined.cli import command, render
from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.api.bindings import v1AllocationSummary
from determined.common.declarative_argparse import Arg, Cmd, Group


def render_tasks(args: Namespace, tasks: Dict[str, v1AllocationSummary]) -> None:
    """Render tasks for JSON, tabulate or csv output.

    The tasks parameter requires a map from allocation IDs to v1AllocationSummary
    describing individual tasks.
    """

    def agent_info(t: v1AllocationSummary) -> Union[str, List[str]]:
        if t.resources is None:
            return "unassigned"
        agents = [a for r in t.resources for a in (r.agentDevices or {})]
        if len(agents) == 1:
            agent = agents[0]  # type: str
            return agent
        return agents

    if args.json:
        render.print_json({a: t.to_json() for (a, t) in tasks.items()})
        return

    headers = [
        "Task ID",
        "Allocation ID",
        "Name",
        "Slots Needed",
        "Registered Time",
        "Agent",
        "Priority",
        "Resource Pool",
        "Ports",
    ]
    values = [
        [
            task.taskId,
            task.allocationId,
            task.name,
            task.slotsNeeded,
            render.format_time(task.registeredTime),
            agent_info(task),
            task.priority if task.schedulerType == "priority" else "N/A",
            task.resourcePool,
            ",".join(
                map(
                    str,
                    sorted([pp.port for pp in (task.proxyPorts or []) if pp.port is not None]),
                )
            ),
        ]
        for task_id, task in sorted(
            tasks.items(),
            key=lambda tup: (render.format_time(tup[1].registeredTime),),
        )
    ]

    render.tabulate_or_csv(headers, values, args.csv)


@authentication.required
def list_tasks(args: Namespace) -> None:
    r = bindings.get_GetTasks(cli.setup_session(args))
    tasks = r.allocationIdToSummary or {}
    render_tasks(args, tasks)


@authentication.required
def logs(args: Namespace) -> None:
    task_id = cast(str, command.expand_uuid_prefixes(args, args.task_id))
    try:
        logs = api.task_logs(
            cli.setup_session(args),
            task_id,
            head=args.head,
            tail=args.tail,
            follow=args.follow,
            agent_ids=args.agent_ids,
            container_ids=args.container_ids,
            rank_ids=args.rank_ids,
            sources=args.sources,
            stdtypes=args.stdtypes,
            min_level=args.level,
            timestamp_before=args.timestamp_before,
            timestamp_after=args.timestamp_after,
        )
        if "json" in args and args.json:
            for log in logs:
                render.print_json(log.to_json())
        else:
            api.pprint_logs(logs)
    finally:
        print(
            colored(
                "Task log stream ended. To reopen log stream, run: "
                "det task logs -f {}".format(task_id),
                "green",
            )
        )


common_log_options: List[Any] = [
    Arg(
        "-f",
        "--follow",
        action="store_true",
        help="follow the logs of a running task, similar to tail -f",
    ),
    Group(
        cli.output_format_args["json"],
    ),
    Group(
        Arg(
            "--head",
            type=int,
            help="number of lines to show, counting from the beginning of the log",
        ),
        Arg(
            "--tail",
            type=int,
            help="number of lines to show, counting from the end of the log",
        ),
    ),
    Arg(
        "--allocation-id",
        dest="allocation_ids",
        action="append",
        help="allocations to show logs from (repeat for multiple values)",
    ),
    Arg(
        "--agent-id",
        dest="agent_ids",
        action="append",
        help="agents to show logs from (repeat for multiple values)",
    ),
    Arg(
        "--container-id",
        dest="container_ids",
        action="append",
        help="containers to show logs from (repeat for multiple values)",
    ),
    Arg(
        "--rank-id",
        dest="rank_ids",
        type=int,
        action="append",
        help="containers to show logs from (repeat for multiple values)",
    ),
    Arg(
        "--timestamp-before",
        help="show logs only from before (RFC 3339 format), e.g. '2021-10-26T23:17:12Z'",
    ),
    Arg(
        "--timestamp-after",
        help="show logs only from after (RFC 3339 format), e.g. '2021-10-26T23:17:12Z'",
    ),
    Arg(
        "--level",
        dest="level",
        help="show logs with this level or higher "
        + "(TRACE, DEBUG, INFO, WARNING, ERROR, CRITICAL)",
        choices=["TRACE", "DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"],
    ),
    Arg(
        "--source",
        dest="sources",
        action="append",
        help="sources to show logs from (repeat for multiple values)",
    ),
    Arg(
        "--stdtype",
        dest="stdtypes",
        action="append",
        help="output stream to show logs from (repeat for multiple values)",
    ),
]


args_description: List[Any] = [
    Cmd(
        "task",
        None,
        "manage tasks (commands, experiments, notebooks, shells, tensorboards)",
        [
            Cmd(
                "list ls",
                list_tasks,
                "list tasks in cluster",
                [
                    Group(
                        cli.output_format_args["csv"],
                        cli.output_format_args["json"],
                    )
                ],
                is_default=True,
            ),
            Cmd(
                "logs",
                # Since declarative argparse tries to attach the help_str to the func itself:
                # ./harness/determined/common/declarative_argparse.py#L57
                # Each func must be unique.
                partial(logs),
                "fetch task logs",
                [
                    Arg("task_id", help="task ID"),
                    *common_log_options,
                ],
            ),
        ],
    ),
]
