from typing import Any, Dict, Optional

import requests
import urllib3

from determined.common import util
from determined.common.api import authentication, certs, request


class Session:
    def __init__(
        self,
        master: Optional[str],
        user: Optional[str],
        auth: Optional[authentication.Authentication],
        cert: Optional[certs.Cert],
        max_retries: Optional[urllib3.util.retry.Retry] = None,
    ) -> None:
        self._master = master or util.get_default_master_address()
        self._user = user
        self._auth = auth
        self._cert = cert
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
        return request.do_request(
            method,
            self._master,
            path,
            params=params,
            json=json,
            data=data,
            auth=self._auth,
            cert=self._cert,
            headers=headers,
            timeout=timeout,
            stream=stream,
            max_retries=self._max_retries,
        )

    def get(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
        stream: bool = False,
    ) -> requests.Response:
        return self._do_request("GET", path, params, None, None, headers, timeout, stream)

    def delete(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("DELETE", path, params, None, None, headers, timeout, False)

    def post(
        self,
        path: str,
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
        params: Optional[Dict[str, Any]] = None,
        json: Any = None,
        data: Optional[str] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("PUT", path, params, json, data, headers, timeout, False)
