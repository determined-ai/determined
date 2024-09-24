import copy
import json
import logging
import pathlib
from typing import Any, Dict, List, Optional, Union

import filelock

from determined import util as det_util

CURR_DIR = pathlib.Path(".")
CONFIG_KEY = "deepspeed_config"
OVERWRITE_KEY = "overwrite_deepspeed_args"


def get_ds_config_from_hparams(
    hparams: Dict[str, Any],
    base_dir: Union[pathlib.Path, str] = CURR_DIR,
) -> Dict[str, Any]:
    """Gets the DS config dictionary after merging with overwrite values.

    Follows the rules as described here:
    https://docs.determined.ai/latest/training/apis-howto/deepspeed/deepspeed.html#configuration
    Args:
        hparams (Dict):
            Hyperparameters dictionary
        base_dir (pathlib.Path):
            Base directory relattive to which hparams.deepspeed_config is defined
    Returns:
        The Deepspeed Configuration for this experiment following the overwriting rules
    """
    assert CONFIG_KEY in hparams, (
        f"Expected to find {CONFIG_KEY} in the Hyperparameters section. " f"Instead found {hparams}"
    )
    ds_config_relative_path = hparams[CONFIG_KEY]
    base_dir = pathlib.Path(base_dir)
    full_path = base_dir.joinpath(ds_config_relative_path)
    with open(full_path, "r") as f:
        base_ds_config: Dict[str, Any] = json.load(f)
    overwrite_ds_config = hparams.get(OVERWRITE_KEY, {})
    final_ds_config = det_util.merge_dicts(base_ds_config, overwrite_ds_config)
    return final_ds_config


def get_hf_ds_config_path_from_args(args: List[str]) -> Optional[str]:
    for idx in range(len(args)):
        if args[idx] == "--deepspeed":
            ds_config_idx = idx + 1
            ds_config_path = args[ds_config_idx]
            return ds_config_path
    return None


def update_hf_args(args: List[str], ds_config_dict: Dict[str, Any]) -> List[str]:
    """
    Updates batch-size-related HF CLI args to be consistent with the values specified in the
    provided DeepSpeed config dictionary.

    Args:
        args: list of CLI arguments passed to the HF entrypoint
        ds_config_dict: the DeepSpeed configuration as a dictionary
    """
    hf_flag_to_ds_key = {
        "--per_device_train_batch_size": "train_micro_batch_size_per_gpu",
        "--gradient_accumulation_steps": "gradient_accumulation_steps",
    }
    # Overwrite CLI args
    args = copy.deepcopy(args)
    for idx in range(len(args)):
        if args[idx] in hf_flag_to_ds_key:
            ds_key = hf_flag_to_ds_key[args[idx]]
            overwrite_value = ds_config_dict[ds_key]
            # Need to avoid copying possible "auto" value from json config to HF CLI.
            is_auto = isinstance(overwrite_value, str) and overwrite_value.strip() == "auto"
            if not is_auto:
                overwrite_value_str = str(overwrite_value)
                if args[idx + 1] != overwrite_value_str:
                    logging.warning(
                        f"Changing {args[idx]} from {args[idx +1]} to {overwrite_value_str}"
                        " to match the deespspeed config values."
                    )
                    args[idx + 1] = overwrite_value_str
                del hf_flag_to_ds_key[args[idx]]

    # Any remaining keys in hf_flag_to_ds_key were not provided as args to the HF CLI entrypoint,
    # but they must be added in explicitly, to avoid falling back to HF defaults.
    for hf_flag, ds_key in hf_flag_to_ds_key.items():
        hf_flag_value = ds_config_dict[ds_key]
        is_auto = isinstance(hf_flag_value, str) and hf_flag_value.strip() == "auto"
        if not is_auto:
            hf_flag_value_str = str(hf_flag_value)
            args.extend([hf_flag, hf_flag_value_str])
            logging.warning(
                f"Adding {hf_flag} {hf_flag_value_str} to HF CLI args to reflect overwrite values."
            )
    return args


def get_hf_args_with_overwrites(args: List[str], hparams: Dict[str, Any]) -> List[str]:
    """Updates the submitted HF CLI Args to account for overwrite values.

    Primarily intended as a helper function for Determined AI DeepSpeed (DS) which provides
    overwrite values through the `hparams["overwrite_deepspeed_args"]` which possibly include DS
    batch-size related arguments (`train_batch_size`, `train_micro_batch_size_per_gpu`, and
    `gradient_accumulation_steps`) which are in conflict with the corresponding HF CLI batch-size
    related arguments(`--per_device_train_batch_size` and `--gradient_accumulation_steps`). This
    function updates the HF CLI args to relect any such overwrite values. This process also requires
    overwriting the corresponding DS json file on-cluster.

    Args:
        args: the original HF CLI arguments
        hparams: hyperparameter dictionary generated through Determined AI

    Returns:
        args: updated HF CLI arguments
    """
    if OVERWRITE_KEY not in hparams:
        logging.info(
            f"{OVERWRITE_KEY} key not found in hparams, `get_hf_args_with_overwrites` " "is a no-op"
        )
        return args

    ds_config_path = get_hf_ds_config_path_from_args(args)
    assert ds_config_path is not None, "--deepspeed flag not found in HuggingFace args!"

    # A file lock is required during both the writing and reading.
    with filelock.FileLock(ds_config_path + ".lock"):
        with open(ds_config_path, "r") as f:
            ds_config_dict = json.load(f)

        # Then merge all overwrites into the ds_config
        overwritten_ds_config_dict = det_util.merge_dicts(ds_config_dict, hparams[OVERWRITE_KEY])

        # We need to actually overwrite the ds json config file, due to how HF processes args.
        with open(ds_config_path, "w") as f:
            json.dump(overwritten_ds_config_dict, f)
        # Finally overwrite the CLI args
        args = update_hf_args(args, overwritten_ds_config_dict)

    return args
