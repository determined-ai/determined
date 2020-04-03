import logging
import os
import shutil
import uuid
from abc import ABCMeta, abstractmethod
from contextlib import contextmanager
from typing import Any, Dict, Generator, Optional, Tuple

from determined_common.check import check_gt, check_not_none, check_true, check_type
from determined_common.util import sizeof_fmt


class StorageMetadata:
    def __init__(
        self, storage_id: str, resources: Dict[str, int], labels: Optional[Dict[str, str]] = None
    ) -> None:
        check_gt(len(storage_id), 0, "Invalid storage ID")
        if labels is None:
            labels = {}
        self.storage_id = storage_id
        self.resources = resources
        self.labels = labels

    def __json__(self) -> Dict[str, Any]:
        return {"uuid": self.storage_id, "resources": self.resources, "labels": self.labels}

    def __str__(self) -> str:
        return "<storage {}, labels {}>".format(self.storage_id, self.labels)

    def __repr__(self) -> str:
        return "<storage {}, labels {}, resources {}>".format(
            self.storage_id, self.labels, self.resources
        )

    @staticmethod
    def from_json(record: Dict[str, Any]) -> "StorageMetadata":
        check_not_none(record["uuid"], "Storage ID is undefined")
        check_not_none(record["resources"], "Resources are undefined")
        return StorageMetadata(record["uuid"], record["resources"], record.get("labels"))


class Storable(metaclass=ABCMeta):
    """
    An interface for objects (like trials) that support being stored.
    This interface can be passed to a `StorageManager` to be saved to or
    loaded from persistent storage.
    """

    @abstractmethod
    def save(self, storage_dir: str) -> None:
        """
        Persist the object to the storage directory. The storage
        directory is not created prior to saving the storage.
        """
        raise NotImplementedError()

    @abstractmethod
    def load(self, storage_dir: str) -> None:
        """
        Load the object from the storage directory.
        """
        raise NotImplementedError()


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

    def store(self, store_data: Storable, storage_id: str = "") -> StorageMetadata:
        """
        Save the object to the backing persistent storage.
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

        storage_dir = os.path.join(self._base_path, storage_id)
        store_data.save(storage_dir)

        check_true(os.path.exists(storage_dir), "Checkpoint did not create a storage directory")

        directory_list = StorageManager._list_directory(storage_dir)

        logging.info(
            "Storing checkpoint {} ({})".format(
                storage_id, sizeof_fmt(sum(directory_list.values()))
            )
        )

        return StorageMetadata(storage_id, directory_list)

    def restore(self, storage_data: Storable, metadata: StorageMetadata) -> None:
        """
        Load the object from the backing persistent storage.
        """
        storage_dir = os.path.join(self._base_path, metadata.storage_id)

        check_true(
            os.path.exists(storage_dir),
            "Storage directory does not exist: {}. Please verify "
            "that you are using the correct configuration value for "
            "checkpoint_storage.host_path.".format(storage_dir),
        )
        check_true(
            os.path.isdir(storage_dir), "Checkpoint path is not a directory: {}".format(storage_dir)
        )

        storage_data.load(storage_dir)

    @contextmanager
    def store_path(self, storage_id: str = "") -> Generator[Tuple[str, str], None, None]:
        """
        Prepare a local directory that will become a checkpoint.

        This base implementation creates the temporary directory and chooses a
        random checkpoint ID, but subclasses whose storage backends are in
        remote places are responsible for uploading the data after the files are
        created and deleting the temporary checkpoint directory.
        """

        if storage_id == "":
            storage_id = str(uuid.uuid4())

        os.makedirs(self._base_path, exist_ok=True)
        storage_dir = os.path.join(self._base_path, storage_id)

        yield (storage_id, storage_dir)
        check_true(os.path.exists(storage_dir), "Checkpoint did not create a storage directory")

    @contextmanager
    def restore_path(self, metadata: StorageMetadata) -> Generator[str, None, None]:
        """
        Prepare a local directory exposing the checkpoint.

        This base implementation does some simple checks to make sure the
        checkpoint has been prepared properly, but subclasses whose storage
        backends are in remote places are responsible for downloading the
        checkpoint before calling this method and deleting the temporary
        checkpoint directory after it is no longer useful.
        """

        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        check_true(
            os.path.exists(storage_dir),
            "Storage directory does not exist: {}. Please verify "
            "that you are using the correct configuration value for "
            "checkpoint_storage.host_path and "
            "tensorboard_storage.host_path.".format(storage_dir),
        )
        check_true(
            os.path.isdir(storage_dir), "Checkpoint path is not a directory: {}".format(storage_dir)
        )
        yield storage_dir

    def delete(self, metadata: StorageMetadata) -> None:
        """
        Delete the stored data from persistent storage.
        """
        storage_dir = os.path.join(self._base_path, metadata.storage_id)

        check_true(
            os.path.exists(storage_dir), "Storage directory does not exist: {}".format(storage_dir)
        )
        check_true(
            os.path.isdir(storage_dir), "Storage path is not a directory: {}".format(storage_dir)
        )

        self._remove_checkpoint_directory(metadata.storage_id, ignore_errors=False)

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
