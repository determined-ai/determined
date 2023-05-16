import argparse
import collections
import json
import logging
import pathlib
import random
from contextlib import contextmanager
from typing import Any, Dict, Generator, List, Optional, Union, cast

import torch
from ruamel import yaml

import determined as det
from determined.pytorch.dsat import _defaults
from determined.util import merge_dicts


def get_base_parser() -> argparse.ArgumentParser:
    # TODO: better args everyhwere, namely shorter flags and abbrevs
    base_parser = argparse.ArgumentParser(add_help=False)
    base_parser.add_argument("config_path")
    base_parser.add_argument("model_dir")
    base_parser.add_argument("-i", "--include", type=str, nargs="+")

    base_parser.add_argument(
        "-mt", "--max-trials", type=int, default=_defaults.AUTOTUNING_ARG_DEFAULTS["max-trials"]
    )
    base_parser.add_argument("-ms", "--max-slots", type=int)
    base_parser.add_argument(
        "-mct",
        "--max-concurrent-trials",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["max-concurrent-trials"],
    )
    base_parser.add_argument("-sc", "--search-runner-config", type=str)
    base_parser.add_argument("-msrr", "--max-search-runner-restarts", type=int)
    base_parser.add_argument(
        "-z",
        "--zero-stages",
        type=int,
        nargs="+",
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["zero-stages"],
        choices=list(range(4)),
    )

    # DS-specific args.
    base_parser.add_argument(
        "-sps",
        "--start_profile-step",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["start-profile-step"],
    )
    base_parser.add_argument(
        "-eps",
        "--end-profile-step",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["end-profile-step"],
    )
    base_parser.add_argument(
        "-m",
        "--metric",
        type=str,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["metric"],
        choices=_defaults.SMALLER_IS_BETTER_METRICS + _defaults.LARGER_IS_BETTER_METRICS,
    )
    base_parser.add_argument(
        "-r",
        "--random_seed",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["random-seed"],
    )
    base_parser.add_argument("--run-full-experiment", action="store_true")

    return base_parser


def get_full_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(required=True, dest="search_method")
    base_parser = get_base_parser()

    subparsers.add_parser("_test", parents=[base_parser])

    random_subparser = subparsers.add_parser("random", parents=[base_parser])
    random_subparser.add_argument(
        "-trc",
        "--trials-per-random-config",
        type=int,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["trials-per-random-config"],
    )
    random_subparser.add_argument("-es", "--early-stopping", type=int)

    binary_subparser = subparsers.add_parser("binary", parents=[base_parser])
    binary_subparser.add_argument(
        "-srf",
        "--search-range-factor",
        type=float,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["search-range-factor"],
    )

    # TODO: just using the paper's notation now, but should make align with existing Determined.
    asha_subparser = subparsers.add_parser("asha", parents=[base_parser])
    asha_subparser.add_argument(
        "--max-binary-search-trials",
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["max-binary-search-trials"],
    )
    asha_subparser.add_argument(
        "--min-binary-search-trials",
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["min-binary-search-trials"],
    )
    asha_subparser.add_argument(
        "--asha-early-stopping",
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["asha-early-stopping"],
    )
    asha_subparser.add_argument(
        "--divisor",
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["divisor"],
    )
    asha_subparser.add_argument(
        "-srf",
        "--search-range-factor",
        type=float,
        default=_defaults.AUTOTUNING_ARG_DEFAULTS["search-range-factor"],
    )

    return parser


def smaller_is_better(metric: str) -> bool:
    if metric in _defaults.SMALLER_IS_BETTER_METRICS:
        return True
    elif metric in _defaults.LARGER_IS_BETTER_METRICS:
        return False
    else:
        raise ValueError(
            "metric must be one of "
            f"{_defaults.SMALLER_IS_BETTER_METRICS + _defaults.LARGER_IS_BETTER_METRICS}, "
            f"not {metric}"
        )


