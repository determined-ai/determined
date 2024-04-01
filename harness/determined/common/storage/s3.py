import logging
import os
import tempfile
from typing import Dict, List, Optional, Union

import requests

from determined import errors
from determined.common import storage, util

logger = logging.getLogger("determined.common.storage.s3")


class S3StorageManager(storage.CloudStorageManager):
    """
    Store and load checkpoints from S3.
    """

    def __init__(
        self,
        bucket: str,
        access_key: Optional[str] = None,
        secret_key: Optional[str] = None,
        endpoint_url: Optional[str] = None,
        prefix: Optional[str] = None,
        temp_dir: Optional[str] = None,
    ) -> None:
        super().__init__(temp_dir if temp_dir is not None else tempfile.gettempdir())
        import boto3

        from determined.common.storage import boto3_credential_manager

        boto3_credential_manager.initialize_boto3_credential_providers()
        self.bucket_name = bucket
        self.s3 = boto3.resource(
            "s3",
            endpoint_url=endpoint_url,
            aws_access_key_id=access_key,
            aws_secret_access_key=secret_key,
        )
        self.bucket = self.s3.Bucket(self.bucket_name)

        self.prefix = storage.normalize_prefix(prefix)

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
                    logger.info(
                        "MinIO backend detected.  To work around a boto3 bug, empty directories"
                        "will not be uploaded in checkpoints."
                    )

    def get_storage_prefix(self, storage_id: str) -> str:
        return os.path.join(self.prefix, storage_id)

    @util.preserve_random_state
    def upload(
        self, src: Union[str, os.PathLike], dst: str, paths: Optional[storage.Paths] = None
    ) -> None:
        src = os.fspath(src)
        prefix = self.get_storage_prefix(dst)
        logger.info(f"Uploading to s3: prefix={prefix}")
        upload_paths = paths if paths is not None else self._list_directory(src)
        for rel_path in sorted(upload_paths):
            key_name = f"{prefix}/{rel_path}"
            logger.debug(f"Uploading {rel_path} to s3://{self.bucket_name}/{key_name}")

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
                abs_path = os.path.join(src, rel_path)
                self.bucket.upload_file(abs_path, key_name)

    @util.preserve_random_state
    def download(
        self,
        src: str,
        dst: Union[str, os.PathLike],
        selector: Optional[storage.Selector] = None,
    ) -> None:
        import botocore

        dst = os.fspath(dst)
        prefix = self.get_storage_prefix(src)
        logger.info(f"Downloading {prefix} from S3")
        found = False

        try:
            for obj in self.bucket.objects.filter(Prefix=prefix):
                found = True
                relname = os.path.relpath(obj.key, prefix)
                if obj.key.endswith("/"):
                    relname = os.path.join(relname, "")

                if selector is not None and not selector(relname):
                    continue
                _dst = os.path.join(dst, relname)
                dst_dir = os.path.dirname(_dst)
                os.makedirs(dst_dir, exist_ok=True)

                logger.debug(f"Downloading s3://{self.bucket_name}/{obj.key} to {_dst}")

                # Only create empty directory for keys that end with "/".
                # See `upload` method for more context.
                if obj.key.endswith("/"):
                    os.makedirs(_dst, exist_ok=True)
                    continue

                self.bucket.download_file(obj.key, _dst)

        except botocore.exceptions.ClientError as e:
            if e.response["Error"]["Code"] == "AccessDenied":
                raise errors.NoDirectStorageAccess(
                    "Unable to access cloud checkpoint storage"
                ) from e
            raise

        except botocore.exceptions.NoCredentialsError as e:
            raise errors.NoDirectStorageAccess("Unable to access cloud checkpoint storage") from e

        if not found:
            raise errors.CheckpointNotFound(f"Did not find {prefix} in S3")

    @util.preserve_random_state
    def delete(self, tgt: str, globs: List[str]) -> Dict[str, int]:
        prefix = self.get_storage_prefix(tgt)
        logger.info(f"Deleting {prefix} from S3")

        objects = {obj.key: obj.size for obj in self.bucket.objects.filter(Prefix=prefix)}

        resources = {}
        if "**/*" not in globs:  # Partial delete case.
            prefixed_resources = self._apply_globs_to_resources(objects, prefix, globs)
            for obj in list(objects):
                if obj in prefixed_resources:
                    resources[obj.replace(f"{prefix}/", "")] = objects[obj]
                    del objects[obj]

        # S3 delete_objects has a limit of 1000 objects.
        for chunk in util.chunks([{"Key": o} for o in objects], 1000):
            logger.debug(f"Deleting {len(chunk)} objects from S3")
            self.bucket.delete_objects(Delete={"Objects": chunk})

        return resources
