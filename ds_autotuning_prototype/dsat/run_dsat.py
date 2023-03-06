import argparse
import logging
import os
import pathlib
import shutil

import determined as det
from determined import searcher
from dsat import dsat_search_method, utils


def get_parsed_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("-c", "--config_path", type=str)
    parser.add_argument("-md", "--model_dir", type=str)
    args = parser.parse_args()
    # Only need the base names.
    args.config_path = os.path.basename(args.config_path)
    args.model_dir = os.path.basename(args.model_dir)

    return args


def dsat_copy_hack(model_dir: str) -> None:
    """
    Temporary hack to make the dsat directory available for follow-on experiments. TODO: remove.
    Won't be needed when we can import dsat from determined.
    """
    src = pathlib.Path("dsat")
    shutil.copytree(src, pathlib.Path(model_dir).joinpath(src), dirs_exist_ok=True)


def main(core_context: det.core.Context) -> None:
    args = get_parsed_args()
    submitted_config_dict = utils.get_config_dict_from_yaml_path(args.config_path)

    all_search_method_classes = {"random": dsat_search_method.DSATRandomSearchMethod}
    tuner_type = submitted_config_dict["hyperparameters"]["ds_config"]["autotuning"]["tuner_type"]
    assert (
        tuner_type in all_search_method_classes
    ), f"search_method must be one of {list(all_search_method_classes)}"

    search_method = all_search_method_classes[tuner_type](submitted_config_dict)
    search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)

    # TODO: remove hack.
    dsat_copy_hack(args.model_dir)

    search_runner.run(submitted_config_dict, model_dir=args.model_dir)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    with det.core.init() as core_context:
        main(core_context)
