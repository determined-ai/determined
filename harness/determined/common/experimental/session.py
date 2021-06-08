from typing import Any, Dict, Optional

import requests

from determined.common import util
from determined.common.api import authentication, certs, request


class Session:
    def __init__(
        self,
        master: Optional[str],
        user: Optional[str],
        auth: authentication.Authentication,
        cert: certs.Cert,
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
        body: Optional[Dict[str, Any]],
    ) -> requests.Response:
        return request.do_request(
            method,
            self._master,
            path,
            params=params,
            body=body,
            auth=self._auth,
            cert=self._cert,
        )

    def get(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        body: Optional[Dict[str, Any]] = None,
    ) -> requests.Response:
        return self._do_request("GET", path, params, body)

    def delete(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        body: Optional[Dict[str, Any]] = None,
    ) -> requests.Response:
        return self._do_request("DELETE", path, params, body)

    def post(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        body: Optional[Dict[str, Any]] = None,
    ) -> requests.Response:
        return self._do_request("POST", path, params, body)

    def patch(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        body: Optional[Dict[str, Any]] = None,
    ) -> requests.Response:
        return self._do_request("PATCH", path, params, body)

    def put(
        self,
        path: str,
        params: Optional[Dict[str, Any]] = None,
        body: Optional[Dict[str, Any]] = None,
    ) -> requests.Response:
        return self._do_request("PUT", path, params, body)
