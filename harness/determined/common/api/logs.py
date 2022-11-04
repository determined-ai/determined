from typing import Iterable, List, Optional

from termcolor import colored

from determined.common import api
from determined.common.api import bindings


def pprint_task_logs(task_id: str, logs: Iterable[bindings.v1TaskLogsResponse]) -> None:
    try:
        for log in logs:
            print(log.message, end="")
    except KeyboardInterrupt:
        pass
    finally:
        print(
            colored(
                "Task log stream ended. To reopen log stream, run: "
                "det task logs -f {}".format(task_id),
                "green",
            )
        )


def pprint_trial_logs(trial_id: int, logs: Iterable[bindings.v1TrialLogsResponse]) -> None:
    try:
        for log in logs:
            print(log.message, end="")
    except KeyboardInterrupt:
        pass
    finally:
        print(
            colored(
                "Trial log stream ended. To reopen log stream, run: "
                "det trial logs -f {}".format(trial_id),
                "green",
            )
        )


def trial_logs(
    session: api.Session,
    trial_id: int,
    head: Optional[int] = None,
    tail: Optional[int] = None,
    follow: bool = False,
    agent_ids: Optional[List[str]] = None,
    container_ids: Optional[List[str]] = None,
    rank_ids: Optional[List[int]] = None,
    sources: Optional[List[str]] = None,
    stdtypes: Optional[List[str]] = None,
    min_level: Optional[bindings.v1LogLevel] = None,
    timestamp_before: Optional[str] = None,
    timestamp_after: Optional[str] = None,
) -> Iterable[bindings.v1TrialLogsResponse]:
    if sum((head is not None, tail is not None, follow)) > 1:
        raise ValueError("at most one of head, tail, or follow may be set")
    logs = bindings.get_TrialLogs(
        session,
        trialId=trial_id,
        agentIds=agent_ids,
        containerIds=container_ids,
        follow=follow,
        levels=levels_at_or_above(min_level),
        limit=head or tail,
        orderBy=tail is not None and bindings.v1OrderBy.ORDER_BY_DESC or None,
        rankIds=rank_ids,
        searchText=None,
        sources=sources,
        stdtypes=stdtypes,
        timestampBefore=timestamp_before,
        timestampAfter=timestamp_after,
    )
    yield from (logs if tail is None else reversed(list(logs)))


def task_logs(
    session: api.Session,
    task_id: str,
    head: Optional[int] = None,
    tail: Optional[int] = None,
    follow: bool = False,
    allocation_ids: Optional[List[str]] = None,
    agent_ids: Optional[List[str]] = None,
    container_ids: Optional[List[str]] = None,
    rank_ids: Optional[List[int]] = None,
    sources: Optional[List[str]] = None,
    stdtypes: Optional[List[str]] = None,
    min_level: Optional[bindings.v1LogLevel] = None,
    timestamp_before: Optional[str] = None,
    timestamp_after: Optional[str] = None,
) -> Iterable[bindings.v1TaskLogsResponse]:
    if sum((head is not None, tail is not None, follow)) > 1:
        raise ValueError("at most one of head, tail, or follow may be set")
    logs = bindings.get_TaskLogs(
        session,
        taskId=task_id,
        agentIds=agent_ids,
        allocationIds=allocation_ids,
        containerIds=container_ids,
        follow=follow,
        levels=levels_at_or_above(min_level),
        limit=head or tail,
        orderBy=tail is not None and bindings.v1OrderBy.ORDER_BY_DESC or None,
        rankIds=rank_ids,
        searchText=None,
        sources=sources,
        stdtypes=stdtypes,
        timestampBefore=timestamp_before,
        timestampAfter=timestamp_after,
    )
    yield from (logs if tail is None else reversed(list(logs)))


def levels_at_or_above(
    min_level: Optional[bindings.v1LogLevel],
) -> Optional[List[bindings.v1LogLevel]]:
    if min_level is None:
        return min_level
    # This is reliably ordered because Enum.__members__ is an OrderedDict.
    levels = list(bindings.v1LogLevel.__members__.values())
    return levels[levels.index(min_level) :]
