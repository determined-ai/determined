import collections
import datetime
import enum
import inspect
import json
import math
import numbers
import os
import pathlib
import random
import shutil
import time
import uuid
import warnings
from typing import Any, Callable, Dict, List, Optional, Set, SupportsFloat, TypeVar, cast

from determined import constants
from determined.common import check, util


@util.preserve_random_state
def download_gcs_blob_with_backoff(blob: Any, n_retries: int = 32, max_backoff: int = 32) -> Any:
    for n in range(n_retries):
        try:
            return blob.download_as_string()
        except Exception:
            time.sleep(min(2 ** n + random.random(), max_backoff))
    raise Exception("Max retries exceeded for downloading blob.")


def is_overridden(full_method: Any, parent_class: Any) -> bool:
    """Check if a function is overriden over the given parent class.

    Note that full_method should always be the name of a method, but users may override
    that name with a variable anyway. In that case we treat full_method as not overridden.
    """
    if callable(full_method):
        return cast(bool, full_method.__qualname__.partition(".")[0] != parent_class.__name__)
    return False


def has_param(fn: Callable[..., Any], name: str, pos: Optional[int] = None) -> bool:
    """
    Inspects function fn for presence of an argument.
    """
    args = inspect.getfullargspec(fn)[0]
    if name in args:
        return True
    if pos is not None:
        return pos < len(args)
    return False


def get_member_func(obj: Any, func_name: str) -> Any:
    member = getattr(obj, func_name, None)
    if callable(member):
        return member
    return None


def _list_to_dict(list_of_dicts: List[Dict[str, Any]]) -> Dict[str, List[Any]]:
    """Transpose list of dicts to dict of lists."""
    dict_of_lists = collections.defaultdict(list)  # type: Dict[str, List[Any]]
    for d in list_of_dicts:
        for key, value in d.items():
            dict_of_lists[key].append(value)
    return dict_of_lists


def _dict_to_list(dict_of_lists: Dict[str, List]) -> List[Dict[str, Any]]:
    """Transpose a dict of lists to a list of dicts.

        dict_to_list({"a": [1, 2], "b": [3, 4]})) -> [{"a": 1, "b": 3}, {"a": 2, "b": 4}]

    In some cases _dict_to_list is the inverse of _list_to_dict. This function assumes that
    all lists have the same length.
    """

    list_len = len(list(dict_of_lists.values())[0])
    for lst in dict_of_lists.values():
        check.check_len(lst, list_len, "All lists in the dict must be the same length.")

    output_list = [{} for _ in range(list_len)]  # type: List[Dict[str, Any]]
    for i in range(list_len):
        for k in dict_of_lists.keys():
            output_list[i][k] = dict_of_lists[k][i]

    return output_list


def validate_batch_metrics(batch_metrics: List[Dict[str, Any]]) -> None:
    metric_dict = _list_to_dict(batch_metrics)

    # We expect that all batches have the same set of metrics.
    metric_dict_keys = metric_dict.keys()
    for idx, metric_dict in zip(range(len(batch_metrics)), batch_metrics):
        keys = metric_dict.keys()
        if metric_dict_keys == keys:
            continue

        check.eq(metric_dict_keys, keys, "inconsistent training metrics: index: {}".format(idx))


