import collections
import json
import pathlib
import re
from typing import Any, Dict, Optional, Sequence, Union

import determined as det
from dsat import constants
from ruamel import yaml


def get_config_dict_from_yaml_path(path: str) -> Dict[str, any]:
    config = yaml.YAML(typ="safe")
    with open(path, "r") as f:
        config_dict = config.load(f)
    return config_dict


def replace_dict_in_place(d: Dict[str, Any], u: Dict[str, Any]):
    """Replaces values in dict d with values in dict u."""
    # TODO: Double check  logic.
    for k, v in u.items():
        if isinstance(v, collections.abc.Mapping) and k in d:
            replace_dict_in_place(d[k], v)
        else:
            d[k] = v


# TODO: The following two dict functions are needed as hacks around the `type` key
# used by DS for their optimizer with conflicts with our own special usage of this key
# in the config.
def upper_case_dict_key(d: Dict[str, Any], key: str) -> Dict[str, Any]:
    upper_d = {}
    for k, v in d.items():
        new_k = k.upper() if key == k else k
        if isinstance(v, dict):
            upper_d[new_k] = upper_case_dict_key(v, key)
        else:
            upper_d[new_k] = v
    return upper_d


def lower_case_dict_key(d: Dict[str, Any], key: str) -> Dict[str, Any]:
    lower_d = {}
    for k, v in d.items():
        new_k = k.lower() if key == k else k
        if isinstance(v, dict):
            lower_d[new_k] = lower_case_dict_key(v, key)
        else:
            lower_d[new_k] = v
    return lower_d


def get_non_decimal_number_in_line(line: str) -> float:
    num_str = re.search(r"\b\d+\b", line).group()
    num = float(num_str)
    return num


def get_decimal_number_in_line(line: str) -> float:
    num_str = re.search(r"\b\d*\.\d+\b", line).group()
    num = float(num_str)
    return num


def get_gpu_mem_bytes(path: Union[str, pathlib.Path] = constants.GPU_MEM_BYTES_FILE_PATH) -> int:
    with open(path, "r") as f:
        gpu_mem_bytes = int(f.read())
    return gpu_mem_bytes


def dsat_forward(core_context, op, model_engine, *args, **kwargs):
    try:
        output = model_engine(*args, **kwargs)
    except SystemExit:
        is_chief = core_context.distributed.rank == 0
        try:
            # TODO: Need some sleep checks/retries to ensure the file was written? Timing issues?
            model_info_profiling_results_dict = get_model_profiling_info_results_dict()
            print(model_info_profiling_results_dict)
            if is_chief:
                op.report_completed(
                    model_info_profiling_results_dict
                )  # TODO: Placeholder, will eventually pass entire results dict.
            exit()
        except Exception as e:
            print(f"Caught additional error after catching DS exit.")
            raise e
    return output


def get_model_profiling_info_results_dict(
    path: Union[str, pathlib.Path] = constants.AUTOTUNING_MODEL_PROFILE_OUTPUT_FILE_PATH
):
    with open(path, "r") as output:
        results_dict = json.load(output)
        return results_dict


def get_ds_profiler_results_dict(
    path: Union[str, pathlib.Path] = constants.PROFILER_OUTPUT_FILE_PATH
):
    metrics_with_units = {"iter latency", "FLOPS per GPU", "params per gpu"}
    metrics_without_units = {
        "samples/second",
        "world size",
        "data parallel size",
        "model parallel size",
        "batch size per GPU",
    }
    # The FLOPS and latency computations are reported with units.  We convert everything to
    # FLOPS and seconds.
    units_map = {
        "TFLOPS": 1e12,
        "GFLOPS": 1e9,
        "MFLOPS": 1e6,
        "KFLOPS": 1e3,
        "M": 1e6,
        "K": 1e3,
        "k": 1e3,
        "s": 1,
        "ms": 1e-3,
        "us": 1e-6,
    }
    results_dict = {}
    with open(path, "r") as output:
        for line in output:
            line = line.strip()
            for metric in metrics_with_units:
                if line.startswith(metric + ":") or line.startswith(metric + " ="):
                    units_factor = units_map[line.split()[-1]]
                    results_dict[metric] = get_decimal_number_in_line(line) * units_factor
            for metric in metrics_without_units:
                if line.startswith(metric + ":"):
                    results_dict[metric] = get_non_decimal_number_in_line(line)
    return results_dict


def get_flattened_dict(d: dict, concat_str: str = "_") -> Dict[str, Any]:
    """Flattens a nested dict into a single level dict with concatenated keys."""
    flat_dict = {}

    def flatten(d: dict, parent_key: str = "") -> None:
        for key, val in d.items():
            if parent_key:
                key = parent_key + concat_str + key
            if not isinstance(val, dict):
                assert key not in flat_dict, f'Key "{key}" already exists in dict!!!'
                flat_dict[key] = val
            else:
                flatten(val, key)

    flatten(d)
    return flat_dict


def dsat_metrics_converter(result: Union[float, Dict[str, Any]]):
    info = det.get_cluster_info()
    assert info is not None, "Must be run on cluster"
    searcher_config = info._trial_info._config["searcher"]
    # TODO: Prevent clashes w/ other non-DSAT custom searchers.
    is_autotuning = searcher_config["name"] == "custom"
    if not is_autotuning:
        print("REGULAR REPORT COMPLETED")
        return result
    else:
        print("DSAT REPORT COMPLETED")
        # TODO: Need some sleep checks/retries to ensure the file was written? Timing issues?
        profiler_results_dict = get_ds_profiler_results_dict()
        return profiler_results_dict
