import time
from typing import Optional

import requests

from determined.common import api
from determined.common.api import authentication, certs
from determined.deploy import errors

DEFAULT_TIMEOUT = 100


def wait_for_master(
    master_url: str,
    timeout: int = DEFAULT_TIMEOUT,
    cert: Optional[certs.Cert] = None,
) -> None:
    POLL_INTERVAL = 2
    polling = False
    start_time = time.time()

    try:
        while time.time() - start_time < timeout:
            try:
                sess = api.UnauthSession(master_url, cert=cert)
                r = sess.get("info")
                if r.status_code == requests.codes.ok:
                    return
            except api.errors.MasterNotFoundException:
                pass
            if not polling:
                polling = True
                print("Waiting for master instance to be available...", end="", flush=True)
            time.sleep(POLL_INTERVAL)
            print(".", end="", flush=True)

        raise errors.MasterTimeoutExpired
    finally:
        if polling:
            print()


def wait_for_genai_url(
    master_url: str,
    timeout: int = DEFAULT_TIMEOUT,
    cert: Optional[certs.Cert] = None,
) -> None:
    POLL_INTERVAL = 2
    polling = False
    start_time = time.time()

    # Hopefully we have an active session to this master, or we can make a default one.
    sess = authentication.login_with_cache(master_url, cert=cert)

    try:
        while time.time() - start_time < timeout:
            try:
                r = sess.get("genai/api/v1/workspaces")
                if r.status_code == requests.codes.ok:
                    _ = r.json()
                    return
            except (api.errors.MasterNotFoundException, api.errors.APIException):
                pass
            if not polling:
                polling = True
                print("Waiting for GenAI instance to be available...", end="", flush=True)
            time.sleep(POLL_INTERVAL)
            print(".", end="", flush=True)
        raise errors.MasterTimeoutExpired
    finally:
        if polling:
            print()
