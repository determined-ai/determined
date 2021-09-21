import contextlib
import logging
import os
import tempfile
from typing import Iterator, Optional

import boto3
import requests

from determined import errors
from determined.common import util
from determined.common.storage.base import StorageManager

from .boto3_credential_manager import initialize_boto3_credential_providers


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
        initialize_boto3_credential_providers()
        self.bucket_name = bucket
        self.s3 = boto3.resource(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )
        self.bucket = self.s3.Bucket(self.bucket_name)

        # Detect if we are talking to minio, because boto3 has a client-side bug parsing the output
        # of the minio server.
        self._use_minio_workaround = False
        if endpoint_url is not None:
            try:
                r = requests.get(endpoint_url)
            except ConnectionError:
                pass
            else:
                if r.headers.get("Server", "").lower() == "minio":
                    self._use_minio_workaround = True
                    logging.info(
                        "MinIO backend detected.  To work around a boto3 bug, empty directories"
                        "will not be uploaded in checkpoints."
                    )

    def post_store_path(self, storage_id: str, storage_dir: str) -> None:
        """post_store_path uploads the checkpoint to s3 and deletes the original files."""
        try:
            logging.info(f"Uploading checkpoint {storage_id} to s3")
            self.upload(storage_id, storage_dir)
        finally:
            self._remove_checkpoint_directory(storage_id)

    @contextlib.contextmanager
    def restore_path(self, storage_id: str) -> Iterator[str]:
        storage_dir = os.path.join(self._base_path, storage_id)
        os.makedirs(storage_dir, exist_ok=True)

        logging.info(f"Downloading checkpoint {storage_id} from S3")
        self.download(storage_id, storage_dir)

        try:
            yield os.path.join(self._base_path, storage_id)
        finally:
            self._remove_checkpoint_directory(storage_id)

    @util.preserve_random_state
    def upload(self, storage_id: str, storage_dir: str) -> None:
        storage_prefix = storage_id
        for rel_path in sorted(self._list_directory(storage_dir)):
            key_name = f"{storage_prefix}/{rel_path}"
            logging.debug(f"Uploading {rel_path} to s3://{self.bucket_name}/{key_name}")

            if rel_path.endswith("/"):
                # Create empty S3 keys for each subdirectory to mimic what the S3 console does to
                # represent empty directories.
                if not self._use_minio_workaround:
                    self.bucket.put_object(Key=key_name, Body=b"")
                else:
                    # boto3 will puke on the following MinIO response if you ever create a
                    # directory by uploading an empty blob.  Uploading a normal file in the
                    # directory and then deleting it seems to cause MinIO to prune the empty
                    # directory.  The AWS authentication scheme is complex and not worth the
                    # effort for supporting empty directories, so... just ignore empty directories.
                    pass
            else:
                abs_path = os.path.join(storage_dir, rel_path)
                self.bucket.upload_file(abs_path, key_name)

    @util.preserve_random_state
    def download(self, storage_id: str, storage_dir: str) -> None:
        storage_prefix = storage_id
        found = False
        for obj in self.bucket.objects.filter(Prefix=storage_prefix):
            found = True
            dst = os.path.join(storage_dir, os.path.relpath(obj.key, storage_prefix))
            dst_dir = os.path.dirname(dst)
            if not os.path.exists(dst_dir):
                os.makedirs(dst_dir, exist_ok=True)

            logging.debug(f"Downloading s3://{self.bucket_name}/{obj.key} to {dst}")

            # Only create empty directory for keys that end with "/".
            # See `upload` method for more context.
            if obj.key.endswith("/"):
                os.makedirs(dst, exist_ok=True)
                continue

            self.bucket.download_file(obj.key, dst)

        if not found:
            raise errors.CheckpointNotFound(f"Did not find checkpoint {storage_id} in S3")

    @util.preserve_random_state
    def delete(self, storage_id: str) -> None:
        logging.info(f"Deleting checkpoint {storage_id} from S3")

        storage_prefix = storage_id
        objects = [{"Key": obj.key} for obj in self.bucket.objects.filter(Prefix=storage_prefix)]

        # S3 delete_objects has a limit of 1000 objects.
        for chunk in util.chunks(objects, 1000):
            logging.debug(f"Deleting {len(chunk)} objects from S3")
            self.bucket.delete_objects(Delete={"Objects": chunk})
