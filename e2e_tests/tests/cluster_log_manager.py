import multiprocessing
from types import TracebackType  # noqa:I2041
from typing import Any, Callable, Optional


class ClusterLogManager:
    def __init__(self, logs_func: Callable[..., Any]) -> None:
        self._logs_process: Optional[multiprocessing.Process] = None
        self.logs_func = logs_func

    def __enter__(self) -> "ClusterLogManager":
        self.setup_logs()
        return self

    def __exit__(self, exc_type: type, exc_value: Exception, traceback: TracebackType) -> None:
        self.stop_logs()

    def setup_logs(self) -> None:
        if self._logs_process is not None:
            self._logs_process.terminate()
        self._logs_process = multiprocessing.Process(target=self.logs_func, daemon=True)
        self._logs_process.start()

    def stop_logs(self) -> None:
        if self._logs_process:
            self._logs_process.terminate()
