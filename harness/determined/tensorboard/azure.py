import logging
from typing import Any, Optional

from azure.core.exceptions import HttpResponseError, ResourceExistsError
from azure.storage.blob import BlobServiceClient

from determined.common import util
from determined.tensorboard import base


class AzureTensorboardManager(base.TensorboardManager):
    """
    Store and load TF Event Logs from Azure Blob Storage.
    """

    def __init__(
        self,
        container: str,
        connection_string: Optional[str] = None,
        account_url: Optional[str] = None,
        credential: Optional[str] = None,
        *args: Any,
        **kwargs: Any,
    ) -> None:
        super().__init__(*args, **kwargs)
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
                        + "ignore this warning if this is because the account url provided points "
                        + "to a container instead of a storage account; otherwise, it may be "
                        + "necessary to fix your config.yaml."
                    ).format(e)
                )
            else:
                logging.error("Failed while trying to create container {}.".format(container))
                raise e
        self.container = container if not container.endswith("/") else container[:-1]

    @util.preserve_random_state
    def sync(self) -> None:
        for path in self.to_sync():
            whole_path = self.sync_path.joinpath(path.relative_to(self.base_path))
            with open(whole_path, "rb") as f:
                self.client.get_blob_client(
                    "{}/{}".format(self.container, str(whole_path.parent)), whole_path.name
                ).upload_blob(f.read())
            self._synced_event_sizes[path] = path.stat().st_size

    def delete(self) -> None:
        for path in self._synced_event_sizes.keys():
            whole_path = self.sync_path.joinpath(path.relative_to(self.base_path))
            self.client.get_blob_client(self.container, str(whole_path)).delete_blob()
