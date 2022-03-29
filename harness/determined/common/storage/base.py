import abc
import contextlib
import os
import pathlib
from typing import Any, Dict, Iterator, Optional, Union

from determined.common.check import check_gt, check_true, check_type


class StorageManager(metaclass=abc.ABCMeta):
    """
    Abstract base class for storage managers. Storage managers need to
    support five operations:

       1.  Upload an existing directory to storage
       2.  Download from storage to a target directory
       3.  Provide a path to write to, in order to upload
       4.  Provide a path to read from, after a download
       5.  Delete a directory in storage

    Advanced methods 3. and 4.  allow shared_fs to do do optimized zero-copy checkpointing.
    Cloud-based implementations can subclass the CloudStorageManager, which will define 3. and 4. in
    terms of 1. and 2.

    Configuration for storage managers is represented as a dictionary of key value pairs. The
    primary key in the dictionary is the `type` defining which storage backend to use. Additional
    keys may be required to instantiate some implementations of the storage manager.
    """

    def __init__(self, base_path: str) -> None:
        check_type(base_path, str)
        check_gt(len(base_path), 0)
        self._base_path = base_path

    @classmethod
    def from_config(cls, config: Dict[str, Any], container_path: Optional[str]) -> "StorageManager":
        """from_config() just calls __init__() unless it is overridden in a subclass."""
        return cls(**config)

    @abc.abstractmethod
    def post_store_path(self, src: str, dst: str) -> None:
        """
        post_store_path is a hook that will be called after store_path(). Subclasess of
        StorageManager should override this in order to customize the behavior of store_path().
        """
        pass

    @contextlib.contextmanager
    def store_path(self, dst: str) -> Iterator[pathlib.Path]:
        """
        Prepare a local directory to be written to the storage backend.

        This base implementation creates the dst directory, but subclasses whose storage
        backends are in remote places are responsible for uploading the data after the files are
        created and deleting the temporary dst directory.
        """

        # Set umask to 0 in order that the storage dir allows future containers of any owner to
        # create new checkpoints. Administrators wishing to control the permissions more
        # specifically should just create the storage path themselves; this will not interfere.
        old_umask = os.umask(0)
        os.makedirs(self._base_path, exist_ok=True, mode=0o777)
        # Restore the original umask.
        os.umask(old_umask)

        os.makedirs(self._base_path, exist_ok=True)
        storage_dir = os.path.join(self._base_path, dst)

        yield pathlib.Path(storage_dir)
        check_true(os.path.exists(storage_dir), "Checkpoint did not create a storage directory")

        self.post_store_path(storage_dir, dst)

    @abc.abstractmethod
    @contextlib.contextmanager
    def restore_path(self, storage_id: str) -> Iterator[pathlib.Path]:
        """
        restore_path should prepare a checkpoint, yield the path to the checkpoint, and do any
        necessary cleanup afterwards. Subclasess of StorageManager must implement this.
        """
        pass

    @abc.abstractmethod
    def upload(self, src: Union[str, os.PathLike], dst: str) -> None:
        pass

    @abc.abstractmethod
    def download(self, src: str, dst: Union[str, os.PathLike]) -> None:
        pass

    @abc.abstractmethod
    def delete(self, tgt: str) -> None:
        """
        Delete the stored data from persistent storage.
        """
        pass

    @staticmethod
    def _list_directory(root: Union[str, os.PathLike]) -> Dict[str, int]:
        """
        Returns a dict mapping path names to file sizes for all files
        and subdirectories in the directory `root`. Directories are
        signified by a trailing "/". Returned path names are relative to
        `root`; directories are included but have a file size of 0.
        """
        root = os.fspath(root)
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
