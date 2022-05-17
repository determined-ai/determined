import json
import os
from typing import Any, Dict, Optional

import requests
import urllib3

from determined.common import util
from determined.common.api import authentication, certs, request


def get_max_retries_config() -> urllib3.util.retry.Retry:
    # Allow overriding retry settings when necessary.
    # `DET_RETRY_CONFIG` env variable can contain `urllib3` `Retry` parameters,
    # encoded as JSON.
    # For example:
    #  - disable retries: {"total":0}
    #  - shorten the wait times {"total":10,"backoff_factor":0.5,"method_whitelist":false}

    config_data = os.environ.get("DET_RETRY_CONFIG")
    if config_data is not None:
        config = json.loads(config_data)
        return urllib3.util.retry.Retry(**config)

    # Defaults.
    try:
        return urllib3.util.retry.Retry(
            total=20,
            backoff_factor=0.5,
            allowed_methods=False,
        )
    except TypeError:  # Support urllib3 prior to 1.26
        return urllib3.util.retry.Retry(
            total=20,
            backoff_factor=0.5,
            method_whitelist=False,  # type: ignore
        )


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
            max_retries=self._max_retries,
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
