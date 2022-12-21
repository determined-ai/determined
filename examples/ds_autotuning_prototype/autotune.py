import argparse
import collections
import copy
import os

import constants
from ruamel import yaml

from determined.experimental import client


def parse_args():

    parser = argparse.ArgumentParser(description="DS Autotuning")
    parser.add_argument("-m", "--master", type=str, default="")
    parser.add_argument("-u", "--user", type=str, default="determined")
    parser.add_argument("-p", "--password", type=str, default="")

    parser.add_argument("config")
    parser.add_argument("model_dir")
    args = parser.parse_args()
    return args


def replace_dict(d, u, ignored_keys=[]):
    """Replaces values in dict d with values in dict u.

    Args:
        d (dict): the target dict to overwrite
        u (dict): the dict containing the values to overwrite the target dict

    Returns:
        dict d with values overwritten by the corresponding ones in dict u.
    """
    if u is not None:
        for k, v in u.items():
            if k not in ignored_keys:
                if isinstance(v, collections.abc.Mapping):
                    d[k] = replace_dict(d.get(k, {}), v, ignored_keys)
                else:
                    d[k] = v
    return d


def get_mem_per_gpu(num_params, total_gpus, fp16_enabled, mp_size, zero_stage):

    # assume the model uses Adam optimizer (GG: inherited assump from DS)
    params_mem = num_params * (2 if fp16_enabled else 4)
    gradients_mem = num_params * (2 if fp16_enabled else 4)
    optimizer_mem = num_params * (16 if fp16_enabled else 8)

    if zero_stage >= 0:
        optimizer_mem = optimizer_mem / total_gpus

    if zero_stage >= 1:
        gradients_mem = gradients_mem / total_gpus

    if zero_stage >= 2:
        params_mem = params_mem / total_gpus

    mem_per_gpu = (params_mem + gradients_mem + optimizer_mem) / mp_size()
    return mem_per_gpu


def run_autotuning(args):
    model_info_config = copy.deepcopy(args.config)
    replace_dict(
        model_info_config["hyperparameters"]["ds_config"],
        constants.MODEL_INFO_DS_CONFIG,
    )
    model_info_config["searcher"] = {
        "name": "single",
        "metric": "placeholder",
        "max_length": constants.MODEL_INFO_MAX_LENGTH,
    }
    model_info_config["name"] += "_model_info"
    project_name = model_info_config.get("project", "")
    workspace_name = model_info_config.get("workspace", "")
    exp_name = model_info_config.get("name", "")
    model_info_config["entrypoint"] += (
        "; python3 -m determined.launch.torch_distributed python3 ds_profiler_logger.py"
        f" -p {project_name} -e {exp_name} -w {workspace_name}"
    )
    model_profile_exp = client.create_experiment(config=model_info_config, model_dir=args.model_dir)


def run_other_experiment(args):
    exp = client.create_experiment(config=args.config, model_dir=args.model_dir)


if __name__ == "__main__":
    args = parse_args()

    # Convert config to python dict
    config = yaml.YAML(typ="safe")
    with open(args.config, "r") as f:
        args.config = config.load(f)

    if not args.master:
        args.master = os.getenv("DET_MASTER", "localhost:8000")

    if args.config["searcher"]["name"] == "custom":
        run_autotuning(args)
    else:
        run_other_experiment(args)
