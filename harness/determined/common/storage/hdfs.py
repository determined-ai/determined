import logging
import os
import tempfile
import warnings
from typing import Optional, Union

from hdfs.client import InsecureClient

from determined.common import storage, util


class HDFSStorageManager(storage.CloudStorageManager):
    """
    Store and load checkpoints from HDFS.
    """

    def __init__(
        self,
        hdfs_url: str,
        hdfs_path: str,
        user: Optional[str] = None,
        temp_dir: Optional[str] = None,
    ) -> None:
        warnings.warn(
            "HDFS checkpoint storage support has been deprecated and will be removed in a future "
            "version.  Please contact Determined if you still need it, or migrate to a different "
            "storage backend.",
            FutureWarning,
            stacklevel=2,
        )
        super().__init__(temp_dir if temp_dir is not None else tempfile.gettempdir())

        self.hdfs_url = hdfs_url
        self.hdfs_path = hdfs_path
        self.user = user

        self.client = InsecureClient(self.hdfs_url, root=self.hdfs_path, user=self.user)

    @util.preserve_random_state
    def upload(
        self, src: Union[str, os.PathLike], dst: str, paths: Optional[storage.Paths] = None
    ) -> None:
        if paths is not None:
            raise NotImplementedError("HDFSStorageManager does not support partial uploads")

        src = os.fspath(src)
        logging.info(f"Uploading to HDFS: {dst}")
        self.client.upload(dst, src)

    @util.preserve_random_state
    def download(
        self,
        src: str,
        dst: Union[str, os.PathLike],
        selector: Optional[storage.Selector] = None,
    ) -> None:
        if selector is not None:
            raise NotImplementedError("HDFSStorageManager does not support partial downloads")
        dst = os.fspath(dst)
        logging.info(f"Downloading {src} from HDFS")
        self.client.download(src, dst, overwrite=True)

    @util.preserve_random_state
    def delete(self, tgt: str) -> None:
        logging.info(f"Deleting {tgt} from HDFS")
        self.client.delete(tgt, recursive=True)
