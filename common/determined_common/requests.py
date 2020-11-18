"""
A drop-in replacement for requests.request() which supports server name overriding.
"""
from typing import Any, Optional

import requests


class HTTPAdapter(requests.adapters.HTTPAdapter):
    """A new HTTPAdapter which honors the ServerName as a value for the verify arg."""

    def __init__(self, server_hostname: Optional[str]) -> None:
        super().__init__()
        self.server_hostname = server_hostname

    def cert_verify(self, conn: Any, url: Any, verify: Any, cert: Any) -> None:
        super().cert_verify(conn, url, verify, cert)  # type: ignore
        if self.server_hostname is not None:
            # Set the server_hostname value of the urllib3 connection.
            conn.assert_hostname = self.server_hostname


class Session(requests.sessions.Session):
    def __init__(self, server_hostname: Optional[str]) -> None:
        super().__init__()
        # Override the https adapter.
        self.mount("https://", HTTPAdapter(server_hostname))


def request(method: str, url: str, **kwargs: Any) -> requests.Response:
    server_hostname = kwargs.pop("server_hostname", None)
    with Session(server_hostname) as session:
        return session.request(method=method, url=url, **kwargs)
