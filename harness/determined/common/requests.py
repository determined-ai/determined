"""
A drop-in replacement for requests.request() which supports server name overriding.
"""
from typing import Any, Optional

import requests
import urllib3


class HTTPAdapter(requests.adapters.HTTPAdapter):
    """A new HTTPAdapter which honors the ServerName as a value for the verify arg."""

    def __init__(self, server_hostname: Optional[str], **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.server_hostname = server_hostname

    def cert_verify(self, conn: Any, url: Any, verify: Any, cert: Any) -> None:
        super().cert_verify(conn, url, verify, cert)  # type: ignore
        if self.server_hostname is not None:
            # Set the server_hostname value of the urllib3 connection.
            conn.assert_hostname = self.server_hostname


class Session(requests.sessions.Session):
    def __init__(
        self, server_hostname: Optional[str], max_retries: Optional[urllib3.util.retry.Retry]
    ) -> None:
        super().__init__()
        if max_retries is None:
            # Override the https adapter.
            self.mount("https://", HTTPAdapter(server_hostname))
        else:
            self.mount("https://", HTTPAdapter(server_hostname, max_retries=max_retries))
            self.mount("http://", requests.adapters.HTTPAdapter(max_retries=max_retries))


def request(method: str, url: str, **kwargs: Any) -> requests.Response:
    server_hostname = kwargs.pop("server_hostname", None)
    max_retries = kwargs.pop("max_retries", None)
    with Session(server_hostname, max_retries) as session:
        out = session.request(method=method, url=url, **kwargs)  # type: requests.Response
        return out
