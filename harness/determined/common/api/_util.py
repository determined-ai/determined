import enum
from typing import Callable, Iterator, Optional, Tuple, TypeVar, Union

import urllib3

from determined.common import api, util
from determined.common.api import bindings

# from determined.cli.render import Animator


class PageOpts(str, enum.Enum):
    single = "1"
    all = "all"


# HTTP status codes that will force request retries.
RETRY_STATUSES = [502, 503, 504]  # Bad Gateway, Service Unavailable, Gateway Timeout

# Default max number of times to retry a request.
MAX_RETRIES = 5

# Default seconds for an NTSC task to become ready before timeout.
DEFAULT_NTSC_TIMEOUT = 60 * 5


# Not that read_paginated requires the output of get_with_offset to be a Paginated type to work.
# The Paginated union type is generated based on response objects with a .pagination attribute.
T = TypeVar("T", bound=bindings.Paginated)

# Map of launch warnings to the warning message shown to users.
WARNING_MESSAGE_MAP = {
    bindings.v1LaunchWarning.CURRENT_SLOTS_EXCEEDED: (
        "Warning: The requested job requires more slots than currently available. "
        "You may need to increase cluster resources in order for the job to run."
    )
}


def read_paginated(
    get_with_offset: Callable[[int], T],
    offset: int = 0,
    pages: PageOpts = PageOpts.all,
) -> Iterator[T]:
    while True:
        resp = get_with_offset(offset)
        pagination = resp.pagination
        assert pagination is not None
        assert pagination.endIndex is not None
        assert pagination.total is not None
        yield resp
        if pagination.endIndex >= pagination.total or pages == PageOpts.single:
            break
        assert pagination.endIndex is not None
        offset = pagination.endIndex


def default_retry(max_retries: int = MAX_RETRIES) -> urllib3.util.retry.Retry:
    retry = urllib3.util.retry.Retry(
        total=max_retries,
        backoff_factor=0.5,  # {backoff factor} * (2 ** ({number of total retries} - 1))
        status_forcelist=RETRY_STATUSES,
    )
    return retry


# Literal["notebook", "tensorboard", "shell", "command"]
class NTSC_Kind(enum.Enum):
    notebook = "notebook"
    tensorboard = "tensorboard"
    shell = "shell"
    command = "command"


AnyNTSC = Union[bindings.v1Notebook, bindings.v1Tensorboard, bindings.v1Shell, bindings.v1Command]


def get_ntsc_details(session: api.Session, typ: NTSC_Kind, ntsc_id: str) -> AnyNTSC:
    if typ == NTSC_Kind.notebook:
        return bindings.get_GetNotebook(session, notebookId=ntsc_id).notebook
    elif typ == NTSC_Kind.tensorboard:
        return bindings.get_GetTensorboard(session, tensorboardId=ntsc_id).tensorboard
    elif typ == NTSC_Kind.shell:
        return bindings.get_GetShell(session, shellId=ntsc_id).shell
    elif typ == NTSC_Kind.command:
        return bindings.get_GetCommand(session, commandId=ntsc_id).command
    else:
        raise ValueError("unknown type")


def wait_for_ntsc_state(
    session: api.Session,
    typ: NTSC_Kind,
    ntsc_id: str,
    predicate: Callable[[bindings.taskv1State], bool],
    timeout: int = 10,  # seconds
) -> bindings.taskv1State:
    """wait for ntsc to reach a state that satisfies the predicate"""

    def get_state() -> Tuple[bool, bindings.taskv1State]:
        last_state = get_ntsc_details(session, typ, ntsc_id).state
        return predicate(last_state), last_state

    return util.wait_for(get_state, timeout)


def task_is_ready(
    session: api.Session,
    task_id: str,
    progress_report: Optional[Callable] = None,
    timeout: int = DEFAULT_NTSC_TIMEOUT,
) -> Optional[str]:
    """
    wait until a task is ready
    return: None if task is ready, otherwise return an error message
    """

    def _task_is_done_loading() -> Tuple[bool, Optional[str]]:
        task = bindings.get_GetTask(session, taskId=task_id).task
        if progress_report:
            progress_report()
        assert task is not None, "task must be present."
        if task.endTime is not None:
            return True, "task has been terminated."

        if len(task.allocations) == 0:
            return False, None
        is_ready = task.allocations[0].isReady
        if is_ready:
            return True, None

        return False, ""

    err_msg = util.wait_for(_task_is_done_loading, timeout=timeout, interval=1)
    return err_msg
