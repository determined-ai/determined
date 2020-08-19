import logging
import pathlib
from typing import Dict, List


class TensorboardManager:
    """
    TensorboardManager stores tfevent logs to a supported storage backend. The
    trial will generate tfevent files during training. The tfevent files must
    be written to the same base_path as the base_path used in the contructor of
    this class.

    Each supported persistent storage backend must define a subclass which
    implements the sync method.
    """

    def __init__(self, base_path: pathlib.Path, sync_path: pathlib.Path):
        self.base_path = base_path
        self.sync_path = sync_path
        self._synced_event_sizes: Dict[pathlib.Path, int] = {}

    def list_tfevents(self) -> List[pathlib.Path]:
        """
        list_tfevents returns tfevent file names located in the base_path directory.
        """

        if not self.base_path.exists():
            logging.warning(
                f"{self.base_path} directory does not exist. "
                "Trial does not include the correct callback for TensorBoard"
            )
            return []

        return [f.resolve() for f in self.base_path.glob("**/*tfevents*")]

    def to_sync(self) -> List[pathlib.Path]:
        """
        to_sync returns tfevent files that have not been exported to the persistent
        storage backend.
        """

        sync_paths = []
        for path in self.list_tfevents():
            if path not in self._synced_event_sizes:
                sync_paths.append(path)
            elif path.stat().st_size > self._synced_event_sizes[path]:
                sync_paths.append(path)

        return sync_paths

    def sync(self) -> None:
        """
        Save the object to the backing persistent storage.
        """
        pass
