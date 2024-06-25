import contextlib
import dataclasses
import io
import os
import socket
import socketserver
import sys
import threading
import time
from typing import Dict, Iterator, List, Optional

import lomond

from determined.common import api, detlomond
from determined.common.api import bindings, certs


@dataclasses.dataclass
class ListenerConfig:
    service_id: str
    local_port: int
    local_addr: str = "0.0.0.0"


def copy_to_websocket(
    ws: lomond.WebSocket, f: io.RawIOBase, ready_sem: threading.Semaphore
) -> None:
    ready_sem.acquire()

    try:
        while True:
            chunk = f.read(4096)
            if not chunk:
                break
            ws.send_binary(chunk)
    finally:
        f.close()
        ws.close()


def copy_to_websocket2(
    ws: lomond.WebSocket, f: socket.socket, ready_sem: threading.Semaphore
) -> None:
    ready_sem.acquire()

    try:
        while True:
            chunk = f.recv(4096)
            if not chunk:
                break
            ws.send_binary(chunk)
    finally:
        f.close()
        ws.close()


def copy_from_websocket(
    f: io.RawIOBase,
    ws: lomond.WebSocket,
    ready_sem: threading.Semaphore,
    cert: Optional[certs.Cert],
) -> None:
    try:
        for event in ws.connect(ping_rate=0):
            if isinstance(event, lomond.events.Binary):
                f.write(event.data)
            elif isinstance(event, lomond.events.Ready):
                ready_sem.release()
            elif isinstance(
                event,
                (lomond.events.ConnectFail, lomond.events.Rejected, lomond.events.ProtocolError),
            ):
                raise Exception("Connection failed: {}".format(event))
            elif isinstance(event, (lomond.events.Closing, lomond.events.Disconnected)):
                break
    finally:
        f.close()


def copy_from_websocket2(
    f: socket.socket,
    ws: lomond.WebSocket,
    ready_sem: threading.Semaphore,
    cert: Optional[certs.Cert],
) -> None:
    try:
        for event in ws.connect(ping_rate=0):
            if isinstance(event, lomond.events.Binary):
                f.send(event.data)
            elif isinstance(event, lomond.events.Ready):
                ready_sem.release()
            elif isinstance(
                event,
                (lomond.events.ConnectFail, lomond.events.Rejected, lomond.events.ProtocolError),
            ):
                if isinstance(event, lomond.events.Rejected):
                    print(event.response)
                raise Exception(f"Connection {ws.url} failed: {event}")
            elif isinstance(event, (lomond.events.Closing, lomond.events.Disconnected)):
                break
    finally:
        f.close()


def http_connect_tunnel(sess: api.BaseSession, service: str) -> None:
    ws = detlomond.WebSocket(sess, f"proxy/{service}/")

    # We can't send data to the WebSocket before the connection becomes ready, which takes a bit of
    # time; this semaphore lets the sending thread wait for that to happen.
    ready_sem = threading.Semaphore(0)

    # Directly using sys.stdin.buffer.read or sys.stdout.buffer.write would block due to
    # buffering; instead, we use unbuffered file objects based on the same file descriptors.
    unbuffered_stdin = os.fdopen(sys.stdin.fileno(), "rb", buffering=0)
    unbuffered_stdout = os.fdopen(sys.stdout.fileno(), "wb", buffering=0)

    c1 = threading.Thread(target=copy_to_websocket, args=(ws, unbuffered_stdin, ready_sem))
    c2 = threading.Thread(
        target=copy_from_websocket, args=(unbuffered_stdout, ws, ready_sem, sess.cert)
    )
    c1.start()
    c2.start()
    c1.join()
    c2.join()


class ReuseAddrServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True


