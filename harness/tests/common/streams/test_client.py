import collections
import json
from typing import Any, Deque, Iterable, Iterator, List, Optional, Tuple

import pytest
from lomond import events

from determined.common import streams
from determined.common.streams import wire


class MockWebSocket(streams.StreamWebSocket):
    def __init__(self) -> None:
        self._dq: Deque[str] = collections.deque()
        # _excpected and _history contain tuples of (call, (args...), retval)
        self._expected: List[Tuple[str, Tuple, Any]] = []
        self._history: List[Tuple[str, Tuple, Any]] = []

    def connect(self) -> Iterable:
        self._check_call("connect", ())
        return self

    def __iter__(self) -> Iterator:
        return self

    def __next__(self) -> Any:
        resp = self._check_call("__next__", ())
        if resp == StopIteration:
            raise StopIteration
        return resp

    def _print_history(self) -> str:
        def print_resp(resp: Any) -> str:
            if isinstance(resp, events.Text):
                return f"Text(text={repr(resp.text)})"
            return repr(resp)

        return "\n".join(
            f" - {call}{args} -> {print_resp(resp)}" for call, args, resp in self._history
        )

    def _check_call(self, call: str, args: Tuple) -> Any:
        if not self._expected:
            raise ValueError(
                "after successful calls:\n"
                + self._print_history()
                + f"\ngot unexpected call to {call}{args}"
            )
        exp_call, exp_args, resp = self._expected.pop(0)
        if call != exp_call or args != exp_args:
            raise ValueError(
                "after successful calls:\n"
                + self._print_history()
                + f"\nexpected: {exp_call}{exp_args}\nbut got:  {call}{args}"
            )
        self._history.append((exp_call, exp_args, resp))
        return resp

    def send_text(self, text: str) -> None:
        self._check_call("send_text", (json.loads(text),))

    def close(self) -> None:
        self._check_call("close", ())

    def get_backoff(self, retries: int) -> Optional[float]:
        backoff = self._check_call("get_backoff", (retries,))
        assert backoff is None or isinstance(backoff, int), backoff
        return backoff

    def expect_connect(self) -> None:
        self._expected.append(("connect", (), None))
        self.expect_next(retval=events.Connecting(None))
        self.expect_next(retval=events.Connected(None))
        self.expect_next(retval=events.Ready(None, None, None))

    def expect_send_text(self, obj: Any) -> None:
        self._expected.append(("send_text", (obj,), None))

    def expect_close(self) -> None:
        self._expected.append(("close", (), None))

    def expect_next(self, *, retval: Any) -> None:
        if not isinstance(retval, events.Event) and retval != StopIteration:
            retval = events.Text(json.dumps(retval))
        self._expected.append(("__next__", (), retval))

    def enqueue_close(self) -> None:
        self.expect_next(retval=events.Closing(None, None))
        self.expect_next(retval=events.Closed(None, None))
        self.expect_next(retval=events.Disconnected(None, None))
        self.expect_next(retval=StopIteration)

    def expect_get_backoff(self, retries: int, *, retval: Optional[float]) -> None:
        self._expected.append(("get_backoff", (retries,), retval))

    def complete_mock(self) -> None:
        msg = ""
        if self._expected:
            msg += "\nMissing expected calls:\n"
            msg += "\n".join(f"  - {call}{arg} -> {retval}" for call, arg, retval in self._expected)
        if self._dq:
            msg += "\nSome events were not drained from the websocket:\n"
            msg += "\n".join(f"  - {event}" for event in self._dq)
        if msg:
            raise ValueError("MockWebSocket detected errors!" + msg)


def test_reject_iter_before_subscribe() -> None:
    ws = MockWebSocket()
    stream = streams.Stream(ws)

    with pytest.raises(RuntimeError, match="before calling .subscribe()"):
        next(stream)


