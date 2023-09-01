import json as _json
import os
import types
import webbrowser
from typing import Any, Dict, Iterator, Optional, Tuple, Union
from urllib import parse

import requests
import urllib3

import determined as det
import determined.common.requests
from determined.common.api import authentication, certs, errors


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
    """@deprecated use make_url_new instead"""
    parsed = parse_master_address(master_address)
    return parse.urljoin(parsed.geturl(), suffix)


def make_url_new(master_address: str, suffix: str) -> str:
    parsed_suffix = parse.urlparse(suffix)
    if parsed_suffix.scheme and parsed_suffix.netloc:
        return make_url(master_address, suffix)
    parsed = parse_master_address(master_address)
    master_url = parsed.geturl().rstrip("/")
    suffix = suffix.lstrip("/")
    separator = "/" if suffix or master_address.endswith("/") else ""
    return "{}{}{}".format(master_url, separator, suffix)


def maybe_upgrade_ws_scheme(master_address: str) -> str:
    parsed = parse.urlparse(master_address)
    if parsed.scheme == "https":
        return parsed._replace(scheme="wss").geturl()
    elif parsed.scheme == "http":
        return parsed._replace(scheme="ws").geturl()
    else:
        return master_address


def make_interactive_task_url(
    task_id: str,
    service_address: str,
    description: str,
    resource_pool: str,
    task_type: str,
    currentSlotsExceeded: bool,
) -> str:
    wait_path = (
        "/jupyter-lab/{}/events".format(task_id)
        if task_type == "jupyter-lab"
        else "/tensorboard/{}/events?tail=1".format(task_id)
    )
    wait_path_url = service_address + wait_path
    public_url = os.environ.get("PUBLIC_URL", "/det")
    wait_page_url = "{}/wait/{}/{}?eventUrl={}&serviceAddr={}".format(
        public_url, task_type, task_id, wait_path_url, service_address
    )
    task_web_url = "{}/interactive/{}/{}/{}/{}/{}?{}".format(
        public_url,
        task_id,
        task_type,
        parse.quote(description),
        resource_pool,
        parse.quote_plus(wait_page_url),
        f"currentSlotsExceeded={str(currentSlotsExceeded).lower()}",
    )
    return task_web_url


def do_request(
    method: str,
    host: str,
    path: str,
    params: Optional[Dict[str, Any]] = None,
    json: Any = None,
    data: Optional[str] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
    auth: Optional[authentication.Authentication] = None,
    cert: Optional[certs.Cert] = None,
    stream: bool = False,
    timeout: Optional[Union[Tuple, float]] = None,
    max_retries: Optional[urllib3.util.retry.Retry] = None,
) -> requests.Response:
    if headers is None:
        h: Dict[str, str] = {}
    else:
        h = headers

    if cert is None:
        cert = certs.cli_cert

    # set the token and username based on this order:
    # - argument `auth`
    # - header `Authorization`
    # - existing cli_auth
    # - allocation_token

    username = ""
    if auth is not None:
        if authenticated:
            h["Authorization"] = "Bearer {}".format(auth.get_session_token())
        username = auth.get_session_user()
    elif h.get("Authorization") is not None:
        pass
    elif authentication.cli_auth is not None:
        if authenticated:
            h["Authorization"] = "Bearer {}".format(authentication.cli_auth.get_session_token())
        username = authentication.cli_auth.get_session_user()
    elif authenticated and h.get("Grpc-Metadata-x-allocation-token") is None:
        allocation_token = authentication.get_allocation_token()
        if allocation_token:
            h["Grpc-Metadata-x-allocation-token"] = "Bearer {}".format(allocation_token)

    if params is None:
        params = {}

    # Allow the json json to come pre-encoded, if we need custom encoding.
    if json is not None and data is not None:
        raise ValueError("json and data must not be provided together")

    if json:
        data = det.util.json_encode(json)

    try:
        r = determined.common.requests.request(
            method,
            make_url(host, path),
            params=params,
            data=data,
            headers=h,
            verify=cert.bundle if cert else None,
            stream=stream,
            timeout=timeout,
            server_hostname=cert.name if cert else None,
            max_retries=max_retries,
        )
    except requests.exceptions.SSLError:
        raise
    except requests.exceptions.ConnectionError as e:
        raise errors.MasterNotFoundException(str(e))
    except requests.exceptions.RequestException as e:
        raise errors.BadRequestException(str(e))

    def _get_error_str(r: requests.models.Response) -> str:
        try:
            json_resp = _json.loads(r.text)
            mes = json_resp.get("message")
            if mes is not None:
                return str(mes)
            # Try getting GRPC error description if message does not exist.
            return str(json_resp.get("error").get("error"))
        except Exception:
            return ""

    if r.status_code == 403:
        raise errors.ForbiddenException(username=username, message=_get_error_str(r))
    if r.status_code == 401:
        raise errors.UnauthenticatedException(username=username)
    elif r.status_code == 404:
        raise errors.NotFoundException(_get_error_str(r))
    elif r.status_code >= 300:
        raise errors.APIException(r)

    return r


