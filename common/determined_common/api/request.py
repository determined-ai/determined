import webbrowser
from types import TracebackType
from typing import Any, Dict, Iterator, Optional
from urllib import parse

import lomond
import requests
import simplejson

from determined_common.api import authentication, errors

# The path to a file containing an SSL certificate to trust specifically for the master, if any.
_master_cert_bundle = None


def set_master_cert_bundle(path: str) -> None:
    global _master_cert_bundle
    _master_cert_bundle = path


def get_master_cert_bundle() -> Optional[str]:
    return _master_cert_bundle


def parse_master_address(master_address: str) -> parse.ParseResult:
    if master_address.startswith("https://"):
        default_port = 443
    elif master_address.startswith("http://"):
        default_port = 80
    else:
        default_port = 8080
        master_address = "http://{}".format(master_address)
    parsed = parse.urlparse(master_address)
    if not parsed.port:
        parsed = parsed._replace(netloc="{}:{}".format(parsed.netloc, default_port))
    return parsed


def make_url(master_address: str, suffix: str) -> str:
    parsed = parse_master_address(master_address)
    return parse.urljoin(parsed.geturl(), suffix)


def maybe_upgrade_ws_scheme(master_address: str) -> str:
    parsed = parse.urlparse(master_address)
    if parsed.scheme == "https":
        return parsed._replace(scheme="wss").geturl()
    elif parsed.scheme == "http":
        return parsed._replace(scheme="ws").geturl()
    else:
        return master_address


def add_token_to_headers(headers: Dict[str, str]) -> Dict[str, str]:
    token = authentication.Authentication.instance().get_session_token()

    return {**headers, "Authorization": "Bearer {}".format(token)}


def do_request(
    method: str,
    host: str,
    path: str,
    params: Optional[Dict[str, Any]] = None,
    body: Optional[Dict[str, Any]] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
) -> requests.Response:
    if headers is None:
        h = {}  # type: Dict[str, str]
    else:
        h = headers

    if params is None:
        params = {}

    if authenticated:
        h = add_token_to_headers(h)

    try:
        r = requests.request(
            method,
            make_url(host, path),
            params=params,
            json=body,
            headers=h,
            verify=_master_cert_bundle,
        )
    except requests.exceptions.SSLError:
        raise
    except requests.exceptions.ConnectionError as e:
        raise errors.MasterNotFoundException(str(e))
    except requests.exceptions.RequestException as e:
        raise errors.BadRequestException(str(e))

    if r.status_code == 403:
        username = authentication.Authentication.instance().get_session_user()
        raise errors.UnauthenticatedException(username=username)

    if r.status_code >= 300:
        raise errors.APIException(r)

    return r


def get(
    host: str,
    path: str,
    params: Optional[Dict[str, Any]] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
) -> requests.Response:
    """
    Send a GET request to the remote API.
    """
    return do_request(
        "GET", host, path, params=params, headers=headers, authenticated=authenticated
    )


def delete(
    host: str,
    path: str,
    params: Optional[Dict[str, Any]] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
) -> requests.Response:
    """
    Send a DELETE request to the remote API.
    """
    return do_request(
        "DELETE", host, path, params=params, headers=headers, authenticated=authenticated
    )


def post(
    host: str,
    path: str,
    body: Optional[Dict[str, Any]] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
) -> requests.Response:
    """
    Send a POST request to the remote API.
    """
    return do_request("POST", host, path, body=body, headers=headers, authenticated=authenticated)


def patch(
    host: str,
    path: str,
    body: Dict[str, Any],
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
) -> requests.Response:
    """
    Send a PATCH request to the remote API.
    """
    return do_request("PATCH", host, path, body=body, headers=headers, authenticated=authenticated)


def put(
    host: str,
    path: str,
    body: Optional[Dict[str, Any]] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
) -> requests.Response:
    """
    Send a PUT request to the remote API.
    """
    return do_request("PUT", host, path, body=body, headers=headers, authenticated=authenticated)


def open(host: str, path: str) -> str:
    url = make_url(host, path)
    webbrowser.open(url)
    return url


class WebSocket:
    def __init__(self, socket: lomond.WebSocket) -> None:
        self.socket = socket

    def __enter__(self) -> "WebSocket":
        return self

    def __iter__(self) -> Iterator[Any]:
        for event in self.socket.connect(ping_rate=0):
            if isinstance(event, lomond.events.Connected):
                # Ignore the initial connection event.
                pass
            elif isinstance(event, lomond.events.Closing) or isinstance(
                event, lomond.events.Disconnected
            ):
                # The socket was successfully closed so we just return.
                return
            elif (
                isinstance(event, lomond.events.ConnectFail)
                or isinstance(event, lomond.events.Rejected)
                or isinstance(event, lomond.events.ProtocolError)
            ):
                # Any unexpected failures raise the standard API exception.
                raise errors.BadRequestException(message="WebSocket failure: {}".format(event))
            elif isinstance(event, lomond.events.Text):
                # All web socket connections are expected to be in a JSON
                # format.
                yield simplejson.loads(event.text)

    def __exit__(
        self,
        exc_type: Optional[type],
        exc_val: Optional[BaseException],
        exc_tb: Optional[TracebackType],
    ) -> None:
        if not self.socket.is_closed:
            self.socket.close()


def ws(host: str, path: str) -> WebSocket:
    """
    Connect to a web socket at the remote API.
    """
    websocket = lomond.WebSocket(maybe_upgrade_ws_scheme(make_url(host, path)))
    token = authentication.Authentication.instance().get_session_token()
    websocket.add_header("Authorization".encode(), "Bearer {}".format(token).encode())
    return WebSocket(websocket)
