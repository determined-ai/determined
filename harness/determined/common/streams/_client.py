import abc
import collections
import json
import time
from typing import (
    Any,
    Callable,
    Deque,
    Dict,
    Iterable,
    List,
    Optional,
    Sequence,
    Set,
    Tuple,
    Type,
    Union,
)

import lomond
from lomond import events

from determined.common import api, detlomond, streams
from determined.common.streams import wire


class StreamWebSocket(metaclass=abc.ABCMeta):
    """
    StreamWebSocket encapsulates all IO for the Stream so that it's easy to unit test.
    """

    @abc.abstractmethod
    def connect(self) -> Iterable:
        pass

    @abc.abstractmethod
    def send_text(self, text: str) -> None:
        pass

    @abc.abstractmethod
    def close(self) -> None:
        pass

    @abc.abstractmethod
    def get_backoff(self, retries: int) -> Optional[float]:
        pass


class LomondStreamWebSocket(StreamWebSocket):
    """
    The "real" StreamWebSocket, used outside of tests.
    """

    def __init__(self, sess: api.Session) -> None:
        self.ws: lomond.WebSocket = None
        self.sess = sess

        # About 60 seconds of auto-retry.
        self._backoffs = [0, 1, 2, 4, 8, 10, 10, 10, 15]

    def connect(self) -> Iterable:
        if self.ws:
            self.ws.close()

        self.ws = detlomond.WebSocket(self.sess, "stream")

        it = self.ws.connect(ping_rate=0)
        assert isinstance(it, Iterable)
        return it

    def send_text(self, text: str) -> None:
        assert self.ws
        self.ws.send_text(text)

    def close(self) -> None:
        if not self.ws:
            return
        self.ws.close()
        self.ws = None

    def get_backoff(self, retries: int) -> Optional[float]:
        try:
            return self._backoffs[retries]
        except IndexError:
            return None


def int_or_list(x: Optional[Union[int, Sequence[int]]]) -> Optional[List[int]]:
    if x is None:
        return None
    if isinstance(x, int):
        return [x]
    return list(x)


class ProjectSpec:
    def __init__(
        self,
        workspace_id: Optional[Union[int, Sequence[int]]] = None,
        project_id: Optional[Union[int, Sequence[int]]] = None,
    ) -> None:
        self.workspace_id = workspace_id
        self.project_id = project_id

    def _copy(self) -> "ProjectSpec":
        return ProjectSpec(self.workspace_id, self.project_id)

    def _to_wire(self) -> Dict[str, Any]:
        return wire.ProjectSubscriptionSpec(
            workspace_ids=int_or_list(self.workspace_id),
            project_ids=int_or_list(self.project_id),
        ).to_json()


class ModelSpec:
    def __init__(
        self,
        workspace_id: Optional[Union[int, Sequence[int]]] = None,
        model_id: Optional[Union[int, Sequence[int]]] = None,
        user_id: Optional[Union[int, Sequence[int]]] = None,
    ) -> None:
        self.workspace_id = workspace_id
        self.model_id = model_id
        self.user_id = user_id

    def _copy(self) -> "ModelSpec":
        return ModelSpec(self.workspace_id, self.model_id, self.user_id)

    def _to_wire(self) -> Dict[str, Any]:
        return wire.ModelSubscriptionSpec(
            workspace_ids=int_or_list(self.workspace_id),
            model_ids=int_or_list(self.model_id),
            user_ids=int_or_list(self.user_id),
        ).to_json()


class ModelVersionSpec:
    def __init__(
        self,
        model_version_id: Optional[Union[int, Sequence[int]]] = None,
        model_id: Optional[Union[int, Sequence[int]]] = None,
        user_id: Optional[Union[int, Sequence[int]]] = None,
    ) -> None:
        self.model_version_id = model_version_id
        self.model_id = model_id
        self.user_id = user_id

    def _copy(self) -> "ModelSpec":
        return ModelSpec(self.model_version_id, self.model_id, self.user_id)

    def _to_wire(self) -> Dict[str, Any]:
        return wire.ModelVersionSubscriptionSpec(
            model_version_ids=int_or_list(self.model_version_id),
            model_ids=int_or_list(self.model_id),
            user_ids=int_or_list(self.user_id),
        ).to_json()


