import enum
import logging
import threading
import time
from typing import Any, Optional

import requests

from determined import core
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.core")


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


class PreemptMode(enum.Enum):
    """
    ``PreemptMode`` defines the calling behavior of the ``PreemptContext.should_preempt()`` call.

    When mode is ``WorkersAskChief`` (the default), all workers must call ``should_preempt()``
    in step.  Only the chief actually communicates with the master, then the chief broadcasts its
    decision to all workers.  This guarantees that all workers decide to preempt at the exact same
    time.

    When mode is ``ChiefOnly``, only the chief is allowed to call
    ``PreemptContext.should_preempt()``.  Usually this implies you must manually inform the workers
    if they should preempt or not.

    When mode is ``WorkersAskMaster``, each worker contacts the master independently in order to
    decide to preempt or not.  Each worker receives the preemption signal at roughly the same time,
    but it becomes your responsibility to tolerate situations where some workers have exited due to
    preemption and others have not.
    """

    WorkersAskChief = "WORKERS_ASK_CHIEF"
    ChiefOnly = "CHIEF_ONLY"
    WorkersAskMaster = "WORKERS_ASK_MASTER"


class PreemptContext:
    """
    ``PreemptContext`` gives access to preemption signals that originate from a user action, such as
    pausing an experiment using the WebUI or the CLI, from in the Determined scheduler.
    """

    def __init__(
        self,
        session: Session,
        allocation_id: str,
        dist: core.DistributedContext,
        preempt_mode: PreemptMode = PreemptMode.WorkersAskChief,
    ) -> None:
        self._session = session
        self._allocation_id = allocation_id
        self._dist = dist
        self._preempt_mode = PreemptMode(preempt_mode)
        self._watcher = None
        self._started = False
        if self._dist.get_rank() == 0 or self._preempt_mode == PreemptMode.WorkersAskMaster:
            self._watcher = _PreemptionWatcher(session, allocation_id)
        self._ack_sent = False

    def start(self) -> "PreemptContext":
        if self._started:
            raise RuntimeError("you cannot call PreemptContext.start() multiple times")
        self._started = True
        if self._watcher is not None:
            self._watcher.start()
        return self

    def close(self) -> None:
        if self._watcher is not None:
            self._watcher.close()

    def __enter__(self) -> "PreemptContext":
        return self.start()

    def __exit__(self, *_: Any) -> None:
        self.close()

    def should_preempt(self, auto_ack: bool = True) -> bool:
        """
        ``should_preempt()`` returns ``True`` if the task should shut itself down, or ``False``
        otherwise.

        The requirements on the the caller and the synchronization between workers during a call
        to ``should_preempt()`` are defined by the ``preempt_mode`` argument passed to the
        ``PreemptContext`` constructor.

        Arguments:
            auto_ack (``bool``, optional): In order for the task to be restarted by the
                Determined master after shutting down due to preemption, the task must acknowledge
                the preemption signal to the Determined master.  When ``auto_ack`` is ``True`` this
                acknowledgement is automatically sent the first time that ``should_preempt()``
                returns ``True``.  If you might choose not to exit after receiving the preemption
                signal (but you still want to check the signal for some purpose), then you should
                set ``auto_ack`` to ``False``.  Then if you later do decide to comply with the
                preemption signal, it is your responsibility to call
                ``acknowledge_preemption_signal()`` manually any time before exiting.  Defaults to
                ``True``.

        .. note::

            Currently, only blocking behavior is supported when checking ``should_preempt()``, so it
            is not performant enough to call every batch.
        """
        if not self._started:
            # Calling should_preempt on the watcher without starting the watcher would hang.
            raise RuntimeError(
                "you cannot call PreemptContext.should_preempt() before PreemptContext.start()"
            )
        if self._watcher is not None:
            # Have watcher; either this is the chief or we are in WorkersAskMaster mode.
            out = self._watcher.should_preempt()
            if auto_ack and out and not self._ack_sent:
                # Tell the master that user code has received the preemption signal.
                self.acknowledge_preemption_signal()
                self._ack_sent = True
            if self._preempt_mode == PreemptMode.WorkersAskChief:
                _ = self._dist.broadcast(out)
        else:
            # No watcher; we should ask the chief or either we should not be here.
            if self._preempt_mode == PreemptMode.ChiefOnly:
                raise RuntimeError(
                    "PreemptContext was configured with preempt_mode=ChiefOnly but "
                    ".should_preempt() was called from non-chief worker of "
                    f"rank={self._dist.get_rank()}"
                )
            out = self._dist.broadcast(None)
            assert isinstance(out, bool)

        logger.debug(f"should_preempt() -> {out}")
        return out

    def acknowledge_preemption_signal(self) -> None:
        """
        ``acknowledge_preemption_signal()`` tells the Determined master that you are shutting down,
        but you have not finished your work and you expect to be restarted later to complete it.

        This is important to tell the master explicitly because otherwise if the python process
        exits with a zero exit code, the master interprets that as a completed task, and the task
        does not get rescheduled.

        By default, ``acknowledge_preemption_signal()`` is called automatically the first time that
        ``should_preempt()`` returns ``True``, unless ``should_preempt()`` is called with
        ``auto_ack=False``.
        """
        logger.debug("acknowledge_preemption_signal()")
        self._session.post(f"/api/v1/allocations/{self._allocation_id}/signals/ack_preemption")


class DummyPreemptContext(PreemptContext):
    """
    DummyPreemptContext is a PreemptContext where should_preempt() always returns False.
    """

    def __init__(
        self,
        dist: core.DistributedContext,
        preempt_mode: PreemptMode = PreemptMode.WorkersAskChief,
    ) -> None:
        self._dist = dist
        self._preempt_mode = PreemptMode(preempt_mode)
        self._started = False

    def start(self) -> "PreemptContext":
        if self._started:
            raise RuntimeError("you cannot call PreemptContext.start() multiple times")
        self._started = True
        return self

    def close(self) -> None:
        pass

    def __enter__(self) -> "PreemptContext":
        return self.start()

    def __exit__(self, *_: Any) -> None:
        pass

    def should_preempt(self, auto_ack: bool = True) -> bool:
        if not self._started:
            # Match the requirements of the real PreemptContext class.
            raise RuntimeError(
                "you cannot call PreemptContext.should_preempt() before PreemptContext.start()"
            )
        if self._dist.rank == 0:
            if self._preempt_mode == PreemptMode.WorkersAskChief:
                # Even though we always return False, preserve the synchronization behavior to avoid
                # giving the user a weird inconsistency between managed and unmanaged dtrain code.
                _ = self._dist.broadcast(False)
        else:
            if self._preempt_mode == PreemptMode.ChiefOnly:
                raise RuntimeError(
                    "PreemptContext was configured with preempt_mode=ChiefOnly but "
                    ".should_preempt() was called from non-chief worker of "
                    f"rank={self._dist.get_rank()}"
                )
            if self._preempt_mode == PreemptMode.WorkersAskChief:
                _ = self._dist.broadcast(None)
        return False

    def acknowledge_preemption_signal(self) -> None:
        pass
