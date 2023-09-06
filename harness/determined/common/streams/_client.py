from typing import Dict, Generic, Iterable, TypeVar, Union, Optional
import collections
import urllib.request
import json

import lomond
from lomond import events

from determined.common import streams
from determined.common.api import request
from determined.common.streams import wire

"""
TODO:
  x write the type generation code
  x demonstrate working client
  - demonstrate automatic reconnects
  - add syncs
"""


# class CustomSSLWebsocketSession(lomond.session.WebsocketSession):  # type: ignore
#     """
#     A session class that allows for the TLS verification mode of a WebSocket connection to be
#     configured.
#     """
#
#     def __init__(
#         self, socket: lomond.WebSocket, cert_file: Union[str, bool, None], cert_name: Optional[str]
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

Record = Union[
    wire.TrialMsg,
]


class Deleted:
    def __init__(self, keys: str) -> None:
        self._keys = keys

    def ids(self) -> Iterable[int]:
        yield from streams.range_encoded_keys(self._keys)

    def __repr__(self) -> str:
        return f"{type(self).__name__}({self._keys})"


class TrialsDeleted(Deleted):
    pass


class TrialSubscriptionSpec:
    def __init__(self, trial_ids=None, experiment_ids=None):
        self.trial_ids = set(trial_ids) if trial_ids else set()
        self.experiment_ids = set(experiment_ids) if experiment_ids else set()

    def add(self, other):
        if other is None:
            return
        self.trial_ids = self.trial_ids.union(other.trial_ids)
        self.experiment_ids = self.experiment_ids.union(other.experiment_ids)

    def drop(self, other):
        if other is None:
            return
        self.trial_ids = self.trial_ids.difference(other.trial_ids)
        self.experiment_ids = self.experiment_ids.difference(other.experiment_ids)

    def to_json(self):
        return {k: sorted(v) for k, v in vars(self).items() if v}


class SubscriptionSpecSet:
    def __init__(self, trials=None):
        self.trials = trials or TrialSubscriptionSpec()

    def add(self, other):
        self.trials.add(other.trials)
        # self.experiments.add(other.experiments)

    def drop(self, other):
        self.trials.drop(other.trials)
        # self.experiments.drop(other.experiments)

    def to_json(self):
        def to_json_items(dct):
            for k, v in dct.items():
                yield k, v.to_json()

        return {k: v for k, v in to_json_items(vars(self)) if v}


