import argparse
import json
import logging
import os
import pathlib
import shutil

import determined as det
from determined import searcher
from determined.pytorch.deepspeed import get_ds_config_from_hparams, overwrite_deepspeed_config
from determined.pytorch.deepspeed.dsat import _dsat_search_method, _utils

ALL_SEARCH_METHOD_CLASSES = {
    "random": _dsat_search_method.DSATRandomSearchMethod,
}


def get_parsed_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("-c", "--config_path", type=str)
    parser.add_argument("-md", "--model_dir", type=str)
    args = parser.parse_args()
    # Only need the base names.
    args.config_path = os.path.basename(args.config_path)
    args.model_dir = os.path.basename(args.model_dir)
    return args


def main(core_context: det.core.Context) -> None:
    args = get_parsed_args()

    model_dir = pathlib.Path(args.model_dir)
    submitted_config_dict = _utils.get_config_dict_from_yaml_path(args.config_path)
    ds_config = get_ds_config_from_hparams(
        submitted_config_dict["hyperparameters"], model_dir=model_dir
    )
    assert "autotuning" in ds_config and "tuner_type" in ds_config["autotuning"], (
        f"deepspeed_config must include an 'autotuning.tuner_type' section when running Deepspeed Autotune!"
        + f" Found:\n{json.dumps(ds_config, indent='  ')}"
    )

    tuner_type = ds_config["autotuning"]["tuner_type"]
    assert (
        tuner_type in ALL_SEARCH_METHOD_CLASSES
    ), f"search_method must be one of {list(ALL_SEARCH_METHOD_CLASSES)}"

    # From the Searcher onward, we are serialized into the "ds_config" format
    submitted_config_dict["hyperparameters"]["ds_config"] = ds_config
    # TODO: Hack because golang searchers enforce constraints around "type"
    submitted_config_dict = _utils.upper_case_dict_key(submitted_config_dict, "type")

    search_method = ALL_SEARCH_METHOD_CLASSES[tuner_type](submitted_config_dict)
    search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)

    search_runner.run(submitted_config_dict, model_dir=args.model_dir)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    with det.core.init() as core_context:
        main(core_context)
