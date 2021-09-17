import contextlib
import logging
import os
import tempfile
from typing import Iterator, Optional

from hdfs.client import InsecureClient

from determined.common import util
from determined.common.storage.base import StorageManager


class HDFSStorageManager(StorageManager):
    """
    Store and load checkpoints from HDFS.
    """

    def __init__(
        self,
        hdfs_url: str,
        hdfs_path: str,
        user: Optional[str] = None,
        temp_dir: Optional[str] = None,
    ) -> None:
        super().__init__(temp_dir if temp_dir is not None else tempfile.gettempdir())

        self.hdfs_url = hdfs_url
        self.hdfs_path = hdfs_path
        self.user = user

        self.client = InsecureClient(self.hdfs_url, root=self.hdfs_path, user=self.user)

    @util.preserve_random_state
    def post_store_path(self, storage_id: str, storage_dir: str) -> None:
        """post_store_path uploads the checkpoint to hdfs and deletes the original files."""
        try:
            logging.info(f"Uploading storage {storage_id} to HDFS")
            result = self.client.upload(storage_id, storage_dir)

            logging.info(f"Uploaded storage {storage_id} to HDFS path {result}")
        finally:
            self._remove_checkpoint_directory(storage_id)

    @contextlib.contextmanager
    def restore_path(self, storage_id: str) -> Iterator[str]:
        logging.info(f"Downloading storage {storage_id} from HDFS")

        self.download(storage_id)

        try:
            yield os.path.join(self._base_path, storage_id)
        finally:
            self._remove_checkpoint_directory(storage_id)

    @util.preserve_random_state
    def download(self, storage_id: str) -> None:
        self.client.download(storage_id, self._base_path, overwrite=True)

    @util.preserve_random_state
    def delete(self, storage_id: str) -> None:
        logging.info(f"Deleting storage {storage_id} from HDFS")
        self.client.delete(storage_id, recursive=True)
