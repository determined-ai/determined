import collections
import json
import logging
import os
import pathlib
import re
import time
from contextlib import contextmanager
from random import choice
from typing import Any, Dict, List, Tuple, Union

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
    core_context: det.core._context.Context,
    op: det.core._searcher.SearcherOperation,
    steps_completed: int,
) -> None:
    try:
        yield
    except RuntimeError as rte:
        oom_error_string = str(rte)
        if "out of memory" in oom_error_string:
            report_oom_and_exit(core_context, op, steps_completed, oom_error_string)
    except SystemExit as se:
        if file_or_dir_exists(constants.MODEL_INFO_PROFILING_PATH):
            report_json_results_and_exit(
                core_context=core_context,
                op=op,
                steps_completed=steps_completed,
                add_gpu_info=True,
                path=constants.MODEL_INFO_PROFILING_PATH,
            )
        elif file_or_dir_exists(constants.AUTOTUNING_RESULTS_PATH):
            report_json_results_and_exit(
                core_context=core_context,
                op=op,
                steps_completed=steps_completed,
                add_gpu_info=False,
                path=constants.AUTOTUNING_RESULTS_PATH,
            )
        else:
            raise se
    finally:
        # TODO: also catch and report ds profiling here?
        pass


def report_oom_and_exit(
    core_context: det.core._context.Context,
    op: det.core._searcher.SearcherOperation,
    steps_completed: int,
    oom_error_string: str,
) -> None:
    is_chief = core_context.distributed.rank == 0
    if is_chief:
        logging.info(
            "******************* GPU Out of Memory: Shutting down Trial ******************"
        )
        logging.info(oom_error_string)
        # TODO: use the information in the error string somehow?
        report_oom_dict = {constants.OOM_KEY: True, "OOM_message": oom_error_string}
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics=report_oom_dict
        )
        op.report_completed(report_oom_dict)
    exit()


def report_json_results_and_exit(
    core_context: det.core._context.Context,
    op: det.core._searcher.SearcherOperation,
    steps_completed: int,
    add_gpu_info: bool,
    path: Union[str, pathlib.Path],
) -> None:
    is_chief = core_context.distributed.rank == 0
    if is_chief:
        with open(path, "r") as f:
            results_dict = json.load(f)
        if add_gpu_info:
            gpu_mem_in_bytes = torch.cuda.get_device_properties(0).total_memory
            results_dict["gpu_mem_in_bytes"] = gpu_mem_in_bytes
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics=results_dict
        )
        op.report_completed(results_dict)
    exit()


def file_or_dir_exists(
    path: Union[str, pathlib.Path], check_limit: int = 1, sleep_time: int = 0
) -> bool:
    # TODO: Clean up, verify needed.
    for _ in range(check_limit):
        if os.path.isfile(path) or os.path.isdir(path):
            return True
        else:
            time.sleep(sleep_time)
    return False


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


def get_tbs_mps_gas(ds_config: Dict[str, Any]) -> Tuple[int, int, int]:
    """
    Verifies that the batch size configuration is valid and returns the Tuple
    `(train_batch_size, train_micro_batch_size_per_gpu, gradient_accumulation_steps)`.
    """
    tbs, mbs, gas = (
        ds_config.get("train_batch_size", None),
        ds_config.get("train_micro_batch_size_per_gpu", None),
        ds_config.get("gradient_accumulation_steps", 1),  # Uses the DS default.
    )
    # TODO: assert messages.
    if tbs is not None:
        if mbs is not None:
            assert tbs == mbs * gas
        else:
            mbs, remainder = divmod(tbs, gas)
            assert not remainder
    elif mbs is not None:
        tbs = mbs * gas

    return tbs, mbs, gas
