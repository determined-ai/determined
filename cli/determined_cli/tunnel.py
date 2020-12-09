"""
tunnel.py will tunnel a TCP connection to the service (typically a shell) with ID equal to
SERVICE_UUID over a WebSocket connection to a Determined master at MASTER_ADDR.
"""

import argparse
import io
import os
import socket
import ssl
import sys
import threading
from typing import Optional

import lomond

from determined_common.api import request


class CustomSSLWebsocketSession(lomond.session.WebsocketSession):  # type: ignore
    """
    A session class that allows for the TLS verification mode of a WebSocket connection to be
    configured.
    """

    def __init__(
        self, socket: lomond.WebSocket, cert_file: Optional[str], cert_name: Optional[str]
    ) -> None:
        super().__init__(socket)
        self.ctx = ssl.create_default_context()
        if cert_file == "False":
            self.ctx.verify_mode = ssl.CERT_NONE
            return

        if cert_file is not None:
            self.ctx.load_verify_locations(cafile=cert_file)
        self.cert_name = cert_name

    def _wrap_socket(self, sock: socket.SocketType, host: str) -> socket.SocketType:
        return self.ctx.wrap_socket(sock, server_hostname=self.cert_name or host)


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


def copy_from_websocket(
    f: io.RawIOBase,
    ws: lomond.WebSocket,
    ready_sem: threading.Semaphore,
    cert_file: Optional[str],
    cert_name: Optional[str],
) -> None:
    try:
        for event in ws.connect(
            ping_rate=0,
            session_class=lambda socket: CustomSSLWebsocketSession(socket, cert_file, cert_name),
        ):
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


def http_connect_tunnel(
    master: str, service: str, cert_file: Optional[str], cert_name: Optional[str]
) -> None:
    parsed_master = request.parse_master_address(master)
    assert parsed_master.hostname is not None, "Failed to parse master address: {}".format(master)
    url = request.make_url(master, "proxy/{}/".format(service))
    ws = lomond.WebSocket(request.maybe_upgrade_ws_scheme(url))

    # We can't send data to the WebSocket before the connection becomes ready, which takes a bit of
    # time; this semaphore lets the sending thread wait for that to happen.
    ready_sem = threading.Semaphore(0)

    # Directly using sys.stdin.buffer.read or sys.stdout.buffer.write would block due to
    # buffering; instead, we use unbuffered file objects based on the same file descriptors.
    unbuffered_stdin = os.fdopen(sys.stdin.fileno(), "rb", buffering=0)
    unbuffered_stdout = os.fdopen(sys.stdout.fileno(), "wb", buffering=0)

    c1 = threading.Thread(target=copy_to_websocket, args=(ws, unbuffered_stdin, ready_sem))
    c2 = threading.Thread(
        target=copy_from_websocket, args=(unbuffered_stdout, ws, ready_sem, cert_file, cert_name)
    )
    c1.start()
    c2.start()
    c1.join()
    c2.join()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Tunnel through a Determined master")
    parser.add_argument("master_addr")
    parser.add_argument("service_uuid")
    parser.add_argument("--cert-file")
    parser.add_argument("--cert-name")
    args = parser.parse_args()

    http_connect_tunnel(args.master_addr, args.service_uuid, args.cert_file, args.cert_name)
