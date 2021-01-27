import collections
import datetime
import enum
import os
import pathlib
import random
import shutil
import time
import uuid
from typing import Any, Dict, List, Optional, cast

import numpy as np
import simplejson

import determined as det
from determined_common import check, util


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


def wrap_metrics(metrics: det.workload.Response, stop_requested: bool) -> det.workload.Response:
    """Make workload response with metrics and stop_requested flag (or is Skipped if not chief)."""
    if isinstance(metrics, det.workload.Skipped):
        return metrics
    else:
        return {
            "metrics": metrics,
            "stop_requested": stop_requested,
        }


def json_encode(obj: Any, indent: Optional[str] = None, sort_keys: bool = False) -> str:
    def json_serializer(obj: Any) -> Any:
        if isinstance(obj, datetime.datetime):
            return obj.isoformat()
        if isinstance(obj, enum.Enum):
            return obj.name
        if isinstance(obj, np.float64):
            return float(obj)
        if isinstance(obj, np.float32):
            return float(obj)
        if isinstance(obj, np.int64):
            return int(obj)
        if isinstance(obj, np.int32):
            return int(obj)
        if isinstance(obj, uuid.UUID):
            return str(obj)
        if isinstance(obj, np.ndarray):
            return obj.tolist()
        # Objects that provide their own custom JSON serialization.
        if hasattr(obj, "__json__"):
            return obj.__json__()

        raise TypeError("Unserializable object {} of type {}".format(obj, type(obj)))

    # NB: We serialize NaN, Infinity, and -Infinity as `null`, because
    # those are not allowed by the JSON spec.
    s = simplejson.dumps(
        obj, default=json_serializer, ignore_nan=True, indent=indent, sort_keys=sort_keys
    )  # type: str
    return s


def write_user_code(path: pathlib.Path) -> None:
    code_path = path.joinpath("code")

    # When restarting from checkpoint, it is possible that the code path is already present
    # in the checkpoint directory. This happens for EstimatorTrial because we overwrite the
    # estimator model directory with the checkpoint folder at the start of training.
    if code_path.exists():
        shutil.rmtree(str(code_path))

    # Pytorch and tf.1 keras models can only be restored from a checkpoint if
    # the original code is present. The model code is the current working
    # directory. Therefore we save the current directory with the checkpoint.
    shutil.copytree(os.getcwd(), code_path, ignore=shutil.ignore_patterns("__pycache__"))
    os.chmod(code_path, 0o755)
