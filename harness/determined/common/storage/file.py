import contextlib
import os
from typing import Iterator

from determined import errors
from determined.common.storage.base import StorageManager


class FileStorageManager(StorageManager):
    """
    FileStorageManager stores artifacts files.

    This StorageManager is not exposed via the normal checkpoint_storage config mechanism.
    Presently, the only way to create one is to create one explicitly, or to use the
    SharedFSStorageManager's .from_config() using a shared_fs checkpoint storage config.
    """

    def post_store_path(self, storage_id: str, storage_dir: str) -> None:
        """
        FileStorageManager has nothing special to do at this point.
        """

    @contextlib.contextmanager
    def restore_path(self, storage_id: str) -> Iterator[str]:
        """
        FileStorageManager's restore_ath just verifies that the checkoint exists.
        """
        storage_dir = os.path.join(self._base_path, storage_id)
        if not os.path.exists(storage_dir):
            raise errors.CheckpointNotFound(
                f"Did not find checkpoint {storage_id} in shared_fs storage"
            )

        yield storage_dir

    def delete(self, storage_id: str) -> None:
        """
        Delete a checkpoint from the filesystem.
        """
        storage_dir = os.path.join(self._base_path, storage_id)

        if not os.path.exists(storage_dir):
            raise ValueError("Storage directory does not exist: {}".format(storage_dir))

        if not os.path.isdir(storage_dir):
            raise ValueError("Storage path is not a directory: {}".format(storage_dir))

        self._remove_checkpoint_directory(storage_id, ignore_errors=False)
