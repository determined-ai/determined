import datetime
import logging
import os
from typing import Any, Callable, Dict, Generator, List
from urllib import parse

from determined.tensorboard.fetchers import base

logger = logging.getLogger("determined.tensorboard.s3")


class S3Fetcher(base.Fetcher):
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

    def _list(self, storage_path: str) -> Generator[str, None, None]:
        logger.debug(
            f"Listing keys in bucket: '{self.bucket_name}' with storage_path: '{storage_path}'"
        )
        prefix = parse.urlparse(storage_path).path.lstrip("/")

        paginator = self.client.get_paginator("list_objects_v2")
        page_iterator = paginator.paginate(Bucket=self.bucket_name, Prefix=prefix)
        page_count = 0
        for page in page_iterator:
            page_count += 1
            for s3_obj in page.get("Contents", []):
                filepath, mdatetime = s3_obj["Key"], s3_obj["LastModified"]
                prev_mdatetime = self._file_records.get(filepath)
                if prev_mdatetime is not None and prev_mdatetime >= mdatetime:
                    continue
                self._file_records[filepath] = mdatetime
                yield filepath
        if page_count > 1:
            logger.info(f"Fetched {page_count} number of list_objects_v2 pages")

    def _fetch(self, filepath: str, new_file_callback: Callable) -> None:
        local_path = os.path.join(self.local_dir, self.bucket_name, filepath)
        dir_path = os.path.dirname(local_path)
        os.makedirs(dir_path, exist_ok=True)

        with open(local_path, "wb") as local_file:
            self.client.download_fileobj(self.bucket_name, filepath, local_file)

        logger.debug(f"Downloaded s3 file to local: {local_path}")
        new_file_callback()
