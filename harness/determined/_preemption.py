import logging
import threading
import time
from typing import Any, Optional

import requests

import determined as det
from determined.common import api
from determined.common.api import certs


class _PreemptionWatcher(threading.Thread):
    """
    _PreemptionWatcher connects to the master and asynchronously waits for a preemption signal.

    _PreemptionWatcher.should_preempt() is non-blocking (after the initial contact is made with the
    master) and returns a bool indicating if a preemption signal has been received yet.

    Example usage:

    .. code:: python

       with _PreemptionWatcher(session, trial_id) as p:
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

    def __init__(self, session: str, trial_id: int) -> None:
        self._session = session
        self._trial_id = trial_id

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
        return self._session.get(
            f"/api/v1/trials/{self._trial_id}/signals/preemption",
            params={"timeout_seconds": str(longpoll_time)},
            timeout=longpoll_time + 10,
        ).json()["preempt"] is True

    def run(self) -> None:
        # Do a rapid check for the initial value.
        with self._cond:
            try:
                self._should_preempt = self._get_preemption(0)
            except requests.Timeout:
                logging.exception("timeout during initial preemption API check, continuing")
                self._should_preempt = False
            except Exception:
                logging.exception("failureduring initial preemption API check, continuing")
                self._should_preempt = False
            finally:
                # Wake the main thread in case it was waiting for the initial response.
                self._cond.notify()


        # Continuously poll for preemption status to change.  Always retry after network failures;
        # if the master is unreachable, either user code will exit due to some more critical API
        # failure, or the user will kill the workload.
        while not self._should_preempt and not self._should_quit:
            try:
                self._should_preempt = self._get_preemption(60)
            except requests.Timeout:
                logging.exception("timeout communicating with preemption API, retrying")
            except Exception:
                logging.exception("failure communicating with preemption API, retrying in 10s")
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
        self, session, trial_id, dist: det.DistributedContext,
    ) -> None:
        self._dist = dist
        if self._dist.get_rank() == 0:
            self._watcher = _PreemptionWatcher(
                session, trial_id
            )  # type: Optional[_PreemptionWatcher]
        else:
            self._watcher = None

        self._will_preempt = False

    def start(self):
        if self._watcher is not None:
            self._watcher.start()

    def close(self):
        if self._watcher is not None:
            self._watcher.close()

    def __enter__(self) -> "Preemption":
        self.start()
        return self

    def __exit__(self, *_) -> None:
        self.close()

    def should_preempt(self, broadcast=True) -> bool:
        """
        Currently, we only support blocking behavior when checking should_preempt(), so it is not
        performant enough to call every batch.
        """
        # The chief broadcasts the results to all workers.
        if self._watcher is not None:
            val = self._watcher.should_preempt()  # type: Any
        else:
            val = None
        out = self._dist._zmq_broadcast(val)
        return out
