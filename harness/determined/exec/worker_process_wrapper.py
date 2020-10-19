"""
worker_process_wrapper.py is the entrypoint for Horovod worker processes.
It exists to redirect stdout/stderr to the docker logging without needing
to package a shell script.
"""
import itertools
import os
import subprocess
import sys
import threading
from typing import List, TextIO

from determined import constants


def forward_stream(src_stream: TextIO, dst_stream: TextIO, rank: int) -> None:
    for line in iter(src_stream.readline, None):
        if line is None:
            break
        if not isinstance(line, str):
            line = line.decode("utf-8")
        if not line:
            break
        line = "[rank={rank}] {line}".format(
            line=line,
            rank=str(rank),
        )
        os.write(dst_stream.fileno(), line.encode("utf-8"))


def run_all(fwds: List[threading.Thread]) -> None:
    for fwd in fwds:
        fwd.start()

    done = {}
    for fwd in itertools.cycle(fwds):
        if fwd.name in done:
            continue

        fwd.join(timeout=1)
        if not fwd.isAlive():
            done[fwd.name] = True

        if len(done) == len(fwds):
            return


def main() -> None:
    rank = os.environ["HOROVOD_RANK"]
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
    with open(constants.CONTAINER_STDOUT, "w") as cstdout:
        with open(constants.CONTAINER_STDERR, "w") as cstderr:
            with proc:
                run_all(
                    [
                        threading.Thread(target=forward_stream, args=(proc.stdout, cstdout, rank)),
                        threading.Thread(target=forward_stream, args=(proc.stderr, cstderr, rank)),
                    ]
                )


if __name__ == "__main__":
    main()
