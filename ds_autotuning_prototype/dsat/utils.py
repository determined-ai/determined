import collections
import json
import logging
import os
import pathlib
import re
import time
from contextlib import contextmanager
from random import choice
from typing import Any, Dict, List, Union

import determined as det
import torch
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


@contextmanager
def dsat_reporting_context(
    core_context,
    op,
    steps_completed,
    model_info_profiling_path: Union[str, pathlib.Path] = constants.MODEL_INFO_PROFILING_PATH,
    ds_profiler_output_path: Union[str, pathlib.Path] = constants.DS_PROFILER_OUTPUT_PATH,
):
    try:
        yield
    except RuntimeError as rte:
        if "out of memory" in str(rte):
            report_oom_and_exit(core_context, op, steps_completed)
    except SystemExit as se:
        if file_exists(model_info_profiling_path):
            report_model_profiling_info_and_exit(
                core_context, op, steps_completed, model_info_profiling_path
            )
        else:
            raise se
    finally:
        print("CALLING finallly block")


def report_oom_and_exit(
    core_context,
    op,
    steps_completed,
):
    is_chief = core_context.distributed.rank == 0
    if is_chief:
        logging.info(
            "******************* GPU Out of Memory: Shutting down Trial ******************"
        )
        report_oom_dict = {constants.OOM_KEY: True}
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics=report_oom_dict
        )
        op.report_completed(report_oom_dict)
    exit()


def report_model_profiling_info_and_exit(
    core_context,
    op,
    steps_completed,
    model_info_profiling_path: Union[str, pathlib.Path] = constants.MODEL_INFO_PROFILING_PATH,
):
    is_chief = core_context.distributed.rank == 0
    if is_chief:
        model_info_profiling_results_dict = get_model_profiling_info_results_dict(
            path=model_info_profiling_path
        )
        gpu_mem_in_bytes = torch.cuda.get_device_properties(0).total_memory
        model_info_profiling_results_dict["gpu_mem_in_bytes"] = gpu_mem_in_bytes
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics=model_info_profiling_results_dict
        )
        op.report_completed(model_info_profiling_results_dict)
    exit()


def report_ds_profiling_info_and_exit(
    core_context,
    op,
    steps_completed,
    ds_profiler_output_path: Union[str, pathlib.Path] = constants.DS_PROFILER_OUTPUT_PATH,
):
    is_chief = core_context.distributed.rank == 0
    if is_chief:
        model_info_profiling_results_dict = get_model_profiling_info_results_dict(
            path=ds_profiler_output_path
        )
        gpu_mem_in_bytes = torch.cuda.get_device_properties(0).total_memory
        model_info_profiling_results_dict["gpu_mem_in_bytes"] = gpu_mem_in_bytes
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics=model_info_profiling_results_dict
        )
        op.report_completed(model_info_profiling_results_dict)
    exit()


def file_exists(path: Union[str, pathlib.Path], check_limit: int = 3, sleep_time: int = 1):
    # TODO: Clean up, verify needed.
    for _ in range(check_limit):
        if os.path.isfile(path):
            return True
        else:
            time.sleep(sleep_time)
    return False


def get_model_profiling_info_results_dict(
    path: Union[str, pathlib.Path] = constants.MODEL_INFO_PROFILING_PATH
):
    with open(path, "r") as output:
        results_dict = json.load(output)
        return results_dict


def get_ds_profiler_results_dict(
    path: Union[str, pathlib.Path] = constants.DS_PROFILER_OUTPUT_PATH
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


def dsat_metrics_converter(
    result: Union[float, Dict[str, Any]],
    profiler_output_path: Union[str, pathlib.Path] = constants.DS_PROFILER_OUTPUT_PATH,
):

    info = det.get_cluster_info()
    assert info is not None, "Must be run on cluster"
    searcher_config = info._trial_info._config["searcher"]
    # TODO: Prevent clashes w/ other non-DSAT custom searchers.
    is_autotuning = searcher_config["name"] == "custom"
    if not is_autotuning or not file_exists(profiler_output_path):
        return result
    else:
        # TODO: Need some sleep checks/retries to ensure the file was written? Timing issues?
        profiler_results_dict = get_ds_profiler_results_dict()
        return profiler_results_dict


def get_zero_optim_keys_and_defaults_per_stage(
    zero_stage: int,
) -> Dict[str, List[Union[bool, float]]]:
    defaults = constants.NEW_ZERO_OPTIM_KEYS_AND_DEFAULTS_PER_STAGE
    assert zero_stage in defaults, f"Invalid zero_stage, must be one of {list(defaults)}"
    keys_and_defaults = defaults[0]
    for stage in range(1, zero_stage + 1):
        keys_and_defaults = {**keys_and_defaults, **defaults[stage]}
    return keys_and_defaults


def get_random_zero_optim_dict_for_zero_stage(zero_stage: int) -> Dict[str, Union[bool, float]]:
    keys_and_defaults = get_zero_optim_keys_and_defaults_per_stage(zero_stage)
    zero_optim_dict = {key: choice(defaults) for key, defaults in keys_and_defaults.items()}
    zero_optim_dict["stage"] = zero_stage
    return zero_optim_dict
