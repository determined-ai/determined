import datetime
import logging
import os
import urllib
from typing import Any, Dict, Generator, List, Tuple

from .base import Fetcher

logger = logging.getLogger(__name__)


class AzureFetcher(Fetcher):
    def __init__(self, storage_config: Dict[str, Any], storage_paths: List[str], local_dir: str):
        from azure.storage import blob

        connection_string = storage_config.get("connection_string")
        container = storage_config.get("container")
        account_url = storage_config.get("account_url")
        credential = storage_config.get("credential")

        if storage_config.get("connection_string"):
            self.client = blob.BlobServiceClient.from_connection_string(connection_string)
        elif account_url:
            self.client = blob.BlobServiceClient(account_url, credential)
        else:
            raise ValueError("Either 'container_string' or 'account_url' must be specified.")

        if container is None:
            raise ValueError("'container' must be specified.")

        self.container_name = container if not container.endswith("/") else container[:-1]

        self.local_dir = local_dir
        self.storage_paths = storage_paths
        self._file_records = {}  # type: Dict[str, datetime.datetime]

    def _list(self, prefix: str) -> Generator[Tuple[str, datetime.datetime], None, None]:
        logger.debug(f"Listing keys in container '{self.container_name}' with '{prefix}'")
        container = self.client.get_container_client(self.container_name)
        prefix = urllib.parse.urlparse(prefix).path.lstrip("/")

        blobs = container.list_blobs(name_starts_with=prefix)
        for blob in blobs:
            yield (blob["name"], blob["last_modified"])

    def fetch_new(self) -> int:
        new_files = []

        # Look at all files in our storage location.
        for storage_path in self.storage_paths:
            for filepath, mtime in self._list(storage_path):
                prev_mtime = self._file_records.get(filepath)

                if prev_mtime is not None and prev_mtime >= mtime:
                    continue

                new_files.append(filepath)
                self._file_records[filepath] = mtime

        # Download the new or updated files.
        for filepath in new_files:
            local_path = os.path.join(self.local_dir, self.container_name, filepath)

            dir_path = os.path.dirname(local_path)
            os.makedirs(dir_path, exist_ok=True)

            with open(local_path, "wb") as local_file:
                stream = self.client.get_blob_client(self.container_name, filepath).download_blob()
                stream.readinto(local_file)

            logger.debug(f"Downloaded file to local: {local_path}")

        return len(new_files)
