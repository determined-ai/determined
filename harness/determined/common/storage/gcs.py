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

from determined import errors
from determined.common import util
from determined.common.storage.base import StorageManager

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

    def post_store_path(self, storage_id: str, storage_dir: str) -> None:
        """post_store_path uploads the checkpoint to gcs and deletes the original files."""
        try:
            logging.info(f"Uploading checkpoint {storage_id} to GCS")
            self.upload(storage_id, storage_dir)
        finally:
            self._remove_checkpoint_directory(storage_id)

    @contextlib.contextmanager
    def restore_path(self, storage_id: str) -> Iterator[str]:
        storage_dir = os.path.join(self._base_path, storage_id)
        os.makedirs(storage_dir, exist_ok=True)

        logging.info(f"Downloading checkpoint {storage_id} from GCS")
        self.download(storage_id, storage_dir)

        try:
            yield os.path.join(self._base_path, storage_id)
        finally:
            self._remove_checkpoint_directory(storage_id)

    @util.preserve_random_state
    def upload(self, storage_id: str, storage_dir: str) -> None:
        storage_prefix = storage_id
        for rel_path in sorted(self._list_directory(storage_dir)):
            blob_name = f"{storage_prefix}/{rel_path}"
            blob = self.bucket.blob(blob_name)

            logging.debug(f"Uploading to GCS: {blob_name}")

            if rel_path.endswith("/"):
                # Create empty blobs for subdirectories. This ensures
                # that empty directories are checkpointed correctly.
                retry_network_errors(blob.upload_from_string)(b"")
            else:
                abs_path = os.path.join(storage_dir, rel_path)
                retry_network_errors(blob.upload_from_filename)(abs_path)

    @util.preserve_random_state
    def download(self, storage_id: str, storage_dir: str) -> None:
        storage_prefix = storage_id
        found = False
        # Listing blobs with prefix set and no delimiter is equivalent to a recursive listing.  If
        # you include a `delimiter="/"` you will get only the file-like blobs inside of a
        # directory-like blob.
        for blob in self.bucket.list_blobs(prefix=storage_prefix):
            found = True
            dst = os.path.join(storage_dir, os.path.relpath(blob.name, storage_prefix))
            dst_dir = os.path.dirname(dst)
            if not os.path.exists(dst_dir):
                os.makedirs(dst_dir, exist_ok=True)

            # Only create empty directory for keys that end with "/".
            # See `upload` method for more context.
            if blob.name.endswith("/"):
                os.makedirs(dst, exist_ok=True)
                continue

            logging.debug(f"Downloading from GCS: {blob.name}")

            blob.download_to_filename(dst)

        if not found:
            raise errors.CheckpointNotFound(f"Did not find checkpoint {storage_id} in GCS")

    @util.preserve_random_state
    def delete(self, storage_id: str) -> None:
        logging.info(f"Deleting checkpoint {storage_id} from GCS")

        storage_prefix = storage_id
        for blob in self.bucket.list_blobs(prefix=storage_prefix):
            logging.debug(f"Deleting {blob.name} from GCS")
            blob.delete()
