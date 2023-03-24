import contextlib
import os
import pathlib
import shutil
from typing import Iterator, Optional, Union

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
            shutil.rmtree(dst, ignore_errors=True)

    def post_store_path(self, src: Union[str, os.PathLike], dst: str) -> None:
        """
        post_store_path uploads the checkpoint to cloud storage and deletes the original files.
        """
        try:
            self.upload(src, dst)
        finally:
            shutil.rmtree(src, ignore_errors=True)

    def store_path_is_direct_access(self) -> bool:
        return False
