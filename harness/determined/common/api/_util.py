import enum
from typing import Callable, Iterator, TypeVar

from determined.common.api import bindings


class PageOpts(str, enum.Enum):
    single = "1"
    all = "all"


# Not that read_paginated requires the output of get_with_offset to be a Paginated type to work.
# The Paginated union type is generated based on response objects with a .pagination attribute.
T = TypeVar("T", bound=bindings.Paginated)


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
