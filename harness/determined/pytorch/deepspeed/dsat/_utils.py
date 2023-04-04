import copy
import json
import logging
import pathlib
import random
from contextlib import contextmanager
from typing import Any, Dict, Generator, Optional, Union

import torch
from ruamel import yaml

import determined as det
from determined.common.experimental import user
from determined.pytorch.deepspeed.dsat import _defaults
from determined.util import merge_dicts


# TODO: move this to determined.util?
def get_dict_from_yaml_or_json_path(
    path: str, convert_json_keys_to_int: bool = True
) -> Dict[Any, Any]:
    """
    Load a json or yaml file as a dict. Optionally convert all json dict keys to ints, where possible.
    """
    p = pathlib.Path(path)
    if p.suffix == ".json":
        try:
            with open(p, "r") as f:
                json_dict: Dict[Any, Any] = json.load(f)
            if convert_json_keys_to_int:

                def try_str_to_int(s: str) -> Union[str, int]:
                    try:
                        return int(s)
                    except ValueError:
                        return s

                json_dict = {try_str_to_int(k): v for k, v in json_dict.items()}
            return json_dict
        except Exception as e:
            logging.info(f"Exception {e} raised when loading {path} with json. Attempting yaml.")
    else:
        with open(p, "r") as f:
            yaml_dict: Dict[Any, Any] = yaml.YAML(typ="safe").load(f)
        return yaml_dict


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
        model_profiling_path = pathlib.Path(_defaults.MODEL_INFO_PROFILING_PATH)
        autotuning_results_path = pathlib.Path(_defaults.AUTOTUNING_RESULTS_PATH)
        possible_paths = [model_profiling_path, autotuning_results_path]
        existing_paths = [p for p in possible_paths if p.exists()]
        # Exactly one of these files should be generated for each properly exited DS AT Trial.
        if len(existing_paths) == 1:
            path = existing_paths[0]
            add_gpu_info = path == model_profiling_path
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


def get_zero_optim_search_space(
    zero_search_config: Optional[Dict[int, Dict[str, Any]]] = None,
) -> Dict[int, Dict[str, Any]]:
    """
    Creates a search space for every provided zero stage (key) in `zero_search_config` whose
    corresponding values are dictionaries whch either specify every configuration for that stage
    or specify a diff on top of all lower stage configurations, merged in numerical order.
    Any lists in the individual stage configurations can be randomly and uniformly sampled from
    using `get_random_zero_optim_dict_from_search_space`.
    TODO: Explain better.
    """
    if zero_search_config is None:
        zero_search_config = _defaults.NEW_ZERO_OPTIM_KEYS_AND_DEFAULTS_PER_STAGE
    user_specified_stages = list(zero_search_config)
    search_space = copy.deepcopy(zero_search_config)
    for s1, s2 in zip(user_specified_stages[:-1], user_specified_stages[1:]):
        search_space[s2] = merge_dicts(zero_search_config[s1], zero_search_config[s2])
    return search_space


def get_random_zero_optim_dict_from_search_space(
    zero_stage: int, search_space: Dict[int, Dict[str, Any]]
) -> Dict[str, Any]:
    zero_optim_dict = copy.deepcopy(search_space[zero_stage])
    zero_optim_dict["stage"] = zero_stage
    for k, v in zero_optim_dict.items():
        # Randomly draw from any provided lists.
        zero_optim_dict[k] = v if not isinstance(v, list) else random.choice(v)
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
