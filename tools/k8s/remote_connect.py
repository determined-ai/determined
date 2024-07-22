#!/usr/bin/env python

"""
- set up reverse proxy
    - port collision
    - err handling
    - clean up
- check and set up gateway
    - record ip
- updated devcluster config

TODO:
- what if there are multiple gateways on the cluster?
- gateway cleanup if we provision it
- run devcluster here for you?
- or migrate some of this into devcluster stages?
"""


import argparse
import copy
import os
import pathlib
import random
import shutil
import socket
import string
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from typing import Any, Dict, List, Optional, Tuple

import kubernetes as k8s
import kubernetes.client.exceptions as client_exceptions
import yaml

DET_ROOT = pathlib.Path("~/projects/da/determined").expanduser()


def expand_env(value: Any, env: Dict[str, str]) -> Any:
    """
    Expand string variables in the config file.
    Borrowed from devcluster.
    """
    if isinstance(value, str):
        return string.Template(value).safe_substitute(env)
    if isinstance(value, dict):
        return {k: expand_env(v, env) for k, v in value.items()}
    if isinstance(value, list):
        return [expand_env(l, env) for l in value]
    return value


@dataclass
class Config:
    # TODO: set up a shared ec2 instance for this usage.
    reverse_proxy_host: str
    k8s_context: str
    ssh_key_path: str
    determined_root: str
    base_devcluster_path: str
    ssh_user: str = "ubuntu"
    local_master_port: int = 8080
    remote_port_range: Tuple[int, int] = (8000, 9000)

    @classmethod
    def from_yaml(cls, path: pathlib.Path) -> "Config":
        with open(path, "r") as f:
            data = yaml.safe_load(f)
            data = expand_env(data, env=dict(os.environ))
        cfg = cls(**data)
        print(f"Config loaded: {cfg}")
        return cfg


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
            yaml.dump(self.data, f, default_flow_style=False, sort_keys=False, indent=2)

    def get_stage(self, stage_name: str) -> dict:
        matching_stages = [stage for stage in self.data["stages"] if (stage_name in stage)]
        assert len(matching_stages) == 1
        return copy.deepcopy(matching_stages[0][stage_name])

    def set_stage(self, stage_name: str, new_data: dict):
        for stage in self.data["stages"]:
            if stage_name in stage:
                stage[stage_name] = new_data
                return
        raise NotImplementedError(f"Stage {stage_name} not found in devcluster config.")


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
    print(
        f"Setting up reverse proxy on {cfg.reverse_proxy_host}:{remote_port}"
        + f" to localhost:{cfg.local_master_port}"
    )
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
        ],
        stderr=subprocess.PIPE,
    )
    if not wait_for_tunnel(cfg.reverse_proxy_host, remote_port):
        print("Failed to establish tunnel")
        proc.terminate()
        raise Exception("Failed to establish tunnel")
    print("Reverse proxy is up.")
    return proc, remote_port


def get_gateway_info(cfg: Config) -> Optional[Gateway]:
    """
    Check if the cluster has a gateway set up.
    """
    k8s.config.load_kube_config(context=cfg.k8s_context)
    custom_api = k8s.client.CustomObjectsApi()

    group = "gateway.networking.k8s.io"
    version = "v1"  # or the appropriate version for your setup
    plural = "gateways"

    try:
        gateways = custom_api.list_cluster_custom_object(
            group=group, version=version, plural=plural
        )
        for gateway in gateways.get("items", []):
            if "status" in gateway and "addresses" in gateway["status"]:
                addresses = gateway["status"]["addresses"]
                if addresses:
                    ip = addresses[0].get("value")
                    if ip:
                        gw = Gateway(
                            ip=ip,
                            name=gateway["metadata"]["name"],
                            namespace=gateway["metadata"]["namespace"],
                        )
                        print(f"Found active gateway: {gw}")
                        return gw
    except client_exceptions.ApiException as e:
        print(f"Exception when calling CoreV1Api->list_service_for_all_namespaces: {e}")
        return None
    print("No gateway service found.")
    return None


def provision_gateway(cfg: Config) -> Gateway:
    """
    Provision a gateway service in the cluster.
    """
    contour_provisioner_url = "https://raw.githubusercontent.com/projectcontour/contour/release-1.29/examples/render/contour-gateway-provisioner.yaml"
    contour_yaml_path = pathlib.Path(cfg.determined_root) / "tools" / "k8s" / "contour.yaml"

    kctl = ["kubectl", "--context", cfg.k8s_context]
    try:
        subprocess.run(kctl + ["apply", "-f", contour_provisioner_url], check=True)
        subprocess.run(kctl + ["apply", "-f", contour_yaml_path], check=True)
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


def warn(msg: str, *args, **kwargs):
    print(f"Warning: {msg}", *args, **kwargs, file=sys.stderr)


def update_devcluster(cfg: Config, gateway: Gateway, remote_port: int) -> pathlib.Path:
    """
    Update the devcluster config to use the gateway and a reverse proxy.
    """
    devc = DevClusterConf.from_yaml(pathlib.Path(cfg.base_devcluster_path))
    master_stage = devc.get_stage("master")
    resource_manager = master_stage["config_file"]["resource_manager"]
    if "additional_resource_managers" in master_stage:
        warn(
            "setting up additional resource managers are not supported yet."
            + "These will be ignored."
        )
    assert resource_manager["type"] == "kubernetes"
    resource_manager["determined_master_ip"] = cfg.reverse_proxy_host
    resource_manager["determined_master_port"] = remote_port
    assert gateway.ip is not None, "Gateway IP is not set"
    resource_manager.update(gateway.to_config())
    master_stage["config_file"]["resource_manager"] = resource_manager
    devc.set_stage("master", master_stage)
    temp_conf_path = pathlib.Path(tempfile.mkdtemp()) / "devcluster.yaml"
    devc.save(temp_conf_path)
    return temp_conf_path


def workflow_1(cfg: Config):
    rev_proxy, proxy_port = setup_reverse_proxy(cfg)
    try:
        gateway = get_gateway_info(cfg)
        if not gateway:
            gateway = provision_gateway(cfg)
        config_path = update_devcluster(cfg, gateway, proxy_port)
        print("Workflow 1 ready.")
        devc_run_cmd = ["devcluster", "-c", config_path]
        if shutil.which("devcluster"):
            print("Running devcluster...")
            subprocess.run(devc_run_cmd)
        else:
            print(f"devcluster -c {config_path}")
            input("Press Enter to terminate and cleanup once done.")
    finally:
        rev_proxy.terminate()


def main():
    parser = argparse.ArgumentParser(description="Set the configuration file path.")
    parser.add_argument(
        "--config",
        type=pathlib.Path,
        default=DET_ROOT / "tools" / "k8s" / "remote_connect.yaml",
        help="Path to the configuration file.",
    )
    args = parser.parse_args()
    cfg = Config.from_yaml(args.config)
    workflow_1(cfg)


if __name__ == "__main__":
    main()
