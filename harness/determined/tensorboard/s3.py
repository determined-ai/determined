import logging
import os
from typing import Any, List, Optional

from determined.common import storage
from determined.tensorboard import base

logger = logging.getLogger("determined.tensorboard.s3")


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

        self.prefix = storage.normalize_prefix(prefix)

    def _sync_impl(
        self,
        path_info_list: List[base.PathUploadInfo],
    ) -> None:
        for path_info in path_info_list:
            path = path_info.path
            mangled_relative_path = path_info.mangled_relative_path
            mangled_path = self.sync_path.joinpath(mangled_relative_path)
            tbd_filename = str(mangled_path)
            key_name = os.path.join(self.prefix, tbd_filename)

            url = f"s3://{self.bucket}/{key_name}"
            logger.debug(f"Uploading {path} to {url}")

            self.client.upload_file(str(path), self.bucket, key_name)

    def delete(self) -> None:
        prefix_path = os.path.join(self.prefix, self.sync_path)
        self.resource.Bucket(self.bucket).objects.filter(Prefix=prefix_path).delete()