def get_search_runner_config_from_args(args: argparse.Namespace) -> Dict[str, Any]:
    if args.search_runner_config is not None:
        submitted_search_runner_config = get_dict_from_yaml_or_json_path(args.search_runner_config)
        return submitted_search_runner_config

    submitted_exp_config_dict = get_dict_from_yaml_or_json_path(args.config_path)
    assert "deepspeed_config" in submitted_exp_config_dict["hyperparameters"], (
        "DS AT requires a `hyperparameters.deepspeed_config` key which points to the ds config"
        " json file"
    )

    # Also sanity check that if a --deepspeed_config (or in the case of HF
    # --deepspeed) arg is passed in, both configs match. Probably some gotchas here because
    # --deepspeed is also a boolean arg for vanilla deepspeed.
    possible_config_flags = ("--deepspeed", "--deepspeed_config")
    submitted_entrypoint = submitted_exp_config_dict["entrypoint"]
    # The entrypoint may be a string or list of strings. Strip all white space from each entry and
    # convert to a list, in either case.
    if isinstance(submitted_entrypoint, str):
        split_entrypoint = submitted_entrypoint.split(" ")
    elif isinstance(submitted_entrypoint, list):
        # Join and re-split to remove any possile white space.
        split_entrypoint = " ".join(submitted_entrypoint)
        split_entrypoint = split_entrypoint.split(" ")
    else:
        raise ValueError(
            f"Expected a str or list for an entrypoint, but received {type(submitted_entrypoint)}"
        )

    split_entrypoint = [s.strip() for s in split_entrypoint if s.strip()]

    for idx in range(len(split_entrypoint) - 1):
        curr_arg, next_arg = split_entrypoint[idx : idx + 2]
        next_arg_is_not_a_flag = next_arg != "-"
        if curr_arg in possible_config_flags and next_arg_is_not_a_flag:
            entrypoint_deepspeed_config = next_arg
            hp_deepspeed_config = submitted_exp_config_dict["hyperparameters"]["deepspeed_config"]
            if entrypoint_deepspeed_config != hp_deepspeed_config:
                raise ValueError(
                    f"The deepspeed config path in the `hyperparameters` section, "
                    f"{hp_deepspeed_config}, does not match the path in the entrypoint, "
                    f"{entrypoint_deepspeed_config}."
                )

    default_search_runner_config = _defaults.DEFAULT_SEARCH_RUNNER_CONFIG
    if args.max_search_runner_restarts is not None:
        default_search_runner_config["max_restarts"] = args.max_search_runner_restarts
    # Merge with the submitted experiment config so that the search runner shares the project,
    # workspace, etc.
    search_runner_config = merge_dicts(submitted_exp_config_dict, default_search_runner_config)
    search_runner_config["name"] = f"(DSAT) {search_runner_config['name']}"
    search_runner_config["hyperparameters"] = vars(args)

    return search_runner_config


