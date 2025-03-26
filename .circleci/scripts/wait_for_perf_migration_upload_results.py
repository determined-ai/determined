import datetime
import os
import pathlib
import re
import subprocess
import time

import requests

from determined.common import api, util
from determined.common.api import authentication, bindings, certs


def _wait_for_master() -> None:
    print("Checking for master")
    cert = certs.Cert(noverify=True)
    sess = api.UnauthSession("http://127.0.0.1:8080", cert).with_retry(0)

    # 15 minutes is the most a migration can take, with this setup.
    # If a migration takes longer than that we have hit an issue a customer will likely hit too.
    time_start = time.time()
    time_last_report = time_start
    while time_start - time.time() < 15 * 60:
        try:
            r = sess.get("info")
            if r.status_code == requests.codes.ok:
                print(f"Master up and available after {int(time.time() - time_start) / 60} minutes")
                return
        except api.errors.MasterNotFoundException:
            pass
        if time.time() - time_last_report > 60:
            time_last_report = time.time()
            print(f"Waiting for master ({int(time_last_report - time_start) / 60} minutes elapsed)")
        time.sleep(1)
    raise ConnectionError("Timed out connecting to Master")


if __name__ == "__main__":
    _wait_for_master()
