import logging
import pathlib
import shutil

import constants
import determined as det

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    hparams = info.trial.hparams
    with det.core.init() as core_context:
        core_context.train.report_validation_metrics(steps_completed=0, metrics=hparams["results"])
        checkpoint_metadata_dict = {"steps_completed": 0}
        with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (
            path,
            storage_id,
        ):
            src = pathlib.Path(constants.OUTPUT_FILE_PATH)
            dst = pathlib.Path(path).joinpath(src.name)
            shutil.copy(src=src, dst=dst)
