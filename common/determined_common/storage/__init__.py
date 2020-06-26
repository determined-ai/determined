import copy
import os
import uuid
from typing import Any, Dict, Optional, Type

from determined_common.check import check_eq, check_in, check_type

from .base import StorageManager, StorageMetadata
from .gcs import GCSStorageManager
from .hdfs import HDFSStorageManager
from .s3 import S3StorageManager
from .shared import SharedFSStorageManager

__all__ = [
    "GCSStorageManager",
    "StorageManager",
    "StorageMetadata",
    "S3StorageManager",
    "SharedFSStorageManager",
]


_STORAGE_MANAGERS = {
    "gcs": GCSStorageManager,
    "s3": S3StorageManager,
    "shared_fs": SharedFSStorageManager,
    "hdfs": HDFSStorageManager,
}  # type: Dict[str, Type[StorageManager]]


def build(config: Dict[str, Any], container_path: Optional[str]) -> StorageManager:
    """
    Return a checkpoint manager defined by the value of the `type` key in
    the configuration dictionary. Throws a `TypeError` if no storage manager
    with `type` is defined.
    """
    check_in("type", config, "Missing 'type' parameter of storage configuration")

    # Make a deep copy of the config because we are removing items to
    # pass to the constructor of the `StorageManager`.
    config = copy.deepcopy(config)
    identifier = config.pop("type")
    check_type(identifier, str, "`type` parameter of storage configuration must be a string")

    try:
        subclass = _STORAGE_MANAGERS[identifier]
    except KeyError:
        raise TypeError("Unknown storage type: {}".format(identifier))

    # Remove configurations that should not be directly passed to
    # subclasses. Keeping these would result in the subclass __init__()
    # function failing to a TypeError with an unexpected keyword.
    config.pop("save_experiment_best", None)
    config.pop("save_trial_best", None)
    config.pop("save_trial_latest", None)

    # For shared_fs maintain backwards compatibility by folding old keys into
    # storage_path.
    if identifier == "shared_fs" and "storage_path" not in config:
        if "tensorboard_path" in config:
            config["storage_path"] = config.get("tensorboard_path", None)
        else:
            config["storage_path"] = config.get("checkpoint_path", None)

    config.pop("tensorboard_path", None)
    config.pop("checkpoint_path", None)

    try:
        return subclass.from_config(config, container_path)
    except TypeError as e:
        raise TypeError(
            "Failed to instantiate {} checkpoint storage: {}".format(identifier, str(e))
        )


def validate_manager(manager: StorageManager) -> None:
    """
    Validate that the StorageManager can be written to, restored from, and
    deleted from. Throws an exception if any of the operations fail.
    """

    class Validater:
        """
        Validater for reading and writing a UUID to a checkpoint. The
        UUID saved must match the UUID that is loaded.
        """

        def __init__(self) -> None:
            self.uuid = str(uuid.uuid4())

        def save(self, storage_dir: str) -> None:
            os.makedirs(storage_dir)
            with open(os.path.join(storage_dir, "VALIDATE.txt"), "w") as fp:
                fp.write(self.uuid)

        def load(self, storage_dir: str) -> None:
            with open(os.path.join(storage_dir, "VALIDATE.txt"), "r") as fp:
                check_eq(fp.read(), self.uuid, "Unable to properly load from storage")

    validater = Validater()
    with manager.store_path() as (storage_id, path):
        validater.save(path)
        metadata = StorageMetadata(storage_id, StorageManager._list_directory(path))
    with manager.restore_path(metadata) as path:
        validater.load(path)
    manager.delete(metadata)


def validate_config(config: Dict[str, Any], container_path: Optional[str]) -> None:
    validate_manager(build(config, container_path))
