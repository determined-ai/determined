import logging
import subprocess
import sys
import time

import determined as det


def launcher_main(cross_rank, chief_ip):
    if cross_rank == 0:
        subprocess.run(["ray", "start", "--head", "--dashboard-host", "0.0.0.0"], check=True)
    else:
        subprocess.run(["ray", "start", f"--address={chief_ip}:6379"], check=True)
    while True:
        time.sleep(1)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    slots_per_node = len(info.slot_ids)
    num_nodes = len(info.container_addrs)
    cross_rank = info.container_rank
    chief_ip = info.container_addrs[0]

    exitcode = launcher_main(cross_rank, chief_ip)
    sys.exit(exitcode)
