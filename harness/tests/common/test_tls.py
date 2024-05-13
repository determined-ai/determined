import contextlib
import http.server
import os
import ssl
import threading
from typing import Dict, Iterator, Tuple

import pytest
import requests

from determined.common import api
from determined.common.api import certs

TRUSTED_DOMAIN = "https://google.com:443"
UNTRUSTED_DIR = os.path.join(os.path.dirname(__file__), "untrusted-root")
UNTRUSTED_CA = os.path.join(UNTRUSTED_DIR, "127.0.0.1-ca.crt")
UNTRUSTED_KEY = os.path.join(UNTRUSTED_DIR, "127.0.0.1-key.pem")
UNTRUSTED_CERT = os.path.join(UNTRUSTED_DIR, "127.0.0.1-cert.pem")
with open(UNTRUSTED_CERT) as f:
    UNTRUSTED_PEM = f.read()


@contextlib.contextmanager
def run_test_server(
    address: Tuple[str, int], cert: str = UNTRUSTED_CERT, key: str = UNTRUSTED_KEY
) -> Iterator[str]:
    server = http.server.HTTPServer(address, http.server.SimpleHTTPRequestHandler)

    server.socket = ssl.wrap_socket(
        server.socket,
        keyfile=key,
        certfile=cert,
        server_side=True,
    )

    thread = threading.Thread(target=server.serve_forever, args=[0.1])
    thread.start()
    try:
        host = address[0]
        port = address[1]
        yield f"https://{host}:{port}"
    finally:
        server.shutdown()
        thread.join()


@pytest.mark.parametrize(
    "cert_kwargs,raises_ssl_error",
    [
        ({"noverify": True}, False),
        ({"noverify": False}, True),
        ({"cert_pem": UNTRUSTED_PEM}, False),
        ({}, True),
    ],
)
def test_custom_tls_certs(cert_kwargs: Dict, raises_ssl_error: bool) -> None:
    with run_test_server(
        ("127.0.0.1", 12345), cert=UNTRUSTED_CERT, key=UNTRUSTED_KEY
    ) as untrusted_url:
        assert isinstance(cert_kwargs, dict)
        cert = certs.Cert(**cert_kwargs)

        # Trusted domains should always work.
        api.UnauthSession(TRUSTED_DOMAIN, cert=cert).get(path="")

        with contextlib.ExitStack() as ctx:
            if raises_ssl_error:
                ctx.enter_context(pytest.raises(requests.exceptions.SSLError))
            sess = api.UnauthSession(untrusted_url, cert=cert, max_retries=1)
            sess.get(path="")
