import logging
import os
import tempfile
from typing import Dict, List, Optional, Union

from determined import errors
from determined.common import storage, util

import posixpath  # isort:skip


logger = logging.getLogger("determined.common.storage.azure")


class AzureStorageManager(storage.CloudStorageManager):
    """
    Store and load checkpoints from Azure Blob Storage.

    Checkpoints are stored as a collection of Block Blobs,
    with each block blob corresponding to one checkpoint resource.
    """

    def __init__(
        self,
        container: str,
        connection_string: Optional[str] = None,
        account_url: Optional[str] = None,
        credential: Optional[str] = None,
        temp_dir: Optional[str] = None,
    ) -> None:
        super().__init__(temp_dir if temp_dir is not None else tempfile.gettempdir())
        from determined.common.storage import azure_client

        self.client = azure_client.AzureStorageClient(
            container, connection_string, account_url, credential
        )
        self.container = container if not container.endswith("/") else container[:-1]

    @util.preserve_random_state
    def upload(
        self, src: Union[str, os.PathLike], dst: str, paths: Optional[storage.Paths] = None
    ) -> None:
        src = os.fspath(src)
        logger.info(f"Uploading to Azure Blob Storage: {dst}")
        upload_paths = paths if paths is not None else self._list_directory(src)
        for rel_path in sorted(upload_paths):
            # Use posixpath so that we always use forward slashes, even on Windows.
            container_blob = posixpath.join(self.container, dst, rel_path)

            if rel_path.endswith("/"):
                blob_dir, blob_base = posixpath.split(container_blob.rstrip("/"))
                blob_base = f"{blob_base}/"
                abs_path = "/dev/null"
                logger.debug(f"Uploading blob empty {blob_base} to container {blob_dir}.")
            else:
                blob_dir, blob_base = posixpath.split(container_blob)
                abs_path = os.path.join(src, rel_path)
                logger.debug(f"Uploading blob {blob_base} to container {blob_dir}.")

            self.client.put(blob_dir, blob_base, abs_path)

    @util.preserve_random_state
    def download(
        self,
        src: str,
        dst: Union[str, os.PathLike],
        selector: Optional[storage.Selector] = None,
    ) -> None:
        dst = os.fspath(dst)
        logger.info(f"Downloading {src} from Azure Blob Storage")
        found = False
        for blob in self.client.list_files(self.container, file_prefix=src):
            found = True
            relname = os.path.relpath(blob, src)
            if blob.endswith("/"):
                relname = os.path.join(relname, "")
            if selector is not None and not selector(relname):
                continue
            _dst = os.path.join(dst, relname)
            dst_dir = os.path.dirname(_dst)
            os.makedirs(dst_dir, exist_ok=True)

            # Only create empty directory for keys that end with "/".
            if blob.endswith("/"):
                os.makedirs(_dst, exist_ok=True)
                continue

            # Use posixpath so that we always use forward slashes, even on Windows.
            container_blob = posixpath.join(self.container, blob)
            blob_dir, blob_base = posixpath.split(container_blob)
            self.client.get(blob_dir, blob_base, _dst)

        if not found:
            raise errors.CheckpointNotFound(f"Did not find checkpoint {src} in Azure Blob Storage")

    @util.preserve_random_state
    def delete(self, tgt: str, globs: List[str]) -> Dict[str, int]:
        storage_prefix = tgt
        logger.info(f"Deleting {tgt} from Azure Blob Storage")

        objects = self.client.list_files(self.container, file_prefix=storage_prefix)

        resources = {}
        if "**/*" not in globs:  # Partial delete case.
            prefixed_resources = self._apply_globs_to_resources(objects, storage_prefix, globs)
            for obj in list(objects):
                if obj in prefixed_resources:
                    resources[obj.replace(f"{storage_prefix}/", "")] = objects[obj]
                    del objects[obj]

        self.client.delete_files(self.container, list(objects))

        return resources
