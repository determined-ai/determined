#!/usr/bin/env python

# README: ./remote_connect.md


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
from dataclasses import MISSING, dataclass, fields
from typing import Any, Dict, List, Optional, Tuple

import kubernetes as k8s
import kubernetes.client.exceptions as client_exceptions
import yaml

DET_ROOT = pathlib.Path(__file__).resolve().parents[1]
CONFIG_DIR = (
    pathlib.Path(os.getenv("XDG_CONFIG_HOME", pathlib.Path.home() / ".config")) / "determined"
)
CONFIG_DIR.mkdir(parents=True, exist_ok=True)


def current_k8s_context() -> str:
    return subprocess.check_output(["kubectl", "config", "current-context"], text=True).strip()


def expand_env(value: Any, env: Dict[str, str]) -> Any:
    """
    Expand string and user variables in the config file.
    Borrowed from devcluster.
    """
    if isinstance(value, str):
        value = os.path.expanduser(value)
        return string.Template(value).safe_substitute(env)
    if isinstance(value, dict):
        return {k: expand_env(v, env) for k, v in value.items()}
    if isinstance(value, list):
        return [expand_env(item, env) for item in value]
    return value


@dataclass
class Config:
    """
    Configuration for the remote connection setup.
    """

    ssh_key_path: str
    reverse_proxy_host: str
    k8s_context: str = current_k8s_context()
    determined_root: str = str(DET_ROOT)
    base_devcluster_path: str = str(DET_ROOT / "tools" / "k8s" / "devcluster.yaml")
    ssh_user: str = "ubuntu"
    remote_port_range: Tuple[int, int] = (8000, 9000)

    @classmethod
    def _from_arg_dict(cls, data) -> "Config":
        data = expand_env(data, env=dict(os.environ))
        cfg = cls(**data)
        print(f"Config loaded: {cfg}")
        return cfg

    @classmethod
    def from_yaml(cls, path: pathlib.Path) -> "Config":
        with open(path, "r") as f:
            data = yaml.safe_load(f)
        return cls._from_arg_dict(data)

    @classmethod
    def from_args(cls) -> "Config":
        parser = argparse.ArgumentParser(description="Configure remote connection settings.")
        parser.add_argument(
            "--config-file",
            "-c",
            type=pathlib.Path,
            help="Path to a YAML config file. CLI args will override values in the file.",
        )

        for field in fields(cls):
            field_name = field.name.replace("_", "-")
            field_type = field.type
            default_value = field.default if field.default != MISSING else None
            help_text = f"(default: {default_value})" if default_value is not None else None
            parser.add_argument(
                f"--{field_name}", type=field_type, default=default_value, help=help_text
            )

        args = parser.parse_args()
        arg_dict = vars(args)

        config_data = {}
        if arg_dict.get("config_file"):
            config_path = arg_dict.pop("config_file")
            if config_path.exists():
                config_data = yaml.safe_load(config_path.read_text())

        for key, value in arg_dict.items():
            if value is not None:
                config_data[key] = value

        for field in fields(cls):
            if isinstance(config_data.get(field.name), str) and field.type == Tuple[int, int]:
                config_data[field.name] = tuple(map(int, config_data[field.name].split(",")))

        return cls._from_arg_dict(config_data)


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
    def __init__(self, data: dict, path: Optional[pathlib.Path] = None):
        self.original_data = copy.deepcopy(data)
        self.original_path = path
        self.data = data

    @classmethod
    def from_yaml(cls, path: pathlib.Path) -> "DevClusterConf":
        with open(path) as f:
            data = yaml.safe_load(f)
        return cls(data=data, path=path)

    def save(self, path: Optional[pathlib.Path] = None) -> pathlib.Path:
        if path is None:
            path = pathlib.Path(tempfile.mkdtemp()) / "devcluster.yaml"
        with open(path, "w") as f:
            yaml.dump(self.data, f, default_flow_style=False, sort_keys=False, indent=2)
        return path

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

    def master_port(self) -> int:
        master_config = self.get_stage("master")["config_file"]
        return master_config.get("port", 8080)

    def run(self) -> Optional[subprocess.Popen]:
        run_path = self.save(pathlib.Path(tempfile.mkdtemp()) / "devcluster.yaml")
        devc_run_cmd = ["devcluster", "-c", run_path]
        if not shutil.which("devcluster"):
            print("`devcluster` not found in PATH.")
            print(f"run using {' '.join(devc_run_cmd)}")
            return None
        print("Running devcluster...")
        return subprocess.Popen(devc_run_cmd)