def make_metrics(num_inputs: Optional[int], batch_metrics: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Make metrics dict including aggregates given individual data points."""
    import numpy as np

    metric_dict = _list_to_dict(batch_metrics)
    validate_batch_metrics(batch_metrics)

    avg_metrics = {}  # type: Dict[str, Optional[float]]
    for name, values in metric_dict.items():
        m = None  # type: Optional[float]
        try:
            values = np.array(values)
            filtered_values = values[values != None]  # noqa: E711
            m = np.mean(filtered_values)
        except (TypeError, ValueError):
            # If we get here, values are non-scalars, which cannot be averaged.
            # We keep the key so consumers can see all the metric names but
            # leave the value as None.
            pass
        avg_metrics[name] = m

    metrics = {"batch_metrics": batch_metrics, "avg_metrics": avg_metrics}
    if num_inputs is not None:
        metrics["num_inputs"] = num_inputs

    return metrics


def json_encode(obj: Any, indent: Optional[str] = None, sort_keys: bool = False) -> str:
    """
    Encode things as json, with an extra preprocessing step to handle some non-standard types.

    Note: json has a "default" argument that accepts something like our preprocessing step,
    except it is only invoked for non-native types (i.e. no catching nan or inf floats).
    """
    import numpy as np

    def jsonable(obj: Any) -> Any:
        if isinstance(obj, (str, bool, type(None))):
            # Needs no fancy encoding.
            return obj
        if isinstance(obj, numbers.Integral):
            # int, np.int64, etc.
            return int(obj)
        if isinstance(obj, numbers.Number):
            obj = cast(SupportsFloat, obj)
            # float, np.float64, etc.  Serialize nan/±infinity as strings.
            if math.isnan(obj):
                return "NaN"
            if math.isinf(obj):
                return "Infinity" if obj > 0 else "-Infinity"
            return float(obj)
        if isinstance(obj, bytes):
            # Assume bytes are utf8 (json can't encode arbitrary binary data).
            return obj.decode("utf8")
        if isinstance(obj, (list, tuple)):
            # Recurse into lists.
            return [jsonable(v) for v in obj]
        if isinstance(obj, dict):
            # Recurse into dicts.
            return {k: jsonable(v) for k, v in obj.items()}
        if isinstance(obj, np.ndarray):
            # Expand arrays into lists, then recurse.
            return jsonable(obj.tolist())
        if isinstance(obj, datetime.datetime):
            return obj.isoformat()
        if isinstance(obj, enum.Enum):
            return obj.name
        if isinstance(obj, uuid.UUID):
            return str(obj)
        # Objects that provide their own custom JSON serialization.
        if hasattr(obj, "__json__"):
            return obj.__json__()
        raise TypeError("Unserializable object {} of type {}".format(obj, type(obj)))

    return json.dumps(jsonable(obj), indent=indent, sort_keys=sort_keys)


def write_user_code(path: pathlib.Path, on_cluster: bool) -> None:
    code_path = path.joinpath("code")

    # When restarting from checkpoint, it is possible that the code path is already present
    # in the checkpoint directory. This happens for EstimatorTrial because we overwrite the
    # estimator model directory with the checkpoint folder at the start of training.
    if code_path.exists():
        shutil.rmtree(str(code_path))

    # Most models can only be restored from a checkpoint if the original code is present. However,
    # since it is rather common that users mount large, non-model files into their working directory
    # (like data or their entire HOME directory), when we are training on-cluster we use a
    # specially-prepared clean copy of the model rather than the working directory.
    if on_cluster:
        model_dir = constants.MANAGED_TRAINING_MODEL_COPY
    else:
        model_dir = "."
    shutil.copytree(model_dir, code_path, ignore=shutil.ignore_patterns("__pycache__"))
    os.chmod(code_path, 0o755)


def filter_duplicates(
    in_list: List[Any], sorter: Callable[[List[Any]], List[Any]] = sorted
) -> Set[Any]:
    """
    Find and return a set of duplicates from the list.
    """
    in_list = sorter(in_list)
    last_item = None
    duplicates = set()
    for item in in_list:
        if last_item == item:
            duplicates.add(item)
        last_item = item
    return duplicates


T = TypeVar("T", bound=Callable[..., Any])


def deprecated(msg: str) -> Callable[[T], T]:
    def make_wrapper(fn: T) -> T:
        def wrapper(*arg: List, **kwarg: Dict) -> Any:
            warnings.warn(msg, FutureWarning)
            return fn(*arg, **kwarg)

        return cast(T, wrapper)

    return make_wrapper


def humanize_float(n: float) -> float:
    """
    Take a float and convert it to a more human-friendly float.
    """
    if n == 0 or not math.isfinite(n):
        return n
    digits = int(math.ceil(math.log10(abs(n))))
    # Since we don't do scientific notation, we only round decimal parts, not integer parts.
    # That is, 1.3 seconds instead of 1.3333333 is ok but 3000 seconds instead of 3333 is less so.
    sigfigs = max(digits, 4)
    return round(n, sigfigs - digits)


def make_timing_log(verb: str, duration: float, num_inputs: int, num_batches: int) -> str:
    rps = humanize_float(num_inputs / duration if duration else math.inf)
    bps = humanize_float(num_batches / duration if duration else math.inf)
    return (
        f"{verb}: {num_inputs} records in {humanize_float(duration)}s ({rps} records/s), "
        f"in {num_batches} batches ({bps} batches/s)"
    )
