import argparse
import datetime
import distutils.util
import json
import os
import queue
import re
import socket
import sys
import threading
import time
from typing import Any, Dict, Iterator

from determined.common import api
from determined.common.api import errors

# Example log message given below.
# 2022-05-12 16:32:48,757:gc_checkpoints: [rank=0] INFO: Determined checkpoint GC, version 0.17.16-dev0
# Below regex is used to extract the rank field from the log message.
# Excluding empty spaces this regex matches rank in the above example as [rank=0]
rank = re.compile("(?P<space1> ?)\[rank=(?P<rank_id>([0-9]+))\](?P<space2> ?)(?P<log>.*)")
# Below regex is used to extract the message severity from the log message.
# Excluding empty spaces and delimiter(:) this regex matches message severity level in the above example as INFO
level = re.compile(
    "(?P<space1> ?)(?P<level>(DEBUG|INFO|WARNING|ERROR|CRITICAL)):(?P<space2> ?)(?P<log>.*)"
)


# Interval at which to force a flush.
SHIPPER_FLUSH_INTERVAL = 1  # How often to make API calls

# Full jitter time on encountering an API exception.
SHIPPER_FAILURE_BACKOFF_SECONDS = 1

# Max size of the log buffer before forcing a flush.
LOG_BATCH_MAX_SIZE = 1000

# Max size of the shipping queue before we start to apply backpressure by blocking sends. We would
# only hit this if we got underwater by three full batches while trying to ship a batch.
SHIP_QUEUE_MAX_SIZE = 3 * LOG_BATCH_MAX_SIZE


class ShutdownMessage:
    pass


class LogCollector(threading.Thread):
    def __init__(
        self,
        ship_queue: queue.Queue,
        task_logging_metadata: Dict[str, Any],
        emit_stdout_logs: bool,
    ):
        self.ship_queue = ship_queue
        self.task_logging_metadata = task_logging_metadata
        self.emit_stdout_logs = emit_stdout_logs
        super().__init__()

    def run(self) -> None:
        try:
            for line in sys.stdin:
                if self.emit_stdout_logs:
                    print(line, flush=True, end="")
                try:
                    parsed_metadata = {}

                    m = rank.match(line)
                    if m:
                        try:
                            parsed_metadata["rank_id"] = int(m.group("rank_id"))
                            line = m.group("log")
                        except ValueError:
                            pass

                    m = level.match(line)
                    if m:
                        parsed_metadata["level"] = m.group("level")
                        line = m.group("log")

                    self.ship_queue.put(
                        {
                            "timestamp": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                            "log": line if line.endswith("\n") else line + "\n",
                            **self.task_logging_metadata,
                            **parsed_metadata,
                        }
                    )
                except Exception as e:
                    print(f"fatal error collecting log {e}", file=sys.stderr)
        finally:
            self.ship_queue.put(ShutdownMessage())


class LogShipper(threading.Thread):
    """
    This is a thread that exists solely so that we can batch logs and ship them to the
    SenderThread every FLUSH_INTERVAL seconds.
    """

    def __init__(
        self,
        ship_queue: queue.Queue,
        master_url: str,
    ) -> None:
        self.ship_queue = ship_queue
        self.logs = []
        self.master_url = master_url
        super().__init__()

    def run(self) -> None:
        while True:
            deadline = time.time() + SHIPPER_FLUSH_INTERVAL
            for m in pop_until_deadline(self.ship_queue, deadline):
                if isinstance(m, ShutdownMessage):
                    self.ship()
                    return

                self.logs.append(m)
                if len(self.logs) >= LOG_BATCH_MAX_SIZE:
                    self.ship()

            # Timeout met.
            self.ship()

    def ship(self) -> None:
        if len(self.logs) <= 0:
            return

        max_tries = 3
        tries = 0
        while tries < max_tries:
            try:
                api.post(self.master_url, "task-logs", self.logs)
                self.logs = []
                return
            except errors.APIException as e:
                tries += 1
                if tries == max_tries:
                    raise e
                time.sleep(SHIPPER_FAILURE_BACKOFF_SECONDS)


def pop_until_deadline(q: queue.Queue, deadline: float) -> Iterator[Any]:
    while True:
        timeout = deadline - time.time()
        if timeout <= 0:
            break

        try:
            yield q.get(timeout=timeout)
        except queue.Empty:
            break


def main(
    master_url: str,
    task_logging_metadata: Dict[str, Any],
    emit_stdout_logs: bool,
) -> None:
    ship_queue = queue.Queue(maxsize=SHIP_QUEUE_MAX_SIZE)
    collector = LogCollector(ship_queue, task_logging_metadata, emit_stdout_logs)
    shipper = LogShipper(ship_queue, master_url)

    collector.start()
    shipper.start()

    # Collector will exit when it sees the end of stdin.
    collector.join()
    shipper.join()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="read a stream and enrich it with the standard logging metadata"
    )
    parser.add_argument("--stdtype", type=str, help="the stdtype of this stream", required=True)
    args = parser.parse_args()

    master_url = os.environ.get("DET_MASTER", os.environ.get("DET_MASTER_ADDR"))
    assert master_url is not None, "DET_MASTER and DET_MASTER_ADDR unset"

    task_logging_metadata_json = os.environ.get("DET_TASK_LOGGING_METADATA")
    assert task_logging_metadata_json is not None, "DET_TASK_LOGGING_METADATA unset"

    task_logging_metadata = json.loads(task_logging_metadata_json)
    task_logging_metadata["stdtype"] = args.stdtype
    task_logging_metadata["agent_id"] = socket.gethostname()
    task_logging_metadata["source"] = "task"
    container_id = os.environ.get("DET_CONTAINER_ID")
    if container_id is not None:
        task_logging_metadata["container_id"] = container_id
    # If trial exists, just drop it since it could mess with de-ser on the API end.
    task_logging_metadata.pop("trial_id", None)
    emit_stdout_logs = distutils.util.strtobool(
        os.environ.get("DET_SHIPPER_EMIT_STDOUT_LOGS", "True"),
    )

    main(master_url, task_logging_metadata, emit_stdout_logs)
