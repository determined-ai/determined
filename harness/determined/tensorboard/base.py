import abc
import logging
import pathlib
import time
from typing import List


class TensorboardManager(metaclass=abc.ABCMeta):
    """
    TensorboardManager stores tfevent logs to a supported storage backend. The
    trial will generate tfevent files during training. The tfevent files must
    be written to the same base_path as the base_path used in the contructor of
    this class.

    Each supported persistent storage backend must define a subclass which
    implements the sync method.
    """

    def __init__(
        self,
        base_path: pathlib.Path,
        sync_path: pathlib.Path,
    ) -> None:
        self.base_path = base_path
        self.sync_path = sync_path
        self.last_sync = 0.0

    def list_tfevents(self, since: float) -> List[pathlib.Path]:
        """
        list_tfevents returns tfevent file names located in the base_path directory that have been
        modified since a certain time.

        If many tfevent files have been created, the syscall to stat on each of them can be quite
        expensive, taking on the order of 1ms for every 100 files. Each file gets stat'd every call
        to this function, which can be a bottleneck late in training or in local training when many
        tfevent files exist but few or none need to be resynced.
        """

        if not self.base_path.exists():
            logging.warning(
                f"{self.base_path} directory does not exist. "
                "Trial does not include the correct callback for TensorBoard"
            )
            return []

        return [f for f in self.base_path.glob("**/*tfevents*") if f.stat().st_mtime > since]

    def to_sync(self) -> List[pathlib.Path]:
        sync_start = time.time()
        sync_paths = self.list_tfevents(self.last_sync)
        self.last_sync = sync_start

        return sync_paths

    @abc.abstractmethod
    def sync(self) -> None:
        """
        Save the object to the backing persistent storage.
        """
        pass

    @abc.abstractmethod
    def delete(self) -> None:
        """
        Delete all objects from the backing persistent storage.
        """
        pass
