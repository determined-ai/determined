import logging
from typing import Any, List, Optional

from hdfs.client import InsecureClient

from determined.common import util
from determined.tensorboard import base


class HDFSTensorboardManager(base.TensorboardManager):
    """
    Store and tfevents files to HDFS.
    """

    @util.preserve_random_state
    def __init__(
        self,
        hdfs_url: str,
        hdfs_path: str,
        user: Optional[str] = None,
        *args: Any,
        **kwargs: Any,
    ) -> None:
        super().__init__(*args, **kwargs)
        self.hdfs_url = hdfs_url
        self.hdfs_path = hdfs_path
        self.user = user

        self.client = InsecureClient(self.hdfs_url, root=self.hdfs_path, user=self.user)
        self.client.makedirs(str(self.sync_path))

    def _sync_impl(
        self,
        path_info_list: List[base.PathUploadInfo],
    ) -> None:
        for path_info in path_info_list:
            path = path_info.path
            mangled_relative_path = path_info.mangled_relative_path
            mangled_path = self.sync_path.joinpath(mangled_relative_path)
            file_name = str(mangled_path)
            logging.debug(f"Uploading {path} to {self.hdfs_path}")

            self.client.upload(file_name, str(path))

    def delete(self) -> None:
        self.client.delete(self.sync_path, recursive=True)
