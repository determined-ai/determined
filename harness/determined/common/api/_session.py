import abc
import copy
import json as _json
from types import TracebackType  # noqa:I2041
from typing import Any, Dict, Optional, TypeVar, Union

import requests
import urllib3
from requests import adapters

import determined as det
from determined.common import api
from determined.common.api import certs, errors

GeneralizedRetry = Union[urllib3.util.retry.Retry, int]
T = TypeVar("T", bound="BaseSession")

# Default retry logic
DEFAULT_MAX_RETRIES = urllib3.util.retry.Retry(
    total=5,
    backoff_factor=0.5,  # {backoff factor} * (2 ** ({number of total retries} - 1))
    status_forcelist=[502, 503, 504],  # Bad Gateway, Service Unavailable, Gateway Timeout
)


def _make_requests_session(
    server_hostname: Optional[str] = None,
    verify: Optional[Union[str, bool]] = None,
    max_retries: Optional[GeneralizedRetry] = None,
    headers: Optional[Dict[str, Any]] = None,
) -> requests.Session:
    if verify is None:
        verify = True
    requests_session = requests.Session()
    requests_session.mount(
        "https://", _HTTPSAdapter(server_hostname=server_hostname, max_retries=max_retries)
    )
    requests_session.mount("http://", adapters.HTTPAdapter(max_retries=max_retries))
    requests_session.verify = verify
    if headers:
        requests_session.headers.update(headers)

    return requests_session


def _get_error_str(r: requests.models.Response) -> str:
    try:
        json_resp = _json.loads(r.text)
        mes = json_resp.get("message")
        if mes is not None:
            return str(mes)

        # Try getting GRPC error description if message does not exist.
        # GRPC errors can have an optional {"error": {"message": "..."}} field.
        # Look here first then fallback to {"error": {"error": "..."}} if there is no message.
        error = json_resp.get("error")
        if "message" in error:
            return str(error.get("message"))
        return str(error.get("error"))
    except Exception:
        return ""


