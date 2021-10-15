import logging
import threading
import time
from typing import Any, Optional

import requests

import determined as det
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.generic")


class _PreemptionWatcher(threading.Thread):
    """
    _PreemptionWatcher connects to the master and asynchronously waits for a preemption signal.

    _PreemptionWatcher.should_preempt() is non-blocking (after the initial contact is made with the
    master) and returns a bool indicating if a preemption signal has been received yet.

    Example usage:

    .. code:: python

       with _PreemptionWatcher(session, allocation_id) as p:
           print("started!")
           for i in range(10):
               if p.should_preempt():
                   print("preempted!")
                   break
               print('no preemption yet, waiting...')
               time.sleep(1)
           else:
               print('finished without preemption signal')
    """

    def __init__(self, session: Session, allocation_id: str) -> None:
        self._session = session
        self._allocation_id = allocation_id

        self._should_preempt = None  # type: Optional[bool]
        self._should_quit = False

        self._cond = threading.Condition()

        # Set daemon=True, since the requests library only supports blocking reads.  Under the hood,
        # the requests library uses buffered IO on top of the socket, which means that we can't even
        # use select() to know if a read would block; select() won't know that some data is
        # available in the buffer.  We would probably have to move to an async-based HTTP library
        # to make the PreemptionWatcher properly preemptible.
        super().__init__(daemon=True)

    def _get_preemption(self, longpoll_time: int) -> bool:
        logger.debug(f"(_PreemptionWatcher thread) _get_preemption({longpoll_time})")
        return (
            self._session.get(
                f"/api/v1/allocations/{self._allocation_id}/signals/preemption",
                params={"timeout_seconds": str(longpoll_time)},
                timeout=longpoll_time + 10,
            ).json()["preempt"]
            is True
        )

    def run(self) -> None:
        # Do a rapid check for the initial value.
        with self._cond:
            try:
                self._should_preempt = self._get_preemption(0)
            except requests.Timeout:
                logging.warning(
                    "timeout during initial preemption API check (continuing):", exc_info=True
                )
                self._should_preempt = False
            except Exception:
                logging.warning(
                    "failure during initial preemption API check (continuing):", exc_info=True
                )
                self._should_preempt = False
            finally:
                # Wake the main thread in case it was waiting for the initial response.
                self._cond.notify()

        # Continuously poll for preemption status to change.  Always retry after network failures;
        # if the master is unreachable, either user code will exit due to some more critical API
        # failure, or the user will kill the task.
        while not self._should_preempt and not self._should_quit:
            try:
                self._should_preempt = self._get_preemption(60)
            except requests.Timeout:
                logging.warning(
                    "timeout communicating with preemption API (retrying):", exc_info=True
                )
            except Exception:
                logging.warning(
                    "failure communicating with preemption API (retrying in 10s):", exc_info=True
                )
                time.sleep(10)

    def close(self) -> None:
        # TODO: For now we have to set daemon=True for the thread, so there's no point in joining.
        # self.join()
        self._should_quit = True

    def __enter__(self) -> "_PreemptionWatcher":
        self.start()
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()

    def should_preempt(self) -> bool:
        # Optimize to avoid locking the threading.Conditional object if we can avoid it.
        if self._should_preempt is not None:
            return self._should_preempt

        # Block until the Preemption API has streamed the initial response.
        with self._cond:
            while self._should_preempt is None:
                self._cond.wait()
        return self._should_preempt


class Preemption:
    """
    Some preemption-related APIs.
    """

    def __init__(
        self,
        session: Session,
        allocation_id: str,
        dist: det.DistributedContext,
    ) -> None:
        self._session = session
        self._allocation_id = allocation_id
        self._dist = dist
        if self._dist.get_rank() == 0:
            self._watcher = _PreemptionWatcher(
                session, allocation_id
            )  # type: Optional[_PreemptionWatcher]
        else:
            self._watcher = None
        self._ack_sent = False

    def start(self) -> "Preemption":
        if self._watcher is not None:
            self._watcher.start()
        return self

    def close(self) -> None:
        if self._watcher is not None:
            self._watcher.close()

    def __enter__(self) -> "Preemption":
        return self.start()

    def __exit__(self, *_: Any) -> None:
        self.close()

    def should_preempt(self, chief_only: bool = False, auto_ack: bool = True) -> bool:
        """
        Currently, we only support blocking behavior when checking should_preempt(), so it is not
        performant enough to call every batch.

        Arguments:
            chief_only (``bool``, default ``False``): In multi-worker training, chief_only
                configures whether or not the result of should_preempt() is being checked on every
                worker or whether it is only being checked on the chief worker.  When all workers
                are checking should_preempt() (the default case), the chief will broadcast its
                decision to all the workers in order to guarantee that all workers agree on the
                decision to preempt.  You should set chief_only if only the chief worker is making
                the call to should_preempt(), in which case it is your responsiblity to have the
                chief shut down training on all workers via some other means.
            auto_ack (``bool``, default ``True``): In order for the task to be restarted by the
                Determined master after shutting down due to preemption, the task must acknowledge
                the preemption signal to the Determined master.  When auto_ack is True (the default
                case) this acknowledgement is automatically sent the first time that
                should_preempt() returns True.  If you might choose not to exit after receiving the
                preemption signal (but you still want to check the signal for some purpose), then
                you should set auto_ack to False.  Then if you later do decide to comply with the
                preemption signal, it will be your responsibility to call
                acknowledge_preemption_signal() manually any time before exiting.
        """
        if self._watcher is not None:
            # Chief.
            out = self._watcher.should_preempt()
            if auto_ack and out and not self._ack_sent:
                # Tell the master that user code has received the preemption signal.
                self.acknowledge_preemption_signal()
            if not chief_only:
                _ = self._dist._zmq_broadcast(out)
        else:
            # Non-chief.
            if chief_only:
                raise ValueError(
                    "should_preempt(chief_only=True) was called from non-chief worker of "
                    f"rank={self._dist.get_rank()}"
                )
            out = self._dist._zmq_broadcast(None)
            assert isinstance(out, bool)

        logger.debug(f"should_preempt() -> {out}")
        return out

    def acknowledge_preemption_signal(self) -> None:
        """
        acknowledge_preemption_signal() tells the Determined master that you are shutting down, but
        you have not finished your work and you expect to be restarted later to complete it.

        This is important to tell the master explicitly, since normally if the python process exits
        with a zero exit code, the master would interpret that as a completed task, and it
        would not get rescheduled at a later time.

        acknowledge_preemption_signal() is normally called automatically the first time that
        should_preempt() returns True, unless should_preempt() is called with auto_ack=False.
        """
        logger.debug("acknowledge_preemption_signal()")
        self._session.post(f"/api/v1/allocations/{self._allocation_id}/signals/ack_preemption")
