import json
from argparse import SUPPRESS, FileType, Namespace
from functools import partial
from pathlib import Path
from typing import Any, Dict, List, Union, cast

from termcolor import colored

from determined import cli
from determined.cli import ntsc, render
from determined.common import api, context, util
from determined.common.api import authentication, bindings
from determined.common.api.bindings import v1AllocationSummary, v1CreateGenericTaskResponse
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
    task_id = cast(str, ntsc.expand_uuid_prefixes(args, args.task_id))
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


@authentication.required
def kill(args: Namespace) -> None:
    sess = cli.setup_session(args)
    req = bindings.v1KillGenericTaskRequest(taskId=args.task_id, killFromRoot=args.root)
    bindings.post_KillGenericTask(sess, taskId=args.task_id, body=req)
    print(f"Sucessfully killed task: {args.task_id}")


def task_creation_output(
    session: api.Session, task_resp: v1CreateGenericTaskResponse, follow: bool
) -> None:
    print(f"Created task {task_resp.taskId}")

    if task_resp.warnings:
        cli.print_launch_warnings(task_resp.warnings)

    if follow:
        try:
            logs = api.task_logs(session, task_resp.taskId, follow=True)
            api.pprint_logs(logs)
        finally:
            print(
                colored(
                    "Task log stream ended. To reopen log stream, run: "
                    "det task logs -f {}".format(task_resp.taskId),
                    "green",
                )
            )


@authentication.required
def create(args: Namespace) -> None:
    config = ntsc.parse_config(args.config_file, None, args.config, [])
    config_text = util.yaml_safe_dump(config)
    context_directory = context.read_v1_context(args.context, args.include)

    sess = cli.setup_session(args)
    req = bindings.v1CreateGenericTaskRequest(
        config=config_text,
        contextDirectory=context_directory,
        projectId=args.project_id,
        forkedFrom=args.fork,
        parentId=args.parent,
        inheritContext=args.inherit_context,
        noPause=args.no_pause,
    )
    task_resp = bindings.post_CreateGenericTask(sess, body=req)
    task_creation_output(session=sess, task_resp=task_resp, follow=args.follow)


@authentication.required
def config(args: Namespace) -> None:
    sess = cli.setup_session(args)
    config_resp = bindings.get_GetGenericTaskConfig(sess, taskId=args.task_id)
    if args.json:
        render.print_json(config_resp.config)
    else:
        yaml_dict = json.loads(config_resp.config)
        print(util.yaml_safe_dump(yaml_dict, default_flow_style=False))


@authentication.required
def fork(args: Namespace) -> None:
    sess = cli.setup_session(args)
    req = bindings.v1CreateGenericTaskRequest(
        config="",
        contextDirectory=[],
        projectId=args.project_id,
        forkedFrom=args.parent_task_id,
        inheritContext=False,
    )
    task_resp = bindings.post_CreateGenericTask(sess, body=req)
    task_creation_output(session=sess, task_resp=task_resp, follow=args.follow)


@authentication.required
def pause(args: Namespace) -> None:
    sess = cli.setup_session(args)
    bindings.post_PauseGenericTask(sess, taskId=args.task_id)
    print(f"Paused task: {args.task_id}")


@authentication.required
def unpause(args: Namespace) -> None:
    sess = cli.setup_session(args)
    bindings.post_UnpauseGenericTask(sess, taskId=args.task_id)
    print(f"Unpaused task: {args.task_id}")


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
            Cmd(
                "create",
                create,
                SUPPRESS,
                [
                    Arg("config_file", type=FileType("r"), help="task config file (.yaml)"),
                    Arg(
                        "--context",
                        "-c",
                        type=Path,
                        help=ntsc.CONTEXT_DESC,
                    ),
                    Arg(
                        "-i",
                        "--include",
                        action="append",
                        default=[],
                        type=Path,
                        help=ntsc.INCLUDE_DESC,
                    ),
                    Arg("--project_id", type=int, help="place this task inside this project"),
                    Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                    Arg(
                        "-f",
                        "--follow",
                        action="store_true",
                        help="follow the logs of the task that is created",
                    ),
                    Arg("--fork", type=str, help="id of parent task to fork from"),
                    Arg(
                        "-p",
                        "--parent",
                        type=str,
                        help="task id of parent task",
                    ),
                    Arg(
                        "--inherit_context",
                        action="store_true",
                        help="inherits context directory from parent task (parent flag required)",
                    ),
                    Arg(
                        "--no_pause",
                        action="store_true",
                        help="make task unpausable",
                    ),
                ],
            ),
            Cmd(
                "config",
                config,
                SUPPRESS,
                [
                    Arg("task_id", type=str, help="ID of task to pull config from"),
                    Arg(
                        "--json",
                        action="store_true",
                        help="return config in JSON format",
                    ),
                ],
            ),
            Cmd(
                "fork",
                fork,
                SUPPRESS,
                [
                    Arg("parent_task_id", type=str, help="Id of parent task to fork from"),
                    Arg(
                        "-f",
                        "--follow",
                        action="store_true",
                        help="follow the logs of the task that is created",
                    ),
                    Arg("--project_id", type=int, help="place this task inside this project"),
                ],
            ),
            Cmd(
                "kill",
                kill,
                SUPPRESS,
                [
                    Arg("task_id", type=str, help=""),
                    Arg(
                        "--root",
                        action="store_true",
                        help="",
                    ),
                ],
            ),
            Cmd(
                "pause",
                pause,
                SUPPRESS,
                [
                    Arg("task_id", type=str, help=""),
                ],
            ),
            Cmd(
                "unpause",
                unpause,
                SUPPRESS,
                [
                    Arg("task_id", type=str, help=""),
                ],
            ),
        ],
    ),
]
