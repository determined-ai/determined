"""
detlomond contains helpers for using the lomond websocket library with Determined.
"""

import socket
import ssl
from typing import Any, Optional
from urllib import parse, request

import lomond

from determined.common import api
from determined.common.api import certs


class CustomSSLWebsocketSession(lomond.session.WebsocketSession):  # type: ignore
    """
    A session class that allows for the TLS verification mode of a WebSocket connection to be
    configured.
    """

    def __init__(self, socket: lomond.WebSocket, cert: Optional[certs.Cert]) -> None:
        super().__init__(socket)
        self.ctx = ssl.create_default_context()

        self.cert_name = cert.name if cert else None

        bundle = cert.bundle if cert else None
        if bundle is False:
            self.ctx.check_hostname = False
            self.ctx.verify_mode = ssl.CERT_NONE
            return

        if bundle is not None:
            assert isinstance(bundle, str)
            self.ctx.load_verify_locations(cafile=bundle)

    def _wrap_socket(self, sock: socket.SocketType, host: str) -> socket.SocketType:
        return self.ctx.wrap_socket(sock, server_hostname=self.cert_name or host)


class WebSocket(lomond.WebSocket):  # type: ignore
    """
    WebSocket extends lomond.WebSocket with Determined-specific features:

      - support for NO_PROXY
      - our custom TLS verification
      - automatic authentication
    """

    def __init__(self, sess: api.BaseSession, path: str, **kwargs: Any):
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
        parsed = parse.urlparse(sess.master)
        proxies = {} if request.proxy_bypass(parsed.hostname) else None  # type: ignore

        # Prepare a session_class for the eventual .connect() method.
        self._default_session_class = lambda socket: CustomSSLWebsocketSession(socket, sess.cert)

        # Replace http with ws for a ws:// or wss:// url
        assert sess.master.startswith("http"), f"unable to convert non-http url ({sess.master})"
        baseurl = sess.master[4:]
        super().__init__(f"ws{baseurl}/{path}", proxies=proxies, **kwargs)

        # Possibly include authorization headers.
        if isinstance(sess, api.Session):
            self.add_header(b"Authorization", f"Bearer {sess.token}".encode())

    def connect(
        self,
        session_class: Optional[lomond.session.WebsocketSession] = None,
        *args: Any,
        **kwargs: Any,
    ) -> Any:
        session_class = session_class or self._default_session_class
        return super().connect(session_class, *args, **kwargs)
