import contextlib
import logging
import os
import tempfile
from typing import Iterator, Optional

import google.api_core.exceptions
import requests.exceptions
import urllib3.exceptions
from google.api_core import retry
from google.cloud import storage

from determined.common import util
from determined.common.storage.base import StorageManager, StorageMetadata

retry_network_errors = retry.Retry(
    retry.if_exception_type(
        ConnectionError,
        google.api_core.exceptions.ServerError,
        urllib3.exceptions.ProtocolError,
        requests.exceptions.ConnectionError,
    )
)


class GCSStorageManager(StorageManager):
    """
    Store and load checkpoints on GCS. Although GCS is similar to S3, some
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

    def post_store_path(self, storage_id: str, storage_dir: str, metadata: StorageMetadata) -> None:
        """post_store_path uploads the checkpoint to gcs and deletes the original files."""
        try:
            logging.info("Uploading checkpoint {} to GCS".format(storage_id))
            self.upload(metadata, storage_dir)
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @contextlib.contextmanager
    def restore_path(self, metadata: StorageMetadata) -> Iterator[str]:
        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        os.makedirs(storage_dir, exist_ok=True)

        logging.info("Downloading checkpoint {} from GCS".format(metadata.storage_id))
        self.download(metadata, storage_dir)

        try:
            yield os.path.join(self._base_path, metadata.storage_id)
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @util.preserve_random_state
    def upload(self, metadata: StorageMetadata, storage_dir: str) -> None:
        for rel_path in metadata.resources.keys():
            blob_name = "{}/{}".format(metadata.storage_id, rel_path)
            blob = self.bucket.blob(blob_name)

            logging.debug("Uploading to GCS: {}".format(blob_name))

            if rel_path.endswith("/"):
                # Create empty blobs for subdirectories. This ensures
                # that empty directories are checkpointed correctly.
                retry_network_errors(blob.upload_from_string)(b"")
            else:
                abs_path = os.path.join(storage_dir, rel_path)
                retry_network_errors(blob.upload_from_filename)(abs_path)

    @util.preserve_random_state
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

    @util.preserve_random_state
    def delete(self, metadata: StorageMetadata) -> None:
        logging.info("Deleting checkpoint {} from GCS".format(metadata.storage_id))

        for rel_path in metadata.resources.keys():
            logging.debug("Deleting {} from GCS".format(rel_path))
            blob_name = "{}/{}".format(metadata.storage_id, rel_path)
            blob = self.bucket.blob(blob_name)
            blob.delete()
