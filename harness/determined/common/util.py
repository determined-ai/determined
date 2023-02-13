import datetime
import functools
import io
import json
import os
import pathlib
import platform
import random
import sys
from typing import (
    IO,
    Any,
    Callable,
    Iterator,
    Optional,
    Sequence,
    TypeVar,
    Union,
    no_type_check,
    overload,
)

import urllib3

from determined.common import yaml

_yaml = yaml.YAML(typ="safe", pure=True)

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


def get_det_username_from_env() -> Optional[str]:
    return os.environ.get("DET_USER")


def get_det_user_token_from_env() -> Optional[str]:
    return os.environ.get("DET_USER_TOKEN")


def get_det_password_from_env() -> Optional[str]:
    return os.environ.get("DET_PASS")


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


def safe_load_yaml_with_exceptions(yaml_file: Union[io.FileIO, IO[Any], str]) -> Any:
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
        config = _yaml.load(yaml_file)
    except (
        yaml.error.MarkedYAMLWarning,
        yaml.error.MarkedYAMLError,
        yaml.error.MarkedYAMLFutureWarning,
    ) as e:
        err_msg = (
            f"Error: invalid experiment config file.\n"
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


def get_max_retries_config() -> urllib3.util.retry.Retry:
    # Allow overriding retry settings when necessary.
    # `DET_RETRY_CONFIG` env variable can contain `urllib3` `Retry` parameters,
    # encoded as JSON.
    # For example:
    #  - disable retries: {"total":0}
    #  - shorten the wait times {"total":10,"backoff_factor":0.5,"method_whitelist":false}

    config_data = os.environ.get("DET_RETRY_CONFIG")
    if config_data is not None:
        config = json.loads(config_data)
        return urllib3.util.retry.Retry(**config)

    # Default retry is different with different versions of urllib3, which mypy doesn't understand.
    @no_type_check
    def make_default_retry():
        try:
            return urllib3.util.retry.Retry(
                total=20,
                backoff_factor=0.5,
                allowed_methods=False,
            )
        except TypeError:  # Support urllib3 prior to 1.26
            return urllib3.util.retry.Retry(
                total=20,
                backoff_factor=0.5,
                method_whitelist=False,
            )

    return make_default_retry()  # type: ignore


def parse_protobuf_timestamp(ts: str) -> datetime.datetime:
    # Protobuf emits timestamps in RFC3339 format, which are identical to canonical JavaScript date
    # stamps [1].  datetime.datetime.fromisoformat parses a subset of ISO8601 timestamps, but
    # notably does not handle the trailing Z to signify the UTC timezone [2].
    #
    # [1] https://tc39.es/ecma262/#sec-date-time-string-format
    # [2] https://bugs.python.org/issue35829
    if ts.endswith("Z"):
        ts = ts[:-1] + "+00:00"
    return datetime.datetime.fromisoformat(ts)
