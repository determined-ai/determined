import logging
import os
import tempfile
from contextlib import contextmanager
from typing import Generator, Optional, Tuple

from hdfs.client import InsecureClient

from determined_common.storage.base import Storable, StorageManager, StorageMetadata


class HDFSStorageManager(StorageManager):
    """
    Store and load storages from HDFS.
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

    def store(self, store_data: Storable, storage_id: str = "") -> StorageMetadata:
        metadata = super().store(store_data, storage_id)

        logging.info("Uploading storage {} to HDFS".format(metadata.storage_id))

        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        result = self.client.upload(metadata, storage_dir)

        logging.info("Uploaded storage {} to HDFS path {}".format(metadata.storage_id, result))

        self._remove_checkpoint_directory(metadata.storage_id)
        return metadata

    def restore(self, storage_data: Storable, metadata: StorageMetadata) -> None:
        logging.info("Downloading storage {} from HDFS".format(metadata.storage_id))

        self.client.download(metadata.storage_id, self._base_path, overwrite=True)

        super().restore(storage_data, metadata)

        self._remove_checkpoint_directory(metadata.storage_id)

    @contextmanager
    def store_path(self, storage_id: str = "") -> Generator[Tuple[str, str], None, None]:
        with super().store_path(storage_id) as (storage_id, path):
            yield (storage_id, path)

        metadata = StorageMetadata(storage_id, StorageManager._list_directory(path))

        try:
            logging.info("Uploading storage {} to HDFS".format(storage_id))
            result = self.client.upload(metadata, path)

            logging.info("Uploaded storage {} to HDFS path {}".format(storage_id, result))
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @contextmanager
    def restore_path(self, metadata: StorageMetadata) -> Generator[str, None, None]:
        logging.info("Downloading storage {} from HDFS".format(metadata.storage_id))

        self.client.download(metadata.storage_id, self._base_path, overwrite=True)

        try:
            with super().restore_path(metadata) as path:
                yield path
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    def delete(self, metadata: StorageMetadata) -> None:
        logging.info("Deleting storage {} from HDFS".format(metadata.storage_id))
        self.client.delete(metadata.storage_id, recursive=True)
