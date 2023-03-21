import argparse
import os
import tempfile
from typing import Any, Dict

from determined.experimental import client
from determined.pytorch.deepspeed.dsat import _utils
from determined.util import merge_dicts


def parse_args():
    # TODO: Allow for additional includes args to be specified, as in the CLI.
    # TODO: Allow the user to pass an optional `searcher_config` to override default DS AT search.
    parser = argparse.ArgumentParser(description="DS Autotuning")
    parser.add_argument("-m", "--master", type=str)
    parser.add_argument("-u", "--user", type=str, default="determined")
    parser.add_argument("-p", "--password", type=str, default="")

    parser.add_argument("config_path")
    parser.add_argument("model_dir")
    args = parser.parse_args()
    return args


def run_autotuning(args: argparse.Namespace, config_dict: Dict[str, Any]):
    config_path_absolute = os.path.abspath(args.config_path)
    model_dir_absolute = os.path.abspath(args.model_dir)

    # Build the SearchRunner's config from the submitted config. The original config yaml file
    # is added as an include and is reimported by the SearchRunner later.
    # TODO: Revisit this choice. Might be worth giving the user the ability to specify some parts of
    # the SearchRunner config separately, despite the annoying double-config workflow.
    # TODO: let users have more fine control over the searcher config.
    search_runner_overrides = {
        "searcher": {"name": "single", "max_length": 0},
        # TODO: don't hardcode the searcher's max_restarts.
        "max_restarts": 3,
        # TODO: taking slots_per_trial: 0 to imply cpu-only here, but that's apparently an unsafe assumption
        # e.g. on Grenoble.
        "resources": {"slots_per_trial": 0},
        "entrypoint": f"python3 -m determined.pytorch.deepspeed.dsat._run_dsat "
        + f"-c {config_path_absolute} -md {model_dir_absolute}",
        # TODO: remove the environment section; just needed for GG's GCP cluster.
        "environment": {
            "image": {
                "cpu": "determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-deepspeed-0.7.0-gpu-0.20.1",
                "gpu": "determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-deepspeed-0.7.0-gpu-0.20.1",
            }
        },
    }
    search_runner_config_dict = merge_dicts(config_dict, search_runner_overrides)
    search_runner_config_dict["name"] += " (DS AT Searcher)"

    # TODO: early sanity check the submitted config. E.g. makesure that searcher.metric and
    # hyperparameters.ds_config.autotuning.metric coincide.

    # TODO: Account for cases where DS is not initialized with yaml config file.
    # Create empty tempdir as the model_dir and upload everything else as an includes in order to
    # avoid unwanted double directory explosions.
    with tempfile.TemporaryDirectory() as temp_dir:
        includes = [model_dir_absolute, config_path_absolute]
        client.create_experiment(
            config=search_runner_config_dict, model_dir=temp_dir, includes=includes
        )


def run():
    args = parse_args()
    config_dict = _utils.get_config_dict_from_yaml_path(args.config_path)
    run_autotuning(args, config_dict)


if __name__ == "__main__":
    run()