class Sync:
    def __init__(self, sync_id: Any, complete: bool) -> None:
        self.sync_id = sync_id
        self.complete = complete

    def to_json(self) -> Dict[str, Any]:
        return dict(vars(self))

    def __repr__(self) -> str:
        return f"Sync(sync_id={self.sync_id}, complete={self.complete})"

    def __eq__(self, other: object) -> bool:
        return (
            isinstance(other, Sync)
            and other.sync_id == self.sync_id
            and other.complete == self.complete
        )


MsgHandler = Callable[[Any], Any]

IterResult = Union[wire.ServerMsg, wire.DeleteMsg, Sync]


class KeyCache:
    """
    KeyCache caches only primary keys.

    KeyCache is just enough caching to allow Stream to automatically reconnect.
    """

    def __init__(self, keys: Optional[Set[int]] = None):
        self.keys = keys or set()
        self.maxseq = 0

    def upsert(self, id_: int, seq: int) -> None:
        self.keys.add(id_)
        self.maxseq = max(self.maxseq, seq)

    def delete_one(self, id_: int) -> None:
        self.keys.discard(id_)

    def delete_msg(self, deleted: str) -> None:
        for i in streams.range_encoded_keys(deleted):
            self.delete_one(i)

    def known(self) -> str:
        if not self.keys:
            return ""

        out: List[str] = []
        keys = iter(sorted(self.keys))
        start = next(keys)
        end = start

        def emit(start: int, end: int, out: List[str]) -> None:
            if start == end:
                # single integer
                out.append(str(start))
            else:
                # range of integers
                out.append(f"{start}-{end}")

        for k in keys:
            if k == end + 1:
                # continue the same range as before
                end = k
                continue
            # end of a range
            emit(start, end, out)
            # start on the next range
            start = k
            end = start
        # deal with the final range
        emit(start, end, out)

        return ",".join(out)


