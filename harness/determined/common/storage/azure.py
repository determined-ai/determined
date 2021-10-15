import contextlib
import logging
import os
import tempfile
from typing import Iterator, Optional

from determined import errors
from determined.common import util
from determined.common.storage.base import StorageManager

from .azure_client import AzureStorageClient

import posixpath  # isort:skip


class AzureStorageManager(StorageManager):
    """
    Store and load checkpoints from Azure Blob Storage.

    Checkpoints are stored as a collection of Block Blobs,
    with each block blob corresponding to one checkpoint resouce.
    """

    def __init__(
        self,
        container: str,
        connection_string: Optional[str] = None,
        account_url: Optional[str] = None,
        credential: Optional[str] = None,
        temp_dir: Optional[str] = None,
    ) -> None:
        super().__init__(temp_dir if temp_dir is not None else tempfile.gettempdir())
        self.client = AzureStorageClient(container, connection_string, account_url, credential)
        self.container = container if not container.endswith("/") else container[:-1]

    def post_store_path(self, storage_id: str, storage_dir: str) -> None:
        """post_store_path uploads the checkpoint to Azure Blob Storage and deletes the original
        files.
        """
        try:
            logging.info(f"Uploading checkpoint {storage_id} to Azure Blob Storage.")
            self.upload(storage_id, storage_dir)
        finally:
            self._remove_checkpoint_directory(storage_id)

    @contextlib.contextmanager
    def restore_path(self, storage_id: str) -> Iterator[str]:
        storage_dir = os.path.join(self._base_path, storage_id)
        os.makedirs(storage_dir, exist_ok=True)

        logging.info(f"Downloading checkpoint {storage_id} from Azure Blob Storage")
        self.download(storage_id, storage_dir)

        try:
            yield os.path.join(self._base_path, storage_id)
        finally:
            self._remove_checkpoint_directory(storage_id)

    @util.preserve_random_state
    def upload(self, storage_id: str, storage_dir: str) -> None:
        storage_prefix = storage_id
        for rel_path in sorted(self._list_directory(storage_dir)):
            if rel_path.endswith("/"):
                continue
            # Use posixpath so that we always use forward slashes, even on Windows.
            container_blob = posixpath.join(self.container, storage_prefix, rel_path)
            blob_dir, blob_base = posixpath.split(container_blob)
            abs_path = os.path.join(storage_dir, rel_path)
            logging.debug(f"Uploading blob {blob_base} to container {blob_dir}.")
            self.client.put(blob_dir, blob_base, abs_path)

    @util.preserve_random_state
    def download(self, storage_id: str, storage_dir: str) -> None:
        storage_prefix = storage_id
        found = False
        for blob in self.client.list_files(self.container, file_prefix=storage_prefix):
            found = True
            dst = os.path.join(storage_dir, os.path.relpath(blob, storage_prefix))
            dst_dir = os.path.dirname(dst)
            if not os.path.exists(dst_dir):
                os.makedirs(dst_dir, exist_ok=True)

            # Only create empty directory for keys that end with "/".
            if blob.endswith("/"):
                continue

            # Use posixpath so that we always use forward slashes, even on Windows.
            container_blob = posixpath.join(self.container, blob)
            blob_dir, blob_base = posixpath.split(container_blob)
            self.client.get(blob_dir, blob_base, dst)

        if not found:
            raise errors.CheckpointNotFound(
                f"Did not find checkpoint {storage_id} in Azure Blob Storage"
            )

    @util.preserve_random_state
    def delete(self, storage_id: str) -> None:
        storage_prefix = storage_id
        logging.info(f"Deleting checkpoint {storage_id} from Azure Blob Storage")

        files = self.client.list_files(self.container, file_prefix=storage_prefix)
        self.client.delete_files(self.container, files)
