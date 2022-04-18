import contextlib
import os
import pathlib
import shutil
from typing import Any, Dict, Iterator, Optional, Union

from determined import errors
from determined.common import check
from determined.common.storage.base import StorageManager


def _full_storage_path(
    host_path: str,
    storage_path: Optional[str] = None,
    container_path: Optional[str] = None,
) -> str:
    """
    Return the full path to the storage_path, either as a subdirectory of the host_path in the
    host environment, where container_path must be None, or as a subdirectory of the container_path
    when in the container enviornment, where container_path must not be None.
    """
    check.true(os.path.isabs(host_path), "`host_path` must be an absolute path.")

    if storage_path is None:
        return host_path if container_path is None else container_path

    # Note that os.path.join() will just return storage_path when it is absolute.
    abs_path = os.path.normpath(os.path.join(host_path, storage_path))
    check.true(abs_path.startswith(host_path), "storage path must be a subdirectory of host path.")
    storage_path = os.path.relpath(abs_path, host_path)

    return os.path.join(host_path if container_path is None else container_path, storage_path)


class SharedFSStorageManager(StorageManager):
    """
    Store and load storages from a shared file system. Each agent should
    have this shared file system mounted in the same location defined by the
    `host_path`.
    """

    @classmethod
    def from_config(cls, config: Dict[str, Any], container_path: Optional[str]) -> "StorageManager":
        allowed_keys = {"host_path", "storage_path", "container_path", "propagation"}
        for key in config.keys():
            check.is_in(key, allowed_keys, "extra key in shared_fs config")
        check.is_in("host_path", config, "shared_fs config is missing host_path")
        # Ignore legacy configuration values propagation and container_path.
        base_path = _full_storage_path(
            config["host_path"], config.get("storage_path"), container_path
        )
        return cls(base_path)

    def post_store_path(self, src: str, dst: str) -> None:
        """
        Nothing to clean up after writing directly to shared_fs.
        """
        pass

    @contextlib.contextmanager
    def restore_path(self, src: str) -> Iterator[pathlib.Path]:
        """
        Prepare a local directory exposing the checkpoint. Do some simple checks to make sure the
        configuration seems reasonable.
        """
        check.true(
            os.path.exists(self._base_path),
            f"Storage directory does not exist: {self._base_path}. Please verify that you are "
            "using the correct configuration value for checkpoint_storage.host_path",
        )
        storage_dir = os.path.join(self._base_path, src)
        if not os.path.exists(storage_dir):
            raise errors.CheckpointNotFound(f"Did not find checkpoint {src} in shared_fs storage")
        yield pathlib.Path(storage_dir)

    def delete(self, tgt: str) -> None:
        """
        Delete the stored data from persistent storage.
        """
        storage_dir = os.path.join(self._base_path, tgt)

        if not os.path.exists(storage_dir):
            raise errors.CheckpointNotFound(f"Storage directory does not exist: {storage_dir}")
        if not os.path.isdir(storage_dir):
            raise errors.CheckpointNotFound(f"Storage path is not a directory: {storage_dir}")
        shutil.rmtree(storage_dir, ignore_errors=False)

    def upload(self, src: Union[str, os.PathLike], dst: str) -> None:
        src = os.fspath(src)
        shutil.copytree(src, os.path.join(self._base_path, dst))

    def download(self, src: str, dst: Union[str, os.PathLike]) -> None:
        dst = os.fspath(dst)
        try:
            shutil.copytree(os.path.join(self._base_path, src), dst)
        except FileNotFoundError:
            raise errors.CheckpointNotFound(
                f"Did not find checkpoint {src} in shared_fs storage"
            ) from None