# TODO: move this to determined.util?
def get_dict_from_yaml_or_json_path(
    path: str, convert_json_keys_to_int: bool = True
) -> Dict[Any, Any]:
    """
    Load a json or yaml file as a dict. Optionally convert all json dict keys to ints,
    where possible.
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
    steps_completed: Optional[int] = None,
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
    if steps_completed is None:
        steps_completed = op.length
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


def get_zero_stage_search_space(
    zero_stage: int,
) -> Dict[str, List[Union[bool, float]]]:
    default_settings: dict = _defaults.DEFAULT_ZERO_SEARCH_SPACE
    assert (
        zero_stage in default_settings
    ), f"Invalid zero_stage, must be one of {list(default_settings)}"
    search_space: dict = default_settings[1]
    for stage in range(2, zero_stage + 1):
        search_space = {**search_space, **default_settings[stage]}
    return search_space


def get_random_zero_optim_config(zero_stage: int) -> Dict[str, Union[bool, float]]:
    search_space = get_zero_stage_search_space(zero_stage)
    zero_optim_dict = {k: random.choice(v) for k, v in search_space.items()}
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
    # TODO: remove this auto hack needed for hf by an actual solution.
    if gas == "auto":
        gas = 1
    tbs = mbs * gas * slots
    return {
        "train_batch_size": tbs,
        "train_micro_batch_size_per_gpu": mbs,
        "gradient_accumulation_steps": gas,
    }


def dict_raise_error_on_duplicate_keys(ordered_pairs):
    """Reject duplicate keys."""
    d = dict((k, v) for k, v in ordered_pairs)
    if len(d) != len(ordered_pairs):
        counter = collections.Counter([pair[0] for pair in ordered_pairs])
        keys = [key for key, value in counter.items() if value > 1]
        raise ValueError("Duplicate keys in DeepSpeed config: {}".format(keys))
    return d


def normalize_base_ds_config(
    base_ds_config: Union[str, Dict], model_dir: pathlib.Path = pathlib.Path(".")
) -> Dict[str, Any]:
    if isinstance(base_ds_config, str):
        full_path = model_dir.joinpath(pathlib.Path(base_ds_config))
        with open(full_path, "r") as f:
            base_ds_config = json.load(
                f,
                object_pairs_hook=dict_raise_error_on_duplicate_keys,
            )
    else:
        if not isinstance(base_ds_config, dict):
            raise TypeError("Expected string or dict for base_ds_config argument.")
    return base_ds_config


def get_ds_config_from_hparams(
    hparams: Dict[str, Any],
    model_dir: Union[pathlib.Path, str] = pathlib.Path("."),
    config_key: str = _defaults.CONFIG_KEY,
    overwrite_key: str = _defaults.OVERWRITE_KEY,
) -> Dict[str, Any]:
    """Fetch and recursively merge the deepspeed config from the experiment config
    Follows the rules as described here:
    https://docs.determined.ai/latest/training/apis-howto/deepspeed/deepspeed.html#configuration
    Arguments:
        hparams (Dict): Hyperparameters dictionary
        model_dir (pathlib.Path): Base path for the Experiment Model
    Returns:
        The Deepspeed Configuration for this experiment following the overwriting rules
    """
    model_dir = pathlib.Path(model_dir)
    assert config_key in hparams, (
        f"Expected to find {config_key} in the Hyperparameters section. " f"Instead found {hparams}"
    )
    base_config_file_name = hparams[config_key]
    base_ds_config = normalize_base_ds_config(base_config_file_name, model_dir=model_dir)
    overwrite_ds_config = hparams.get(overwrite_key, {})
    ds_config = merge_dicts(cast(Dict[str, Any], base_ds_config), overwrite_ds_config)
    return ds_config


def overwrite_deepspeed_config(
    base_ds_config: Union[str, Dict],
    source_ds_dict: Dict[str, Any],
    model_dir: pathlib.Path = pathlib.Path("."),
) -> Dict[str, Any]:
    """Overwrite a base_ds_config with values from a source_ds_dict.
    You can use source_ds_dict to overwrite leaf nodes of the base_ds_config.
    More precisely, we will iterate depth first into source_ds_dict and if a node corresponds to
    a leaf node of base_ds_config, we copy the node value over to base_ds_config.
    Arguments:
        base_ds_config (str or Dict): either a path to a DeepSpeed config file or a dictionary.
        source_ds_dict (Dict): dictionary with fields that we want to copy to base_ds_config
        model_dir (pathlib.Path): Base path for the Experiment Model
    Returns:
        The resulting dictionary when base_ds_config is overwritten with source_ds_dict.
    """
    normalized_base_ds_config = normalize_base_ds_config(base_ds_config, model_dir=model_dir)
    return merge_dicts(cast(Dict[str, Any], normalized_base_ds_config), source_ds_dict)


def get_ds_config_path_from_args(args: List[str]) -> Optional[str]:
    for idx in range(len(args)):
        if args[idx] == "--deepspeed":
            ds_config_idx = idx + 1
            ds_config_path = args[ds_config_idx]
            return ds_config_path


def replace_ds_config_file_using_overwrites(
    args: List[str], hparams: Dict[str, Any], overwrite_key: str = _defaults.OVERWRITE_KEY
):
    """
    Gets the deepspeed json config path from the list of HF args, overwrites its values using
    the provided overwrite values, and the re-writes the result to the original config path.
    Intended primarily for use with DeepSpeed Autotuning. The resulting DeepSpeed config will have
    a consistent batch size configuration.
    """
    ds_config_path = get_ds_config_path_from_args(args)
    with open(ds_config_path, "r") as f:
        ds_config_dict_with_overwrites = json.load(f)
        # If overwrites are provided, use them. The deepspeed configuration is assumed to have a
        # consistent batch size configuration at this point, with all of train_batch_size,
        # train_micro_batch_size_per_gpu, and gradient_accumulation_steps filled in.
        ds_config_dict_with_overwrites = overwrite_deepspeed_config(
            ds_config_dict_with_overwrites, hparams.get(overwrite_key, {})
        )
        # overwrite the original config
        with open(ds_config_path, "w") as f:
            json.dump(ds_config_dict_with_overwrites, f)
