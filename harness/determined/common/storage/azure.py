import contextlib
from azure.storage.blob import BlobServiceClient
from azure.core.exceptions import ResourceExistsError, HttpResponseError
import contextlib
import logging
import os
import tempfile
from typing import Iterator, Optional

from determined.common import util
from determined.common.storage.base import StorageManager, StorageMetadata


class AzureStorageManager(StorageManager):
    """
    Store and load checkpoints from Azure Blob Storage.

    Checkpoints are stored as a collection of Block Blobs,
    with each block blob corresponding to one checkpoint resouce.
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
        if connection_string:
            self.client = BlobServiceClient.from_connection_string(connection_string)
        elif account_url:
            self.client = BlobServiceClient(account_url, credential)

        logging.info("Trying to create Azure Blob Storage Container: {}.".format(container))
        try:
            self.client.create_container(container.split("/")[0])
            logging.info("Successfully created container {}.".format(container))
        except ResourceExistsError:
            logging.info(
                "Container {} already exists, and will be used to store checkpoints.".format(
                    container
                )
            )
        except HttpResponseError as e:
            if e.reason == "The requested URI does not represent any resource on the server.":
                logging.warning(
                    (
                        "The storage client raised the following HttpResponseError:\n{}\nPlease "
                        + "ignore this warning if this is because the account url provided points to "
                        + "a container instead of a storage account; otherwise, it may be necessary "
                        + "to fix your config.yaml."
                    ).format(e)
                )
            else:
                logging.error("Failed while trying to create container {}.".format(container))
                raise e
        self.container = container if not container.endswith("/") else container[:-1]

    def post_store_path(self, storage_id: str, storage_dir: str, metadata: StorageMetadata) -> None:
        """post_store_path uploads the checkpoint to Azure Blob Storage and deletes the original files."""
        try:
            logging.info("Uploading checkpoint {} to Azure Blob Storage.".format(storage_id))
            self.upload(metadata, storage_dir)
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @contextlib.contextmanager
    def restore_path(self, metadata: StorageMetadata) -> Iterator[str]:
        storage_dir = os.path.join(self._base_path, metadata.storage_id)
        os.makedirs(storage_dir, exist_ok=True)

        logging.info(
            "Downloading checkpoint {} from Azure Blob Storage".format(metadata.storage_id)
        )
        self.download(metadata, storage_dir)

        try:
            yield os.path.join(self._base_path, metadata.storage_id)
        finally:
            self._remove_checkpoint_directory(metadata.storage_id)

    @util.preserve_random_state
    def upload(self, metadata: StorageMetadata, storage_dir: str) -> None:
        for rel_path in metadata.resources.keys():
            if not rel_path.endswith("/"):
                rel_path_parent = (
                    "{}/{}".format(metadata.storage_id, "/".join(rel_path.split("/")[:-1]))
                ).rstrip("/")
                container_name = "{}/{}".format(self.container, rel_path_parent)
                blob_name = rel_path.split("/")[-1]
                abs_path = os.path.join(storage_dir, rel_path)
                logging.debug(
                    "Uploading blob {} to container {}.".format(blob_name, container_name)
                )
                with open(abs_path, "rb") as f:
                    self.client.get_blob_client(container_name, blob_name).upload_blob(f.read())

    @util.preserve_random_state
    def download(self, metadata: StorageMetadata, storage_dir: str) -> None:
        for rel_path in metadata.resources.keys():
            abs_path = os.path.join(storage_dir, rel_path)

            os.makedirs(os.path.dirname(abs_path), exist_ok=True)

            # Only create empty directory for keys that end with "/".
            if rel_path.endswith("/"):
                continue

            rel_path_parent = (
                "{}/{}".format(metadata.storage_id, "/".join(rel_path.split("/")[:-1]))
            ).rstrip("/")
            container_name = "{}/{}".format(self.container, rel_path_parent)
            blob_name = rel_path.split("/")[-1]
            logging.debug(
                "Downloading blob {} from container {}.".format(blob_name, container_name)
            )
            with open(abs_path, "wb") as f:
                stream = self.client.get_blob_client(container_name, blob_name).download_blob()
                stream.readinto(f)

    @util.preserve_random_state
    def delete(self, metadata: StorageMetadata) -> None:
        logging.info("Deleting checkpoint {} from Azure Blob Storage".format(metadata.storage_id))
        for rel_path in metadata.resources.keys():
            if not rel_path.endswith("/"):
                rel_path = "{}/{}".format(metadata.storage_id, rel_path)
                self.client.get_blob_client(self.container, rel_path).delete_blob()
