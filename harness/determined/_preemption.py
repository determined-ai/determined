import logging
import socket
import threading
import time
from typing import Any, Optional

import requests

from determined.common import api
from determined.common.api import certs


class _PreemptionWatcher(threading.Thread):
    """
    _PreemptionWatcher connects to the master and asynchronously waits for a preemption signal.

    _PreemptionWatcher.should_preempt() is non-blocking (after the initial contact is made with the
    master) and returns a bool indicating if a preemption signal has been received yet.

    Example usage:

    .. code:: python

       with _PreemptionWatcher(master_url, trial_id, cert) as p:
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

    def __init__(self, master_url: str, trial_id: int, cert: certs.Cert) -> None:
        self._master_url = master_url
        self._trial_id = trial_id
        self._cert = cert

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
        return api.get(
            self._master_url,
            f"/api/v1/trials/{self._trial_id}/signals/preemption",
            cert=self._cert,
            params={"timeout": longpoll_time},
            timeout=70,
        ).json()["result"]["preempt"] is True

    def run(self) -> None:
        # Do a rapid check for the initial value.
        with self._cond:
            self._should_preempt = self._get_preemption(0)
            # Wake the main thread in case it was waiting for the initial response.
            self._cond.notify()

        # Continuously poll for preemption status to change.  Always retry after network failures;
        # if the master is unreachable, either user code will exit due to some more critical API
        # failure, or the user will kill the workload.
        while self._should_preempt and not self._should_quit:
            try:
                self._get_preemption(60)
            except requests.Timeout:
                logging.exception("timeout communicating with preemption API, retrying")
            except Exception:
                logging.exception("failure communicating with preemption API, retrying in 10s")
                time.sleep(10)

    def start(self) -> None:
        self._rsock, self._wsock = socket.socketpair()
        super().start()

    def close(self) -> None:
        if self._wsock is None:
            return
        # Sending any message at all will wake up the selector loop.
        self._wsock.send(b"quit")
        self.join()

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
