import argparse
import logging
import pathlib
import shutil
import sys

import constants
import utils

import determined as det
from determined.experimental.client import create_experiment


def get_parsed_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("-w", "--workspace_name", type=str, default="")
    parser.add_argument("-p", "--project_name", type=str, default="")
    parser.add_argument("-e", "--exp_name", type=str, default="")

    parsed_args = parser.parse_args()

    return parsed_args


def main(core_context: det.core.Context, args: argparse.Namespace) -> None:
    is_chief = core_context.distributed.get_rank() == 0
    if is_chief:
        checkpoint_metadata_dict = {"steps_completed": 0}
        with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (
            path,
            storage_id,
        ):
            src = pathlib.Path(constants.OUTPUT_FILE_PATH)
            dst = pathlib.Path(path).joinpath(src.name)
            shutil.copy(src=src, dst=dst)
        ds_profiler_results = utils.DSProfilerResults(path=src)
        config = ds_profiler_results.get_config(
            workspace_name=args.workspace_name,
            project_name=args.project_name,
            exp_name=args.exp_name,
            entrypoint="python3 single_ds_profiler_logger_exp.py",
        )
        logging_exp = create_experiment(config=config, model_dir=".")
        exit_status = logging_exp.wait()
        if not exit_status:
            logging.info("Experiment completed successfully")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    distributed = det.core.DistributedContext.from_torch_distributed()
    with det.core.init(distributed=distributed) as core_context:
        args = get_parsed_args()
        main(core_context, args=args)
