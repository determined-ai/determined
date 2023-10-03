import atexit
import logging
import sys
import threading
import time
import types
from typing import Any, Optional, Union

from determined.common import api
from determined.common.api import bindings

logger = logging.getLogger("determined.core")


class _Heartbeat:
    """Heartbeat controls the userspace trial state updates.

    For unmanaged / detached trials, this class will report the state changes
    to the determined-master.

    For managed trials, it'll not do anything as the state is controlled by
    determined-master itself.
    """

    def __init__(self, *, session: api.Session, trial_id: int) -> None:
        self._session = session
        self._trial_id = trial_id

    def start(self) -> "_Heartbeat":
        return self

    def close(
        self,
        exc_type: Optional[type],
        exc_val: Optional[BaseException],
        exc_tb: Optional[types.TracebackType],
    ) -> "_Heartbeat":
        return self

    def __enter__(self) -> "_Heartbeat":
        return self.start()

    def __exit__(
        self,
        exc_type: Optional[type] = None,
        exc_val: Optional[BaseException] = None,
        exc_tb: Optional[types.TracebackType] = None,
    ) -> "_Heartbeat":
        return self.close(exc_type, exc_val, exc_tb)


class _ManagedTrialHeartbeat(_Heartbeat):
    """
    ManagedTrialHeartbeat leaves the state management to the server.
    """

    pass


class _HeartbeatReporter(threading.Thread):
    def __init__(self, session: api.Session, trial_id: int) -> None:
        self._session = session
        self._trial_id = trial_id
        self._should_quit = False

        super().__init__(daemon=True, name="HeartbeatReporterThread")

    def _post_heartbeat(self) -> None:
        body = bindings.v1PatchTrialRequest(trialId=self._trial_id)
        bindings.patch_PatchTrial(session=self._session, body=body, trialId=self._trial_id)

    def run(self) -> None:
        while not self._should_quit:
            try:
                self._post_heartbeat()
            except Exception:
                logger.warning(
                    "failure communicating with heartbeat API (retrying in 10s):", exc_info=True
                )
                time.sleep(10)
            else:
                time.sleep(60)

    def close(self) -> None:
        self._should_quit = True

    def __enter__(self) -> "_HeartbeatReporter":
        self.start()
        return self

    def __exit__(self, *_: Any) -> None:
        self.close()


class _ExitHook(object):
    def __init__(self) -> None:
        self.exit_code = None  # typing: Optional[Union[int, str]]
        self.exception = None

    def hook(self) -> None:
        self._orig_exit = sys.exit
        self._orig_excepthook = sys.excepthook
        sys.exit = self.exit  # type: ignore
        sys.excepthook = self.exc_handler

    def exit(self, code: Optional[Union[int, str]] = 0) -> None:
        self.exit_code = code  # type: ignore
        self._orig_exit(code)

    def exc_handler(self, exc_type: Any, exc: Any, *args: Any) -> None:
        self.exception = exc
        self._orig_excepthook(exc_type, exc, *args)


class _UnmanagedTrialHeartbeat(_Heartbeat):
    """
    UnmanagedTrialHeartbeat updates the state on context enter/exit.
    """

    def start(self) -> "_Heartbeat":
        self._update_state(bindings.trialv1State.RUNNING)
        self._heartbeat = _HeartbeatReporter(self._session, self._trial_id)
        self._heartbeat.start()

        self._hook = _ExitHook()
        self._hook.hook()
        atexit.register(self._exit_handler)

        return self

    def _update_state(self, state: bindings.trialv1State) -> None:
        body = bindings.v1PatchTrialRequest(trialId=self._trial_id, state=state)
        bindings.patch_PatchTrial(session=self._session, body=body, trialId=self._trial_id)

    def _exit_handler(self) -> None:
        # TODO(ilia): check if we need to explicitly log the exception, e.g. if it was not
        # intercepted by the stdout/stderr capture.
        if self._hook.exception:
            exc = self._hook.exception
            self.close(type(exc), exc, None)
        elif self._hook.exit_code is not None and self._hook.exit_code != 0:
            exc = RuntimeError(f"exit code {self._hook.exit_code}")
            self.close(type(exc), exc, None)
        else:
            self.close()

    def close(
        self,
        exc_type: Optional[type] = None,
        exc_val: Optional[BaseException] = None,
        exc_tb: Optional[types.TracebackType] = None,
    ) -> "_Heartbeat":
        atexit.unregister(self._exit_handler)

        self._heartbeat.close()

        if exc_type is None:
            self._update_state(bindings.trialv1State.COMPLETED)
        else:
            self._update_state(bindings.trialv1State.ERROR)

        return self
