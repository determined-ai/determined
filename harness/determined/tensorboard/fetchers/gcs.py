import datetime
import logging
import os
import posixpath
import urllib
from typing import Any, Dict, Generator, List, Tuple

from .base import Fetcher

logger = logging.getLogger(__name__)


class GCSFetcher(Fetcher):
    def __init__(self, storage_config: Dict[str, Any], storage_paths: List[str], local_dir: str):
        import google.cloud.storage

        self.client = google.cloud.storage.Client()
        self.bucket_name = str(storage_config["bucket"])

        self.local_dir = local_dir
        self.storage_paths = storage_paths
        self._file_records = {}  # type: Dict[str, datetime.datetime]

    def _list(self, prefix: str) -> Generator[Tuple[str, datetime.datetime], None, None]:
        logger.debug(f"Listing keys in bucket '{self.bucket_name}' with '{prefix}'")
        prefix = urllib.parse.urlparse(prefix).path.lstrip("/")
        blobs = self.client.list_blobs(self.bucket_name, prefix=prefix)

        for blob in blobs:
            yield (blob.name, blob.updated)

    def fetch_new(self) -> int:
        new_files = []
        bucket = self.client.bucket(self.bucket_name)

        # Look at all files in our storage location.
        for storage_path in self.storage_paths:
            logger.debug(f"Looking at path: {storage_path}")

            for filepath, mtime in self._list(storage_path):
                prev_mtime = self._file_records.get(filepath)

                if prev_mtime is not None and prev_mtime >= mtime:
                    continue

                new_files.append(filepath)
                self._file_records[filepath] = mtime

        # Download the new or updated files.
        for filepath in new_files:
            local_path = posixpath.join(self.local_dir, self.bucket_name, filepath)
            dir_path = os.path.dirname(local_path)
            os.makedirs(dir_path, exist_ok=True)

            bucket.blob(filepath).download_to_filename(local_path)

            logger.debug(f"Downloaded file to local: {local_path}")

        return len(new_files)
