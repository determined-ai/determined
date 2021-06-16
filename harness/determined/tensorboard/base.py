import abc
import pathlib
import time
from typing import List

from determined.tensorboard import util


class TensorboardManager(metaclass=abc.ABCMeta):
    """
    TensorboardManager stores tensorboard logs (tfevent files, .gz zipped archives,
    and .pb protobuf graph and model definition files) to a supported storage backend. The
    trial will generate tfevent files during training. If a profiling callback is used,
    .pb and .gz files will also be generated. These files must be written to the same
    base_path as the base_path used in the constructor of this class.

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

    def list_tb_files(self, since: float) -> List[pathlib.Path]:
        """
        list_files returns Tensorboard-relevant file names located in the base_path directory
        and all sub-directories that have been modified since a certain time.

        If many files have been created, the syscall to stat on each of them can be quite
        expensive, taking on the order of 1ms for every 100 files. Each file gets stat'd every call
        to this function, which can be a bottleneck late in training or in local training when many
        files exist but few or none need to be re-synced.
        """

        tb_files = util.find_tb_files(self.base_path)
        return list(filter(lambda file: file.stat().st_mtime > since, tb_files))

    def to_sync(self) -> List[pathlib.Path]:
        sync_start = time.time()
        sync_paths = self.list_tb_files(self.last_sync)
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
