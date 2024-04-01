import logging
import pathlib
from typing import Dict, List, Optional, Union

from determined.common import util

# Prevents Azure's HTTP logs from appearing in our trial logs.
logging.getLogger("azure").setLevel(logging.ERROR)


logger = logging.getLogger("determined.common.storage.azure")


class AzureStorageClient(object):
    """Connects to an Azure Blob Storage service account."""

    def __init__(
        self,
        container: str,
        connection_string: Optional[str] = None,
        account_url: Optional[str] = None,
        credential: Optional[str] = None,
    ) -> None:
        import azure.core.exceptions
        from azure.storage import blob

        if connection_string:
            self.client = blob.BlobServiceClient.from_connection_string(connection_string)
        elif account_url:
            self.client = blob.BlobServiceClient(account_url, credential)
        else:
            raise ValueError("Either 'connection_string' or 'account_url' must be specified.")

        logger.info(f"Trying to create Azure Blob Storage Container: {container}.")
        try:
            self.client.create_container(container.split("/")[0])
            logger.info(f"Successfully created container {container}.")
        except azure.core.exceptions.ResourceExistsError:
            logger.info(
                f"Container {container} already exists, and will be used to store checkpoints."
            )
        except azure.core.exceptions.HttpResponseError as e:
            if e.error_code == blob.StorageErrorCode.invalid_uri:  # type: ignore
                logger.warning(
                    f"The storage client raised the following HttpResponseError:\n{e}\nPlease "
                    "ignore this warning if this is because the account url provided points to a "
                    "container instead of a storage account; otherwise, it may be necessary to fix "
                    "your config.yaml."
                )
            else:
                logger.error(f"Failed while trying to create container {container}.")
                raise e

    @util.preserve_random_state
    def put(self, container_name: str, blob_name: str, filename: Union[str, pathlib.Path]) -> None:
        """Upload a file to the specified blob in the specified container."""
        with open(filename, "rb") as file:
            self.client.get_blob_client(container_name, blob_name).upload_blob(file, overwrite=True)

    @util.preserve_random_state
    def get(self, container_name: str, blob_name: str, filename: str) -> None:
        """Download the specified blob in the specified container to a file."""
        with open(filename, "wb") as file:
            stream = self.client.get_blob_client(container_name, blob_name).download_blob()
            stream.readinto(file)

    @util.preserve_random_state
    def delete_files(self, container_name: str, files: List[str]) -> None:
        """Deletes the specified files from the specified container."""
        for file in files:
            self.client.get_blob_client(container_name, file).delete_blob()

    @util.preserve_random_state
    def list_files(
        self, container_name: str, file_prefix: Optional[Union[str, pathlib.Path]] = None
    ) -> Dict[str, int]:
        """Lists files within the specified container that have the specified file prefix.
        Lists all files if file_prefix is None.
        """
        container = self.client.get_container_client(container_name)
        files = {
            blob["name"]: blob["size"]
            for blob in container.list_blobs(name_starts_with=file_prefix)
        }
        return files
