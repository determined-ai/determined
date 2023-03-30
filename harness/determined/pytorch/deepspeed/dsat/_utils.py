import json
import os
import pathlib
import random
import time
from contextlib import contextmanager
from typing import Any, Dict, Generator, List, Union

import torch
from ruamel import yaml

import determined as det
from determined.pytorch.deepspeed.dsat import _defaults


def get_config_dict_from_yaml_path(path: str) -> Dict[str, Any]:
    config = yaml.YAML(typ="safe")
    with open(path, "r") as f:
        config_dict: dict = config.load(f)
    return config_dict


@contextmanager
def dsat_reporting_context(
    core_context: det.core._context.Context,
    op: det.core._searcher.SearcherOperation,
    steps_completed: int,
) -> Generator[None, None, None]:
    """
    Call the DeepSpeed model engine's `forward` method within this context to intercept the `exit`
    call utilized by DS when autotuning and report the results back to Determined.  All other pieces
    of code which can potentially result in a GPU out-of-memory error should also be wrapped in
    the same context manager.

    TODO: the `report_validation_metrics` calls are needed for Web UI rendering, but they can also
    generate `duplicate key value` errors due to calling this method twice on the same
    `steps_completed`. Not sure if the solution should lie in code or documentation.
    """
    try:
        yield
    except SystemExit as se:
        possible_paths = [_defaults.MODEL_INFO_PROFILING_PATH, _defaults.AUTOTUNING_RESULTS_PATH]
        existing_paths = [path for path in possible_paths if file_or_dir_exists(path)]
        # Exactly one of these files should be generated for each properly exited DS AT Trial.
        if len(existing_paths) == 1:
            path = existing_paths[0]
            add_gpu_info = path == _defaults.MODEL_INFO_PROFILING_PATH
            report_json_results(
                core_context=core_context,
                op=op,
                steps_completed=steps_completed,
                add_gpu_info=add_gpu_info,
                path=path,
            )
        raise se


def report_json_results(
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
            gpu_mem = torch.cuda.get_device_properties(0).total_memory
            results_dict["gpu_mem"] = gpu_mem
        # TODO: solve potential problems with double reporting on the same time step.
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics=results_dict
        )
        op.report_completed(results_dict)
    # Ensure the operations generator is empty to complete sanity checks.
    try:
        next(core_context.searcher.operations())
    except StopIteration:
        pass
    else:
        raise AssertionError("Unexpected additional operations found!")


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
    default_settings: dict = _defaults.NEW_ZERO_OPTIM_KEYS_AND_DEFAULTS_PER_STAGE
    assert (
        zero_stage in default_settings
    ), f"Invalid zero_stage, must be one of {list(default_settings)}"
    keys_and_defaults: dict = default_settings[0]
    for stage in range(1, zero_stage + 1):
        keys_and_defaults = {**keys_and_defaults, **default_settings[stage]}
    return keys_and_defaults


def get_random_zero_optim_dict_for_zero_stage(zero_stage: int) -> Dict[str, Union[bool, float]]:
    keys_and_defaults = get_zero_optim_keys_and_defaults_per_stage(zero_stage)
    zero_optim_dict = {key: random.choice(defaults) for key, defaults in keys_and_defaults.items()}
    zero_optim_dict["stage"] = zero_stage
    return zero_optim_dict


def get_batch_config_from_mbs_gas_and_slots(
    ds_config: Dict[str, Any], slots: int
) -> Dict[str, int]:
    """
    Returns a consistent batch size configuration by adjusting `train_batch_size` according to the
    number of `slots`, `train_micro_batch_size_per_gpu`, and `gradient_accumulation_steps`  (or its
    default value, if not specified).
    """
    mbs = ds_config["train_micro_batch_size_per_gpu"]
    gas = ds_config.get("gradient_accumulation_steps", _defaults.GAS_DEFAULT)
    tbs = mbs * gas * slots
    return {
        "train_batch_size": tbs,
        "train_micro_batch_size_per_gpu": mbs,
        "gradient_accumulation_steps": gas,
    }
