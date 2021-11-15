"""
check_ready.py accepts a task's logs as STDIN, runs a regex to determine readiness
and reemits the logs to STDOUT
"""
import argparse
import os
import sys
import re
import os
from time import sleep
from determined.common import api
from determined.common.api import certs
from typing import Any, Pattern
import backoff
from requests.exceptions import RequestException


BACKOFF_SECONDS = 5

# Since the service is virtually inaccessible by the user unless
# the call completes, we may as well try forever or just wait for
# them to kill us.
@backoff.on_exception(  # type: ignore
    lambda: backoff.constant(interval=BACKOFF_SECONDS),
    RequestException,
    giveup=lambda e: e.response is not None and e.response.status_code < 500,
)
def post_ready(master_url: str, cert: certs.Cert, allocation_id: str):
    api.post(
        master_url,
        f"/api/v1/allocations/{allocation_id}/ready",
        {},
        cert=cert,
    )


def main(ready: Pattern) -> int:
    master_url = str(os.environ["DET_MASTER"])
    cert = certs.default_load(master_url)
    allocation_id = str(os.environ["DET_ALLOCATION_ID"])
    for line in sys.stdin:
        if ready.match(line):
            post_ready(master_url, cert, allocation_id)
            return


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Read STDIN for a match and mark a task as ready')
    parser.add_argument('--ready-regex', type=str, help='the pattern to match', required=True)
    args = parser.parse_args()

    ready_regex = re.compile(args.ready_regex)
    main(ready_regex)
