import os
import pathlib
import urllib
from typing import Any, Dict, Optional, Union

from determined.common.storage.shared import _full_storage_path
from determined.tensorboard import azure, base, gcs, s3, shared


def get_sync_path(cluster_id: str, experiment_id: str, trial_id: str) -> pathlib.Path:
    return pathlib.Path(
        get_experiment_sync_path(cluster_id, experiment_id),
        "trial",
        trial_id,
    )


def get_experiment_sync_path(cluster_id: str, experiment_id: str) -> pathlib.Path:
    return pathlib.Path(
        cluster_id,
        "tensorboard",
        "experiment",
        experiment_id,
    )


def get_rank_if_horovod_process_else_return_zero() -> Optional[int]:
    return int(os.getenv("HOROVOD_RANK", 0))


def get_base_path(checkpoint_config: Dict[str, Any]) -> pathlib.Path:
    allocation_id = os.environ.get("DET_ALLOCATION_ID", "")
    rank = get_rank_if_horovod_process_else_return_zero()

    if checkpoint_config.get("base_path"):
        base_path = pathlib.Path(checkpoint_config["base_path"])
    else:
        base_path = pathlib.Path("/", "tmp")

    return base_path.joinpath(f"tensorboard-{allocation_id}-{rank}")


def _shortcut_to_config(shortcut: str) -> Dict[str, Any]:
    p: urllib.parse.ParseResult = urllib.parse.urlparse(shortcut)
    if any((p.params, p.query, p.fragment)):
        raise ValueError(f'Malformed checkpoint_storage string "{shortcut}"')

    scheme = p.scheme.lower()

    if scheme in ["", "file"]:
        return {
            "type": "shared_fs",
            "host_path": p.path,
        }
    elif scheme in ["s3", "gs"]:
        bucket = p.netloc
        prefix = p.path.lstrip("/")
        storage_type = {
            "s3": "s3",
            "gs": "gcs",
        }[scheme]

        return {
            "type": storage_type,
            "bucket": bucket,
            "prefix": prefix,
        }
    else:
        raise NotImplementedError(
            "tensorboard only supports shared_fs, s3, and gs " "shortcuts at the moment"
        )


def build(
    cluster_id: str,
    experiment_id: str,
    trial_id: Optional[str],
    checkpoint_config: Union[Dict[str, Any], str],
    container_path: Optional[str] = None,
    async_upload: bool = True,
    sync_on_close: bool = True,
) -> base.TensorboardManager:
    """
    Return a tensorboard manager defined by the value of the `type` key in
    the configuration dictionary. Throws a `TypeError` if no tensorboard manager
    with `type` is defined.

    container_path, if set, will replace the host_path when determining the storage_path for the
    SharedFSTensorboardManager.
    """
    if isinstance(checkpoint_config, str):
        checkpoint_config = _shortcut_to_config(checkpoint_config)

    type_name = checkpoint_config.get("type")

    if not type_name:
        raise TypeError("Missing 'type' parameter of storage configuration")

    if not isinstance(type_name, str):
        raise TypeError("`type` parameter of storage configuration must be a string")

    base_path = get_base_path(checkpoint_config)

    if trial_id:
        sync_path = get_sync_path(cluster_id, experiment_id, trial_id)
    else:
        sync_path = get_experiment_sync_path(cluster_id, experiment_id)

    if type_name == "shared_fs":
        host_path = checkpoint_config["host_path"]
        storage_path = checkpoint_config.get("storage_path")
        return shared.SharedFSTensorboardManager(
            _full_storage_path(host_path, storage_path, container_path),
            base_path,
            sync_path,
            async_upload=async_upload,
            sync_on_close=sync_on_close,
        )

    elif type_name == "gcs":
        return gcs.GCSTensorboardManager(
            checkpoint_config["bucket"],
            checkpoint_config.get("prefix", None),
            base_path,
            sync_path,
            async_upload=async_upload,
            sync_on_close=sync_on_close,
        )

    elif type_name == "s3":
        return s3.S3TensorboardManager(
            checkpoint_config["bucket"],
            checkpoint_config.get("access_key", None),
            checkpoint_config.get("secret_key", None),
            checkpoint_config.get("endpoint_url", None),
            checkpoint_config.get("prefix", None),
            base_path,
            sync_path,
            async_upload=async_upload,
            sync_on_close=sync_on_close,
        )

    elif type_name == "azure":
        if not checkpoint_config.get("connection_string") and checkpoint_config.get("access_url"):
            raise ValueError(
                """At least one of [connection_string, account_url] must be specified for Azure
                 Tensorboard Manager, but none were."""
            )
        return azure.AzureTensorboardManager(
            checkpoint_config["container"],
            checkpoint_config.get("connection_string", None),
            checkpoint_config.get("access_url", None),
            checkpoint_config.get("credential", None),
            base_path,
            sync_path,
            async_upload=async_upload,
            sync_on_close=sync_on_close,
        )

    else:
        raise TypeError(f"Unknown storage type: {type_name}")
