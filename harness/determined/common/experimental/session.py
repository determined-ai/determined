from typing import Any, Dict, Optional

import requests

from determined.common import util
from determined.common.api import authentication, certs, request


class Session:
    def __init__(
        self,
        master: Optional[str],
        user: Optional[str],
        auth: Optional[authentication.Authentication],
        cert: Optional[certs.Cert],
    ) -> None:
        self._master = master or util.get_default_master_address()
        self._user = user
        self._auth = auth
        self._cert = cert

    def _do_request(
        self,
        method: str,
        path: str,
        params: Optional[Dict[str, Any]],
        json: Any,
        data: Optional[str],
        headers: Optional[Dict[str, Any]],
        timeout: Optional[int],
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
        )

    def get(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("GET", path, params, None, None, headers, timeout)

    def delete(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("DELETE", path, params, None, None, headers, timeout)

    def post(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        json: Any = None,
        data: Optional[str] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("POST", path, params, json, data, headers, timeout)

    def patch(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        json: Any = None,
        data: Optional[str] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("PATCH", path, params, json, data, headers, timeout)

    def put(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        json: Any = None,
        data: Optional[str] = None,
        headers: Optional[Dict[str, Any]] = None,
        timeout: Optional[int] = None,
    ) -> requests.Response:
        return self._do_request("PUT", path, params, json, data, headers, timeout)
