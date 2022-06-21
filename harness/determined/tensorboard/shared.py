import logging
import os
import pathlib
import shutil
from typing import Any, Callable

from determined.tensorboard import base

logger = logging.getLogger("determined.tensorboard")


class SharedFSTensorboardManager(base.TensorboardManager):
    """
    SharedFSTensorboardManager stores tfevent logs from a shared file system.
    The host_path must be present on each agent machine.
    """

    def __init__(self, storage_path: str, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self.storage_path = pathlib.Path(storage_path)
        self.shared_fs_base = self.storage_path.joinpath(self.sync_path)

        # Set umask to 0 in order that the storage dir allows future containers of any owner to
        # create new checkpoints. Administrators wishing to control the permissions more
        # specifically should just create the storage path themselves; this will not interfere.
        old_umask = os.umask(0)
        self.shared_fs_base.mkdir(parents=True, exist_ok=True, mode=0o777)
        # Restore the original umask.
        os.umask(old_umask)

    def sync(
        self,
        selector: Callable[[pathlib.Path], bool] = lambda _: True,
        mangler: Callable[[pathlib.Path, int], pathlib.Path] = lambda p, __: p,
        rank: int = 0,
    ) -> None:
        for path in self.to_sync(selector):
            relative_path = path.relative_to(self.base_path)
            mangled_relative_path = mangler(relative_path, rank)
            mangled_path = self.shared_fs_base.joinpath(mangled_relative_path)
            pathlib.Path.mkdir(mangled_path.parent, parents=True, exist_ok=True)
            logger.debug(f"SharedFSTensorboardManager saving {path} to {mangled_path}")

            shutil.copy(path, mangled_path)

    def delete(self) -> None:
        shutil.rmtree(self.shared_fs_base, False)
