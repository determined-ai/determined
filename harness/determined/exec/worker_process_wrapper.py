"""
worker_process_wrapper.py is the entrypoint for Horovod worker processes.
It exists to redirect stdout/stderr to the docker logging without needing
to package a shell script.
"""
import os
import sys

from determined import constants


def main() -> None:
    os.dup2(os.open(constants.CONTAINER_STDOUT, os.O_WRONLY), sys.stdout.fileno())
    os.dup2(os.open(constants.CONTAINER_STDERR, os.O_WRONLY), sys.stderr.fileno())
    os.execv(
        sys.executable,
        [
            sys.executable,
            "-m",
            "determined.exec.worker_process",
            *sys.argv[1:],
        ],
    )


if __name__ == "__main__":
    main()
