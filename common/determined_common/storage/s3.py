import contextlib
import logging
import os
import tempfile
from typing import Iterator, Optional

import boto3

from determined_common import util
from determined_common.storage.base import StorageManager, StorageMetadata


class S3StorageManager(StorageManager):
    """
    Store and load checkpoints from S3.
    """

    def __init__(
        self,
        bucket: str,
        access_key: Optional[str] = None,
        secret_key: Optional[str] = None,
        endpoint_url: Optional[str] = None,
        temp_dir: Optional[str] = None,
    ) -> None:
        super().__init__(temp_dir if temp_dir is not None else tempfile.gettempdir())
        self.bucket = bucket
        self.client = boto3.client(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )

    def post_store_path(self, storage_id: str, storage_dir: str, metadata: StorageMetadata) -> None:
        """post_store_path uploads the checkpoint to s3 and deletes the original files."""
        try:
            logging.info("Uploading checkpoint {} to s3".format(storage_id))
            self.upload(metadata, storage_dir)
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @contextlib.contextmanager
    def restore_path(self, metadata: StorageMetadata) -> Iterator[str]:
        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        os.makedirs(storage_dir, exist_ok=True)

        logging.info("Downloading checkpoint {} from S3".format(metadata.storage_id))
        self.download(metadata, storage_dir)

        try:
            yield os.path.join(self._base_path, metadata.storage_id)
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @util.preserve_random_state
    def upload(self, metadata: StorageMetadata, storage_dir: str) -> None:
        for rel_path in metadata.resources.keys():
            key_name = "{}/{}".format(metadata.storage_id, rel_path)
            url = "s3://{}/{}".format(self.bucket, key_name)

            logging.debug("Uploading {} to {}".format(rel_path, url))

            if rel_path.endswith("/"):
                # Create empty S3 keys for each subdirectory.
                self.client.put_object(Bucket=self.bucket, Key=key_name, Body=b"")
            else:
                abs_path = os.path.join(storage_dir, rel_path)
                self.client.upload_file(abs_path, self.bucket, key_name)

    @util.preserve_random_state
    def download(self, metadata: StorageMetadata, storage_dir: str) -> None:
        for rel_path in metadata.resources.keys():
            abs_path = os.path.join(storage_dir, rel_path)

            os.makedirs(os.path.dirname(abs_path), exist_ok=True)

            # Only create empty directory for keys that end with "/".
            # See `upload` method for more context.
            if rel_path.endswith("/"):
                continue

            key_name = "{}/{}".format(metadata.storage_id, rel_path)
            url = "s3://{}/{}".format(self.bucket, key_name)
            logging.debug("Downloading {} from {}".format(url, rel_path))

            self.client.download_file(self.bucket, key_name, abs_path)

    @util.preserve_random_state
    def delete(self, metadata: StorageMetadata) -> None:
        logging.info("Deleting checkpoint {} from S3".format(metadata.storage_id))

        objects = [
            {"Key": "{}/{}".format(metadata.storage_id, rel_path)}
            for rel_path in metadata.resources.keys()
        ]

        # S3 delete_objects has a limit of 1000 objects.
        for chunk in util.chunks(objects, 1000):
            logging.debug("Deleting {} objects from S3".format(len(chunk)))
            self.client.delete_objects(Bucket=self.bucket, Delete={"Objects": chunk})
