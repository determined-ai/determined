import os
from typing import Any, Dict, Optional

from determined_common.check import check_gt, check_true, check_type
from determined_common.storage.base import StorageManager


def _full_storage_dir(host_path: str, container_path: str, storage_path: Optional[str]) -> str:
    """
    Return the full path to the storage base directory.
    """
    if storage_path is not None:
        abs_path = os.path.normpath(os.path.join(host_path, storage_path))
        check_true(
            abs_path.startswith(host_path), "storage path must be a subdirectory of host path."
        )
        storage_path = os.path.relpath(abs_path, host_path)

    if storage_path is not None:
        return os.path.join(container_path, storage_path)

    return container_path


class SharedFSStorageManager(StorageManager):
    """
    Store and load storages from a shared file system. Each agent should
    have this shared file system mounted in the same location defined by the
    `host_path`.
    """

    def __init__(
        self,
        host_path: str,
        container_path: str = "/determined_shared_fs",
        storage_path: Optional[str] = None,
        propagation: str = "rprivate",
    ) -> None:
        super().__init__(_full_storage_dir(host_path, container_path, storage_path))
        check_type(host_path, str, "`host_path` must be a str.")
        check_true(os.path.isabs(host_path), "`host_path` must be an absolute path.")
        check_type(container_path, str, "`container_path` must be a str.")
        check_true(os.path.isabs(container_path), "`container_path` must be an absolute path.")
        check_type(propagation, str, "`propagation` must be a str.")
        check_gt(len(host_path), 0, "`host_path` must be non-empty.")
        check_gt(len(container_path), 0, "`container_path` must be non-empty.")
        self.host_path = host_path
        self.container_path = container_path
        self.propagation = propagation

    def get_mount_config(self) -> Dict[str, Any]:
        return {
            "Type": "bind",
            "Source": self.host_path,
            "Target": self.container_path,
            "ReadOnly": False,
            "BindOptions": {"Propagation": self.propagation},
        }
