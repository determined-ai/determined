import logging
from typing import Any, Optional

import boto3

from determined.tensorboard import base
from determined_common import util


class S3TensorboardManager(base.TensorboardManager):
    """
    Store and load tf event logs from s3.
    """

    def __init__(
        self,
        bucket: str,
        access_key: Optional[str],
        secret_key: Optional[str],
        endpoint_url: Optional[str],
        *args: Any,
        **kwargs: Any,
    ) -> None:
        super().__init__(*args, **kwargs)
        self.bucket = bucket
        self.client = boto3.client(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )

    @util.preserve_random_state
    def sync(self) -> None:
        for path in self.to_sync():
            key_name = str(self.sync_path.joinpath(path.relative_to(self.base_path)))

            url = f"s3://{self.bucket}/{key_name}"
            logging.debug(f"Uploading {path} to {url}")

            self.client.upload_file(str(path), self.bucket, key_name)
            self._synced_event_sizes[path] = path.stat().st_size
