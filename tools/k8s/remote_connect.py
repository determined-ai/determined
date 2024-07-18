#!/usr/bin/env python

import pathlib
import random
import socket
import subprocess
import time
from dataclasses import dataclass
from typing import Tuple


@dataclass
class Config:
    reverse_proxy_host: str
    k8s_context: str
    ssh_key_path: pathlib.Path
    ssh_user: str = "ubuntu"
    # base_devcluster_path: pathlib.Path
    local_master_port: int = 8080
    remote_port_range: Tuple[int, int] = (8000, 9000)


"""
- set up reverse proxy
    - port collision
    - err handling
    - clean up
- check and set up gateway
    - record ip
- updated devcluster config

"""


def is_port_listening(host: str, port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        result = sock.connect_ex((host, port))
        return result == 0


def wait_for_tunnel(remote_addr: str, remote_port: int, timeout: int = 30) -> bool:
    start_time = time.time()
    while time.time() - start_time < timeout:
        if is_port_listening(remote_addr, remote_port):
            print(f"Tunnel is up on {remote_addr}:{remote_port}")
            return True
        time.sleep(1)
    return False


def setup_reverse_proxy(cfg: Config) -> subprocess.Popen:
    remote_port = random.randint(*cfg.remote_port_range)
    while is_port_listening(cfg.reverse_proxy_host, remote_port):
        remote_port = random.randint(*cfg.remote_port_range)
        print("trying a different port", remote_port)
    print(f"Using remote port {remote_port}")

    print(f"Setting up reverse proxy..")
    proc = subprocess.Popen(
        [
            "ssh",
            "-i",
            cfg.ssh_key_path,
            "-R",
            f"{remote_port}:localhost:{cfg.local_master_port}",
            cfg.ssh_user + "@" + cfg.reverse_proxy_host,
            "-N",
            "-o",
            "ServerAliveInterval=60",
            "-o",
            "ServerAliveCountMax=10",
        ]
    )
    if not wait_for_tunnel(cfg.reverse_proxy_host, remote_port):
        print("Failed to establish tunnel")
        proc.terminate()
        raise Exception("Failed to establish tunnel")
    print("Reverse proxy is up")
    return proc


def main():
    cfg = Config(
        reverse_proxy_host="aws-dev.prv",
        k8s_context="my-k8s-context",
        ssh_key_path=pathlib.Path("~/.ssh/id_ed25519").expanduser(),
        ssh_user="hmd",
    )
    proc = setup_reverse_proxy(cfg)
    print("ready")
    time.sleep(10)
    proc.terminate()


if __name__ == "__main__":
    main()
