import json
import selectors
import socket
import threading
from typing import Any, Optional

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

        # We will use a selector loop as a non-blocking way to stream through the requests library.
        # We use a socketpair for a cross-platform way to wake up a selector loop mid-stream.
        self._rsock = None  # type: Optional[socket.socket]
        self._wsock = None  # type: Optional[socket.socket]

        self._should_preempt = None  # type: Optional[bool]

        self._cond = threading.Condition()
        super().__init__()

    def run(self) -> None:
        assert self._rsock is not None
        assert self._wsock is not None

        with api.get(
            self._master_url,
            f"/api/v1/trials/{self._trial_id}/signals/preemption",
            stream=True,
            cert=self._cert,
        ) as r:
            lines = r.iter_lines()

            # The first response is always immediate, so we always block for it.  This is doubly
            # important because we can only select on the HTTP connection socket _after_ we call
            # next(lines) at least once.
            for line in lines:
                if not line:
                    # Empty keepalive message.
                    continue
                break

            j = json.loads(line)

            with self._cond:
                self._should_preempt = j["result"]["preempt"]
                # Wake the main thread in case it was waiting for the main response.
                self._cond.notify()

            if self._should_preempt:
                return

            # It may be a long time between the first response and the second.  We want to be
            # responsive to .close() calls from the main thread, so we block on input from either
            # the HTTP connection socket or on our own _rsock socket.
            with selectors.DefaultSelector() as sel:
                sel.register(r.raw, selectors.EVENT_READ)
                sel.register(self._rsock, selectors.EVENT_READ)

                while True:
                    for key, _ in sel.select():
                        if key.fileobj == self._rsock:
                            # self._rsock is only written to when we are supposed to close.
                            return

                        assert key.fileobj == r.raw
                        # Calling next() is blocking, but it should only block for a trivial amount
                        # of time; basically only in the case that one message from the preemption
                        # API got broken into two network packets.
                        line = next(lines)

                        if not line:
                            # Empty keepalive message.
                            continue

                        j = json.loads(line)

                        self._should_preempt = j["result"]["preempt"]

                        if self._should_preempt:
                            return

    def start(self) -> None:
        self._rsock, self._wsock = socket.socketpair()
        super().start()

    def close(self) -> None:
        if self._wsock is None:
            return
        # Sending any message at all will wake up the selector loop.
        self._wsock.send(b"quit")
        self.join()
        self._rsock.close()
        self._rsock = None
        self._wsock.close()
        self._wsock = None

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
