from typing import Any, Optional

from determined.common import util
from determined.common.storage.azure_client import AzureStorageClient
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
        self.client = AzureStorageClient(container, connection_string, account_url, credential)
        self.container = container if not container.endswith("/") else container[:-1]

    @util.preserve_random_state
    def sync(self) -> None:
        for path in self.to_sync():
            whole_path = self.sync_path.joinpath(path.relative_to(self.base_path))
            self.client.put(
                "{}/{}".format(self.container, str(whole_path.parent)), whole_path.name, whole_path
            )
            self._synced_event_sizes[path] = path.stat().st_size

    def delete(self) -> None:
        files = [
            str(self.sync_path.joinpath(path.relative_to(self.base_path)))
            for path in self._synced_event_sizes.keys()
        ]
        self.client.delete_files(self.container, files)
