import os
from typing import Iterator, Sequence, TypeVar, Union, overload

T = TypeVar("T")


@overload
def chunks(lst: str, chunk_size: int) -> Iterator[str]:
    ...


@overload
def chunks(lst: Sequence[T], chunk_size: int) -> Iterator[Sequence[T]]:
    ...


def chunks(
    lst: Union[str, Sequence[T]], chunk_size: int
) -> Union[Iterator[str], Iterator[Sequence[T]]]:
    """
    Collect data into fixed-length chunks or blocks.  Adapted from the
    itertools documentation recipes.

    e.g. chunks('ABCDEFG', 3) --> ABC DEF G
    """
    for i in range(0, len(lst), chunk_size):
        yield lst[i : i + chunk_size]


def sizeof_fmt(val: Union[int, float]) -> str:
    val = float(val)
    for unit in ("", "K", "M", "G", "T", "P", "E", "Z"):
        if abs(val) < 1024.0:
            return "%3.1f%sB" % (val, unit)
        val /= 1024.0
    return "%.1f%sB" % (val, "Y")


def get_default_master_address() -> str:
    return os.environ.get("DET_MASTER", os.environ.get("DET_MASTER_ADDR", "localhost:8080"))
