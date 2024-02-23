import enum
import os
from typing import Callable, Iterator, Optional, Tuple, TypeVar, Union
from urllib import parse

from determined.common import api, util
from determined.common.api import bindings


class PageOpts(str, enum.Enum):
    single = "1"
    all = "all"


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


def canonicalize_master_url(url: str) -> str:
    """
    Read a user-provided master url and convert it to a canonical master url.

    It is expected that user inputs are canonicalized once right when the user passes them in, and
    that the master_url remains unchanged throughout the internals of the system.

    A canonical master has the following properties:
      - explicit scheme
      - nonempty host
      - explicit port
      - path does not end in a '/', if it is present at all
      - no username, password, query, or fragment
      - a full url can be trivially formed a la f"{master_url}/path/to/resource"

    In addition to validation, canonicalization is important for the authentication cache, because
    it helps to prevent situations where a use creates multiple sessions for a single master
    instance.  It's not bulletproof though, if they do things like connect to the master as both
    localhost and as 127.0.0.1; we can't help those cases without an inappropriate amount of
    guesswork.
    """

    # We need to prepend a scheme first, because urlparse() doesn't handle that case well.
    if url.startswith("https://"):
        default_port = 443
    elif url.startswith("http://"):
        default_port = 80
    else:
        url = f"http://{url}"
        default_port = 8080

    parsed = parse.urlparse(url)

    if not parsed.hostname:
        raise ValueError(f"invalid master url {url}; master url must contain a nonempty hostname")

    if parsed.username or parsed.password or parsed.query or parsed.fragment:
        raise ValueError(
            f"invalid master url {url}; master url must not contain username, password, query, or "
            "fragment"
        )

    port = parsed.port or default_port
    netloc = f"{parsed.hostname}:{port}"
    return parse.urlunparse((parsed.scheme, netloc, parsed.path, "", "", "")).rstrip("/")


def get_default_master_url() -> str:
    """
    Read supported environment variables for a master address, or pick localhost:8080.

    Note that the result is not canonicalized; that is ok because there's no usage pattern where
    you wouldn't be taking a user-provided value or this value, and you'd need to call
    canonicalize_master_url() afterwards anyway.

    Example:

        master_url = user_requested_master or get_default_master_url()
        master_url = canonicalize_master_url(master_url)
    """
    return os.environ.get("DET_MASTER", os.environ.get("DET_MASTER_ADDR", "localhost:8080"))


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


def wait_for_task_ready(
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
