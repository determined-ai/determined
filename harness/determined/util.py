import collections
import contextlib
import datetime
import enum
import inspect
import json
import logging
import math
import numbers
import os
import pathlib
import random
import re
import shutil
import signal
import socket
import stat
import subprocess
import time
import uuid
from typing import Any, Callable, Dict, Iterator, List, Optional, Set, SupportsFloat, Tuple, cast

import determined as det
from determined import constants
from determined.common import check, util


@util.preserve_random_state
def download_gcs_blob_with_backoff(blob: Any, n_retries: int = 32, max_backoff: int = 32) -> Any:
    from google.cloud import storage

    if not (isinstance(blob, storage.Blob)):
        raise Exception(
            f"Called download_gcs_blob_with_backoff with object of type {type(blob).__name__}"
        )
    for n in range(n_retries):
        try:
            return blob.download_as_string()
        except Exception:
            time.sleep(min(2 ** n + random.random(), max_backoff))
    raise Exception("Max retries exceeded for downloading blob.")


def is_overridden(full_method: Any, parent_class: Any) -> bool:
    """Check if a function is overridden over the given parent class.

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
            np_values = np.array(values)
            filtered_values = np_values[np_values != None]  # noqa: E711
            m = np.mean(filtered_values).item()
        except (TypeError, ValueError):
            # If we get here, values are non-scalars, which cannot be averaged.
            # We keep the key so consumers can see all the metric names but
            # leave the value as None.
            pass
        avg_metrics[name] = m

    metrics = {"batch_metrics": batch_metrics, "avg_metrics": avg_metrics}  # type: Dict[str, Any]
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
                return "Infinity" if float(obj) > 0.0 else "-Infinity"
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


def calculate_batch_sizes(
    hparams: Dict[str, Any],
    slots_per_trial: int,
    trialname: str,
) -> Tuple[int, int]:
    slots_per_trial = max(slots_per_trial, 1)
    if "global_batch_size" not in hparams:
        raise det.errors.InvalidExperimentException(
            "Please specify an integer `global_batch_size` hyperparameter in your experiment "
            f"config.  It is a required hyperparameter for {trialname}-based training."
        )
    global_batch_size = hparams["global_batch_size"]
    if not isinstance(global_batch_size, int):
        raise det.errors.InvalidExperimentException(
            "The `global_batch_size` hyperparameter must be an integer value, not "
            f"{type(global_batch_size).__name__}"
        )

    if global_batch_size < slots_per_trial:
        raise det.errors.InvalidExperimentException(
            "Please set the `global_batch_size` hyperparameter to be greater or equal to the "
            f"number of slots. Current batch_size: {global_batch_size}, slots_per_trial: "
            f"{slots_per_trial}."
        )

    per_gpu_batch_size = global_batch_size // slots_per_trial
    effective_batch_size = per_gpu_batch_size * slots_per_trial
    if effective_batch_size != global_batch_size:
        logging.warning(
            f"`global_batch_size` changed from {global_batch_size} to {effective_batch_size} "
            f"to divide equally across {slots_per_trial} slots."
        )

    return per_gpu_batch_size, effective_batch_size


def check_sshd(peer_addr: str, deadline: float, port: int) -> None:
    """
    Waits for every peer machine to be ready to accept SSHD connections.

    :param peer_addr: address of machine running SSHD
    :param deadline: time to wait until SSHD ready
    :param port: port on addresses running SSHD
    :return: raises Exception if SSHD connection invalid or timeout on any peer
    """
    while True:
        with socket.socket() as sock:
            sock.settimeout(1)
            try:
                # Connect to a socket to ensure sshd is listening.
                sock.connect((peer_addr, port))
                # The ssh protocol requires the server to serve an initial greeting.
                # Receive part of that greeting to know that sshd is accepting/responding.
                data = sock.recv(1)
                if not data:
                    raise ValueError("no sshd greeting")
                # This peer is ready.
                break
            except Exception:
                if time.time() > deadline:
                    raise ValueError(
                        f"Chief machine was unable to connect to sshd on peer machine at "
                        f"{peer_addr}:{port}"
                    )
                time.sleep(0.1)


def match_legacy_trial_class(arg: str) -> bool:
    """
    Legacy trial-class entrypoints are of the form: module.submodule:ClassName
    """
    trial_class_regex = re.compile("^[a-zA-Z0-9_.]+:[a-zA-Z0-9_]+$")
    if trial_class_regex.match(arg):
        return True
    return False


def legacy_trial_entrypoint_to_script(trial_entrypoint: str) -> List[str]:
    return ["python3", "-m", "determined.exec.harness", trial_entrypoint]


def force_create_symlink(src: str, dst: str) -> None:
    os.makedirs(src, exist_ok=True)
    try:
        os.symlink(src, dst, target_is_directory=True)
    except FileExistsError:
        try:
            if os.path.islink(dst) or os.path.isfile(dst):
                os.unlink(dst)
            else:
                shutil.rmtree(dst)

            try:
                os.symlink(src, dst, target_is_directory=True)
                # be nice, make the newly created link world-writable
                file_mode = os.stat(dst).st_mode
                os.chmod(dst, file_mode | stat.S_IWOTH)
            except FileExistsError:
                # in case of a race between two workers
                pass

        except PermissionError as err:
            logging.warning(f"{err} trying to remove {dst}")


@contextlib.contextmanager
def forward_signals(p: subprocess.Popen, *signums: signal.Signals) -> Iterator[None]:
    """Forward a list of signals to a subprocess, restoring the original handlers afterwards."""
    if not signums:
        # Pick a useful default for wrapper processes.
        names = ["SIGINT", "SIGTERM", "SIGHUP", "SIGUSR1", "SIGUSR2", "SIGWINCH", "SIGBREAK"]
        signums = tuple(getattr(signal, name) for name in names if hasattr(signal, name))

    def signal_passthru(signum: Any, frame: Any) -> None:
        p.send_signal(signum)

    old_handlers = [None for n in signums]  # type: List[Any]
    try:
        # Install passthru handlers.
        for i, n in enumerate(signums):
            old_handlers[i] = signal.signal(n, signal_passthru)
        yield
    finally:
        # restore original handlers
        for n, old in zip(signums, old_handlers):
            if old is None:
                continue
            signal.signal(n, old)
