import logging
import os
import pathlib
from typing import Any, Callable, Optional

from determined.common import util
from determined.common.storage.s3 import normalize_prefix
from determined.tensorboard import base

logger = logging.getLogger("determined.tensorboard")


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
        prefix: Optional[str],
        *args: Any,
        **kwargs: Any,
    ) -> None:
        super().__init__(*args, **kwargs)
        import boto3

        self.bucket = bucket
        self.client = boto3.client(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )
        self.resource = boto3.resource(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )

        self.prefix = normalize_prefix(prefix)

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
            tbd_filename = str(mangled_path)
            key_name = os.path.join(self.prefix, tbd_filename)

            url = f"s3://{self.bucket}/{key_name}"
            logger.debug(f"Uploading {path} to {url}")

            self.client.upload_file(str(path), self.bucket, key_name)

    def delete(self) -> None:
        prefix_path = os.path.join(self.prefix, self.sync_path)
        self.resource.Bucket(self.bucket).objects.filter(Prefix=prefix_path).delete()
