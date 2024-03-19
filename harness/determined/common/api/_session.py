import abc
import copy
import json as _json
from typing import Any, Dict, Optional, Tuple, TypeVar, Union

import requests
import urllib3

import determined as det
from determined.common import api
from determined.common import requests as det_requests
from determined.common.api import certs, errors

GeneralizedRetry = Union[urllib3.util.retry.Retry, int]
T = TypeVar("T", bound="BaseSession")

# Default retry logic
DEFAULT_MAX_RETRIES = urllib3.util.retry.Retry(
    total=5,
    backoff_factor=0.5,  # {backoff factor} * (2 ** ({number of total retries} - 1))
    status_forcelist=[502, 503, 504],  # Bad Gateway, Service Unavailable, Gateway Timeout
)


def _do_request(
    method: str,
    host: str,
    path: str,
    max_retries: Optional[GeneralizedRetry],
    params: Optional[Dict[str, Any]] = None,
    json: Any = None,
    data: Optional[str] = None,
    headers: Optional[Dict[str, str]] = None,
    cert: Optional[certs.Cert] = None,
    timeout: Optional[Union[Tuple, float]] = None,
    stream: bool = False,
) -> requests.Response:
    # Allow the json to come pre-encoded, if we need custom encoding.
    if json is not None and data is not None:
        raise ValueError("json and data must not be provided together")

    if json:
        data = det.util.json_encode(json)

    relpath = path.lstrip("/")

    try:
        r = det_requests.request(
            method,
            f"{host}/{relpath}",
            params=params,
            data=data,
            headers=headers,
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
        raise errors.ForbiddenException(message=_get_error_str(r))
    if r.status_code == 401:
        raise errors.UnauthenticatedException()
    elif r.status_code == 404:
        raise errors.NotFoundException(_get_error_str(r))
    elif r.status_code >= 300:
        raise errors.APIException(r)

    return r


class BaseSession(metaclass=abc.ABCMeta):
    """
    BaseSession is a requests-like interface that hides master url, master cert, and authz info.

    There are very few cases where BaseSession is the right type; you probably want a Session.  In
    a few cases, you might be ok with an UnauthSession.  BaseSession is really only to express that
    you don't know what kind of session you need.  For example, the generated bindings take a
    BaseSession because the protos aren't annotated with which endpoints are authenticated.
    """

    master: str
    cert: Optional[certs.Cert]
    _max_retries: Optional[GeneralizedRetry]

    @abc.abstractmethod
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
        pass

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
        return _do_request(
            method=method,
            host=self.master,
            path=path,
            max_retries=self._max_retries,
            params=params,
            json=json,
            data=data,
            headers=headers,
            cert=self.cert,
            timeout=timeout,
            stream=stream,
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
        # Add authentication.
        headers = dict(headers) if headers is not None else {}
        headers["Authorization"] = f"Bearer {self.token}"
        return _do_request(
            method=method,
            host=self.master,
            path=path,
            max_retries=self._max_retries,
            params=params,
            json=json,
            data=data,
            cert=self.cert,
            headers=headers,
            timeout=timeout,
            stream=stream,
        )
