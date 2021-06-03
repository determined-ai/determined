import logging
from typing import Any

from google.cloud import storage

from determined.common import util
from determined.tensorboard import base


class GCSTensorboardManager(base.TensorboardManager):
    """
    Store and load tf event logs from gcs.

    Authentication is currently only supported via the "Application
    Default Credentials" method in GCP [1]. Typical configuration:
    ensure your VM runs in a service account that has sufficient
    permissions to read/write/delete from the GCS bucket where
    checkpoints will be stored (this only works when running in GCE).
    """

    def __init__(self, bucket: str, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self.client = storage.Client()
        self.bucket = self.client.bucket(bucket)

    @util.preserve_random_state
    def sync(self) -> None:
        for path in self.to_sync():
            blob_name = str(self.sync_path.joinpath(path.relative_to(self.base_path)))
            blob = self.bucket.blob(blob_name)
            logging.debug(f"Uploading to GCS: {blob_name}")

            blob.upload_from_filename(str(path))

    def delete(self) -> None:
        self.bucket.delete_blobs(blobs=list(self.bucket.list_blobs(prefix=self.sync_path)))