class BaseSession(metaclass=abc.ABCMeta):
    """
    BaseSession is a requests-like interface that hides master url, master cert, and authz info.

    There are very few cases where BaseSession is the right type; you probably want a Session.  In
    a few cases, you might be ok with an UnauthSession.  BaseSession is really only to express that
    you don't know what kind of session you need.  For example, the generated bindings take a
    BaseSession because the protos aren't annotated with which endpoints are authenticated.

    `BaseSession` and subclasses can be used directly, or as a context manager. When used as a
    context manager, all requests within the context will share a persistent underlying HTTP
    connection. When used directly, each request will create a new HTTP connection.

    TODO (MD-392): Migrate all session usage to persistent, remove context manager usage pattern.

    Example of direct usage:

    .. code:: python

       session = api.Session(...)
       # Each request opens and closes a new HTTP connection.
       session.get(...)
       session.get(...)

    Example of use as a context manager:

    .. code:: python
       with api.Session(...) as session:
           # Each request within the context re-uses the same HTTP connection.
           session.get(...)
           session.get(...)

    """

    master: str
    cert: Optional[certs.Cert]
    _max_retries: Optional[GeneralizedRetry]
    # Persistent HTTP session
    _http_session: Optional[requests.Session]

    def __enter__(self: T) -> T:
        self._persist_http_session()
        return self

    def __exit__(self, exc_type: type, exc_value: Exception, traceback: TracebackType) -> None:
        self.close()

    def _persist_http_session(self) -> None:
        # starts a new persistent HTTP session that will be used
        # for all requests until self.close() or the context exits.
        if self._http_session:
            self._http_session.close()
        self._http_session = self._make_http_session()

    def close(self) -> None:
        if self._http_session:
            self._http_session.close()
            self._http_session = None

    @abc.abstractmethod
    def _make_http_session(self) -> requests.Session:
        pass

    def _do_request(
        self,
        method: str,
        path: str,
        params: Optional[Dict[str, Any]],
        json: Any,
        data: Optional[str],
        headers: Optional[Dict[str, Any]],
        timeout: Optional[int],
        stream: bool,
    ) -> requests.Response:
        # Allow the json to come pre-encoded, if we need custom encoding.
        if json is not None and data is not None:
            raise ValueError("json and data must not be provided together")

        if json:
            data = det.util.json_encode(json)

        relpath = path.lstrip("/")

        # Use persistent session or create a new single-use one.
        session = self._http_session or self._make_http_session()
        try:
            r = session.request(
                method=method,
                url=f"{self.master}/{relpath}",
                params=params,
                data=data,
                headers=headers,
                stream=stream,
                timeout=timeout,
            )
        except requests.exceptions.SSLError:
            raise
        except requests.exceptions.ConnectionError as e:
            raise errors.MasterNotFoundException(str(e))
        except requests.exceptions.RequestException as e:
            raise errors.BadRequestException(str(e))
        finally:
            if not self._http_session:
                # Close the session if not persistent.
                session.close()

        if r.status_code == 403:
            raise errors.ForbiddenException(message=_get_error_str(r))
        if r.status_code == 401:
            raise errors.UnauthenticatedException()
        elif r.status_code == 404:
            raise errors.NotFoundException(_get_error_str(r))
        elif r.status_code >= 300:
            raise errors.APIException(r)

        return r

    def get(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
        stream: bool = False,
    ) -> requests.Response:
        return self._do_request("GET", path, params, None, None, headers, timeout, stream)

    def delete(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("DELETE", path, params, None, None, headers, timeout, False)

    def post(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        json: Any = None,
        data: Optional[str] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("POST", path, params, json, data, headers, timeout, False)

    def patch(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        json: Any = None,
        data: Optional[str] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("PATCH", path, params, json, data, headers, timeout, False)

    def put(
        self,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        json: Any = None,
        data: Optional[str] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("PUT", path, params, json, data, headers, timeout, False)

    def with_retry(self: T, max_retries: GeneralizedRetry) -> T:
        """Generate a new session with a different retry policy."""
        new_session = copy.copy(self)
        new_session._max_retries = max_retries
        new_session._http_session = None
        return new_session


class UnauthSession(BaseSession):
    """
    UnauthSession is mostly only useful to log in or unathenticated endpoints like /info.
    """

    def __init__(
        self,
        master: str,
        cert: Optional[certs.Cert],
        max_retries: Optional[GeneralizedRetry] = DEFAULT_MAX_RETRIES,
    ) -> None:
        if master != api.canonicalize_master_url(master):
            # This check is targeting developers of Determined, not users of Determined.
            raise RuntimeError(
                f"UnauthSession created with non-canonicalized url: {master}; the master url "
                "should have been canonicalized as soon as it was received from the end-user."
            )

        self.master = master
        self.cert = cert
        self._max_retries = max_retries
        self._http_session = None

    def _make_http_session(self) -> requests.Session:
        return _make_requests_session(
            server_hostname=self.cert.name if self.cert else None,
            verify=self.cert.bundle if self.cert else None,
            max_retries=self._max_retries,
        )


class Session(BaseSession):
    """
    Session authenticates every request it makes.

    By far, most BaseSessions in the codebase will be this Session subclass.
    """

    def __init__(
        self,
        master: str,
        username: str,
        token: str,
        cert: Optional[certs.Cert],
        max_retries: Optional[GeneralizedRetry] = DEFAULT_MAX_RETRIES,
    ) -> None:
        if master != api.canonicalize_master_url(master):
            # This check is targeting developers of Determined, not users of Determined.
            raise RuntimeError(
                f"Session created with non-canonicalized url: {master}; the master url should have "
                "been canonicalized as soon as it was received from the end-user."
            )

        self.master = master
        self.username = username
        self.token = token
        self.cert = cert
        self._max_retries = max_retries
        self._http_session = None

    def _make_http_session(self) -> requests.Session:
        return _make_requests_session(
            server_hostname=self.cert.name if self.cert else None,
            verify=self.cert.bundle if self.cert else None,
            max_retries=self._max_retries,
            headers={"Authorization": f"Bearer {self.token}"},
        )


class _HTTPSAdapter(adapters.HTTPAdapter):
    """Overrides the hostname checked against for TLS verification.

    This is used when the request address specified does not match its DNS resolved hostname
    (i.e. custom TLS certificates, private IP addresses).
    """

    def __init__(self, server_hostname: Optional[str], **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.server_hostname = server_hostname

    def cert_verify(self, conn: Any, url: Any, verify: Any, cert: Any) -> None:
        super().cert_verify(conn, url, verify, cert)  # type: ignore
        # Set the server_hostname value of the urllib3 connection.
        conn.assert_hostname = self.server_hostname
