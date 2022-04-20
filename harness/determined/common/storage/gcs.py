import logging
import os
import tempfile
from typing import Optional, Union, no_type_check

import requests.exceptions
import urllib3.exceptions

from determined import errors
from determined.common import storage, util


class GCSStorageManager(storage.CloudStorageManager):
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
        import google.cloud.storage

        self.client = google.cloud.storage.Client()
        self.bucket = self.client.bucket(bucket)

    @no_type_check
    @util.preserve_random_state
    def upload(self, src: Union[str, os.PathLike], dst: str) -> None:
        src = os.fspath(src)
        logging.info(f"Uploading to GCS: {dst}")
        for rel_path in sorted(self._list_directory(src)):
            blob_name = f"{dst}/{rel_path}"
            blob = self.bucket.blob(blob_name)

            logging.debug(f"Uploading to GCS: {blob_name}")

            from google.api_core import exceptions, retry

            retry_network_errors = retry.Retry(
                retry.if_exception_type(
                    ConnectionError,
                    exceptions.ServerError,
                    urllib3.exceptions.ProtocolError,
                    requests.exceptions.ConnectionError,
                )
            )

            if rel_path.endswith("/"):
                # Create empty blobs for subdirectories. This ensures
                # that empty directories are checkpointed correctly.
                retry_network_errors(blob.upload_from_string)(b"")
            else:
                abs_path = os.path.join(src, rel_path)
                retry_network_errors(blob.upload_from_filename)(abs_path)

    @util.preserve_random_state
    def download(self, src: str, dst: Union[str, os.PathLike]) -> None:
        dst = os.fspath(dst)
        logging.info(f"Downloading {src} from GCS")
        found = False
        # Listing blobs with prefix set and no delimiter is equivalent to a recursive listing.  If
        # you include a `delimiter="/"` you will get only the file-like blobs inside of a
        # directory-like blob.
        for blob in self.bucket.list_blobs(prefix=src):
            found = True
            _dst = os.path.join(dst, os.path.relpath(blob.name, src))
            dst_dir = os.path.dirname(_dst)
            if not os.path.exists(dst_dir):
                os.makedirs(dst_dir, exist_ok=True)

            # Only create empty directory for keys that end with "/".
            # See `upload` method for more context.
            if blob.name.endswith("/"):
                os.makedirs(_dst, exist_ok=True)
                continue

            logging.debug(f"Downloading from GCS: {blob.name}")

            blob.download_to_filename(_dst)

        if not found:
            raise errors.CheckpointNotFound(f"Did not find checkpoint {src} in GCS")

    @util.preserve_random_state
    def delete(self, storage_id: str) -> None:
        logging.info(f"Deleting checkpoint {storage_id} from GCS")

        storage_prefix = storage_id
        for blob in self.bucket.list_blobs(prefix=storage_prefix):
            logging.debug(f"Deleting {blob.name} from GCS")
            blob.delete()
