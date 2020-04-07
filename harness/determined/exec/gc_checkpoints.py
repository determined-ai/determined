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
import determined_common.storage
from determined_common.storage import StorageMetadata


def delete_checkpoints(
    config: Dict[str, Any], to_delete: List[Dict[str, Any]], validate: bool, dry_run: bool
) -> None:
    """
    Delete some of the checkpoints associated with a single
    experiment. `config` is the experiment config of the target
    experiment; `to_delete` is a list of two-element dicts,
    {"uuid": str, "resources": List[str]}.
    """
    storage = config["checkpoint_storage"]
    logging.info("Using checkpoint storage: {}".format(storage))

    if validate:
        logging.info("Validating checkpoint storage...")
        determined_common.storage.validate(storage)
        logging.info("Checkpoint storage validation successful")
    else:
        logging.info("Skipping checkpoint validation")

    manager = determined_common.storage.build(storage)

    logging.info("Deleting {} checkpoints".format(len(to_delete)))

    for record in to_delete:
        metadata = StorageMetadata.from_json(record)
        if not dry_run:
            logging.info("Deleting checkpoint {}".format(metadata))
            manager.delete(metadata)
        else:
            logging.info("Dry run: deleting checkpoint {}".format(metadata.storage_id))

    logging.info("Finished deleting {} checkpoints".format(len(to_delete)))


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
        type=json.loads,
        default=os.getenv("DET_EXPERIMENT_CONFIG", {}),
        help="Experiment config (JSON-formatted string)",
    )
    parser.add_argument(
        "--delete",
        type=json.loads,
        default=os.getenv("DET_DELETE", []),
        help="Checkpoints to delete (JSON-formatted string)",
    )
    parser.add_argument(
        "--validate",
        action="store_true",
        default=("DET_VALIDATE" in os.environ),
        help="Validate the checkpoint storage can be deleted "
        "from before starting deletion process",
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

    delete_checkpoints(
        args.experiment_config,
        args.delete["checkpoints"],
        validate=args.validate,
        dry_run=args.dry_run,
    )


if __name__ == "__main__":
    main(sys.argv[1:])
