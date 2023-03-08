import argparse
import logging
import pathlib
import shutil

import determined as det
from dsat import constants


def main(core_context: det.core.Context) -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--prev_exit_code")
    prev_exit_code = int(parser.parse_args().prev_exit_code)
    if prev_exit_code:
        print("EXITING DUE TO PREV EXIT CODE", prev_exit_code)
        exit(prev_exit_code)
    is_chief = core_context.distributed.get_rank() == 0
    if is_chief:
        # Save the profile results as a checkpoint of the calling Trial (Ryan wouldn't approve).
        # This wrapper also doesn't know the actual steps_completed, so it's just using zero, which
        # is bad.
        checkpoint_metadata_dict = {"steps_completed": 0}  # TODO: use the actual steps completed
        with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (
            path,
            _,
        ):
            for src_str in {constants.DS_PROFILER_OUTPUT_PATH}:  # Previously wrote more to ckpt.
                src = pathlib.Path(src_str)
                dst = pathlib.Path(path).joinpath(src.name)
                try:
                    shutil.copy(src=src, dst=dst)
                except FileNotFoundError:
                    logging.info(f"File {src} not found, skipping profiling checkpoint.")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    distributed = det.core.DistributedContext.from_torch_distributed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context)
