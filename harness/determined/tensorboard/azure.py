import logging
import pathlib
from typing import Any, Callable, Optional

from determined.common import util
from determined.tensorboard import base

logger = logging.getLogger("determined.tensorboard")


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
    def sync(
        self,
        selector: Callable[[pathlib.Path], bool] = lambda _: True,
        mangler: Callable[[pathlib.Path, int], pathlib.Path] = lambda p, __: p,
        rank: int = 0,
    ) -> None:
        for path in self.to_sync(selector):
            relative_path = path.relative_to(self.base_path)
            mangled_relative_path = mangler(relative_path, rank)
            mangled_path = self.sync_path.joinpath(mangled_relative_path)

            logger.debug(f"Uploading {path} to Azure: {self.container}/{mangled_path}")
            self.client.put(
                f"{self.container}/{mangled_path.parent}",
                mangled_path.name,
                path,
            )

    def delete(self) -> None:
        files = self.client.list_files(self.container, self.sync_path)
        self.client.delete_files(self.container, files)
