"""
worker_process_wrapper.py is the entrypoint for Horovod worker processes.
It exists to redirect stdout/stderr to the docker logging without needing
to package a shell script.
"""
import os
import subprocess
import sys
import threading
from typing import BinaryIO, List

from determined import constants


def forward_stream(src_stream: BinaryIO, dst_stream: BinaryIO, rank: str) -> None:
    for line in iter(src_stream.readline, b""):
        line = f"[rank={rank}] ".encode() + line
        os.write(dst_stream.fileno(), line)


def run_all(ts: List[threading.Thread]) -> None:
    for t in ts:
        t.start()
    for t in ts:
        t.join()


def main() -> int:
    rank = os.environ.get("HOROVOD_RANK")
    proc = subprocess.Popen(
        [
            sys.executable,
            "-m",
            "determined.exec.worker_process",
            *sys.argv[1:],
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    with open(constants.CONTAINER_STDOUT, "w") as cstdout, open(
        constants.CONTAINER_STDERR, "w"
    ) as cstderr, proc:
        run_all(
            [
                threading.Thread(target=forward_stream, args=(proc.stdout, cstdout, rank)),
                threading.Thread(target=forward_stream, args=(proc.stderr, cstderr, rank)),
            ]
        )

    return proc.returncode


if __name__ == "__main__":
    sys.exit(main())
