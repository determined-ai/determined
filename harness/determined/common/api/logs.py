import collections
import json
from typing import Any, Dict, List, Optional, Tuple
from urllib.parse import urlencode

from termcolor import colored

from determined.common import api


def pprint_task_logs(master_url: str, task_id: str, **kwargs: Any) -> None:
    try:
        for log in task_logs(master_url, task_id, **kwargs):
            print(log["message"], end="")
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


def pprint_trial_logs(master_url: str, trial_id: int, **kwargs: Any) -> None:
    try:
        for log in trial_logs(master_url, trial_id, **kwargs):
            print(log["message"], end="")
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
    master_url: str,
    trial_id: int,
    head: Optional[int] = None,
    tail: Optional[int] = None,
    follow: bool = False,
    agent_ids: Optional[List[str]] = None,
    container_ids: Optional[List[str]] = None,
    rank_ids: Optional[List[str]] = None,
    sources: Optional[List[str]] = None,
    stdtypes: Optional[List[str]] = None,
    level_above: Optional[str] = None,
    timestamp_before: Optional[str] = None,
    timestamp_after: Optional[str] = None,
) -> collections.abc.Iterable:
    path = "/api/v1/trials/{}/logs?{}".format(
        trial_id,
        to_log_query_string(
            head,
            tail,
            follow=follow,
            level_above=level_above,
            extras=[
                ("agent_ids", agent_ids),
                ("container_ids", container_ids),
                ("rank_ids", rank_ids),
                ("sources", sources),
                ("stdtypes", stdtypes),
                ("timestamp_before", timestamp_before),
                ("timestamp_after", timestamp_after),
            ],
        ),
    )
    with api.get(master_url, path, stream=True) as r:
        line_iter = r.iter_lines()
        if tail is not None:
            line_iter = reversed(list(line_iter))
        for line in line_iter:
            yield json.loads(line)["result"]


def trial_log_fields(
    master_url: str,
    trial_id: int,
    follow: bool = False,
) -> collections.abc.Iterable:
    path = "/api/v1/trials/{}/logs/fields?{}".format(
        trial_id,
        to_log_query_string(
            None,
            None,
            follow=follow,
        ),
    )
    with api.get(master_url, path, stream=True) as r:
        for line in r.iter_lines():
            yield json.loads(line)["result"]


def task_logs(
    master_url: str,
    task_id: str,
    head: Optional[int] = None,
    tail: Optional[int] = None,
    follow: bool = False,
    allocation_ids: Optional[List[str]] = None,
    agent_ids: Optional[List[str]] = None,
    container_ids: Optional[List[str]] = None,
    rank_ids: Optional[List[str]] = None,
    sources: Optional[List[str]] = None,
    stdtypes: Optional[List[str]] = None,
    level_above: Optional[str] = None,
    timestamp_before: Optional[str] = None,
    timestamp_after: Optional[str] = None,
) -> collections.abc.Iterable:
    path = "/api/v1/tasks/{}/logs?{}".format(
        task_id,
        to_log_query_string(
            head,
            tail,
            follow=follow,
            level_above=level_above,
            extras=[
                ("allocation_ids", allocation_ids),
                ("agent_ids", agent_ids),
                ("container_ids", container_ids),
                ("rank_ids", rank_ids),
                ("sources", sources),
                ("stdtypes", stdtypes),
                ("timestamp_before", timestamp_before),
                ("timestamp_after", timestamp_after),
            ],
        ),
    )
    with api.get(master_url, path, stream=True) as r:
        line_iter = r.iter_lines()
        if tail is not None:
            line_iter = reversed(list(line_iter))
        for line in line_iter:
            yield json.loads(line)["result"]


def task_log_fields(
    master_url: str,
    task_id: str,
    follow: bool = False,
) -> collections.abc.Iterable:
    path = "/api/v1/tasks/{}/logs/fields?{}".format(
        task_id,
        to_log_query_string(
            None,
            None,
            follow=follow,
        ),
    )
    with api.get(master_url, path, stream=True) as r:
        for line in r.iter_lines():
            yield json.loads(line)["result"]


def to_log_query_string(
    head: Optional[int] = None,
    tail: Optional[int] = None,
    follow: bool = False,
    level_above: Optional[str] = None,
    extras: Optional[List[Tuple[str, Any]]] = None,
) -> str:
    query = {}  # type: Dict[str, Any]
    if head is not None:
        query["limit"] = head
    elif tail is not None:
        query["limit"] = tail
        query["order_by"] = "ORDER_BY_DESC"
    elif follow:
        query["follow"] = "true"

    if extras:
        for key, val in extras:
            if val is not None:
                query[key] = val

    if level_above is not None:
        query["levels"] = to_levels_above(level_above)

    return urlencode(query, doseq=True)


def to_levels_above(level: str) -> List[str]:
    # We should just be using the generated client instead and this is why.
    levels = [
        "LOG_LEVEL_TRACE",
        "LOG_LEVEL_DEBUG",
        "LOG_LEVEL_INFO",
        "LOG_LEVEL_WARNING",
        "LOG_LEVEL_ERROR",
        "LOG_LEVEL_CRITICAL",
    ]
    try:
        return levels[levels.index("LOG_LEVEL_" + level) :]
    except ValueError:
        raise Exception("invalid log level: {}".format(level))
