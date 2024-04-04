import argparse
import functools
import json
import pathlib
from typing import Any, Dict, List, Union, cast

import termcolor

from determined import cli
from determined.cli import ntsc, render
from determined.common import api, context, util
from determined.common.api import bindings


def render_tasks(args: argparse.Namespace, tasks: Dict[str, bindings.v1AllocationSummary]) -> None:
    """Render tasks for JSON, tabulate or csv output.

    The tasks parameter requires a map from allocation IDs to bindings.v1AllocationSummary
    describing individual tasks.
    """

    def agent_info(t: bindings.v1AllocationSummary) -> Union[str, List[str]]:
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


def list_tasks(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    r = bindings.get_GetTasks(sess)
    tasks = r.allocationIdToSummary or {}
    render_tasks(args, tasks)


def logs(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    task_id = cast(str, ntsc.expand_uuid_prefixes(sess, args, args.task_id))
    try:
        logs = api.task_logs(
            sess,
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
            termcolor.colored(
                "Task log stream ended. To reopen log stream, run: "
                "det task logs -f {}".format(task_id),
                "green",
            )
        )


def kill(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    req = bindings.v1KillGenericTaskRequest(taskId=args.task_id, killFromRoot=args.root)
    bindings.post_KillGenericTask(sess, taskId=args.task_id, body=req)
    print(f"Sucessfully killed task: {args.task_id}")


def task_creation_output(
    session: api.Session, task_resp: bindings.v1CreateGenericTaskResponse, follow: bool
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
                termcolor.colored(
                    "Task log stream ended. To reopen log stream, run: "
                    "det task logs -f {}".format(task_resp.taskId),
                    "green",
                )
            )


def create(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    config = ntsc.parse_config(args.config_file, None, args.config, [])
    config_text = util.yaml_safe_dump(config)
    context_directory = context.read_v1_context(args.context, args.include)

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


def config(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    config_resp = bindings.get_GetGenericTaskConfig(sess, taskId=args.task_id)
    if args.json:
        render.print_json(config_resp.config)
    else:
        yaml_dict = json.loads(config_resp.config)
        print(util.yaml_safe_dump(yaml_dict, default_flow_style=False))


def fork(args: argparse.Namespace) -> None:
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


def pause(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    bindings.post_PauseGenericTask(sess, taskId=args.task_id)
    print(f"Paused task: {args.task_id}")


def unpause(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    bindings.post_UnpauseGenericTask(sess, taskId=args.task_id)
    print(f"Unpaused task: {args.task_id}")


def cleanup_logs(args: argparse.Namespace) -> None:
    response = bindings.post_CleanupLogs(cli.setup_session(args))
    print(f"Deleted {response.removedCount} rows of log entries.")


common_log_options: List[Any] = [
    cli.Arg(
        "-f",
        "--follow",
        action="store_true",
        help="follow the logs of a running task, similar to tail -f",
    ),
    cli.Group(
        cli.output_format_args["json"],
    ),
    cli.Group(
        cli.Arg(
            "--head",
            type=int,
            help="number of lines to show, counting from the beginning of the log",
        ),
        cli.Arg(
            "--tail",
            type=int,
            help="number of lines to show, counting from the end of the log",
        ),
    ),
    cli.Arg(
        "--allocation-id",
        dest="allocation_ids",
        action="append",
        help="allocations to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--agent-id",
        dest="agent_ids",
        action="append",
        help="agents to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--container-id",
        dest="container_ids",
        action="append",
        help="containers to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--rank-id",
        dest="rank_ids",
        type=int,
        action="append",
        help="containers to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--timestamp-before",
        help="show logs only from before (RFC 3339 format), e.g. '2021-10-26T23:17:12Z'",
    ),
    cli.Arg(
        "--timestamp-after",
        help="show logs only from after (RFC 3339 format), e.g. '2021-10-26T23:17:12Z'",
    ),
    cli.Arg(
        "--level",
        dest="level",
        help="show logs with this level or higher "
        + "(TRACE, DEBUG, INFO, WARNING, ERROR, CRITICAL)",
        choices=["TRACE", "DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"],
    ),
    cli.Arg(
        "--source",
        dest="sources",
        action="append",
        help="sources to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--stdtype",
        dest="stdtypes",
        action="append",
        help="output stream to show logs from (repeat for multiple values)",
    ),
]


args_description: List[Any] = [
    cli.Cmd(
        "task",
        None,
        "manage tasks (commands, experiments, notebooks, shells, tensorboards)",
        [
            cli.Cmd(
                "list ls",
                list_tasks,
                "list tasks in cluster",
                [
                    cli.Group(
                        cli.output_format_args["csv"],
                        cli.output_format_args["json"],
                    )
                ],
                is_default=True,
            ),
            cli.Cmd(
                "logs",
                # Since declarative argparse tries to attach the help_str to the func itself:
                # ./harness/determined/cli/_declarative_argparse.py#L57
                # Each func must be unique.
                functools.partial(logs),
                "fetch task logs",
                [
                    cli.Arg("task_id", help="task ID"),
                    *common_log_options,
                ],
            ),
            cli.Cmd("cleanup-logs", cleanup_logs, "cleanup expired task logs", []),
            cli.Cmd(
                "create",
                create,
                argparse.SUPPRESS,
                [
                    cli.Arg(
                        "config_file", type=argparse.FileType("r"), help="task config file (.yaml)"
                    ),
                    cli.Arg(
                        "--context",
                        "-c",
                        type=pathlib.Path,
                        help=ntsc.CONTEXT_DESC,
                    ),
                    cli.Arg(
                        "-i",
                        "--include",
                        action="append",
                        default=[],
                        type=pathlib.Path,
                        help=ntsc.INCLUDE_DESC,
                    ),
                    cli.Arg("--project_id", type=int, help="place this task inside this project"),
                    cli.Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                    cli.Arg(
                        "-f",
                        "--follow",
                        action="store_true",
                        help="follow the logs of the task that is created",
                    ),
                    cli.Arg("--fork", type=str, help="id of parent task to fork from"),
                    cli.Arg(
                        "-p",
                        "--parent",
                        type=str,
                        help="task id of parent task",
                    ),
                    cli.Arg(
                        "--inherit_context",
                        action="store_true",
                        help="inherits context directory from parent task (parent flag required)",
                    ),
                    cli.Arg(
                        "--no_pause",
                        action="store_true",
                        help="make task unpausable",
                    ),
                ],
            ),
            cli.Cmd(
                "config",
                config,
                argparse.SUPPRESS,
                [
                    cli.Arg("task_id", type=str, help="ID of task to pull config from"),
                    cli.Arg(
                        "--json",
                        action="store_true",
                        help="return config in JSON format",
                    ),
                ],
            ),
            cli.Cmd(
                "fork",
                fork,
                argparse.SUPPRESS,
                [
                    cli.Arg("parent_task_id", type=str, help="Id of parent task to fork from"),
                    cli.Arg(
                        "-f",
                        "--follow",
                        action="store_true",
                        help="follow the logs of the task that is created",
                    ),
                    cli.Arg("--project_id", type=int, help="place this task inside this project"),
                ],
            ),
            cli.Cmd(
                "kill",
                kill,
                argparse.SUPPRESS,
                [
                    cli.Arg("task_id", type=str, help=""),
                    cli.Arg(
                        "--root",
                        action="store_true",
                        help="",
                    ),
                ],
            ),
            cli.Cmd(
                "pause",
                pause,
                argparse.SUPPRESS,
                [
                    cli.Arg("task_id", type=str, help=""),
                ],
            ),
            cli.Cmd(
                "unpause",
                unpause,
                argparse.SUPPRESS,
                [
                    cli.Arg("task_id", type=str, help=""),
                ],
            ),
        ],
    ),
]
