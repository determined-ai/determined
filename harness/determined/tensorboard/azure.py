from typing import Any, Optional

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
        from determined.common.storage import azure_client

        self.client = azure_client.AzureStorageClient(
            container, connection_string, account_url, credential
        )
        self.container = container if not container.endswith("/") else container[:-1]

    @util.preserve_random_state
    def sync(self) -> None:
        for path in self.to_sync():
            whole_path = self.sync_path.joinpath(path.relative_to(self.base_path))
            self.client.put("{}/{}".format(self.container, str(whole_path.parent)), path.name, path)

    def delete(self) -> None:
        files = self.client.list_files(self.container, self.sync_path)
        self.client.delete_files(self.container, files)
