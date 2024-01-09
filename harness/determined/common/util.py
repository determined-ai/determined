import datetime
import functools
import io
import json
import os
import pathlib
import platform
import random
import re
import sys
import time
import warnings
from typing import (
    IO,
    Any,
    Callable,
    Dict,
    Iterator,
    Optional,
    Sequence,
    Tuple,
    TypeVar,
    Union,
    cast,
    no_type_check,
    overload,
)

import urllib3

from determined.common import yaml

# _yamls keeps a cache of yaml.YAML objects for different values of default_flow_style
_yamls: Dict[Optional[bool], yaml.YAML] = {}


def _get_yaml(default_flow_style: Optional[bool] = None) -> yaml.YAML:
    y = _yamls.get(default_flow_style)
    if y is None:
        y = yaml.YAML(typ="safe", pure=True)
        if default_flow_style is not None:
            y.default_flow_style = default_flow_style
        _yamls[default_flow_style] = y
    return y


_LEGACY_TRAINING = "training"
_LEGACY_VALIDATION = "validation"
_INFERENCE = "inference"

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


def yaml_safe_dump(
    data: Any,
    stream: Optional[Union[io.FileIO, IO[Any], str]] = None,
    default_flow_style: Optional[bool] = None,
) -> Any:
    """
    A utility wrapper to mimick the pre-0.18.0 ruamel.yaml.safe_dump() API.

    The new API is needlessly verbose, and has some benefits we don't care about.
    """

    y = _get_yaml(default_flow_style)
    if stream is not None:
        # Write directly to the provided stream.
        return y.dump(data, stream=stream)
    # Write to a fake stream and return the result as a string.
    stream = io.StringIO()
    y.dump(data, stream=stream)
    return stream.getvalue()


def yaml_safe_load(f: Union[io.FileIO, IO[Any], str]) -> Any:
    """
    A utility wrapper to mimick the pre-0.18.0 ruamel.yaml.safe_load() API.

    In user-facing places like the CLI, safe_load_yaml_with_exceptions may be preferable.

    In internal places like e2e tests, this yaml_safe_load() should be preferred.
    """
    return _get_yaml().load(f)


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
        config = _get_yaml().load(yaml_file)
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
    # Protobuf emits timestamps in RFC3339 format [1], which are identical to canonical JavaScript
    # date stamps [2]. In Python, we use the method datetime.datetime.fromisoformat() to parse a
    # timestamp string using the ISO 8601 format [3]. Python versions below 3.11 have some
    # restrictions while parsing a timestamp string using the ISO 8601 format. Two restrictions
    # are specifically applicable in the case of DeterminedAI and are listed below.
    #     1.  datetime.datetime.fromisoformat() parses a subset of ISO 8601 timestamps, but
    #         notably does not handle the trailing Z to signify the UTC timezone [4].
    #     2.  datetime.datetime.fromisoformat() can only parse milliseconds and microseconds but
    #         not nanoseconds. Any other arbitrary length of the sub-second portion will fail [5].
    # These two restrictions are fixed in the Python 3.11 version, but we have to handle them until
    # the EOL for versions 3.10.x.
    # This method updates the provided RFC3339 format timestamp string to workaround the above
    # mentioned restrictions.
    #
    # [1] https://datatracker.ietf.org/doc/html/rfc3339
    # [2] https://tc39.es/ecma262/#sec-date-time-string-format
    # [3] https://docs.python.org/3/library/datetime.html#datetime.datetime.fromisoformat
    # [4] https://bugs.python.org/issue35829
    # [5] https://discuss.python.org/t/parse-z-timezone-suffix-in-datetime/2220/27
    #
    # Workaround for restriction 1 - replace UTC timezone indicator "Z" with "+00:00".
    if ts.endswith("Z"):
        ts = ts[:-1] + "+00:00"
    # Workaround for restriction 2 - remove any sub-second portion in the timestamp string.
    # Below are the list of examples demonstrating that the regex implementation is safe:
    # >>> re.sub(r"\.[0-9]*", "", "2023-08-22T22:06:45.242391275+00:00")
    # '2023-08-22T22:06:45+00:00'
    # >>> re.sub(r"\.[0-9]*", "", "2023-08-22T22:06:45.242391275+06:00")
    # '2023-08-22T22:06:45+06:00'
    # >>> re.sub(r"\.[0-9]*", "", "2023-08-22T22:06:45.242391275-06:00")
    # '2023-08-22T22:06:45-06:00'
    # >>> re.sub(r"\.[0-9]*", "", "2023-08-22T22:06:45.242391+00:00")
    # '2023-08-22T22:06:45+00:00'
    # >>> re.sub(r"\.[0-9]*", "", "2023-08-22T22:06:45.242+00:00")
    # '2023-08-22T22:06:45+00:00'
    # >>> re.sub(r"\.[0-9]*", "", "2023-08-22T22:06:45+00:00")
    # '2023-08-22T22:06:45+00:00'
    ts = re.sub(r"\.[0-9]*", "", ts)
    return datetime.datetime.fromisoformat(ts)


def is_protobuf_timestamp(ts: str) -> bool:
    """Validates that a string timestamp is in a Protobuf-compatible format.

    Protobuf requires timestamps in a limited RFC3339 format which requires a trailing "Z" to
    indicate UTC timezone.

    Arguments:
        ts (string): timestamp string (eg. ``yyyy-MM-dd'T'HH:mm:ss'Z'``)
    """
    if not ts.endswith("Z"):
        return False

    # Protobuf accepts timestamps with or without microseconds.
    accepted_formats = ["%Y-%m-%dT%H:%M:%S.%fZ", "%Y-%m-%dT%H:%M:%SZ"]
    for fmt in accepted_formats:
        try:
            datetime.datetime.strptime(ts, fmt)
        except (ValueError, TypeError):
            continue
        else:
            # Return if any format is successful.
            return True
    return False


def wait_for(
    predicate: Callable[[], Tuple[bool, T]], timeout: int = 60, interval: float = 0.1
) -> T:
    """
    Wait for the predicate to return (Done, ReturnValue) while
    checking for a timeout. without preempting the predicate.
    """

    start = time.time()
    done = False
    while not done:
        if time.time() - start > timeout:
            raise TimeoutError("timed out waiting for predicate")
        done, rv = predicate()
        time.sleep(interval)
    return rv


U = TypeVar("U", bound=Callable[..., Any])


def deprecated(message: Optional[str] = None) -> Callable[[U], U]:
    def decorator(func: U) -> U:
        @functools.wraps(func)
        def wrapper_deprecated(*args: Any, **kwargs: Any) -> Any:
            warning_message = (
                f"{func.__name__} is deprecated and will be removed in a future version."
            )
            if message:
                warning_message += f" {message}."
            warnings.warn(warning_message, category=DeprecationWarning, stacklevel=2)
            return func(*args, **kwargs)

        return cast(U, wrapper_deprecated)

    return decorator


def strtobool(val: str) -> bool:
    """
    A port of the distutils.util.strtobool function, removed in python 3.12.

    The only difference in this function is that any non-falsy value which is a non-empty string is
    accepted as a true value.  That small difference gives us a small headstart on this todo:

    TODO(MLG-1520): we should instead treat any nonempty string as "true".
    """
    return bool(val and val.lower() not in ("n", "no", "f", "false", "off", "0"))
