import datetime
import logging
import os
from typing import Any, Callable, Dict, Generator, List
from urllib import parse

from determined.tensorboard.fetchers import base

logger = logging.getLogger("determined.tensorboard.azure")


class AzureFetcher(base.Fetcher):
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

    def _list(self, storage_path: str) -> Generator[str, None, None]:
        logger.debug(
            f"Listing keys in container: '{self.container_name}'"
            " with storage_path: '{storage_path}'"
        )
        container = self.client.get_container_client(self.container_name)
        prefix = parse.urlparse(storage_path).path.lstrip("/")

        blobs = container.list_blobs(name_starts_with=prefix)
        for blob in blobs:
            filepath, mtime = blob["name"], blob["last_modified"]
            prev_mtime = self._file_records.get(filepath)

            if prev_mtime is not None and prev_mtime >= mtime:
                continue
            self._file_records[filepath] = mtime
            yield filepath

    def _fetch(self, filepath: str, new_file_callback: Callable) -> None:
        local_path = os.path.join(self.local_dir, self.container_name, filepath)
        dir_path = os.path.dirname(local_path)
        os.makedirs(dir_path, exist_ok=True)

        with open(local_path, "wb") as local_file:
            stream = self.client.get_blob_client(self.container_name, filepath).download_blob()
            stream.readinto(local_file)

        logger.debug(f"Downloaded file to local: {local_path}")
        new_file_callback()
