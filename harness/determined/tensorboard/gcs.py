import logging
import os
import pathlib
from typing import Any, List, Optional, no_type_check

from requests import exceptions as request_exceptions
from urllib3 import exceptions as url_exceptions

from determined.common import storage
from determined.tensorboard import base

logger = logging.getLogger("determined.tensorboard.gcs")


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
        self.prefix = storage.normalize_prefix(prefix)

    def get_storage_prefix(self, storage_id: pathlib.Path) -> str:
        return os.path.join(self.prefix, storage_id)

    @no_type_check
    def _sync_impl(self, path_info_list: List[base.PathUploadInfo]) -> None:
        for path_info in path_info_list:
            path = path_info.path
            mangled_relative_path = path_info.mangled_relative_path
            mangled_path = self.sync_path.joinpath(mangled_relative_path)
            to_path = self.get_storage_prefix(mangled_path)

            from google.api_core import exceptions, retry

            retry_network_errors = retry.Retry(
                retry.if_exception_type(
                    ConnectionError,
                    exceptions.ServerError,
                    exceptions.TooManyRequests,
                    url_exceptions.ProtocolError,
                    request_exceptions.ConnectionError,
                )
            )

            blob = self.bucket.blob(to_path)

            logger.debug(f"Uploading {path} to GCS: {to_path}")
            retry_network_errors(blob.upload_from_filename)(str(path))

    def delete(self) -> None:
        prefix_path = self.get_storage_prefix(self.sync_path)
        self.bucket.delete_blobs(blobs=list(self.bucket.list_blobs(prefix=prefix_path)))
