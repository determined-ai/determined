"""
wrap_rank.py prefixes every line of output from a worker process with the rank
that emitted it.

In distributed training, the rank prefix added by wrap_rank.py is necessary for
the WebUI log viewer's filter-by-rank feature to work.

Additionally, when used in a Determined container, wrap_rank.py redirects stdout
and stderr of the worker process to the stdout and stderr of the container.  The
purpose of this is to save network bandwidth when launchers like mpirun or
horovodrun are used, as they often are configured to send all logs from worker
nodes to the chief node over the network.  This may be disabled with the
``--no-redirect-stdio`` flag.
"""
import argparse
import contextlib
import os
import subprocess
import sys
import threading
from typing import BinaryIO, List

import determined as det
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
    parser = argparse.ArgumentParser(
        usage="wrap_rank.py [-h] [--no-redirect-stdio] RANK SCRIPT...",
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--no-redirect-stdio", action="store_true")
    parser.add_argument(
        "rank",
        metavar="RANK",
        help=(
            "Can be an integer rank or a comma-separated list of "
            "names of environment variables which are tried, in order, "
            "to determine an integer rank."
        ),
    )
    parser.add_argument(
        "script", nargs=argparse.REMAINDER, metavar="SCRIPT...", help="The worker command."
    )
    args = parser.parse_args()

    if set("0123456789") >= set(args.rank):
        # Rank is provided as a number.
        rank = int(args.rank)
    else:
        # Rank is provided as the name of an environment variable.
        for r in args.rank.split(","):
            if r in os.environ:
                rank = int(os.environ[r])
                break
        else:
            print(
                f"rank environment variable is set to {args.rank}, but it is not in os.environ",
                file=sys.stderr,
            )
            return 1

    # Slurm/PBS: Hack to refresh the working directory using the softlink in the
    # current container.  Each container's "/run/determined/workdir" is actually
    # a symlink to a mounted shared directory on the host whose path contains
    # "*/procs/${SLURM_PROCID}/run/determined/workdir". "os.getcwd()" returns
    # the real path pointed to by the symlink, instead of the symlink itself.
    # Because the chief is using "os.getcwd()" to get the working directory, it
    # is propagating its rank-specific directory to all the workers, causing the
    # workers' working directory to be set to "*/procs/0/run/determined/workdir".
    # This results in collisions when the workers are downloading the dataset,
    # because all the workers are downloading to the same directory.  Each worker
    # needs to refresh its working directory by getting the real path of the
    # symlink pointed to by "/run/determined/workdir" to its own working
    # directory (e.g., "*/procs/1/run/determined/workdir",
    # "*/procs/2/run/determined/workdir", and so on).
    cwd = os.getcwd()
    if cwd.endswith("/run/determined/workdir"):
        os.chdir("/run/determined/workdir")

    proc = subprocess.Popen(args.script, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    with det.util.forward_signals(proc):
        with contextlib.ExitStack() as exit_stack:
            if os.path.exists(constants.CONTAINER_STDOUT) and not args.no_redirect_stdio:
                stdout = exit_stack.enter_context(open(constants.CONTAINER_STDOUT, "w"))
                stderr = exit_stack.enter_context(open(constants.CONTAINER_STDERR, "w"))
            else:
                stdout = sys.stdout
                stderr = sys.stderr
            run_all(
                [
                    threading.Thread(target=forward_stream, args=(proc.stdout, stdout, rank)),
                    threading.Thread(target=forward_stream, args=(proc.stderr, stderr, rank)),
                ]
            )

        return proc.wait()


if __name__ == "__main__":
    sys.exit(main())
