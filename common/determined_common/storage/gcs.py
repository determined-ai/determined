import logging
import os
import tempfile
from contextlib import contextmanager
from typing import Generator, Optional, Tuple

from google.cloud import storage

from determined_common.storage.base import Storable, StorageManager, StorageMetadata


class GCSStorageManager(StorageManager):
    """
    Store and load Storables on GCS. Although GCS is similar to S3, some
    S3 APIs are not supported on GCS and vice versa. Moreover, Google
    recommends using the google-storage-python library to access GCS,
    rather than the boto library we use to access S3 -- boto uses
    various S3 features that are not supported by GCS.

    Batching is supported by the GCS API for deletion, however it is not used because
    of observed request failures. Batching is not used for uploading
    or downloading files, because the GCS API does not support it. Upload/download
    performance could be improved by using multiple clients in a multithreaded fashion.

    Authentication is currently only supported via the "Application
    Default Credentials" method in GCP [1]. Typical configuration:
    ensure your VM runs in a service account that has sufficient
    permissions to read/write/delete from the GCS bucket where
    checkpoints will be stored (this only works when running in GCE).
    """

    def __init__(self, bucket: str, temp_dir: Optional[str] = None) -> None:
        super().__init__(temp_dir if temp_dir is not None else tempfile.gettempdir())
        self.client = storage.Client()
        self.bucket = self.client.bucket(bucket)

    def store(self, store_data: Storable, storage_id: str = "") -> StorageMetadata:
        metadata = super().store(store_data, storage_id)

        logging.info("Uploading checkpoint {} to GCS".format(metadata.storage_id))
        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        self.upload(metadata, storage_dir)

        self._remove_checkpoint_directory(metadata.storage_id)

        return metadata

    def restore(self, checkpoint: Storable, metadata: StorageMetadata) -> None:
        logging.info("Downloading checkpoint {} from GCS".format(metadata.storage_id))

        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        os.makedirs(storage_dir, exist_ok=True)
        self.download(metadata, storage_dir)
        super().restore(checkpoint, metadata)

        self._remove_checkpoint_directory(metadata.storage_id)

    @contextmanager
    def store_path(self, storage_id: str = "") -> Generator[Tuple[str, str], None, None]:
        with super().store_path(storage_id) as (storage_id, path):
            yield (storage_id, path)

        metadata = StorageMetadata(storage_id, StorageManager._list_directory(path))

        try:
            logging.info("Uploading checkpoint {} to GCS".format(storage_id))
            self.upload(metadata, path)
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @contextmanager
    def restore_path(self, metadata: StorageMetadata) -> Generator[str, None, None]:
        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        os.makedirs(storage_dir, exist_ok=True)

        logging.info("Downloading checkpoint {} from GCS".format(metadata.storage_id))
        self.download(metadata, storage_dir)

        try:
            with super().restore_path(metadata) as path:
                yield path
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    def upload(self, metadata: StorageMetadata, storage_dir: str) -> None:
        for rel_path in metadata.resources.keys():
            blob_name = "{}/{}".format(metadata.storage_id, rel_path)
            blob = self.bucket.blob(blob_name)

            logging.debug("Uploading to GCS: {}".format(blob_name))

            if rel_path.endswith("/"):
                # Create empty blobs for subdirectories. This ensures
                # that empty directories are checkpointed correctly.
                blob.upload_from_string(b"")
            else:
                abs_path = os.path.join(storage_dir, rel_path)
                blob.upload_from_filename(abs_path)

    def download(self, metadata: StorageMetadata, storage_dir: str) -> None:
        for rel_path in metadata.resources.keys():
            abs_path = os.path.join(storage_dir, rel_path)
            os.makedirs(os.path.dirname(abs_path), exist_ok=True)

            # Only create empty directory for keys that end with "/".
            # See `upload` method for more context.
            if rel_path.endswith("/"):
                continue

            blob_name = "{}/{}".format(metadata.storage_id, rel_path)
            blob = self.bucket.blob(blob_name)

            logging.debug("Downloading from GCS: {}".format(blob_name))

            blob.download_to_filename(abs_path)

    def delete(self, metadata: StorageMetadata) -> None:
        logging.info("Deleting checkpoint {} from GCS".format(metadata.storage_id))

        for rel_path in metadata.resources.keys():
            logging.debug("Deleting {} from GCS".format(rel_path))
            blob_name = "{}/{}".format(metadata.storage_id, rel_path)
            blob = self.bucket.blob(blob_name)
            blob.delete()
