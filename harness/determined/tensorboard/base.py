import abc
import logging
import pathlib
import queue
import threading
import time
from typing import Callable, List

from determined import tensorboard


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

    def list_tb_files(
        self,
        since: float,
        selector: Callable[[pathlib.Path], bool],
    ) -> List[pathlib.Path]:
        """
        list_files returns Tensorboard-relevant file names located in the base_path directory
        and all sub-directories that have been modified since a certain time.

        If many files have been created, the syscall to stat on each of them can be quite
        expensive, taking on the order of 1ms for every 100 files. Each file gets stat'd every call
        to this function, which can be a bottleneck late in training or in local training when many
        files exist but few or none need to be re-synced.
        """

        if not self.base_path.exists():
            logging.warning(f"{self.base_path} directory does not exist.")
            return []
        return [
            file
            for file in self.base_path.rglob("*")
            if file.stat().st_mtime > since and file.is_file() and selector(file)
        ]

    def to_sync(
        self,
        selector: Callable[[pathlib.Path], bool],
    ) -> List[pathlib.Path]:
        sync_start = time.time()
        sync_paths = self.list_tb_files(self.last_sync, selector)
        self.last_sync = sync_start

        return sync_paths

    @abc.abstractmethod
    def sync(
        self,
        selector: Callable[[pathlib.Path], bool] = lambda _: True,
        mangler: Callable[[pathlib.Path, int], pathlib.Path] = lambda p, __: p,
        rank: int = 0,
    ) -> None:
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


def get_metric_writer() -> tensorboard.BatchMetricWriter:
    try:
        from determined.tensorboard.metric_writers import tensorflow

        writer: tensorboard.MetricWriter = tensorflow.TFWriter()

    except ModuleNotFoundError:
        logging.warning("TensorFlow writer not found")
        from determined.tensorboard.metric_writers import pytorch

        writer = pytorch.TorchWriter()

    return tensorboard.BatchMetricWriter(writer)


class _TensorboardUploadThread(threading.Thread):
    def __init__(
        self, upload_function: Callable[[List[pathlib.Path]], None], work_queue_max_size: int = 50
    ) -> None:
        self._upload_function = upload_function

        self._work_queue: queue.Queue = queue.Queue(maxsize=work_queue_max_size)

        super().__init__()

    def run(self) -> None:
        while True:
            file_paths = self._work_queue.get()

            # None is the sentinel value
            # to signal the thread to exit
            if file_paths is None:
                return

            # Try-catch is used to avoid exception from
            # one failed sync attempt to cause the thread to exit.
            try:
                self._upload_function(file_paths)
            except Exception as e:
                logging.warning(f"Sync of Tensorboard files failed with error: {e}")

    def upload(self, paths: List[pathlib.Path]) -> None:
        self._work_queue.put(paths)

    def close(self) -> None:
        self._work_queue.put(None)
