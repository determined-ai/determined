import argparse
import os
from typing import Any, Dict

from determined.experimental import client
from dsat import constants, utils


def parse_args():

    parser = argparse.ArgumentParser(description="DS Autotuning")
    parser.add_argument("-m", "--master", type=str, default="")
    parser.add_argument("-u", "--user", type=str, default="determined")
    parser.add_argument("-p", "--password", type=str, default="")

    parser.add_argument("config_path")
    parser.add_argument("model_dir")
    args = parser.parse_args()
    return args


def run_autotuning(args: argparse.Namespace, config_dict: Dict[str, Any]):
    config_dict["name"] += " (DS AT Searcher)"
    config_dict["searcher"]["name"] = "single"
    config_dict["searcher"]["max_length"] = constants.DSAT_MAX_LENGTH_STEPS
    config_dict["resources"] = {"slots_per_trial": 0}  # Will need to get original resources later.
    config_dict["entrypoint"] = f"python3 -m dsat.run_dsat -c {args.config_path}"

    # TODO: Need to account for case where config isn't in model_dir, in which case
    # we need to pass its path to the `includes` arg of `create_experiment` (rather than config)
    # for later stages to have access the original config file.

    # TODO: Account for cases where DS is not initialized with yaml config file.
    client.create_experiment(config=config_dict, model_dir=args.model_dir)


def run_other_experiment(args: argparse.Namespace, config_dict: Dict[str, Any]):
    client.create_experiment(config=config_dict, model_dir=args.model_dir)


def run():
    args = parse_args()

    # Convert config to python dict
    config_dict = utils.get_config_dict_from_yaml_path(args.config_path)

    if not args.master:
        args.master = os.getenv("DET_MASTER", "localhost:8000")

    client.login(master=args.master, user=args.user, password=args.password)

    if (
        config_dict["searcher"]["name"] == "custom"
    ):  # TODO: Avoid conflict w/ other custom searchers.
        run_autotuning(args, config_dict)
    else:
        run_other_experiment(args, config_dict)


if __name__ == "__main__":
    run()
