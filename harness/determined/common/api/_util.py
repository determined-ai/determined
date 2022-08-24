import enum
from typing import Callable, Iterator, TypeVar, Union

from determined.common.api import bindings


class PageOpts(str, enum.Enum):
    single = "1"
    all = "all"


# All paginated response types which want read_paginated() to work with.
# A Paginated union member must have a .pagination attribute.
# hint: grep 'pagination: .*v1Pagination.*' bindings.py -B 7 | sed -E -n -e 's/class (.*):/\1/p'
Paginated = Union[
    bindings.v1GetAgentsResponse,
    bindings.v1GetCommandsResponse,
    bindings.v1GetExperimentCheckpointsResponse,
    bindings.v1GetExperimentTrialsResponse,
    bindings.v1GetExperimentsResponse,
    bindings.v1GetJobsResponse,
    bindings.v1GetModelVersionsResponse,
    bindings.v1GetModelsResponse,
    bindings.v1GetNotebooksResponse,
    bindings.v1GetResourcePoolsResponse,
    bindings.v1GetShellsResponse,
    bindings.v1GetTemplatesResponse,
    bindings.v1GetTensorboardsResponse,
    bindings.v1GetTrialCheckpointsResponse,
    bindings.v1GetTrialWorkloadsResponse,
    bindings.v1GetUsersResponse,
    bindings.v1GetWorkspaceProjectsResponse,
    bindings.v1GetWorkspacesResponse,
]

T = TypeVar("T", bound=Paginated)


def read_paginated(
    get_with_offset: Callable[[int], T],
    offset: int = 0,
    pages: PageOpts = PageOpts.all,
) -> Iterator[T]:
    while True:
        resp = get_with_offset(offset)
        pagination = resp.pagination
        assert pagination
        yield resp
        if pagination.endIndex == pagination.total or pages == PageOpts.single:
            break
        assert pagination.endIndex is not None
        offset = pagination.endIndex
