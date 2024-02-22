"""
check_ready_logs.py accepts a task's logs as STDIN, runs a regex to determine and report readiness.
Callers should be aware it may terminate early, and stop reading from STDIN.
"""
import argparse
import os
import re
import sys
import time
from typing import Optional, Pattern

from requests.exceptions import RequestException

from determined.common import api
from determined.common.api import authentication, certs

BACKOFF_SECONDS = 5


def post_ready(sess: api.Session, allocation_id: str, state: str):
    # Since the service is virtually inaccessible by the user unless
    # the call completes, we may as well try forever or just wait for
    # them to kill us.
    while True:
        try:
            sess.post(f"/api/v1/allocations/{allocation_id}/{state}")
            return
        except RequestException as e:
            if e.response is not None and e.response.status_code < 500:
                raise e
            time.sleep(BACKOFF_SECONDS)


def main(ready: Pattern, waiting: Optional[Pattern] = None):
    master_url = api.canonicalize_master_url(os.environ["DET_MASTER"])
    cert = certs.default_load(master_url)
    # This only runs on-cluster, so it is expected the username and session token are present in the
    # environment.
    utp = authentication.login_with_cache(master_url, cert=cert)
    sess = api.Session(master_url, utp, cert)
    allocation_id = str(os.environ["DET_ALLOCATION_ID"])
    for line in sys.stdin:
        if ready.match(line):
            post_ready(sess, allocation_id, "ready")
            return
        if waiting and waiting.match(line):
            post_ready(sess, allocation_id, "waiting")
            return

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Read STDIN for a match and mark a task as ready")
    parser.add_argument(
        "--ready-regex", type=str, help="the pattern to match task ready", required=True
    )
    parser.add_argument("--waiting-regex", type=str, help="the pattern to match task waiting")
    args = parser.parse_args()

    ready_regex = re.compile(args.ready_regex)
    if args.waiting_regex:
        waiting_regrex = re.compile(args.waiting_regex)
        main(ready_regex, waiting_regrex)
    else:
        main(ready_regex, None)