class Stream:
    """
    Stream is a client to the Determined streaming updates system featuring auto-reconnects.

    Streaming updates is a mechanism for efficiently monitoring entities in the Determined platform.
    For example, you could stream all metrics from a run or all checkpoints from a trial, or the
    state of one or more experiments.

    The Stream can be iterated over to read events related to subscriptions.  A subscription is set
    with .subscribe().  There is only ever one active subscription at a time.  Each call to
    .subscribe() causes the stream to yield a Sync(complete=False) message, followed by any number
    of subscription-related messages, then a Sync(complete=True) message.  The second Sync indicates
    that the client has loaded all state from the Determined master, and additional
    subscription-related messages are yielded as the master publishes them, unless another call to
    .subscribe() has been made.

    If another .subscribe() call has been made, the Stream will cease to yield messages from the
    first subscription and will begin with the Sync(complete=False) message from the second
    subscription, and then the second subscription continues in the manner of the first.

    The behavior of the Stream while streaming could be described by the following steps:

      1. Begin by sending the subscription message for A

      2. Ignore all messages until the sync start message for A

      3. Collect all offline messages for A, until the sync end message for A.

      4. Collect online messages for A until another .subscribe() call has been made.

      5. Start from step 1 again with the new subscription.
    """

    def __init__(self, ws: StreamWebSocket) -> None:
        self._ws = ws
        # Our stream-level in-memory cache: just enough to handle automatic reconnects.
        self._projects = KeyCache()
        self._models = KeyCache()
        self._model_versions = KeyCache()
        # The websocket events.  We'll connect (and reconnect) lazily.
        self._ws_iter: Optional[Iterable] = None
        self._closed = False
        # Parsed messages we've collected but haven't passed out yet.
        self._pending: Deque[IterResult] = collections.deque()

        # Is our websocket in the "ready" state?
        self._ready = False
        # What was the last sync_id we sent (on this connection)?
        self._sync_sent: Optional[str] = None
        # What was the last sync_id we saw a sync-start message for (on this connection)?
        self._sync_started = ""
        # What was the last sync_id we saw a sync-stop message for (on this connection)?
        self._sync_complete = ""
        # how many syncs have we sent (ever)?
        self._num_syncs = 0

        # What subscription specs have been requested?
        self._specs: List[Tuple[Any, Dict[str, Any]]] = []
        # What was the most recent subscription spec we sent?
        self._prev_spec: Optional[Dict[str, Any]] = None
        # What sync_id was provided by the user during the currently-active .subscribe()?
        self._user_sync_id: Any = None
        # We might see multiple sync-start and sync-end messages for a single subscription, due to
        # automatic reconnects, but we keep that transparent to the end user.
        self._user_sync_start_sent = False
        self._user_sync_end_sent = False

        self.handlers: Dict[str, MsgHandler] = {
            "project": self._make_upsertion_handler(wire.ProjectMsg, self._projects),
            "projects_deleted": self._make_deletion_handler(wire.ProjectsDeleted, self._projects),
            "model": self._make_upsertion_handler(wire.ModelMsg, self._models),
            "models_deleted": self._make_deletion_handler(wire.ModelsDeleted, self._models),
            "modelversion": self._make_upsertion_handler(
                wire.ModelVersionMsg, self._model_versions
            ),
            "modelversions_deleted": self._make_deletion_handler(
                wire.ModelVersionMsg, self._model_versions
            ),
        }

        self._retries = 0
        self._last_conn_failure = ""

    def _make_upsertion_handler(self, typ: Type, cache: KeyCache) -> MsgHandler:
        def handler(val: Any) -> Any:
            record = typ.from_json(val)
            cache.upsert(record.id, record.seq)
            return record

        return handler

    def _make_deletion_handler(self, typ: Type, cache: KeyCache) -> MsgHandler:
        def handler(val: Any) -> Any:
            # ignore pointless deletion messages
            if val == "":
                return None
            cache.delete_msg(val)
            return typ.from_json(val)

        return handler

    def _backoff(self) -> None:
        backoff = self._ws.get_backoff(self._retries)
        if backoff is None:
            raise ConnectionError(self._last_conn_failure)
        self._retries += 1
        time.sleep(backoff)

    def _send_spec(self, spec: Dict[str, Any]) -> None:
        self._num_syncs += 1
        sync_id = str(self._num_syncs)
        # build our startup message
        since = {
            "projects": self._projects.maxseq,
            "models": self._models.maxseq,
            "modelversions": self._model_versions.maxseq,
        }
        subscribe = {k: v._to_wire() for k, v in spec.items()}
        # add since info to our initial subscriptions
        #
        # Note: it is important that we calculate our since values at the moment that we send the
        # new subscription, and that we ignore additional messages which arrive for the old
        # subscription after we send this new subscription.  The reason is that the additional
        # messages for the old subscription would mess with the state of our KeyCache objects (from
        # which we calculated the since values of the new subscription).  However, the messages from
        # our new subscription will be tailored to the state of the KeyCache objects at this moment.
        # So effectively, ignoring additional messages from the old subscription protects the state
        # of our KeyCache objects.
        for k, v in subscribe.items():
            if since[k]:
                v["since"] = since[k]
        startup_msg = {
            "sync_id": sync_id,
            "known": {
                k: v
                for k, v in {
                    "projects": self._projects.known(),
                    "models": self._models.known(),
                    "modelversions": self._models.known(),
                }.items()
                if v
            },
            "subscribe": subscribe,
        }
        self._ws.send_text(json.dumps(startup_msg))
        self._sync_sent = sync_id
        self._prev_spec = spec

    def _advance_subscription(self) -> None:
        if not self._ready:
            return
        if not self._prev_spec and not self._specs:
            return

        # Have we not sent the first spec on this connection?
        if not self._sync_sent:
            # Either resend the last spec or pick the first requested one.
            if self._prev_spec:
                spec = self._prev_spec
            else:
                self._user_sync_id, spec = self._specs.pop(0)
                self._user_sync_start_sent = False
                self._user_sync_end_sent = False
            self._send_spec(spec)
            return

        # Are we ready to send the next sync?
        if self._specs and self._sync_complete == self._sync_sent:
            self._user_sync_id, spec = self._specs.pop(0)
            self._user_sync_start_sent = False
            self._user_sync_end_sent = False
            self._send_spec(spec)
            return

    def __iter__(self) -> "Stream":
        # You can iterate on this Stream directly.
        return self

    def __next__(self) -> IterResult:
        if self._closed:
            raise StopIteration
        if not self._prev_spec and not self._specs:
            raise RuntimeError("you cannot iterate through the Stream before calling .subscribe()")
        while True:
            if self._pending:
                return self._pending.popleft()
            if self._ws_iter is None:
                self._ws_iter = self._ws.connect()
                self._sync_sent = None
                self._sync_started = ""
                self._sync_complete = ""
            for event in self._ws_iter:
                if isinstance(event, (events.ConnectFail, events.Rejected, events.ProtocolError)):
                    # Connection failed; reset connection-related state.
                    self._ready = False
                    self._last_conn_failure = event.name
                elif isinstance(event, events.Poll):
                    pass
                elif isinstance(event, events.Connecting):
                    pass
                elif isinstance(event, events.Connected):
                    pass
                elif isinstance(event, events.Ready):
                    self._ready = True
                    self._retries = 0
                    # Becoming ready should trigger sending a subscription, since this websocket
                    # connection hasn't sent any yet.
                    self._advance_subscription()

                elif isinstance(event, (events.Text)):
                    # parse and process this message
                    msg = json.loads(event.text)

                    # Handle sync messages first.
                    if "sync_id" in msg:
                        if not msg["complete"]:
                            # sync-start message
                            self._sync_started = msg["sync_id"]
                            # do we need to forward the sync-start indication?
                            if not self._user_sync_start_sent:
                                self._user_sync_start_sent = True
                                return Sync(self._user_sync_id, False)
                        else:
                            # sync-done message
                            self._sync_complete = msg["sync_id"]
                            # capture _user_sync_id before advance_subscription can change it
                            old_sync_id = self._user_sync_id
                            # This can trigger sending the next subscription.
                            self._advance_subscription()
                            # do we need to forward the sync-end indication?
                            if not self._user_sync_end_sent:
                                self._user_sync_end_sent = True
                                return Sync(old_sync_id, True)
                        continue

                    # Ignore all messages between when we send a new subscription and when the
                    # sync-start message for that subscription arrives.  These are the online
                    # updates for a subscription we no longer care about.
                    if self._sync_sent != self._sync_started:
                        continue

                    # Process the message.
                    for k, v in msg.items():
                        handler = self.handlers.get(k, None)
                        if handler is None:
                            raise ValueError(f"unhandled msg: key {k} in {msg}")
                        result = handler(v)
                        if result:
                            assert isinstance(result, (wire.ServerMsg, wire.DeleteMsg, Sync))
                            self._pending.append(result)
                    if self._pending:
                        return self._pending.popleft()
                elif isinstance(event, events.Closing):
                    # server is done writing messages, but we're still allowed to
                    pass
                elif isinstance(event, events.Closed):
                    # no longer safe to write events
                    self._ready = False
                    pass
                elif isinstance(event, events.Disconnected):
                    self._ready = False
                else:
                    raise ValueError(f"unexpected event type {type(event).__name__}: {event}")
            self._backoff()
            # reconnect and try again
            self._ws_iter = None

    def __enter__(self) -> "Stream":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()

    def close(self) -> None:
        if self._ws_iter is None:
            return
        self._ws.close()
        # Drain the websocket of events.
        for _ in self._ws_iter:
            pass
        self._ws_iter = None

    def subscribe(
        self,
        sync_id: Any = None,
        *,
        projects: Optional[ProjectSpec] = None,
        models: Optional[ModelSpec] = None,
        model_versions: Optional[ModelVersionSpec] = None,
    ) -> "Stream":
        # Capture what the user asked for immediately, but we won't fill since or known values until
        # we send it.
        spec: Dict[str, Any] = {}
        if projects:
            spec["projects"] = projects._copy()
        if models:
            spec["models"] = models._copy()
        if model_versions:
            spec["modelversions"] = model_versions._copy()
        self._specs.append((sync_id, spec))
        # Adding a spec can trigger sending a subscription.
        self._advance_subscription()
        return self