def _http_tunnel_listener(
    sess: api.BaseSession,
    tunnel: ListenerConfig,
) -> socketserver.ThreadingTCPServer:
    class TunnelHandler(socketserver.BaseRequestHandler):
        def handle(self) -> None:
            ws = detlomond.WebSocket(sess, f"proxy/{tunnel.service_id}/")

            # We can't send data to the WebSocket before the connection becomes ready,
            # which takes a bit of time; this semaphore lets the sending thread
            # wait for that to happen.
            ready_sem = threading.Semaphore(0)

            c1 = threading.Thread(target=copy_to_websocket2, args=(ws, self.request, ready_sem))
            c2 = threading.Thread(
                target=copy_from_websocket2,
                args=(self.request, ws, ready_sem, sess.cert),
            )
            c1.start()
            c2.start()
            c1.join()
            c2.join()

    socket_class = ReuseAddrServer
    if sys.platform == "win32":
        # On Windows, SO_REUSEADDR is a security issue:
        # https://learn.microsoft.com/en-us/windows/win32/winsock/using-so-reuseaddr-and-so-exclusiveaddruse#application-strategies
        socket_class = socketserver.ThreadingTCPServer

    return socket_class((tunnel.local_addr, tunnel.local_port), TunnelHandler)


@contextlib.contextmanager
def http_tunnel_listener(
    sess: api.BaseSession,
    tunnels: List[ListenerConfig],
) -> Iterator[None]:
    servers = [_http_tunnel_listener(sess, tunnel) for tunnel in tunnels]

    threads = [threading.Thread(target=lambda s: s.serve_forever(), args=(s,)) for s in servers]

    try:
        for t in threads:
            t.start()
        # TODO(ilia): should we inform the user when we are up?
        yield
    finally:
        for s in servers:
            s.shutdown()
            s.server_close()
        for t in threads:
            t.join()


@contextlib.contextmanager
def _tunnel_task(sess: api.Session, task_id: str, port_map: "Dict[int, int]") -> Iterator[None]:
    # Args:
    #   port_map: dict of local port => task port.
    #   task_id: tunneled task_id.

    listeners = [
        ListenerConfig(service_id=f"{task_id}:{task_port}", local_port=local_port)
        for local_port, task_port in port_map.items()
    ]

    with http_tunnel_listener(sess, listeners):
        yield


@contextlib.contextmanager
def _tunnel_trial(sess: api.Session, trial_id: int, port_map: "Dict[int, int]") -> Iterator[None]:
    # TODO(DET-9000): perhaps the tunnel should be able to probe master for service status,
    # instead of us explicitly polling for task/trial status.
    while True:
        resp = bindings.get_GetTrial(sess, trialId=trial_id)
        trial = resp.trial

        terminal_states = [
            bindings.trialv1State.COMPLETED,
            bindings.trialv1State.CANCELED,
            bindings.trialv1State.ERROR,
        ]
        if trial.state in terminal_states:
            raise ValueError("Can't tunnel a trial in terminal state")

        task_id = trial.taskId
        if task_id is not None:
            break
        else:
            time.sleep(0.1)

    with _tunnel_task(sess, task_id, port_map):
        yield


@contextlib.contextmanager
def tunnel_experiment(
    sess: api.Session, experiment_id: int, port_map: "Dict[int, int]"
) -> Iterator[None]:
    while True:
        trials = bindings.get_GetExperimentTrials(sess, experimentId=experiment_id).trials
        if len(trials) > 0:
            break
        else:
            time.sleep(0.1)

    first_trial_id = sorted(t.id for t in trials)[0]

    with _tunnel_trial(sess, first_trial_id, port_map):
        yield


def parse_port_map_flag(publish_arg: "list[str]") -> "Dict[int, int]":
    result = {}  # type: Dict[int, int]

    for e in publish_arg:
        try:
            if ":" in e:
                lp, tp = e.split(":")
                local_port, task_port = int(lp), int(tp)
                result[local_port] = task_port
            else:
                port = int(e)
                result[port] = port
        except ValueError as e:
            raise ValueError(f"failed to parse --publish argument: {e}") from e

    return result
