import logging
import os
from pathlib import Path
from typing import Any, Optional

from determined.common import util
from determined.common.storage.s3 import normalize_prefix
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

    def __init__(self, bucket: str, prefix: Optional[str], *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        import google.cloud.storage

        self.client = google.cloud.storage.Client()
        self.bucket = self.client.bucket(bucket)
        self.prefix = normalize_prefix(prefix)

    def get_storage_prefix(self, storage_id: Path) -> str:
        return os.path.join(self.prefix, storage_id)

    @util.preserve_random_state
    def sync(self) -> None:
        for path in self.to_sync():
            blob_name = self.sync_path.joinpath(path.relative_to(self.base_path))
            to_path = self.get_storage_prefix(blob_name)
            blob = self.bucket.blob(to_path)

            logging.debug(f"Uploading {path} to GCS: {to_path}")
            blob.upload_from_filename(str(path))

    def delete(self) -> None:
        prefix_path = self.get_storage_prefix(self.sync_path)
        self.bucket.delete_blobs(blobs=list(self.bucket.list_blobs(prefix=prefix_path)))
