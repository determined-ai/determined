import argparse
import base64
import io
import os
import socket
import tarfile
import uuid
from typing import List, Optional, cast

import psutil

import determined as det
from determined import constants
from determined.common.api import certs, request


def trial_prep(info: det.ClusterInfo, cert: certs.Cert) -> None:
    trial_info = det.TrialInfo._from_env()
    trial_info._to_file()

    path = f"api/v1/experiments/{trial_info.experiment_id}/model_def"
    resp = None
    try:
        resp = request.get(info.master_url, path=path, cert=cert)
        resp.raise_for_status()
    except Exception:
        # Since this is the very first api call in the entrypoint script, and the call is made
        # before you can debug with a startup hook, we offer an overly-detailed explanation to help
        # sysadmins debug their cluster.
        resp_content = str(resp and resp.content)
        noverify = info.master_cert_file == "noverify"
        cert_content = None if noverify else info.master_cert_file
        if cert_content is not None:
            with open(cert_content) as f:
                cert_content = f.read()
        print(
            "Failed to download model definition from master.  This may be due to an address\n"
            "resolution problem, a certificate problem, a firewall problem, or some other\n"
            "networking error.\n"
            "Debug information:\n"
            f"    master_url: {info.master_url}\n"
            f"    endpoint: {path}\n"
            f"    tls_verify_name: {info.master_cert_name}\n"
            f"    tls_noverify: {noverify}\n"
            f"    tls_cert: {cert_content}\n"
            f"    response code: {resp and resp.status_code}\n"
            f"    response content: {resp_content}\n"
        )
        raise

    tgz = base64.b64decode(resp.json()["b64Tgz"])

    with tarfile.open(fileobj=io.BytesIO(tgz), mode="r:gz") as model_def:
        # Ensure all members of the tarball resolve to subdirectories.
        for path in model_def.getnames():
            if os.path.relpath(path).startswith("../"):
                raise ValueError(f"'{path}' in tarball would expand to a parent directory")
        model_def.extractall(path=constants.MANAGED_TRAINING_MODEL_COPY)
        model_def.extractall(path=".")


def do_rendezvous_rm_provided(
    info: det.ClusterInfo, cert: certs.Cert, resources_id: str
) -> "det.RendezvousInfo":
    r = request.get(
        info.master_url,
        path=f"/api/v1/allocations/{info.allocation_id}/resources/{resources_id}/rendezvous",
        cert=cert,
    )
    jri = r.json()["rendezvousInfo"]
    addrs, rank = jri["addresses"], jri["rank"]
    return det.RendezvousInfo(container_addrs=addrs, container_rank=rank)


def do_rendezvous_slurm(
    info: det.ClusterInfo, cert: certs.Cert, resources_id: str
) -> "det.RendezvousInfo":
    rank_str = os.environ.get("SLURM_PROCID")
    assert rank_str, "Unable to complete rendezvous without SLURM_PROCID"
    rank = int(rank_str)

    num_peers_str = os.environ.get("SLURM_NPROCS")
    assert num_peers_str, "Unable to complete rendezvous without SLURM_NPROCS"
    num_peers = int(num_peers_str)

    rendezvous_ip, resolution_error = None, None
    for rendezvous_iface in rendezvous_ifaces():
        try:
            rendezvous_ip = get_ip_from_interface(rendezvous_iface)
            break
        except ValueError as e:
            resolution_error = e
    if not rendezvous_ip:
        raise resolution_error or ValueError("unable to resolve rendezvous ip")

    r = request.post(
        info.master_url,
        path=f"/api/v1/allocations/{info.allocation_id}/all_gather",
        cert=cert,
        json={
            "request_uuid": uuid.uuid4(),
            "num_peers": num_peers,
            "data": {
                "rendezvous_ip": rendezvous_ip,
            },
        },
    )
    addrs = [d["rendezvous_ip"] for d in r.json()["data"]]
    return det.RendezvousInfo(container_addrs=addrs, container_rank=rank)


def rendezvous_ifaces() -> List[str]:
    # First case is a manual override. For maximum flexibility, this can be a comma-delimited list.
    rendezvous_iface = os.environ.get("DET_SLURM_RENDEZVOUS_IFACE")

    # If it doesn't work, fallback to just eth. Rendezvous over eth is fine since horovod will
    # still use DET_INTER_NODE_NETWORK_INTERFACE for everything important, and SSH over IB mostly
    # won't work.
    if not rendezvous_iface:
        rendezvous_iface = get_eth_interface_name()

    # On systems where there is no eth, DET_INTER_NODE_NETWORK_INTERFACE should work, though.
    if not rendezvous_iface:
        rendezvous_iface = os.environ.get("DET_INTER_NODE_NETWORK_INTERFACE")

    # If none of these resolved, we're out of luck.
    if not rendezvous_iface:
        raise ValueError("unable to resolve rendezvous iface")

    return rendezvous_iface.split(",")


def get_eth_interface_name() -> Optional[str]:
    net_if_addrs = list(psutil.net_if_addrs())
    for interface in net_if_addrs:
        if interface.startswith("eth"):
            return cast(str, interface)
    return None


def get_ip_from_interface(interface: str) -> str:
    net_if_addrs = psutil.net_if_addrs()

    if interface not in net_if_addrs:
        available = list(net_if_addrs.keys())
        raise ValueError(
            f"{interface} is not a valid network interface. "
            f"Valid network interfaces are: {available}"
        )

    for info in net_if_addrs[interface]:
        if info.family == socket.AF_INET:
            return cast(str, info.address)

    raise ValueError(f"interface {interface} doesn't have an IPv4 address")


# The canonical definitions of these consts live in Go code.
RESOURCES_TYPE_K8S_POD = "k8s-pod"
RESOURCES_TYPE_DOCKER_CONTAINER = "docker-container"
RESOURCES_TYPE_SLURM_JOB = "slurm-job"


def do_rendezvous(info: det.ClusterInfo, cert: certs.Cert) -> None:
    # Even though resources_id and resources type is not part of the ClusterInfo API, we still
    # depend on them in all current Determined backends.
    r_id = os.environ.get("DET_RESOURCES_ID")
    assert r_id, "Unable to complete rendezvous info without DET_RESOURCES_ID"

    r_type = os.environ.get("DET_RESOURCES_TYPE")
    assert r_type, "Unable to complete rendezvous info without DET_RESOURCES_TYPE"

    rendezvous_info = None
    if r_type == RESOURCES_TYPE_DOCKER_CONTAINER or r_type == RESOURCES_TYPE_K8S_POD:
        rendezvous_info = do_rendezvous_rm_provided(info, cert, r_id)
    elif r_type == RESOURCES_TYPE_SLURM_JOB:
        rendezvous_info = do_rendezvous_slurm(info, cert, r_id)
    else:
        raise ValueError(f"unsupported resources type: {r_type}")

    rendezvous_info._to_file()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--trial", action="store_true")
    parser.add_argument("--resources", action="store_true")
    parser.add_argument("--rendezvous", action="store_true")
    args = parser.parse_args()

    # Avoid reading det.get_cluster_info(), which might (wrongly) set a singleton to None.
    info = det.ClusterInfo._from_file()
    if info is None:
        info = det.ClusterInfo._from_env()
        info._to_file()

    cert = certs.default_load(info.master_url)

    if args.trial:
        trial_prep(info, cert)

    if args.resources:
        det.ResourcesInfo._by_inspection()._to_file()

    if args.rendezvous:
        do_rendezvous(info, cert)
