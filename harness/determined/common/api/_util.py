import enum
from typing import Callable, Iterator, TypeVar, Union

from determined.common.api import Session, bindings
from determined.common import util


class PageOpts(str, enum.Enum):
    single = "1"
    all = "all"


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


# Literal["notebook", "tensorboard", "shell", "command"]
class NTSC_Kind(enum.Enum):
    notebook = "notebook"
    tensorboard = "tensorboard"
    shell = "shell"
    command = "command"


AnyNTSC = Union[bindings.v1Notebook, bindings.v1Tensorboard, bindings.v1Shell, bindings.v1Command]


def get_ntsc_details(session: Session, typ: NTSC_Kind, ntsc_id: str) -> AnyNTSC:
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
    session: Session,
    typ: NTSC_Kind,
    ntsc_id: str,
    predicate: Callable[[bindings.taskv1State], bool],
    timeout: int = 10,  # seconds
) -> bindings.taskv1State:
    """wait for ntsc to reach a state that satisfies the predicate"""

    def get_state():
        last_state = get_ntsc_details(session, typ, ntsc_id).state
        return predicate(last_state), last_state

    return util.wait_for(get_state, timeout)
