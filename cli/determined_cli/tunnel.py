"""
tunnel.py will tunnel a TCP connection through Determined master at MASTER_ADDR to SERVICE_UUID.

This is used to tunnel ssh connections through the master, where the hostname in the SERVICE_UUID
should be the shell ID of the shell in question.
"""

import http.client
import os
import socket
import ssl
import sys
import threading
from typing import Optional

from determined_common.api import request


class Copier(threading.Thread):
    """
    A thread to copy from one file descriptor to another.  Only copies in one direction; use two
    threads to deal with bidirectional file descriptors.  The choice to use a pair of threads as
    opposed to select.select or select.poll ensures that this code will run nicely on Windows.
    """

    def __init__(self, src: int, dst: int):
        super().__init__()
        self.src = src
        self.dst = dst

    def run(self) -> None:
        try:
            while True:
                try:
                    buf = os.read(self.src, 4096)
                except OSError:
                    break
                if not buf:
                    break
                try:
                    os.write(self.dst, buf)
                except OSError:
                    break
        finally:
            # We're ok with double-closing some sockets.
            try:
                os.close(self.src)
            except OSError:
                pass

            try:
                os.close(self.dst)
            except OSError:
                pass


def http_connect_tunnel(master: str, service: str, master_cert: Optional[str]) -> None:
    parsed_master = request.parse_master_address(master)
    assert parsed_master.hostname is not None, "Failed to parse master address: {}".format(master)

    if parsed_master.scheme == "https":
        context = ssl.create_default_context(cafile=master_cert)
        client = http.client.HTTPSConnection(
            parsed_master.hostname, parsed_master.port, context=context
        )  # type: http.client.HTTPConnection
    else:
        client = http.client.HTTPConnection(parsed_master.hostname, parsed_master.port)

    client.set_tunnel(service)

    try:
        client.connect()
    except socket.gaierror:
        print("failed to look up host:", master, file=sys.stderr)
        raise

    with client.sock as sock:
        c1 = Copier(sock.fileno(), sys.stdout.fileno())
        c2 = Copier(sys.stdin.fileno(), sock.fileno())
        c1.start()
        c2.start()
        c1.join()
        c2.join()


if __name__ == "__main__":
    if len(sys.argv) not in (3, 4):
        print(
            "usage: {} MASTER_ADDR SERVICE_UUID [MASTER_CERT]".format(sys.argv[0]), file=sys.stderr
        )
        sys.exit(1)

    master = sys.argv[1]
    service = sys.argv[2]
    master_cert = None if len(sys.argv) < 4 else sys.argv[3]

    http_connect_tunnel(master, service, master_cert)
