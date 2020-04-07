import multiprocessing
import typing
from types import TracebackType

from determined_deploy.local import cluster_utils


class ClusterLogManager:
    def __init__(self, cluster_name: str) -> None:
        self._logs_process: typing.Optional[multiprocessing.Process] = None
        self.cluster_name = cluster_name

    def __enter__(self) -> "ClusterLogManager":
        self.setup_logs()
        return self

    def __exit__(self, type: type, value: Exception, traceback: TracebackType) -> None:
        self.stop_logs()

    def setup_logs(self) -> None:
        if self._logs_process is not None:
            self._logs_process.terminate()
        self._logs_process = multiprocessing.Process(
            target=cluster_utils.logs, args=(self.cluster_name,), daemon=True
        )
        self._logs_process.start()

    def stop_logs(self) -> None:
        if self._logs_process:
            self._logs_process.terminate()
