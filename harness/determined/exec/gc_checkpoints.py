"""
The entrypoint for the GC checkpoints job container.
"""
import argparse
import json
import logging
import os
import sys
from typing import Any, List

import determined as det
from determined import errors, tensorboard
from determined.common import constants, storage


def delete_checkpoints(
    manager: storage.StorageManager, to_delete: List[str], dry_run: bool
) -> None:
    """
    Delete some of the checkpoints associated with a single experiment.
    """
    logging.info("Deleting {} checkpoints".format(len(to_delete)))

    for storage_id in to_delete:
        if not dry_run:
            logging.info(f"Deleting checkpoint {storage_id}")
            try:
                manager.delete(storage_id)
            except errors.CheckpointNotFound as e:
                logging.warn(e)
        else:
            logging.info(f"Dry run: deleting checkpoint {storage_id}")


def delete_tensorboards(manager: tensorboard.TensorboardManager, dry_run: bool = False) -> None:
    """
    Delete all Tensorboards associated with a single experiment.
    """
    if dry_run:
        logging.info(f"Dry run: deleting Tensorboards for {manager.sync_path}")
        return

    try:
        manager.delete()
    except errors.CheckpointNotFound as e:
        logging.warn(e)
    logging.info(f"Finished deleting Tensorboards for {manager.sync_path}")


def json_file_arg(val: str) -> Any:
    with open(val) as f:
        return json.load(f)


def main(argv: List[str]) -> None:
    parser = argparse.ArgumentParser(description="Determined checkpoint GC")

    parser.add_argument(
        "--version",
        action="version",
        version="Determined checkpoint GC, version {}".format(det.__version__),
    )
    parser.add_argument("--experiment-id", help="The experiment ID to run the GC job for")
    parser.add_argument(
        "--log-level",
        default=os.getenv("DET_LOG_LEVEL", "INFO"),
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        help="Set the logging level",
    )
    parser.add_argument(
        "--storage-config",
        type=json_file_arg,
        default=os.getenv("DET_STORAGE_CONFIG", {}),
        help="Storage config (JSON-formatted file)",
    )
    parser.add_argument(
        "--delete",
        type=str,
        default=os.getenv("DET_DELETE", ""),
        help="comma-separated list of checkpoints to delete",
    )
    parser.add_argument(
        "--delete-tensorboards",
        action="store_true",
        default=os.getenv("DET_DELETE_TENSORBOARDS", False),
        help="Delete Tensorboards from storage",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        default=("DET_DRY_RUN" in os.environ),
        help="Do not actually delete any checkpoints from storage",
    )

    args = parser.parse_args(argv)

    logging.basicConfig(
        level=args.log_level, format="%(asctime)s:%(module)s:%(levelname)s: %(message)s"
    )

    logging.info("Determined checkpoint GC, version {}".format(det.__version__))

    storage_config = args.storage_config
    logging.info("Using checkpoint storage: {}".format(storage_config))

    manager = storage.build(storage_config, container_path=constants.SHARED_FS_CONTAINER_PATH)

    args.delete = args.delete.strip()
    storage_ids = []
    if args.delete != "":
        storage_ids = args.delete.split(",")
    delete_checkpoints(manager, storage_ids, dry_run=args.dry_run)

    if args.delete_tensorboards:
        tb_manager = tensorboard.build(
            os.environ["DET_CLUSTER_ID"],
            args.experiment_id,
            None,
            storage_config,
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )
        delete_tensorboards(tb_manager, dry_run=args.dry_run)


if __name__ == "__main__":
    main(sys.argv[1:])
