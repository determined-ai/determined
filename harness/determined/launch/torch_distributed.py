import logging
import subprocess
import sys


def launch():
    harness_cmd = [
        "determined.exec.harness",
        "--distributed",
        "torch"
    ]

    launch_cmd = [
        "python3",
        "-m",
        "torch.distributed.run",
        "--module"
    ]

    return subprocess.Popen(launch_cmd + harness_cmd).wait()


if __name__ == "__main__":
    sys.exit(launch())