def test_reconnect() -> None:
    ws = MockWebSocket()
    stream = streams.Stream(ws)

    stream.subscribe(sync_id="sync1", projects=streams.ProjectSpec(workspace_id=2))

    ws.expect_connect()
    ws.expect_send_text(
        {
            "sync_id": "1",
            "known": {},
            "subscribe": {"projects": {"workspace_ids": [2]}},
        }
    )
    ws.expect_next(retval={"sync_id": "1", "complete": False})
    p1 = wire.ProjectMsg(
        id=1,
        name="p1",
        description="",
        archived=False,
        created_at=0,
        notes=None,
        workspace_id=2,
        user_id=1,
        immutable=False,
        state="",
        seq=1,
    )
    ws.expect_next(retval={"project": p1.to_json()})

    event = next(stream)
    assert event == streams.Sync("sync1", False), event
    event = next(stream)
    assert event == p1, event

    # Pretend the connection broke after sync-start but before sync-end.
    ws.enqueue_close()
    ws.expect_get_backoff(0, retval=0)
    ws.expect_connect()
    ws.expect_send_text(
        {
            "sync_id": "2",
            "known": {"projects": "1"},
            "subscribe": {"projects": {"workspace_ids": [2], "since": 1}},
        }
    )
    ws.expect_next(retval={"sync_id": "2", "complete": False})
    p2 = wire.ProjectMsg(
        id=2,
        name="p2",
        description="",
        archived=False,
        created_at=0,
        notes=None,
        workspace_id=2,
        user_id=1,
        immutable=False,
        state="",
        seq=2,
    )
    ws.expect_next(retval={"project": p2.to_json()})
    ws.expect_next(retval={"sync_id": "2", "complete": True})

    # Note that the caller of Stream doesn't see another sync-start message, they start with p2.
    event = next(stream)
    assert event == p2, event

    # Now they see the sync-end message.
    event = next(stream)
    assert event == streams.Sync("sync1", True), event

    # Pretend the connection broke after sync-end.
    ws.enqueue_close()
    ws.expect_get_backoff(0, retval=0)
    ws.expect_connect()
    ws.expect_send_text(
        {
            "sync_id": "3",
            "known": {"projects": "1-2"},
            "subscribe": {"projects": {"workspace_ids": [2], "since": 2}},
        }
    )
    ws.expect_next(retval={"sync_id": "3", "complete": False})
    ws.expect_next(retval={"sync_id": "3", "complete": True})
    p3 = wire.ProjectMsg(
        id=3,
        name="p3",
        description="",
        archived=False,
        created_at=0,
        notes=None,
        workspace_id=2,
        user_id=1,
        immutable=False,
        state="",
        seq=3,
    )
    ws.expect_next(retval={"project": p3.to_json()})

    # Note that neither sync-start nor sync-end get passed to the end user now.
    event = next(stream)
    assert event == p3, event

    ws.complete_mock()


def test_change_subscription() -> None:
    ws = MockWebSocket()
    stream = streams.Stream(ws)

    stream.subscribe(sync_id="sync1", projects=streams.ProjectSpec(workspace_id=2))

    # Start streaming.
    ws.expect_connect()
    ws.expect_send_text(
        {
            "sync_id": "1",
            "known": {},
            "subscribe": {"projects": {"workspace_ids": [2]}},
        }
    )
    ws.expect_next(retval={"sync_id": "1", "complete": False})
    p1 = wire.ProjectMsg(
        id=1,
        name="p1",
        description="",
        archived=False,
        created_at=0,
        notes=None,
        workspace_id=2,
        user_id=1,
        immutable=False,
        state="",
        seq=1,
    )
    ws.expect_next(retval={"project": p1.to_json()})
    ws.expect_next(retval={"sync_id": "1", "complete": True})

    event = next(stream)
    assert event == streams.Sync("sync1", False), event
    event = next(stream)
    assert event == p1, event
    event = next(stream)
    assert event == streams.Sync("sync1", True), event

    # Start a new subscription, and expect additional old-subscription messages to be ignored.
    ws.expect_send_text(
        {
            "sync_id": "2",
            "known": {"projects": "1"},
            "subscribe": {"projects": {"workspace_ids": [3], "since": 1}},
        }
    )
    stream.subscribe(sync_id="sync2", projects=streams.ProjectSpec(workspace_id=3))

    # Also queue up a third subscription which won't fire until the second subscription's sync-end.
    stream.subscribe(sync_id="sync3", projects=streams.ProjectSpec(workspace_id=4))

    p1b = wire.ProjectMsg(
        id=1,
        name="p1",
        description="updated description",
        archived=False,
        created_at=0,
        notes=None,
        workspace_id=2,
        user_id=1,
        immutable=False,
        state="",
        seq=1000,  # Note the high seq here will be ignored since this is from the old subscription.
    )
    ws.expect_next(retval={"project": p1b.to_json()})

    # messages for the new subscription begin.
    ws.expect_next(retval={"sync_id": "2", "complete": False})
    ws.expect_next(retval={"projects_deleted": "1"})
    p2 = wire.ProjectMsg(
        id=2,
        name="p2",
        description="",
        archived=False,
        created_at=0,
        notes=None,
        workspace_id=3,
        user_id=1,
        immutable=False,
        state="",
        seq=2,
    )
    ws.expect_next(retval={"project": p2.to_json()})
    ws.expect_next(retval={"sync_id": "2", "complete": True})

    event = next(stream)
    assert event == streams.Sync("sync2", False), event
    event = next(stream)
    assert event == wire.ProjectsDeleted("1"), event
    event = next(stream)
    assert event == p2, event

    # When we detect the second subscription's sync-end, we will send the third subscription.
    ws.expect_send_text(
        {
            "sync_id": "3",
            "known": {"projects": "2"},
            "subscribe": {"projects": {"workspace_ids": [4], "since": 2}},
        }
    )

    event = next(stream)
    assert event == streams.Sync("sync2", True), event

    ws.complete_mock()


if __name__ == "__main__":
    test_reject_iter_before_subscribe()
    test_reconnect()
    test_change_subscription()
