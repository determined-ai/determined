import logging
import os
import tempfile
from contextlib import contextmanager
from typing import Generator, Optional, Tuple

import boto3

import determined_common.util as util
from determined_common.storage.base import Storable, StorageManager, StorageMetadata


class S3StorageManager(StorageManager):
    """
    Store and load Storables from S3.
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

    def store(self, store_data: Storable, storage_id: str = "") -> StorageMetadata:
        metadata = super().store(store_data, storage_id)

        logging.info("Uploading checkpoint {} to S3".format(metadata.storage_id))
        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        self.upload(metadata, storage_dir)

        self._remove_checkpoint_directory(metadata.storage_id)

        return metadata

    def restore(self, checkpoint: Storable, metadata: StorageMetadata) -> None:
        logging.info("Downloading checkpoint {} from S3".format(metadata.storage_id))

        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        os.makedirs(storage_dir, exist_ok=True)
        self.download(metadata, storage_dir)
        super().restore(checkpoint, metadata)

        self._remove_checkpoint_directory(metadata.storage_id)

    @contextmanager
    def store_path(self, storage_id: str = "") -> Generator[Tuple[str, str], None, None]:
        with super().store_path(storage_id) as (storage_id, path):
            yield (storage_id, path)

        metadata = StorageMetadata(storage_id, StorageManager._list_directory(path))

        try:
            logging.info("Uploading checkpoint {} to s3".format(storage_id))
            self.upload(metadata, path)
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @contextmanager
    def restore_path(self, metadata: StorageMetadata) -> Generator[str, None, None]:
        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        os.makedirs(storage_dir, exist_ok=True)

        logging.info("Downloading checkpoint {} from S3".format(metadata.storage_id))
        self.download(metadata, storage_dir)

        try:
            with super().restore_path(metadata) as path:
                yield path
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

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
