import abc
import contextlib
import os
import shutil
import uuid
from typing import Any, Dict, Iterator, Optional, Tuple

from determined.common.check import check_gt, check_true, check_type


class StorageManager:
    """
    Abstract base class for storage managers. Storage managers need to
    support three operations: creating, loading, and deleting storages.

    Configuration for storage managers is represented as a dictionary of key
    value pairs. The primary key in the dictionary is the `type` defining
    which storage backend to use. Additional keys may be required to
    instantiate some implementations of the storage manager.
    """

    def __init__(self, base_path: str) -> None:
        check_type(base_path, str)
        check_gt(len(base_path), 0)
        self._base_path = base_path

    @classmethod
    def from_config(cls, config: Dict[str, Any], container_path: Optional[str]) -> "StorageManager":
        """from_config() just calls __init__() unless it is overridden in a subclass."""
        return cls(**config)

    def post_store_path(self, storage_id: str, storage_dir: str) -> None:
        """
        post_store_path is a hook that will be called after store_path(). Subclasess of
        StorageManager should override this in order to customize the behavior of store_path().
        """
        pass

    @contextlib.contextmanager
    def store_path(self, storage_id: str = "") -> Iterator[Tuple[str, str]]:
        """
        Prepare a local directory that will become a checkpoint.

        This base implementation creates the temporary directory and chooses a
        random checkpoint ID, but subclasses whose storage backends are in
        remote places are responsible for uploading the data after the files are
        created and deleting the temporary checkpoint directory.
        """

        if storage_id == "":
            storage_id = str(uuid.uuid4())

        # Set umask to 0 in order that the storage dir allows future containers of any owner to
        # create new checkpoints. Administrators wishing to control the permissions more
        # specifically should just create the storage path themselves; this will not interfere.
        old_umask = os.umask(0)
        os.makedirs(self._base_path, exist_ok=True, mode=0o777)
        # Restore the original umask.
        os.umask(old_umask)

        os.makedirs(self._base_path, exist_ok=True)
        storage_dir = os.path.join(self._base_path, storage_id)

        yield (storage_id, storage_dir)
        check_true(os.path.exists(storage_dir), "Checkpoint did not create a storage directory")

        self.post_store_path(storage_id, storage_dir)

    @abc.abstractmethod
    @contextlib.contextmanager
    def restore_path(self, storage_id: str) -> Iterator[str]:
        """
        restore_path should prepare a checkpoint, yield the path to the checkpoint, and do any
        necessary cleanup afterwards. Subclasess of StorageManager must implement this.
        """
        pass

    def delete(self, storage_id: str) -> None:
        """
        Delete the stored data from persistent storage.
        """
        storage_dir = os.path.join(self._base_path, storage_id)

        check_true(
            os.path.exists(storage_dir), "Storage directory does not exist: {}".format(storage_dir)
        )
        check_true(
            os.path.isdir(storage_dir), "Storage path is not a directory: {}".format(storage_dir)
        )

        self._remove_checkpoint_directory(storage_id, ignore_errors=False)

    def _remove_checkpoint_directory(self, storage_id: str, ignore_errors: bool = True) -> None:
        """
        Recursively delete a checkpoint directory from the local filesystem.
        This is primarily useful when cleaning up temporary files after saving
        or restoring a checkpoint from some remote storage backend, but it is
        also useful for deleting from persistent storage when the storage
        backend is a shared file system.
        """
        storage_dir = os.path.join(self._base_path, storage_id)
        shutil.rmtree(storage_dir, ignore_errors)

    @staticmethod
    def _list_directory(root: str) -> Dict[str, int]:
        """
        Returns a dict mapping path names to file sizes for all files
        and subdirectories in the directory `root`. Directories are
        signified by a trailing "/". Returned path names are relative to
        `root`; directories are included but have a file size of 0.
        """
        check_true(os.path.isdir(root), "{} must be an extant directory".format(root))
        result = {}
        for cur_path, sub_dirs, files in os.walk(root):
            for d in sub_dirs:
                abs_path = os.path.join(cur_path, d)
                rel_path = os.path.relpath(abs_path, root) + "/"
                result[rel_path] = 0

            for f in files:
                abs_path = os.path.join(cur_path, f)
                rel_path = os.path.relpath(abs_path, root)
                result[rel_path] = os.path.getsize(abs_path)

        return result
