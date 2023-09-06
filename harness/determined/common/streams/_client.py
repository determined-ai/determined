import abc
import collections
import json
import time
import urllib.request
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

from determined.common import api, streams
from determined.common.api import request
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


# class CustomSSLWebSocketSession(lomond.session.WebSocketSession):  # type: ignore
#     """
#     A session class that allows for the TLS verification mode of a WebSocket connection to be
#     configured.
#     """
#
#     def __init__(
#        self, socket: lomond.WebSocket, cert_file: Union[str, bool, None], cert_name: Optional[str]
#     ) -> None:
#         super().__init__(socket)
#         self.ctx = ssl.create_default_context()
#         if cert_file == "False" or cert_file is False:
#             self.ctx.verify_mode = ssl.CERT_NONE
#             return
#
#         if cert_file is not None:
#             assert isinstance(cert_file, str)
#             self.ctx.load_verify_locations(cafile=cert_file)
#         self.cert_name = cert_name
#
#     def _wrap_socket(self, sock: socket.SocketType, host: str) -> socket.SocketType:
#         return self.ctx.wrap_socket(sock, server_hostname=self.cert_name or host)


class LomondStreamWebSocket(StreamWebSocket):
    """
    The "real" StreamWebSocket, used outside of tests.
    """

    def __init__(self, session: api.Session) -> None:
        self.ws: lomond.WebSocket = None
        self.session = session

        # The "lomond.WebSocket()" function does not honor the "no_proxy" or
        # "NO_PROXY" environment variables. To work around that, we check if
        # the hostname is in the "no_proxy" or "NO_PROXY" environment variables
        # ourselves using the "proxy_bypass()" function, which checks the "no_proxy"
        # and "NO_PROXY" environment variables, and returns True if the host does
        # not require a proxy server. The "lomond.WebSocket()" function will disable
        # the proxy if the "proxies" parameter is an empty dictionary.  Otherwise,
        # if the "proxies" parameter is "None", it will honor the "HTTP_PROXY" and
        # "HTTPS_PROXY" environment variables. When the "proxies" parameter is not
        # specified, the default value is "None".
        parsed_master = request.parse_master_address(session._master)
        self.proxies: Optional[Dict] = (
            {} if urllib.request.proxy_bypass(parsed_master.hostname) else None  # type: ignore
        )
        http_url = request.make_url_new(session._master, "stream")
        self.url = request.maybe_upgrade_ws_scheme(http_url)

        assert self.session._auth
        self.token = self.session._auth.session.token

        # About 60 seconds of auto-retry.
        self._backoffs = [0, 1, 2, 4, 8, 10, 10, 10, 15]

    def connect(self) -> Iterable:
        # TODO: rewrite this after landing auth refactor
        if self.ws:
            self.ws.close()

        self.ws = lomond.WebSocket(self.url, proxies=self.proxies)
        self.ws.add_header(b"Authorization", f"Bearer {self.token}".encode("utf8"))

        it = self.ws.connect(
            ping_rate=0,
            # session_class=lambda ws: CustomSSLWebSocketSession(ws, cert_file, cert_name),
        )
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

    def _to_wire(self, since: int) -> Dict[str, Any]:
        return wire.ProjectSubscriptionSpec(
            workspace_ids=int_or_list(self.workspace_id),
            project_ids=int_or_list(self.project_id),
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
        }

        self._retries = 0

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

    def _retry(self) -> bool:
        backoff = self._ws.get_backoff(self._retries)
        if backoff is None:
            return False
        self._retries += 1
        time.sleep(backoff)
        return True

    def _send_spec(self, spec: Dict[str, Any]) -> None:
        self._num_syncs += 1
        sync_id = str(self._num_syncs)
        # build our startup message
        since = {
            "projects": self._projects.maxseq,
            # "experiments": self._experiments.maxseq,
        }
        subscribe = {k: v._to_wire(since[k]) for k, v in spec.items()}
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
                    # "experiments": self._experiments.known(),
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
            if not self._retry():
                # XXX: need to capture the failure or disconnect reason to raise here
                raise ConnectionError(event)
            # XXX: should we be calling ws.close()??
            # XXX: just use lomond.persist??
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
        # experiments: Optional[ExperimentSpec] = None,
    ) -> "Stream":
        # Capture what the user asked for immediately, but we won't fill since or known values until
        # we send it.
        spec = {}
        if projects:
            spec["projects"] = projects._copy()
        self._specs.append((sync_id, spec))
        # Adding a spec can trigger sending a subscription.
        self._advance_subscription()
        return self


"""
Failure Modes:
  - Network failure: Stream automatically recovers.  Cache and Application aren't even aware.
  - Just Boot'em: Treated as a network failure.
  - Application failure: MemoryCache is obviously toast.  FileCache can continue streaming from
    when it was last committed to disk.

Fault-tolerant read/write controller (mutating non-idempotent network resources):
  - Application uses a FileCache to read from the Stream.
  - FileCache emits some message
      - Client can crash, recover FileCache, and keep streaming.
  - Client saves an intention to create a task, including a disk store seq, and a request id.
      - Client can crash, realize intention is newer than FileCache, and drop the intention
  - Client commits FileCache.
      - Client can crash, see intention is current with FileCache, then query master for the result
        of the intention, and see it is still necessary
  - Client acts on intention.
      - Client can crash, see intention is current with FileCache, then query the master for the
        result of the intention, and see the intention is no longer necessary.
  - Client may delete intention from disk.

Read-only controller:
  - FileCache emits some message
  - Client commits FileCache to disk.  Done.
"""

"""
Do we need subscription synchronizations?
  - for the webui?
  - for the sdk?
  - What about a general sync mechanism, like a ping/pong sort of thing?
  - Server-side, what forms of synchronization can we easily provide?
      - flushing what is in-memory for the streamer is easy
      - flushing what is propagating through the publishing is hard
          - you wouldn't want to actually publish a sync message, since it would bounce off
            everybody, but maybe there could be a mechanism for a streamer to look at the publisher
            input state and wait for the output state to catch up to the input state?  Like start
            a little go routine to send a wakeup after the publisher finishes?
      - flushing changes in the database feels neigh impossible
          - perhaps you could write your streamer to the postgres channel and the publisher would
            notify you directly?
          - that could be an expensive mechanism!
          - but that's nice because if you see something, the transaction is complete, which means
            the NOTIFY queue has been populated, so you can have provably useful syncs
          - you'd have to write to every NOTIFY channel you had a subscription for, because they're
            all operating independently.
          - perhaps query database for latest seq of a table, and wait for publisher to catch up to
            that?  Yeah, that seems easier, and a lot less intricate.
"""

# XXX: bug: maxseq might not reflect correct value
#           - user subscribes to trial 1 and 2
#           - user sees updates on 1 and 2, updates maxseq
#           - user drops trial 1
#           - server updates 1 and 2
#           - user sees 2 updated, updates maxseq
#           - user adds 1, but maxseq precludes seeing update to 1
