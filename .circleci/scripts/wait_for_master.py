import argparse
import time

import requests

from determined.common import api
from determined.common.api import authentication, certs


def _wait_for_master(address: str) -> None:
    print("Checking for master at", address)
    cert = certs.Cert(noverify=True)
    sess = api.UnauthSession(address, cert)
    for _ in range(150):
        try:
            r = sess.get("info")
            if r.status_code == requests.codes.ok:
                return
        except api.errors.MasterNotFoundException:
            pass
        print("Waiting for master to be available...")
        time.sleep(2)
    raise ConnectionError("Timed out connecting to Master")


def main() -> None:
    parser = argparse.ArgumentParser(description="Wait for master helper.")
    parser.add_argument("address", help="Master address.")
    args = parser.parse_args()
    _wait_for_master(args.address)


if __name__ == "__main__":
    main()
