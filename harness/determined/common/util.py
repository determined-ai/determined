import functools
import io
import os
import pathlib
import platform
import random
import sys
from typing import IO, Any, Callable, Iterator, Sequence, TypeVar, Union, overload

from determined.common import yaml

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


def debug_mode() -> bool:
    return os.getenv("DET_DEBUG", "").lower() in ("true", "1", "yes")


def preserve_random_state(fn: Callable) -> Callable:
    """A decorator to run a function with a fork of the random state."""

    @functools.wraps(fn)
    def wrapped(*arg: Any, **kwarg: Any) -> Any:
        state = random.getstate()
        try:
            return fn(*arg, **kwarg)
        finally:
            random.setstate(state)

    return wrapped


def safe_load_yaml_with_exceptions(yaml_file: Union[io.FileIO, IO[Any]]) -> Any:
    """Attempts to use ruamel.yaml.safe_load on the specified file. If successful, returns
    the output. If not, formats a ruamel.yaml Exception so that the user does not see a traceback
    of our internal APIs.

    ---------------------------------------------------------------------------------------------
    DuplicateKeyError Example:
    Input:
        Traceback (most recent call last):
        ...
        ruamel.yaml.constructor.DuplicateKeyError: while constructing a mapping
        in "<unicode string>", line 1, column 1:
            description: constrained_adaptiv ...
            ^ (line: 1)
        found duplicate key "checkpoint_storage" with value "{}" (original value: "{}")
        in "<unicode string>", line 7, column 1:
            checkpoint_storage:
            ^ (line: 7)
        To suppress this check see:
            http://yaml.readthedocs.io/en/latest/api.html#duplicate-keys
        Duplicate keys will become an error in future releases, and are errors
        by default when using the new API.
        Failed to create experiment
    Output:
        Error: invalid experiment config file constrained_adaptive.yaml.
        DuplicateKeyError: found duplicate key "learning_rate" with value "0.022"
        (original value: "0.025")
        in "constrained_adaptive.yaml", line 23, column 3
    ---------------------------------------------------------------------------------------------
    """
    try:
        config = yaml.safe_load(yaml_file)
    except (
        yaml.error.MarkedYAMLWarning,
        yaml.error.MarkedYAMLError,
        yaml.error.MarkedYAMLFutureWarning,
    ) as e:
        err_msg = (
            f"Error: invalid experiment config file {yaml_file.name}.\n"
            f"{e.__class__.__name__}: {e.problem}\n{e.problem_mark}"
        )
        print(err_msg)
        sys.exit(1)
    return config


def get_config_path() -> pathlib.Path:
    if os.environ.get("DET_DEBUG_CONFIG_PATH"):
        return pathlib.Path(os.environ["DET_DEBUG_CONFIG_PATH"])

    system = platform.system()
    if "Linux" in system and "XDG_CONFIG_HOME" in os.environ:
        config_path = pathlib.Path(os.environ["XDG_CONFIG_HOME"])
    elif "Darwin" in system:
        config_path = pathlib.Path.home().joinpath("Library").joinpath("Application Support")
    elif "Windows" in system and "LOCALAPPDATA" in os.environ:
        config_path = pathlib.Path(os.environ["LOCALAPPDATA"])
    else:
        config_path = pathlib.Path.home().joinpath(".config")

    return config_path.joinpath("determined")
