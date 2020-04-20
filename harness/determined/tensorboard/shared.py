import os
import pathlib
import shutil
from typing import Any, Optional

from determined.tensorboard import base
from determined_common.storage.shared import _full_storage_dir


class SharedFSTensorboardManager(base.TensorboardManager):
    """
    SharedFSTensorboardManager stores tfevent logs from a shared file system.
    The host_path must be present on each agent machine.
    """

    def __init__(
        self,
        host_path: str,
        container_path: str,
        storage_path: Optional[str],
        *args: Any,
        **kwargs: Any
    ) -> None:
        super().__init__(*args, **kwargs)
        self.container_path = pathlib.Path(
            _full_storage_dir(host_path, container_path, storage_path)
        )
        self.shared_fs_base = self.container_path.joinpath(self.sync_path)

        # Set umask to 0 in order that the storage dir allows future containers of any owner to
        # create new checkpoints. Administrators wishing to control the permissions more
        # specifically should just create the storage path themselves; this will not interfere.
        old_umask = os.umask(0)
        self.shared_fs_base.mkdir(parents=True, exist_ok=True, mode=0o777)
        # Restore the original umask.
        os.umask(old_umask)

    def sync(self) -> None:
        for path in self.to_sync():
            shared_fs_path = self.shared_fs_base.joinpath(path.name)
            shutil.copy(path, shared_fs_path)

            self._synced_event_sizes[path] = path.stat().st_size
