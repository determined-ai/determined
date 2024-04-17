import collections
import datetime
import logging
import queue
import sys
import threading
import time
import types
from typing import Any, Callable, Dict, Iterator, List, Optional, TextIO, Union

from determined import core
from determined.common import api

logger = logging.getLogger("determined.core")


class _LogShipper:
    def __init__(
        self,
        *,
        session: api.Session,
        trial_id: int,
        task_id: str,
        distributed: Optional[core.DistributedContext] = None
    ) -> None:
        self._session = session
        self._trial_id = trial_id
        self._task_id = task_id
        self._distributed = distributed

    def start(self) -> "_LogShipper":
        return self

    def close(
        self,
        exc_type: Optional[type],
        exc_val: Optional[BaseException],
        exc_tb: Optional[types.TracebackType],
    ) -> "_LogShipper":
        return self

    def __enter__(self) -> "_LogShipper":
        return self.start()

    def __exit__(
        self,
        exc_type: Optional[type],
        exc_val: Optional[BaseException],
        exc_tb: Optional[types.TracebackType],
    ) -> "_LogShipper":
        return self.close(exc_type, exc_val, exc_tb)


class _ManagedTrialLogShipper(_LogShipper):
    """
    Managed trials will ship their logs normally via fluentd.
    """

    pass


class _Interceptor:
    def __init__(self, original_io: TextIO, handler: Callable[[str], None]) -> None:
        self._original_io = original_io
        self._handler = handler

    def write(self, data: str) -> int:
        self._handler(data)
        return self._original_io.write(data)

    def flush(self) -> None:
        self._original_io.flush()

    def __getattr__(self, attr: str) -> Any:
        return getattr(self._original_io, attr)


SHIPPER_FLUSH_INTERVAL = 1
SHIPPER_FAILURE_BACKOFF_SECONDS = 1
LOG_BATCH_MAX_SIZE = 1000
SHIP_QUEUE_MAX_SIZE = 3 * LOG_BATCH_MAX_SIZE


class _ShutdownMessage:
    pass


_QueueElement = Union[str, _ShutdownMessage]


class _LogSender(threading.Thread):
    def __init__(self, session: api.Session, logs_metadata: Dict) -> None:
        self._queue = queue.Queue(maxsize=SHIP_QUEUE_MAX_SIZE)  # type: queue.Queue[_QueueElement]
        self._logs = collections.deque()  # type: collections.deque[str]
        self._session = session
        self._logs_metadata = logs_metadata
        self._buf = ""

        super().__init__(daemon=True, name="LogSenderThread")

    def write(self, data: str) -> None:
        self._queue.put(data)

    def close(self) -> None:
        self._queue.put(_ShutdownMessage())
        self.join(1)
        if self.is_alive():
            logger.info("Waiting for LogSender...")
            self.join(5)
            if self.is_alive():
                logger.warn("Failed to complete LogSender cleanup")
            else:
                logger.info("LogSender cleanup completed")

    def _pop_until_deadline(self, deadline: float) -> Iterator[_QueueElement]:
        while True:
            timeout = deadline - time.time()
            if timeout <= 0:
                break

            try:
                yield self._queue.get(timeout=timeout)
            except queue.Empty:
                break

    def run(self) -> None:
        while True:
            deadline = time.time() + SHIPPER_FLUSH_INTERVAL
            for m in self._pop_until_deadline(deadline):
                if isinstance(m, _ShutdownMessage):
                    self.ship()
                    return

                self._logs.append(m)
                if len(self._logs) >= LOG_BATCH_MAX_SIZE:
                    self.ship()

            self.ship()

    def ship(self) -> None:
        if len(self._logs) == 0:
            return

        msgs = []

        while len(self._logs):
            data = self._logs.popleft()
            self._buf += data
            while "\n" in self._buf:
                idx = self._buf.index("\n") + 1
                line = self._buf[:idx]
                self._buf = self._buf[idx:]

                msg = dict(self._logs_metadata)
                msg["log"] = line
                msgs.append(msg)

            if len(msgs) > LOG_BATCH_MAX_SIZE:
                self._ship(msgs)
                msgs = []

        if len(msgs) > 0:
            self._ship(msgs)

    def _ship(self, msgs: List[Dict]) -> None:
        self._session.post("task-logs", json=msgs)


class _UnmanagedTrialLogShipper(_LogShipper):
    def start(self) -> "_LogShipper":
        self._original_stdout, self._original_stderr = sys.stdout, sys.stderr

        logs_metadata = {
            "task_id": self._task_id,
            "timestamp": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        }

        if self._distributed:
            logs_metadata["rank"] = str(self._distributed.rank)

        self._log_sender = _LogSender(session=self._session, logs_metadata=logs_metadata)

        sys.stdout = _Interceptor(sys.stdout, self._log_sender.write)  # type: ignore
        sys.stderr = _Interceptor(sys.stderr, self._log_sender.write)  # type: ignore

        self._log_sender.start()

        return self

    def close(
        self,
        exc_type: Optional[type] = None,
        exc_val: Optional[BaseException] = None,
        exc_tb: Optional[types.TracebackType] = None,
    ) -> "_LogShipper":
        sys.stdout, sys.stderr = self._original_stdout, self._original_stderr
        self._log_sender.close()
        self._log_sender.join()

        return self
