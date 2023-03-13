import argparse
import os
import tempfile
from typing import Any, Dict

from determined.experimental import client
from dsat import _defaults, _utils


def parse_args():

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
    search_runner_config_dict = config_dict
    search_runner_config_dict["name"] += " (DS AT Searcher)"
    search_runner_config_dict["searcher"]["name"] = "single"
    search_runner_config_dict["searcher"]["max_length"] = 0  # max_length not used by DS AT.
    # TODO: don't hardcode the searcher's max_restarts.
    search_runner_config_dict["max_restarts"] = 3

    # TODO: let users have more fine control over the searcher config.
    # TODO: taking slots_per_trial: 0 to imply cpu-only here, but that's apparently an unsafe assump
    # e.g. on Grenoble.
    search_runner_config_dict["resources"] = {"slots_per_trial": 0}
    # TODO: remove this Grenoble specific code.
    config_dict["resources"] = {
        "resource_pool": "misc_cpus"
    }  # will need to get original resources later.
    search_runner_config_dict[
        "entrypoint"
    ] = f"python3 -m dsat._run_dsat -c {config_path_absolute} -md {model_dir_absolute}"

    # TODO: early sanity check the submitted config. E.g. makesure that searcher.metric and
    # hyperparameters.ds_config.autotuning.metric coincide.

    # TODO: Account for cases where DS is not initialized with yaml config file.
    # Create empty tempdir as the model_dir and upload everything else as an includes in order to
    # avoid unwanted double directory explosions.
    with tempfile.TemporaryDirectory() as temp_dir:
        includes = [model_dir_absolute, config_path_absolute]
        # TODO: need to append dsat here for searcher logic to be available on-cluster, but this
        # will be removed when the logic lives in determined proper.
        includes.append("dsat")
        client.create_experiment(
            config=search_runner_config_dict, model_dir=temp_dir, includes=includes
        )


def run():
    args = parse_args()
    config_dict = _utils.get_config_dict_from_yaml_path(args.config_path)
    run_autotuning(args, config_dict)


if __name__ == "__main__":
    run()
