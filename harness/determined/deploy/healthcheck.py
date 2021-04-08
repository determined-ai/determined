import time

import requests

from determined.common import api

from .errors import MasterTimeoutExpired

DEFAULT_TIMEOUT = 100


def _make_master_url(master_host: str, master_port: int, suffix: str = "") -> str:
    return "http://{}:{}/{}".format(master_host, master_port, suffix)


def wait_for_master(
    master_host: str, master_port: int = 8080, timeout: int = DEFAULT_TIMEOUT
) -> None:
    master_url = _make_master_url(master_host, master_port)

    return wait_for_master_url(master_url, timeout)


def wait_for_master_url(master_url: str, timeout: int = DEFAULT_TIMEOUT) -> None:
    POLL_INTERVAL = 2
    polling = False
    start_time = time.time()

    try:
        while time.time() - start_time < timeout:
            try:
                r = api.get(master_url, "info", authenticated=False)
                if r.status_code == requests.codes.ok:
                    return
            except api.errors.MasterNotFoundException:
                pass
            if not polling:
                polling = True
                print("Waiting for master instance to be available...", end="", flush=True)
            time.sleep(POLL_INTERVAL)
            print(".", end="", flush=True)

        raise MasterTimeoutExpired
    finally:
        if polling:
            print()
