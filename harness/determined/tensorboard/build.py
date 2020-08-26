import os
import pathlib
from typing import Any, Dict, Optional

import determined as det
from determined.tensorboard import base, gcs, hdfs, s3, shared
from determined_common.storage.shared import _full_storage_path


def get_sync_path(env: det.EnvContext) -> pathlib.Path:
    return pathlib.Path(
        env.det_cluster_id,
        "tensorboard",
        "experiment",
        env.det_experiment_id,
        "trial",
        env.det_trial_id,
    )


def get_rank_if_horovod_process_else_return_zero() -> Optional[int]:
    return int(os.getenv("HOROVOD_RANK", 0))


def get_base_path(checkpoint_config: Dict[str, Any], manager: bool = False) -> pathlib.Path:
    rank = get_rank_if_horovod_process_else_return_zero()

    if checkpoint_config.get("base_path"):
        return pathlib.Path(checkpoint_config["base_path"]).joinpath("tensorboard")

    if manager or rank == 0:
        # In a distributed training job the manager should monitor the chief
        # trials logs and ignore all other trials.
        return pathlib.Path("/", "tmp", "tensorboard")

    return pathlib.Path("/", "tmp", f"tensorboard-{rank}")


def build(
    env: det.EnvContext, checkpoint_config: Dict[str, Any], container_path: Optional[str] = None
) -> base.TensorboardManager:
    """
    Return a tensorboard manager defined by the value of the `type` key in
    the configuration dictionary. Throws a `TypeError` if no tensorboard manager
    with `type` is defined.

    container_path, if set, will replace the host_path when determining the storage_path for the
    SharedFSTensorboardManager.
    """
    type_name = checkpoint_config.get("type")

    if not type_name:
        raise TypeError("Missing 'type' parameter of storage configuration")

    if not isinstance(type_name, str):
        raise TypeError("`type` parameter of storage configuration must be a string")

    base_path = get_base_path(checkpoint_config, manager=True)
    sync_path = get_sync_path(env)

    if type_name == "shared_fs":
        host_path = checkpoint_config["host_path"]
        storage_path = checkpoint_config.get("storage_path")
        return shared.SharedFSTensorboardManager(
            _full_storage_path(host_path, storage_path, container_path),
            base_path,
            sync_path,
        )

    elif type_name == "gcs":
        return gcs.GCSTensorboardManager(checkpoint_config["bucket"], base_path, sync_path)

    elif type_name == "s3":
        return s3.S3TensorboardManager(
            checkpoint_config["bucket"],
            checkpoint_config.get("access_key", None),
            checkpoint_config.get("secret_key", None),
            checkpoint_config.get("endpoint_url", None),
            base_path,
            sync_path,
        )

    # Return the base_path.TensorboardManager for known but unsupported storage
    # backends. This will result in a noop action when the workload_manager
    # attempts to sync the tfevent files to persistent storage.
    elif type_name == "hdfs":
        return hdfs.HDFSTensorboardManager(
            checkpoint_config["hdfs_url"],
            checkpoint_config["hdfs_path"],
            checkpoint_config.get("user"),
            base_path,
            sync_path,
        )

    else:
        raise TypeError(f"Unknown storage type: {type_name}")
