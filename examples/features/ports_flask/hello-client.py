import contextlib
import logging
import re
import subprocess
import time
from typing import Iterator

import requests

import determined as det


def _parse_exp_id(proc: "subprocess.Popen[str]") -> int:
    assert proc.stdout is not None
    for line in iter(proc.stdout.readline, ""):
        if proc.poll() is not None:
            raise ValueError(
                f"Unexpected `det e create` failure before receiving an experiment id, "
                f"return code: f{proc.returncode}"
            )
        m = re.search(r"Created experiment (\d+)\n", line)
        if m is not None:
            return int(m.group(1))
    raise ValueError("Failed to find experiment id in `det e create` output")


@contextlib.contextmanager
def launch_server() -> Iterator[None]:
    print("Starting hello-server...")
    cmd = ["det", "e", "create", "hello-server.yaml", ".", "-f", "-p", "5000"]
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    exp_id = _parse_exp_id(proc)
    print(f"Server experiment id: {exp_id}")
    # TODO: instead of the sleep, we could check if the experiment is running.
    time.sleep(5)
    yield
    print("Killing hello-server...")
    cmd = ["det", "e", "kill", str(exp_id)]
    subprocess.run(cmd)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    slots_per_node = len(info.slot_ids)
    num_nodes = len(info.container_addrs)
    cross_rank = info.container_rank
    chief_ip = info.container_addrs[0]

    # Local port will be used to setup a tunnel.
    URL = "http://localhost:5000/hello"
    with launch_server():
        # Probe the server liveliness.
        print("Probing the server liveliness...")
        for _i in range(3 * 60):
            try:
                r = requests.get(URL, timeout=3)
                if r.status_code == 200:
                    break
                print(f"Bad status code: {r.status_code}, retrying...")
            except requests.exceptions.ConnectionError:
                print("ConnectionError, retrying...")
            except requests.exceptions.ReadTimeout:
                print("ReadTimeout, retrying...")
            time.sleep(1.0)
        else:
            raise ValueError("Probe failure")

        r = requests.get(URL)
        r.raise_for_status()
        resp_json = r.json()
        print("Got server response: ", resp_json)
        assert resp_json["data"] == "Hello World"
        print("SUCCESS!")
