import os
from typing import Any, Dict, Optional

import determined as det
from determined.common import constants
from determined.common.storage.base import StorageManager
from determined.common.storage.file import FileStorageManager


def _full_storage_path(
    on_cluster: bool,
    host_path: str,
    storage_path: Optional[str] = None,
) -> str:
    """
    Return the full path to the storage_path, either as a subdirectory of the host_path in the
    host environment if on_cluster is True, or as a subdirectory of the container_path otherwise.
    """
    if not os.path.isabs(host_path):
        raise ValueError("`host_path` must be an absolute path.")

    container_path = constants.SHARED_FS_CONTAINER_PATH

    if storage_path is None:
        return container_path if on_cluster else host_path

    # Note that os.path.join() will just return storage_path when it is absolute.
    abs_path = os.path.normpath(os.path.join(host_path, storage_path))
    if not abs_path.startswith(host_path):
        raise ValueError(
            f"storage path ({storage_path}) must be a subdirectory of host path ({host_path})."
        )
    storage_path = os.path.relpath(abs_path, host_path)

    return os.path.join(container_path if on_cluster else host_path, storage_path)


class SharedFSStorageManager(StorageManager):
    """
    Store and load checkpoints from a shared file system. Each agent should have this shared file
    system mounted in the same location defined by the `host_path`.

    SharedFSStorageManager is not actually an implementation of a StorageManager; it only implements
    .from_config() and can choose one of two possible base_path values for a FileStorageManager.
    """

    @classmethod
    def from_config(cls, config: Dict[str, Any]) -> "StorageManager":
        """
        SharedFSStorageManager.from_config() actually just decides if we are inside the container or
        not, and builds a base StorageManager accordingly.
        """
        allowed_keys = {"host_path", "storage_path", "container_path", "propagation"}
        extra_keys = allowed_keys.difference(set(config.keys()))
        if extra_keys:
            raise ValueError(f"extra key(s) in shared_fs config: {sorted(extra_keys)}")
        if "host_path" not in config:
            raise ValueError(f"shared_fs config is missing host_path: {config}")
        # Ignore legacy configuration values propagation and container_path.
        on_cluster = det.get_cluster_info() is not None
        base_path = _full_storage_path(on_cluster, config["host_path"], config.get("storage_path"))

        return FileStorageManager(base_path)