class KeyCache:
    """
    KeyCache caches only primary keys.

    KeyCache is just enough caching to allow Stream to automatically reconnect.
    """

    def __init__(self, keys=None):
        self.keys = keys or set()
        self.maxseq = 0

    def upsert(self, id, seq):
        self.keys.add(id)
        self.maxseq = max(self.maxseq, seq)

    def delete_one(self, id):
        try:
            self.keys.remove(id)
        except KeyError:
            pass

    def delete_msg(self, deleted: str) -> list:
        for i in streams.range_encoded_keys(deleted):
            self.delete_one(i)

    def known(self) -> str:
        if not self.keys:
            return ""

        out = []
        keys = iter(sorted(self.keys))
        start = next(keys)
        end = start

        def emit(start, end, out):
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
    def __init__(self, session, spec: Optional[SubscriptionSpecSet] = None):
        self._session = session
        # Our stream-level in-memory cache: just enough to handle automatic reconnects.
        self._trials = KeyCache()
        self._experiments = KeyCache()
        self._spec = spec or SubscriptionSpecSet()
        # The websocket connection.  We'll connect (and reconnect) lazily.
        self._ws = None
        self._ws_iter = None
        self._closed = False
        # Parsed messages we've collected but haven't passed out yet.
        self._pending = collections.deque()

        # Spec changes which we have made but haven't sent yet.
        self._resubs = []

        self.handlers = {
            "trial": self.upsertion_handler(wire.TrialMsg, self._trials),
            "trials_deleted": self.deletion_handler(TrialsDeleted, self._trials),
        }

    def upsertion_handler(self, typ, cache):

        def handler(val):
            record = typ(**val)
            cache.upsert(record.id, record.seq)
            return record

        return handler

    def deletion_handler(self, typ, cache):

        def handler(val):
            cache.delete_msg(val)
            return typ(val)

        return handler

    def __iter__(self):
        # You can iterate on this Stream directly.
        return self

    def _connect(self):
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
        parsed_master = request.parse_master_address(self._session._master)
        proxies = (
            {} if urllib.request.proxy_bypass(parsed_master.hostname) else None
        )  # type: ignore
        http_url = request.make_url_new(self._session._master, "stream")
        url = request.maybe_upgrade_ws_scheme(http_url)

        self._ws = lomond.WebSocket(url, proxies=proxies)

        # is there a better way to get the session?
        token = self._session._auth.session.token
        self._ws.add_header(b"Authorization", f"Bearer {token}".encode("utf8"))

        self._ws_iter = self._ws.connect(
            ping_rate=0,
            # session_class=lambda socket: CustomSSLWebsocketSession(
            #     socket, cert_file, cert_name,
            # ),
        )

    def _send_resub(self, add, drop):
        msg = {}
        addmsg = add.to_json()
        if addmsg:
            msg["add"] = addmsg
        dropmsg = drop.to_json()
        if dropmsg:
            msg["drop"] = dropmsg
        if msg:
            self._ws.send_text(json.dumps(msg))
        self._spec.add(add)
        self._spec.drop(drop)

    def __next__(self):
        if self._closed:
            raise StopIteration
        if self._ws is None:
            self._connect()
        while True:
            if self._pending:
                return self._pending.popleft()
            for add, drop in self._resubs:
                self._send_resub(add, drop)
                self._resubs = []
            for event in self._ws_iter:
                if isinstance(
                    event,
                    (lomond.events.ConnectFail, lomond.events.Rejected, lomond.events.ProtocolError),
                ):
                    # XXX: support a retry policy with a sane default
                    raise ConnectionError(event)
                elif isinstance(event, lomond.events.Poll):
                    pass
                elif isinstance(event, lomond.events.Connecting):
                    pass
                elif isinstance(event, lomond.events.Connected):
                    pass
                elif isinstance(event, lomond.events.Ready):
                    # send our start_msg
                    # XXX: where does since come from??
                    startup_msg = {
                        "known": {k: v for k, v in {
                            "trials": self._trials.known(),
                            #"experiments": self._experiments.known(),
                        }.items() if v},
                        "subscribe": self._spec.to_json(),
                    }
                    print(f"sending startup: {startup_msg}")
                    self._ws.send_text(json.dumps(startup_msg))
                elif isinstance(event, (lomond.events.Text)):
                    # parse and process this message
                    msg = json.loads(event.text)
                    for k, v in msg.items():
                        handler = self.handlers.get(k, None)
                        if handler is None:
                            raise ValueError(f"unhandled msg: key {k} in {msg}")
                        self._pending.append(handler(v))
                        if self._pending:
                            return self._pending.popleft()
                elif isinstance(
                    event,
                    (lomond.events.Closing, lomond.events.Closed, lomond.events.Disconnected)
                ):
                    # XXX: no longer safe to write events
                    pass
                else:
                    raise ValueError(f"unexpected event type {type(event).__name__}: {event}")
            # XXX: should we be calling ws.close()??
            # XXX: implement a backoff?
            # XXX: just use lomond.persist??
            # reconnect and try again
            print("self._connect()!")
            self._connect()

    def __enter__(self):
        return self

    def __exit__(self):
        self.close()

    def close(self):
        if self._ws is None:
            return
        self._ws.close()
        # Drain the websocket of events.
        for event in self._ws_iter:
            pass
        self._ws = None
        self._ws_iter = None

    def resubscribe(self, add=None, drop=None):
        self._resubs.append((add, drop))

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
Focus Items for Streaming Updates:
  - generated Record types
  - Stream
  - automatic network reconnects
  - websocket code
  - synchronizations
      - gonna need enough long-term client-side plans or mocks or whatever to figure out what
        useful synchronization behavior even means
      - talk to webui, what synchronization do they need?  Do they need any?

Won't Do:
  - FileCache: this is just client-side sugar
  - SDK API: not a concern of backend or webui; only relevant to ml-sys.

Should we distinguish Fallout vs Deleted?
  - would make client life a lot easier, since one store could be used for many different filters.
  - well... if the get_keys() accepted an initial subscription set.  That is pretty easy.
  - How hard is it to distinguish offline Fallout vs Deleted on the server?  Do we want to?
      - we'd need a second round of processing intitial known values, where we check if the
        missing values actually exist and are visible to the user (aka if they are fallout)

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
