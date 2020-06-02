"""
The entrypoint for the GC checkpoints job container.
"""
import argparse
import json
import logging
import os
import sys
from typing import Any, Dict, List

import determined as det
from determined_common import constants, storage


def delete_checkpoints(
    manager: storage.StorageManager, to_delete: List[Dict[str, Any]], dry_run: bool
) -> None:
    """
    Delete some of the checkpoints associated with a single
    experiment. `to_delete` is a list of two-element dicts,
    {"uuid": str, "resources": List[str]}.
    """
    logging.info("Deleting {} checkpoints".format(len(to_delete)))

    for record in to_delete:
        metadata = storage.StorageMetadata.from_json(record)
        if not dry_run:
            logging.info("Deleting checkpoint {}".format(metadata))
            manager.delete(metadata)
        else:
            logging.info("Dry run: deleting checkpoint {}".format(metadata.storage_id))

    logging.info("Finished deleting {} checkpoints".format(len(to_delete)))


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
    parser.add_argument(
        "--log-level",
        default=os.getenv("DET_LOG_LEVEL", "INFO"),
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        help="Set the logging level",
    )
    parser.add_argument(
        "--experiment-config",
        type=json_file_arg,
        default=os.getenv("DET_EXPERIMENT_CONFIG", {}),
        help="Experiment config (JSON-formatted file)",
    )
    parser.add_argument(
        "--delete",
        type=json_file_arg,
        default=os.getenv("DET_DELETE", []),
        help="Checkpoints to delete (JSON-formatted file)",
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

    storage_config = args.experiment_config["checkpoint_storage"]
    logging.info("Using checkpoint storage: {}".format(storage_config))

    manager = storage.build(storage_config, container_path=constants.SHARED_FS_CONTAINER_PATH)

    delete_checkpoints(manager, args.delete["checkpoints"], dry_run=args.dry_run)


if __name__ == "__main__":
    main(sys.argv[1:])
