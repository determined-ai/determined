import argparse
import datetime
import json
import os
import re
import socket
import sys
import time
from typing import Any, Dict

from determined.common import api

# Example log message given below.
# 2022-05-12 16:32:48,757:gc_checkpoints: [rank=0] INFO: Determined checkpoint GC, version 0.17.16-dev0
# Below regex is used to extract the rank field from the log message. 
# Excluding empty spaces this regex matches rank in the above example as [rank=0]
rank = re.compile("(?P<space1> ?)\[rank=(?P<rank_id>([0-9]+))\](?P<space2> ?)(?P<log>.*)")
# Below regex is used to extract the message severity from the log message. 
# Excluding empty spaces and delimiter(:) this regex matches message severity level in the above example as INFO
level = re.compile("(?P<space1> ?)(?P<level>(DEBUG|INFO|WARNING|ERROR|CRITICAL)):(?P<space2> ?)(?P<log>.*)")


def main(
    master_url: str,
    task_logging_metadata: Dict[str, Any],
) -> None:
    buffer = []
    # Maximum number of log messages to be send to determined master each time.
    max_buffer_length = 1024
    start_time = time.time()
    # Maximum time in seconds the loop executes, before sending log messages to determined master.
    frequency = 5

    for line in sys.stdin:
        parsed_metadata = {}

        m = rank.match(line)
        if m:
            parsed_metadata["rank"] = m.group("rank_id")
            line = m.group("log")

        m = level.match(line)
        if m:
            parsed_metadata["level"] = m.group("level")
            line = m.group("log")

        log =  {
                "timestamp": datetime.datetime.now(
                    datetime.timezone.utc
                ).isoformat(),
                "log": line,
                **task_logging_metadata,
                **parsed_metadata,
            }
        
        buffer.append(log)

        elapsed_time = time.time() - start_time

        if len(buffer) < max_buffer_length and elapsed_time < frequency:
            continue

        # Send enriched logs to determined master.
        api.post(master_url, "task-logs", buffer)
        # Reset buffer and timer
        buffer = []
        start_time = time.time()

    if len(buffer):
        # If any, send the last outstanding messages to master.
        api.post(master_url, "task-logs", buffer)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="read a stream and enrich it with the standard logging metadata"
    )
    parser.add_argument(
        "--stdtype", type=str, help="the stdtype of this stream", required=True
    )
    args = parser.parse_args()

    master_url = os.environ.get("DET_MASTER", os.environ.get("DET_MASTER_ADDR"))
    assert master_url is not None, "DET_MASTER and DET_MASTER_ADDR unset"

    task_logging_metadata_json = os.environ.get("DET_TASK_LOGGING_METADATA")
    assert task_logging_metadata_json is not None, "DET_TASK_LOGGING_METADATA unset"

    task_logging_metadata = json.loads(task_logging_metadata_json)
    task_logging_metadata["stdtype"] = args.stdtype
    task_logging_metadata["agent_id"] = socket.gethostname()
    # If trial exists, just drop it since it could mess with de-ser on the API end.
    task_logging_metadata.pop("trial_id", None)

    main(master_url, task_logging_metadata)
