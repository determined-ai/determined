import determined as det
import logging
import os
import subprocess
import sys


def launch() -> int:
    info = det.get_cluster_info()

    chief_ip = info.container_addrs[0]

    harness_cmd = [
        "determined.exec.harness",
        "--chief-ip",
        chief_ip,
    ]

    launch_cmd = [
        "python3",
        "-m",
        "torch.distributed.run",
        "--module"
    ]


    os.environ["USE_TORCH_DISTRIBUTED"] = "True"
    os.environ["MASTER_ADDR"] = chief_ip
    os.environ["MASTER_PORT"] = "7555"
    os.environ["RANK"] = str(info.container_rank)
    os.environ["WORLD_SIZE"] = str(len(info.container_addrs))

    return subprocess.Popen(launch_cmd + harness_cmd).wait()


if __name__ == "__main__":
    sys.exit(launch())