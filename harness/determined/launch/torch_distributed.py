import logging
import determined as det
import subprocess
import sys
import os

def parse_url(url):
    split = url.split(":")
    return split[1][2:], split[2]


def launch():
    cluster_info = det.get_cluster_info()
    harness_cmd = [
        "determined.exec.harness"
    ]
    print(f"pwd: {os.getcwd()}")
    master_url, port = parse_url(cluster_info.master_url)
    print(f"master addr {master_url}, port {port}")
    launch_cmd = [
        "python3",
        "-m",
        "torch.distributed.run",
        "--nproc_per_node",
        str(len(cluster_info.slot_ids)),
        "--nnodes",
        str(len(cluster_info.container_addrs)),
        "--rdzv_backend",
        "c10d",
        "--rdzv_endpoint",
        f"{master_url}:{port}",
        "--rdzv_id",
        str(cluster_info.trial._trial_run_id),
        "--module"
    ]

    print(f"full command {launch_cmd + harness_cmd}")

    return subprocess.Popen(launch_cmd + harness_cmd).wait()


if __name__ == "__main__":
    sys.exit(launch())