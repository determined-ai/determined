import collections
import datetime
import enum
import json
import pathlib
import random
import time
import uuid
from typing import Any, Dict, List, Optional, cast

import numpy as np
import simplejson

import determined as det
from determined._env_context import EnvContext
from determined_common import check


def download_gcs_blob_with_backoff(blob: Any, n_retries: int = 32, max_backoff: int = 32) -> Any:
    for n in range(n_retries):
        try:
            return blob.download_as_string()
        except Exception:
            time.sleep(min(2 ** n + random.random(), max_backoff))
    raise Exception("Max retries exceeded for downloading blob.")


def is_overridden(full_method: Any, parent_class: Any) -> bool:
    return cast(bool, full_method.__qualname__.partition(".")[0] != parent_class.__name__)


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

    # We expect that every batch has a metric named "loss".
    check.true(
        any(v for v in metric_dict if v.startswith("loss")),
        "model did not compute 'loss' training metric",
    )

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


def write_checkpoint_metadata(path: pathlib.Path, env: EnvContext, extras: Dict[str, Any]) -> None:
    metadata_path = path.joinpath("metadata.json")
    det_metadata = {
        "cluster_id": env.det_cluster_id,
        "det_version": det.__version__,
        "experiment_id": env.det_experiment_id,
        "trial_id": env.det_trial_id,
        **extras,
    }

    with metadata_path.open("w") as f:
        json.dump(det_metadata, f, indent=2)
