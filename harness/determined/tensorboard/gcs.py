import logging
import os
import pathlib
from typing import Any, Callable, Optional, no_type_check

import requests.exceptions
import urllib3.exceptions

from determined.common import util
from determined.common.storage.s3 import normalize_prefix
from determined.tensorboard import base

logger = logging.getLogger("determined.tensorboard")


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

    def get_storage_prefix(self, storage_id: pathlib.Path) -> str:
        return os.path.join(self.prefix, storage_id)

    @no_type_check
    @util.preserve_random_state
    def sync(
        self,
        selector: Callable[[pathlib.Path], bool] = lambda _: True,
        mangler: Callable[[pathlib.Path, int], pathlib.Path] = lambda p, __: p,
        rank: int = 0,
    ) -> None:
        for path in self.to_sync(selector):
            relative_path = path.relative_to(self.base_path)
            mangled_relative_path = mangler(relative_path, rank)
            mangled_path = self.sync_path.joinpath(mangled_relative_path)
            to_path = self.get_storage_prefix(mangled_path)

            from google.api_core import exceptions, retry

            retry_network_errors = retry.Retry(
                retry.if_exception_type(
                    ConnectionError,
                    exceptions.ServerError,
                    urllib3.exceptions.ProtocolError,
                    requests.exceptions.ConnectionError,
                )
            )

            blob = self.bucket.blob(to_path)

            logger.debug(f"Uploading {path} to GCS: {to_path}")
            retry_network_errors(blob.upload_from_filename)(str(path))

    def delete(self) -> None:
        prefix_path = self.get_storage_prefix(self.sync_path)
        self.bucket.delete_blobs(blobs=list(self.bucket.list_blobs(prefix=prefix_path)))
