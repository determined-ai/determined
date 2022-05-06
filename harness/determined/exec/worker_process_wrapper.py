"""
worker_process_wrapper.py is the entrypoint for distributed worker processes.
It exists to redirect stdout/stderr to the docker logging without needing
to package a shell script.
"""
import argparse
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
    parser = argparse.ArgumentParser()
    # Different launcher may use different environment variable names to indicate rank.
    parser.add_argument("rank_var_name")
    parser.add_argument("cmd", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    rank = os.environ.get(args.rank_var_name)
    proc = subprocess.Popen(args.cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
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