def is_port_listening(host: str, port: int, timeout: float = 1.0) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(timeout)
        try:
            result = sock.connect_ex((host, port))
            return result == 0
        except socket.timeout:
            return False


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
    local_master_port = DevClusterConf.from_yaml(
        pathlib.Path(cfg.base_devcluster_path)
    ).master_port()
    while is_port_listening(cfg.reverse_proxy_host, remote_port):
        remote_port = random.randint(*cfg.remote_port_range)
        print("trying a different port", remote_port)
    print(
        f"Setting up reverse proxy on {cfg.reverse_proxy_host}:{remote_port}"
        + f" to localhost:{local_master_port}"
    )
    rev_proxy_cmd = [
        "ssh",
        "-i",
        cfg.ssh_key_path,
        "-R",
        f"{remote_port}:localhost:{local_master_port}",
        cfg.ssh_user + "@" + cfg.reverse_proxy_host,
        "-N",
        "-o",
        "ServerAliveInterval=60",
        "-o",
        "ServerAliveCountMax=10",
        "-v",
    ]
    proc = subprocess.Popen(rev_proxy_cmd, stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    ssh_stderr: List[str] = []
    success_msg = "forward success"
    for line in iter(proc.stderr.readline, ""):
        ssh_stderr.append(line.decode("utf-8"))
        if success_msg in line.decode("utf-8"):
            break
        exist_code = proc.poll()
        if exist_code is not None:
            print("".join(ssh_stderr), file=sys.stderr)
            raise Exception("Failed to establish tunnel")
    if not wait_for_tunnel(cfg.reverse_proxy_host, remote_port, timeout=1000):
        proc.terminate()
        raise Exception("Failed to establish tunnel")
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
    contour_provisioner_url = (
        "https://raw.githubusercontent.com/projectcontour/contour"
        + "/release-1.29/examples/render/contour-gateway-provisioner.yaml"
    )
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
    resource_manager["determined_master_host"] = cfg.reverse_proxy_host
    resource_manager["determined_master_port"] = remote_port
    assert gateway.ip is not None, "Gateway IP is not set"
    resource_manager.update(gateway.to_config())
    master_stage["config_file"]["resource_manager"] = resource_manager
    devc.set_stage("master", master_stage)
    temp_conf_path = pathlib.Path(tempfile.mkdtemp()) / "devcluster.yaml"
    devc.save(temp_conf_path)
    return temp_conf_path


def workflow_1(cfg: Config):
    master_port = DevClusterConf.from_yaml(pathlib.Path(cfg.base_devcluster_path)).master_port()
    if is_port_listening("localhost", master_port):
        print(f"Another process is listening on localhost:{master_port}.")
        print("Please stop the existing process before running this script.")
        sys.exit(1)

    rev_proxy, proxy_port = setup_reverse_proxy(cfg)
    try:
        gateway = get_gateway_info(cfg)
        if not gateway:
            input(f"Press Enter to provision a gateway service in the cluster: {cfg.k8s_context}.")
            gateway = provision_gateway(cfg)
        config_path = update_devcluster(cfg, gateway, proxy_port)
        print("Workflow 1 ready.")
        proc = DevClusterConf.from_yaml(config_path).run()
        if proc is None:
            input("Press Enter to terminate and cleanup once done.")
        else:
            proc.wait()
    finally:
        rev_proxy.terminate()


def main():
    cfg = Config.from_args()
    workflow_1(cfg)


if __name__ == "__main__":
    main()
