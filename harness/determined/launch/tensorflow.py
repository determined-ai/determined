import argparse
import json
import logging
import os
import subprocess
import sys
from typing import List, Tuple

import determined as det

# We use the same port configure as our torch_distributed launcher, to make network communications
# a little easier for the cluster admin.
C10D_PORT = int(str(os.getenv("C10D_PORT", "29400")))

logger = logging.getLogger("determined.launch.tensorflow")


def create_log_wrapper(rank: int) -> List[str]:
    return [
        "python3",
        "-m",
        "determined.launch.wrap_rank",
        str(rank),
        "--",
    ]


def main(port: int, script: List[str]) -> int:
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"

    chief_ip = info.container_addrs[0]
    env = {**os.environ, "DET_CHIEF_IP": chief_ip}

    if len(info.container_addrs) > 1:
        # Multi-node training means MultiWorkerMirroredStrategy.
        tf_config = {
            "cluster": {"worker": [f"{addr}:{port}" for addr in info.container_addrs]},
            "task": {"type": "worker", "index": info.container_rank},
        }
        env["TF_CONFIG"] = json.dumps(tf_config)
        log_wrapper = create_log_wrapper(info.container_rank)
    else:
        # Single-node training means MirroredStrategy or just the default strategy.
        # (no point in prefixing every log line with "rank=0")
        log_wrapper = []

    launch_cmd = log_wrapper + script

    logger.debug(f"Tensorflow launching with: {launch_cmd}")

    p = subprocess.Popen(launch_cmd, env=env)
    with det.util.forward_signals(p):
        return p.wait()


def parse_args(args: List[str]) -> Tuple[int, List[str]]:
    parser = argparse.ArgumentParser(
        usage="%(prog)s [--port PORT] [--] SCRIPT...",
        description="Launch a script for tensorflow training on a Determined cluster.",
        epilog=(
            "This launcher automatically injects a TF_CONFIG environment variable suitable for "
            "MirroredStrategy or MultiWorkerMirroredStrategy when multiple nodes and or GPUs are "
            "available."
        ),
    )
    parser.add_argument(
        "--port",
        type=int,
        help="the port that TensorFlow should use for distributed training communication",
        default=C10D_PORT,
    )
    parser.add_argument(
        "script",
        metavar="SCRIPT...",
        nargs=argparse.REMAINDER,
        help="script to launch for training",
    )

    # Manually process the -- because argparse doesn't quite handle it right.
    if "--" in args:
        split = args.index("--")
        args, extra_script = args[:split], args[split + 1 :]
    else:
        extra_script = []

    parsed = parser.parse_args(args)

    full_script = parsed.script + extra_script

    if not full_script:
        # There needs to be at least one script argument.
        parser.print_usage()
        print("error: empty script is not allowed", file=sys.stderr)
        sys.exit(1)

    return parsed.port, full_script


if __name__ == "__main__":
    port, script = parse_args(sys.argv[1:])
    sys.exit(main(port, script))
