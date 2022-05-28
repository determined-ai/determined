"""
Make a feeble attempt to post an error directly to the master when the logging setup fails.

Do it in python instead of in curl, we expect more images to have python than curl on our platform.
"""

# Only stdlib dependencies allowed here.
import json
import os
import sys
import urllib.request
import time
import datetime

master_url = os.environ["DET_MASTER"]
static = {
    "level": "ERROR",
    "task_id": os.environ["DET_TASK_ID"],
    "allocation_id": os.environ["DET_ALLOCATION_ID"],
    "agent_id": os.environ.get("DET_AGENT_ID"),
    # XXX: container id?  or no?
    "timestamp": datetime.datetime.now(datetime.timezone.utc).isoformat(),
}

text = " ".join(sys.argv[1:]) + "\n"
body = [{"log": line, **static} for line in text.splitlines(keepends=True)]
data = json.dumps(body).encode("utf8")

req = urllib.request.Request(f"{master_url}/task-logs", method="POST", data=data)
with urllib.request.urlopen(req):
    pass