def get(
    host: str,
    path: str,
    params: Optional[Dict[str, Any]] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
    auth: Optional[authentication.Authentication] = None,
    cert: Optional[certs.Cert] = None,
    stream: bool = False,
    timeout: Optional[Union[Tuple, float]] = None,
) -> requests.Response:
    """
    Send a GET request to the remote API.
    """
    return do_request(
        "GET",
        host,
        path,
        params=params,
        headers=headers,
        authenticated=authenticated,
        auth=auth,
        cert=cert,
        stream=stream,
    )


def delete(
    host: str,
    path: str,
    params: Optional[Dict[str, Any]] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
    auth: Optional[authentication.Authentication] = None,
    cert: Optional[certs.Cert] = None,
    timeout: Optional[Union[Tuple, float]] = None,
) -> requests.Response:
    """
    Send a DELETE request to the remote API.
    """
    return do_request(
        "DELETE",
        host,
        path,
        params=params,
        headers=headers,
        authenticated=authenticated,
        auth=auth,
        cert=cert,
        timeout=timeout,
    )


def post(
    host: str,
    path: str,
    json: Any = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
    auth: Optional[authentication.Authentication] = None,
    cert: Optional[certs.Cert] = None,
    timeout: Optional[Union[Tuple, float]] = None,
) -> requests.Response:
    """
    Send a POST request to the remote API.
    """
    return do_request(
        "POST",
        host,
        path,
        json=json,
        headers=headers,
        authenticated=authenticated,
        auth=auth,
        cert=cert,
        timeout=timeout,
    )


def patch(
    host: str,
    path: str,
    json: Dict[str, Any],
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
    auth: Optional[authentication.Authentication] = None,
    cert: Optional[certs.Cert] = None,
    timeout: Optional[Union[Tuple, float]] = None,
) -> requests.Response:
    """
    Send a PATCH request to the remote API.
    """
    return do_request(
        "PATCH",
        host,
        path,
        json=json,
        headers=headers,
        authenticated=authenticated,
        auth=auth,
        cert=cert,
        timeout=timeout,
    )


def put(
    host: str,
    path: str,
    json: Optional[Dict[str, Any]] = None,
    headers: Optional[Dict[str, str]] = None,
    authenticated: bool = True,
    auth: Optional[authentication.Authentication] = None,
    cert: Optional[certs.Cert] = None,
    timeout: Optional[Union[Tuple, float]] = None,
) -> requests.Response:
    """
    Send a PUT request to the remote API.
    """
    return do_request(
        "PUT",
        host,
        path,
        json=json,
        headers=headers,
        authenticated=authenticated,
        auth=auth,
        cert=cert,
        timeout=timeout,
    )


def browser_open(host: str, path: str) -> str:
    url = make_url(host, path)
    webbrowser.open(url)
    return url


class WebSocket:
    def __init__(self, socket: Any) -> None:
        import lomond

        self.socket = socket  # type: lomond.WebSocket

    def __enter__(self) -> "WebSocket":
        return self

    def __iter__(self) -> Iterator[Any]:
        from lomond import events

        for event in self.socket.connect(ping_rate=0):
            if isinstance(event, events.Connected):
                # Ignore the initial connection event.
                pass
            elif isinstance(event, (events.Closing, events.Disconnected)):
                # The socket was successfully closed so we just return.
                return
            elif isinstance(
                event,
                (events.ConnectFail, events.Rejected, events.ProtocolError),
            ):
                # Any unexpected failures raise the standard API exception.
                raise errors.BadRequestException(message="WebSocket failure: {}".format(event))
            elif isinstance(event, events.Text):
                # All web socket connections are expected to be in a JSON
                # format.
                yield _json.loads(event.text)

    def __exit__(
        self,
        exc_type: Optional[type],
        exc_val: Optional[BaseException],
        exc_tb: Optional[types.TracebackType],
    ) -> None:
        if not self.socket.is_closed:
            self.socket.close()


def ws(host: str, path: str) -> WebSocket:
    """
    Connect to a web socket at the remote API.
    """
    import lomond

    websocket = lomond.WebSocket(maybe_upgrade_ws_scheme(make_url(host, path)))
    token = authentication.must_cli_auth().get_session_token()
    websocket.add_header("Authorization".encode(), "Bearer {}".format(token).encode())
    return WebSocket(websocket)
