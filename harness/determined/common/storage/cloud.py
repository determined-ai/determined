import contextlib
import os
import pathlib
from typing import Iterator, Optional, Set, Union

from determined import util
from determined.common import storage


class CloudStorageManager(storage.StorageManager):
    @contextlib.contextmanager
    def restore_path(
        self, src: str, selector: Optional[storage.Selector] = None
    ) -> Iterator[pathlib.Path]:
        dst = os.path.join(self._base_path, src)
        os.makedirs(dst, exist_ok=True)

        self.download(src, dst, selector)

        try:
            yield pathlib.Path(dst)
        finally:
            util.rmtree_nfs_safe(dst, ignore_errors=True)

    def post_store_path(
        self, src: Union[str, os.PathLike], dst: str, paths: Optional[Set[str]] = None
    ) -> None:
        """
        post_store_path uploads the checkpoint to cloud storage and deletes the original files.
        """
        try:
            self.upload(src, dst, paths)
        finally:
            util.rmtree_nfs_safe(src, ignore_errors=True)

    def store_path_is_direct_access(self) -> bool:
        return False
