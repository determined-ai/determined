import logging
from typing import Any, List, Optional

from determined.tensorboard import base

logger = logging.getLogger("determined.tensorboard.azure")


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

    def _sync_impl(
        self,
        path_info_list: List[base.PathUploadInfo],
    ) -> None:
        for path_info in path_info_list:
            path = path_info.path
            mangled_relative_path = path_info.mangled_relative_path
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
