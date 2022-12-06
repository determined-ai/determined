import json as _json
import os
import sys
import webbrowser
from types import TracebackType
from typing import Any, Dict, Iterator, Optional, Sequence, Tuple, Union
from urllib import parse

import lomond
import requests
import urllib3
from termcolor import colored

import determined as det
import determined.common.requests
from determined.common import api
from determined.common.api import authentication, bindings, certs, errors


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


def make_interactive_task_url(
    task_id: str,
    service_address: str,
    description: str,
    resource_pool: str,
    task_type: str,
    currentSlotsExceeded: bool,
) -> str:
    wait_path = (
        "/notebooks/{}/events".format(task_id)
        if task_type == "notebook"
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


def add_token_to_headers(
    headers: Dict[str, str],
    auth: Optional[authentication.Authentication],
) -> Dict[str, str]:
    # Try to get user token first since it will include the user token that is used
    # for queries in some restful APIs.
    user_token = ""
    if auth is not None:
        user_token = auth.get_session_token()

    if user_token:
        return {**headers, "Authorization": "Bearer {}".format(user_token)}

    allocation_token = authentication.get_allocation_token()
    if allocation_token:
        return {**headers, "Grpc-Metadata-x-allocation-token": "Bearer {}".format(allocation_token)}

    return headers


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
    # If no explicit Authentication object was provided, use the cli's singleton Authentication.
    if auth is None:
        auth = authentication.cli_auth
    if cert is None:
        cert = certs.cli_cert

    if headers is None:
        h = {}  # type: Dict[str, str]
    else:
        h = headers

    if params is None:
        params = {}

    if authenticated:
        h = add_token_to_headers(h, auth)

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
        username = ""
        if auth is not None:
            username = auth.get_session_user()
        raise errors.ForbiddenException(username=username, message=_get_error_str(r))
    if r.status_code == 401:
        username = ""
        if auth is not None:
            username = auth.get_session_user()
        raise errors.UnauthenticatedException(username=username)
    elif r.status_code == 404:
        raise errors.NotFoundException(r)
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
    def __init__(self, socket: lomond.WebSocket) -> None:
        self.socket = socket

    def __enter__(self) -> "WebSocket":
        return self

    def __iter__(self) -> Iterator[Any]:
        for event in self.socket.connect(ping_rate=0):
            if isinstance(event, lomond.events.Connected):
                # Ignore the initial connection event.
                pass
            elif isinstance(event, (lomond.events.Closing, lomond.events.Disconnected)):
                # The socket was successfully closed so we just return.
                return
            elif isinstance(
                event,
                (lomond.events.ConnectFail, lomond.events.Rejected, lomond.events.ProtocolError),
            ):
                # Any unexpected failures raise the standard API exception.
                raise errors.BadRequestException(message="WebSocket failure: {}".format(event))
            elif isinstance(event, lomond.events.Text):
                # All web socket connections are expected to be in a JSON
                # format.
                yield _json.loads(event.text)

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
    token = authentication.must_cli_auth().get_session_token()
    websocket.add_header("Authorization".encode(), "Bearer {}".format(token).encode())
    return WebSocket(websocket)


def handle_warnings(warnings: Optional[Sequence[bindings.v1LaunchWarning]]) -> None:
    if warnings:
        for warning in warnings:
            print(colored(api.WARNING_MESSAGE_MAP[warning], "yellow"), file=sys.stderr)
