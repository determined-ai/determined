import datetime
import logging
import os
import posixpath
import urllib
from typing import Any, Callable, Dict, Generator, List

from .base import Fetcher

logger = logging.getLogger("determined.tensorboard.gcs")


class GCSFetcher(Fetcher):
    def __init__(self, storage_config: Dict[str, Any], storage_paths: List[str], local_dir: str):
        import google.cloud.storage

        self.client = google.cloud.storage.Client()
        self.bucket_name = str(storage_config["bucket"])
        self.bucket = self.client.bucket(self.bucket_name)

        self.local_dir = local_dir
        self.storage_paths = storage_paths
        self._file_records = {}  # type: Dict[str, datetime.datetime]

    def _list(self, storage_path: str) -> Generator[str, None, None]:
        logger.debug(
            f"Listing keys in bucket: '{self.bucket_name}' with storage_path: '{storage_path}'"
        )
        prefix = urllib.parse.urlparse(storage_path).path.lstrip("/")
        blobs = self.client.list_blobs(self.bucket_name, prefix=prefix)

        for blob in blobs:
            filepath, mtime = blob.name, blob.updated
            prev_mtime = self._file_records.get(filepath)
            if prev_mtime is not None and prev_mtime >= mtime:
                continue
            self._file_records[filepath] = mtime
            yield blob.name

    def _fetch(self, filepath: str, new_file_callback: Callable) -> None:
        local_path = posixpath.join(self.local_dir, self.bucket_name, filepath)
        dir_path = os.path.dirname(local_path)
        os.makedirs(dir_path, exist_ok=True)

        self.bucket.blob(filepath).download_to_filename(local_path)

        logger.debug(f"Downloaded GCS file to local: {local_path}")
        new_file_callback()
