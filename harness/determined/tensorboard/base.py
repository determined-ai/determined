import abc
import dataclasses
import logging
import pathlib
import queue
import threading
import time
from typing import Any, Callable, List

from determined import tensorboard
from determined.common import util

logger = logging.getLogger("determined.tensorboard")


@dataclasses.dataclass
class PathUploadInfo:
    path: pathlib.Path
    mangled_relative_path: pathlib.Path


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
        async_upload: bool = True,
        sync_on_close: bool = True,
    ) -> None:
        self.base_path = base_path
        self.sync_path = sync_path
        self.last_sync = 0.0

        self.upload_thread = None
        if async_upload:
            self.upload_thread = _TensorboardUploadThread(self._sync_impl)
        self.sync_on_close = sync_on_close

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
    def _sync_impl(self, path_info_list: List[PathUploadInfo]) -> None:
        """
        Save the object to the backing persistent storage.
        """
        pass

    def sync(
        self,
        selector: Callable[[pathlib.Path], bool] = lambda _: True,
        mangler: Callable[[pathlib.Path, int], pathlib.Path] = lambda p, __: p,
        rank: int = 0,
    ) -> None:
        # Only sync a maximum of once per second to play nice with cloud storage request quotas.
        if time.time() - self.last_sync < 1:
            return
        self._sync(selector, mangler, rank)

    def _sync(
        self,
        selector: Callable[[pathlib.Path], bool] = lambda _: True,
        mangler: Callable[[pathlib.Path, int], pathlib.Path] = lambda p, __: p,
        rank: int = 0,
    ) -> None:
        paths = self.to_sync(selector)
        path_list = []
        for path in paths:
            relative_path = path.relative_to(self.base_path)
            mangled_relative_path = mangler(relative_path, rank)
            path_list.append(PathUploadInfo(path=path, mangled_relative_path=mangled_relative_path))
        if self.upload_thread is not None and self.upload_thread.is_alive():
            self.upload_thread.upload(path_list)
        else:
            util.preserve_random_state(self._sync_impl)(path_list)

    @abc.abstractmethod
    def delete(self) -> None:
        """
        Delete all objects from the backing persistent storage.
        """
        pass

    def start(self) -> None:
        if self.upload_thread is not None:
            self.upload_thread.start()

    def close(self) -> None:
        if self.sync_on_close:
            self._sync()
        if self.upload_thread is not None and self.upload_thread.is_alive():
            self.upload_thread.close()

    def __enter__(self) -> "TensorboardManager":
        self.start()
        return self

    def __exit__(self, exc_type: type, exc_val: Exception, exc_tb: Any) -> None:
        self.close()


def get_metric_writer() -> tensorboard.BatchMetricWriter:
    try:
        from determined.tensorboard.metric_writers import tensorflow

        writer: tensorboard.MetricWriter = tensorflow.TFWriter()

    except ModuleNotFoundError:
        logger.warning("TensorFlow writer not found")
        from determined.tensorboard.metric_writers import pytorch

        writer = pytorch._TorchWriter()

    return tensorboard.BatchMetricWriter(writer)


class _TensorboardUploadThread(threading.Thread):
    def __init__(
        self,
        upload_function: Callable[[List[PathUploadInfo]], None],
        work_queue_max_size: int = 50,
    ) -> None:
        self._upload_function = upload_function

        self._work_queue: queue.Queue = queue.Queue(maxsize=work_queue_max_size)

        super().__init__(daemon=True, name="TensorboardUploadThread")

    def run(self) -> None:
        while True:
            path_info_list = self._work_queue.get()

            # None is the sentinel value
            # to signal the thread to exit
            if path_info_list is None:
                return

            # Try-catch is used to avoid exception from
            # one failed sync attempt to cause the thread to exit.
            try:
                self._upload_function(path_info_list)
            except Exception as e:
                logger.warning(f"Sync of Tensorboard files failed with error: {e}")

    def upload(self, path_info_list: List[PathUploadInfo]) -> None:
        self._work_queue.put(path_info_list)

    def close(self) -> None:
        self._work_queue.put(None)
        self.join(10)
        was_waiting = False
        while self.is_alive():
            was_waiting = True
            logger.info("Waiting for Tensorboard files to finish uploading")
            self.join(10)
        if was_waiting:
            logger.info("Tensorboard upload completed")
