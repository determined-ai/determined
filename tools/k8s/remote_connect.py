#!/usr/bin/env python

"""
- set up reverse proxy
    - port collision
    - err handling
    - clean up
- check and set up gateway
    - record ip
- updated devcluster config

"""


import pathlib
import random
import socket
import subprocess
import tempfile
import time
from dataclasses import dataclass
from typing import Optional, Tuple

import kubernetes as k8s
import kubernetes.client.exceptions as client_exceptions
import yaml


@dataclass
class Config:
    # TODO: set up a shared ec2 instance for this usage.
    reverse_proxy_host: str
    k8s_context: str
    ssh_key_path: pathlib.Path
    determined_root: pathlib.Path
    ssh_user: str = "ubuntu"
    base_devcluster_path: pathlib.Path = pathlib.Path("tools/k8s/devcluster.yaml")
    local_master_port: int = 8080
    remote_port_range: Tuple[int, int] = (8000, 9000)


@dataclass
class Gateway:
    ip: Optional[str]
    name: str
    namespace: str

    def to_config(self) -> dict:
        return {
            "internal_task_gateway": {
                "gateway_name": self.name,
                "gateway_namespace": self.namespace,
                "gateway_ip": self.ip,
            }
        }


class DevClusterConf:
    def __init__(self, data: dict):
        self.original_data = data
        self.data = data

    @classmethod
    def from_yaml(cls, path: pathlib.Path) -> "DevClusterConf":
        with open(path) as f:
            data = yaml.safe_load(f)
        return cls(data)

    def save(self, path: pathlib.Path):
        with open(path, "w") as f:
            yaml.safe_dump(self.data, f)

    def get_stage(self, stage_name: str) -> dict:
        matching_stages = [stage for stage in self.data["stages"] if (stage_name in stage)]
        assert len(matching_stages) == 1
        return matching_stages[0]

    def set_stage(self, stage_name: str, new_data: dict):
        for stage in self.data["stages"]:
            if stage_name in stage:
                stage.update(new_data)
                break
        else:
            self.data["stages"].append(new_data)


def is_port_listening(host: str, port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        result = sock.connect_ex((host, port))
        return result == 0


def wait_for_tunnel(remote_host: str, remote_port: int, timeout: int = 30) -> bool:
    start_time = time.time()
    while time.time() - start_time < timeout:
        if is_port_listening(remote_host, remote_port):
            print(f"Tunnel is up on {remote_host}:{remote_port}")
            return True
        time.sleep(1)
    return False


def setup_reverse_proxy(cfg: Config) -> Tuple[subprocess.Popen, int]:
    """
    Set up a reverse proxy to open access to a local process, eg master.
    """
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
    return proc, remote_port


def get_gateway_info(cfg: Config) -> Optional[Gateway]:
    """
    Check if the cluster has a gateway set up.
    """
    k8s.config.load_kube_config(context=cfg.k8s_context)
    v1 = k8s.client.CoreV1Api()
    try:
        services = v1.list_service_for_all_namespaces(watch=False)
        for svc in services.items:
            if svc.status.load_balancer and svc.status.load_balancer.ingress:
                ingress = svc.status.load_balancer.ingress
                if ingress:
                    ip = ingress[0].ip or ingress[0].hostname
                    if ip:
                        print(
                            f"Found gateway service: {svc.metadata.name} in namespace: {svc.metadata.namespace}"
                        )
                        return Gateway(
                            ip=ip, name=svc.metadata.name, namespace=svc.metadata.namespace
                        )
    except client_exceptions.ApiException as e:
        print(f"Exception when calling CoreV1Api->list_service_for_all_namespaces: {e}")
        return None

    print("No gateway service found.")
    return None


def load_yaml_from_url(url: str) -> dict:
    import requests

    response = requests.get(url)
    response.raise_for_status()
    return yaml.safe_load(response.text)


def provision_gateway(cfg: Config) -> Gateway:
    """
    Provision a gateway service in the cluster.
    """
    k8s.config.load_kube_config(context=cfg.k8s_context)
    k8s_client = k8s.client.ApiClient()

    contour_provisioner_url = "https://raw.githubusercontent.com/projectcontour/contour/release-1.29/examples/render/contour-gateway-provisioner.yaml"
    contour_yaml_path = cfg.determined_root / "tools" / "k8s" / "contour.yaml"

    try:
        provisioner_yaml = load_yaml_from_url(contour_provisioner_url)
        k8s.utils.create_from_yaml(k8s_client, yaml_objects=provisioner_yaml)
        with open(contour_yaml_path) as f:
            contour_yaml = yaml.safe_load(f)
        k8s.utils.create_from_yaml(k8s_client, yaml_objects=contour_yaml)
        print("Contour Gateway Provisioner applied successfully.")
    except Exception as e:
        print(f"Failed to apply Contour Gateway: {e}")
        raise e

    print("Gateway provisioned successfully.")
    start_time = time.time()
    gateway = None
    while time.time() - start_time < 300:
        gateway = get_gateway_info(cfg)
        if gateway:
            break
        time.sleep(10)
    if not gateway:
        raise Exception("Failed to provision gateway")
    return gateway


def update_devcluster(cfg: Config, gateway: Gateway, remote_port: int) -> pathlib.Path:
    """
    Update the devcluster config to use the gateway.
    - create a backup before changing
    - add/update gateway config
    - add/update master address and port
    - save the updated formatted conf somewhere and share the path


    for each resource_manager with type: kubernetes as if it needs updating
    """
    devc = DevClusterConf.from_yaml(cfg.base_devcluster_path)
    master_stage = devc.get_stage("master")
    resource_manager = master_stage["resource_manager"]
    assert resource_manager["type"] == "kubernetes"
    resource_manager["determined_master_ip"] = cfg.reverse_proxy_host
    resource_manager["determined_master_port"] = remote_port
    assert gateway.ip is not None, "Gateway IP is not set"
    resource_manager.update(gateway.to_config())
    master_stage["resource_manager"] = resource_manager
    devc.set_stage("master", master_stage)
    temp_conf_path = pathlib.Path(tempfile.mkdtemp()) / "devcluster.yaml"
    devc.save(temp_conf_path)
    print(f"Updated devcluster config saved to {temp_conf_path}")
    return temp_conf_path


def main():
    cfg = Config(
        reverse_proxy_host="aws-dev.prv",
        k8s_context="gw",
        determined_root=pathlib.Path("$HOME/projects/da/determined"),
        ssh_key_path=pathlib.Path("~/.ssh/id_ed25519").expanduser(),
        ssh_user="hmd",
    )

    rev_proxy, proxy_port = setup_reverse_proxy(cfg)
    rev_proxy.terminate()
    gateway = get_gateway_info(cfg)
    if not gateway:
        gateway = provision_gateway(cfg)
    update_devcluster(cfg, gateway, proxy_port)


if __name__ == "__main__":
    main()
