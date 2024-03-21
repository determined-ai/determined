"""
The entrypoint for the GC checkpoints job container.
"""

import argparse
import json
import logging
import os
import sys
from typing import Any, Dict, List

import urllib3

import determined as det
from determined import errors, tensorboard
from determined.common import constants, storage
from determined.common.api import authentication, bindings, certs

logger = logging.getLogger("determined")


def patch_checkpoints(storage_ids_to_resources: Dict[str, Dict[str, int]]) -> None:
    info = det.ClusterInfo._from_file()
    if info is None:
        info = det.ClusterInfo._from_env()
        info._to_file()

    cert = certs.default_load(info.master_url)
    # With backoff retries for 64 seconds
    sess = authentication.login_with_cache(info.master_url, cert=cert).with_retry(
        urllib3.util.retry.Retry(total=6, backoff_factor=0.5)
    )

    checkpoints = []
    for storage_id, resources in storage_ids_to_resources.items():
        checkpoints.append(
            bindings.v1PatchCheckpoint(
                uuid=storage_id,
                resources=bindings.PatchCheckpointOptionalResources(
                    resources=resources,  # type: ignore
                ),
            )
        )

    bindings.patch_PatchCheckpoints(
        sess, body=bindings.v1PatchCheckpointsRequest(checkpoints=checkpoints)
    )


def delete_checkpoints(
    manager: storage.StorageManager, to_delete: List[str], globs: List[str], dry_run: bool
) -> Dict[str, Dict[str, int]]:
    """
    Delete some of the checkpoints associated with a single experiment.
    """
    logger.info(f"Deleting {len(to_delete)} checkpoints")

    storage_id_to_resources: Dict[str, Dict[str, int]] = {}
    for storage_id in to_delete:
        if not dry_run:
            logger.info(f"Deleting checkpoint {storage_id}")
            try:
                storage_id_to_resources[storage_id] = manager.delete(storage_id, globs)
            except errors.CheckpointNotFound as e:
                logger.warn(e)
        else:
            logger.info(f"Dry run: deleting checkpoint {storage_id}")

    return storage_id_to_resources


def delete_tensorboards(manager: tensorboard.TensorboardManager, dry_run: bool = False) -> None:
    """
    Delete all Tensorboards associated with a single experiment.
    """
    if dry_run:
        logger.info(f"Dry run: deleting Tensorboards for {manager.sync_path}")
        return

    try:
        manager.delete()
    except errors.CheckpointNotFound as e:
        logger.warn(e)
    logger.info(f"Finished deleting Tensorboards for {manager.sync_path}")


def json_file_arg(val: str) -> Any:
    with open(val) as f:
        return json.load(f)


def main(argv: List[str]) -> None:
    parser = argparse.ArgumentParser(description="Determined checkpoint GC")

    parser.add_argument(
        "--version",
        action="version",
        version=f"Determined checkpoint GC, version {det.__version__}",
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
        type=json_file_arg,
        default=os.getenv("DET_DELETE", []),
        help="Checkpoints to delete (JSON-formatted file)",
    )
    parser.add_argument(
        "--globs",
        type=json_file_arg,
        default=os.getenv("DET_GLOB", []),
        help="Glob list to match against checkpoint list (JSON-formatted file)",
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

    logger.info(f"Determined checkpoint GC, version {det.__version__}")

    storage_config = args.storage_config
    masked_config = json.dumps(det.util.mask_checkpoint_storage(storage_config))
    logger.info(f"Using checkpoint storage: {masked_config}")

    storage_ids = [s.strip() for s in args.delete]
    globs = [s.strip() for s in args.globs]

    manager = storage.build(storage_config, container_path=constants.SHARED_FS_CONTAINER_PATH)

    if len(storage_ids) > 0:
        storage_ids_to_resources = delete_checkpoints(
            manager, storage_ids, globs, dry_run=args.dry_run
        )
        patch_checkpoints(storage_ids_to_resources)

    if args.delete_tensorboards:
        tb_manager = tensorboard.build(
            os.environ["DET_CLUSTER_ID"],
            args.experiment_id,
            None,
            storage_config,
            container_path=constants.SHARED_FS_CONTAINER_PATH,
            async_upload=False,
            sync_on_close=False,
        )
        delete_tensorboards(tb_manager, dry_run=args.dry_run)


if __name__ == "__main__":
    main(sys.argv[1:])
