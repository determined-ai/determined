import json
from argparse import Namespace
from typing import Any, Dict, List, Union, cast

from determined.cli import command, render
from determined.common import api
from determined.common.api import authentication
from determined.common.declarative_argparse import Arg, Cmd, Group


def render_tasks(args: Namespace, tasks: Dict[str, Dict[str, Any]]) -> None:
    def agent_info(t: Dict[str, Any]) -> Union[str, List[str]]:
        resources = t.get("resources", [])
        if not resources:
            return "unassigned"
        agents = [a for r in resources for a in r["agent_devices"]]
        if len(agents) == 1:
            agent = agents[0]  # type: str
            return agent
        return agents

    if args.json:
        print(json.dumps(tasks, indent=4))
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
    ]
    values = [
        [
            task["task_id"],
            task["allocation_id"],
            task["name"],
            task["slots_needed"],
            render.format_time(task["registered_time"]),
            agent_info(task),
            task["priority"] if task["scheduler_type"] == "priority" else "N/A",
            task["resource_pool"],
        ]
        for task_id, task in sorted(
            tasks.items(),
            key=lambda tup: (render.format_time(tup[1]["registered_time"]),),
        )
    ]

    render.tabulate_or_csv(headers, values, args.csv)


@authentication.required
def list_tasks(args: Namespace) -> None:
    r = api.get(args.master, "tasks")
    tasks = r.json()
    render_tasks(args, tasks)


@authentication.required
def logs(args: Namespace) -> None:
    task_id = cast(str, command.expand_uuid_prefixes(args, args.task_id))
    api.pprint_task_logs(
        args.master,
        task_id,
        head=args.head,
        tail=args.tail,
        follow=args.follow,
        agent_ids=args.agent_ids,
        container_ids=args.container_ids,
        rank_ids=args.rank_ids,
        sources=args.sources,
        stdtypes=args.stdtypes,
        level_above=args.level,
        timestamp_before=args.timestamp_before,
        timestamp_after=args.timestamp_after,
    )


common_log_options = [
    Arg(
        "-f",
        "--follow",
        action="store_true",
        help="follow the logs of a running task, similar to tail -f",
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
                "list",
                list_tasks,
                "list tasks in cluster",
                [
                    Group(
                        Arg("--csv", action="store_true", help="print as CSV"),
                        Arg("--json", action="store_true", help="print as JSON"),
                    )
                ],
                is_default=True,
            ),
            Cmd(
                "logs",
                # Since declarative argparse tries to attach the help_str to the func itself:
                # ./harness/determined/common/declarative_argparse.py#L57
                # Each func must be unique.
                lambda *args, **kwargs: logs(*args, **kwargs),
                "fetch task logs",
                [
                    Arg("task_id", help="task ID"),
                    *common_log_options,
                ],
            ),
        ],
    ),
]
