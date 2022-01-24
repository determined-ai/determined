import datetime
import logging
import os
import urllib.parse
from typing import Any, Dict, Generator, List, Tuple

from .base import Fetcher

logger = logging.getLogger(__name__)


class S3Fetcher(Fetcher):
    def __init__(
        self, storage_config: Dict[str, Any], storage_paths: List[str], local_dir: str
    ) -> None:
        import boto3

        from determined.common.storage import boto3_credential_manager

        boto3_credential_manager.initialize_boto3_credential_providers()
        self.s3 = boto3.resource(
            "s3",
            endpoint_url=storage_config.get("endpoint_url"),
            aws_access_key_id=storage_config.get("access_key"),
            aws_secret_access_key=storage_config.get("secret_key"),
        )
        self.client = self.s3.meta.client
        self.bucket_name = str(storage_config["bucket"])

        self.local_dir = local_dir
        self.storage_paths = storage_paths
        self._file_records = {}  # type: Dict[str, datetime.datetime]

    def _find_keys(self, prefix: str) -> Generator[Tuple[str, datetime.datetime], None, None]:
        logger.debug(f"Listing keys in bucket '{self.bucket_name}' with prefix '{prefix}'")
        prefix = urllib.parse.urlparse(prefix).path.lstrip("/")

        paginator = self.client.get_paginator("list_objects_v2")
        page_iterator = paginator.paginate(Bucket=self.bucket_name, Prefix=prefix)
        page_count = 0
        for page in page_iterator:
            page_count += 1
            for s3_obj in page.get("Contents", []):
                yield (s3_obj["Key"], s3_obj["LastModified"])
        if page_count > 1:
            logger.info(f"Fetched {page_count} number of list_objects_v2 pages")

    def fetch_new(self) -> int:
        """Fetches changes files found in storage paths to local disk."""
        new_files = []

        # Look at all files in our storage location.
        for storage_path in self.storage_paths:
            for filepath, mtime in self._find_keys(storage_path):
                prev_mtime = self._file_records.get(filepath)

                if prev_mtime is not None and prev_mtime >= mtime:
                    continue

                new_files.append(filepath)
                self._file_records[filepath] = mtime

        # Download the new or updated files.
        for filepath in new_files:
            local_path = os.path.join(self.local_dir, self.bucket_name, filepath)
            dir_path = os.path.dirname(local_path)
            os.makedirs(dir_path, exist_ok=True)

            with open(local_path, "wb") as local_file:
                self.client.download_fileobj(self.bucket_name, filepath, local_file)

            logger.debug(f"Downloaded file to local: {local_path}")

        return len(new_files)
