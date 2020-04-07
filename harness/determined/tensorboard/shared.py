import pathlib
import shutil
from typing import Any

from determined.tensorboard import base


class SharedFSTensorboardManager(base.TensorboardManager):
    """
    SharedFSTensorboardManager stores tfevent logs from a shared file system.
    The host_path must be present on each agent machine.
    """

    def __init__(self, container_path: str, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        self.container_path = pathlib.Path(container_path)
        self.shared_fs_base = self.container_path.joinpath(self.sync_path)

        self.shared_fs_base.mkdir(parents=True, exist_ok=True)

    def sync(self) -> None:
        for path in self.to_sync():
            shared_fs_path = self.shared_fs_base.joinpath(path.name)
            shutil.copy(path, shared_fs_path)

            self._synced_event_sizes[path] = path.stat().st_size
