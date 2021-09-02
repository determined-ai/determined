import json
from argparse import Namespace
from typing import Any, Dict, List, Union

from determined.cli import render
from determined.common import api
from determined.common.api import authentication
from determined.common.declarative_argparse import Arg, Cmd, Group


def render_tasks(args: Namespace, tasks: Dict[str, Dict[str, Any]]) -> None:
    def agent_info(t: Dict[str, Any]) -> Union[str, List[str]]:
        containers = t.get("containers", [])
        if not containers:
            return "unassigned"
        if len(containers) == 1:
            agent = containers[0]["agent"]  # type: str
            return agent
        return [c["agent"] for c in containers]

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
        ],
    ),
]
