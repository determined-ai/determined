import abc
import argparse
import copy
import logging
import uuid
from typing import Any, Dict, List, Set, Union

from dsat import constants, dsat_search_method, utils

import determined as det
from determined import searcher


def get_parsed_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("-c", "--config_path", type=str, default="")
    parsed_args = parser.parse_args()

    return parsed_args


def main(core_context: det.core.Context) -> None:
    args = get_parsed_args()
    submitted_config_dict = utils.get_config_dict_from_yaml_path(args.config_path)
    # Save profiling results w/ wrapper; probably remove eventually, but useful for sanity checking.
    # Needs error handling if we keep this; currently reports success even if the Trial fails.
    submitted_config_dict["entrypoint"] += (
        "; python3 -m determined.launch.torch_distributed"
        " python3 -m dsat.checkpoint_profiling_results_wrapper"
    )
    search_method = dsat_search_method.DSATSearchMethod(submitted_config_dict)
    search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)

    search_runner.run(submitted_config_dict, model_dir=".")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    with det.core.init() as core_context:
        main(core_context)
