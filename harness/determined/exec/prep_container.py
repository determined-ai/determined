import argparse
import base64
import io
import logging
import os
import socket
import tarfile
import uuid
from typing import List, Optional

import psutil

import determined as det
from determined import constants
from determined.common import util
from determined.common.api import bindings, certs, request
from determined.common.experimental import Session, get_max_retries_config


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
    sess: Session, allocation_id: str, resources_id: str
) -> "det.RendezvousInfo":
    resp = bindings.get_AllocationRendezvousInfo(
        sess, allocationId=allocation_id, resourcesId=resources_id
    )
    return det.RendezvousInfo(
        container_addrs=list(resp.rendezvousInfo.addresses), container_rank=resp.rendezvousInfo.rank
    )


def do_rendezvous_slurm(
    sess: Session, allocation_id: str, resources_id: str
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
    else:
        logging.warning(f"falling back to naive ip resolution after:\n\t{resolution_error}")
        rendezvous_ip = socket.gethostbyname(socket.gethostname())

    # Note, rendezvous must be sorted in rank order.
    resp = bindings.post_AllocationAllGather(
        sess,
        allocationId=allocation_id,
        body=bindings.v1AllocationAllGatherRequest(
            allocationId=allocation_id,
            requestUuid=str(uuid.uuid4()),
            numPeers=num_peers,
            data={
                "rank": rank,
                "rendezvous_ip": rendezvous_ip,
            },
        ),
    )
    addrs = [d["rendezvous_ip"] for d in sorted(resp.data, key=lambda d: int(d["rank"]))]
    return det.RendezvousInfo(container_addrs=addrs, container_rank=rank)


def rendezvous_ifaces() -> List[str]:
    # First case is a manual override. For maximum flexibility, this can be a comma-delimited list.
    rendezvous_iface = os.environ.get("DET_SLURM_RENDEZVOUS_IFACE")

    # If it doesn't work, fallback to just eth. Rendezvous over eth is fine since horovod will
    # still use DET_INTER_NODE_NETWORK_INTERFACE for everything important, and SSH over IB mostly
    # won't work. On systems where we need this, 'eth' will need to be the proper name.
    if not rendezvous_iface:
        rendezvous_iface = get_eth_interface_name()

    # If none of these resolved, we can fallback to something naive.
    if not rendezvous_iface:
        return []

    return rendezvous_iface.split(",")


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
            assert isinstance(info.address, str)
            return info.address

    raise ValueError(f"interface {interface} doesn't have an IPv4 address")


def get_eth_interface_name() -> Optional[str]:
    net_if_addrs = list(psutil.net_if_addrs())
    for interface in net_if_addrs:
        if interface.startswith("eth"):
            assert isinstance(interface, str)
            return interface
    return None


# The canonical definitions of these consts live in Go code.
RESOURCES_TYPE_K8S_POD = "k8s-pod"
RESOURCES_TYPE_DOCKER_CONTAINER = "docker-container"
RESOURCES_TYPE_SLURM_JOB = "slurm-job"


def do_rendezvous(sess: Session, allocation_id: str) -> None:
    r_id = os.environ.get("DET_RESOURCES_ID")
    assert r_id, "Unable to complete rendezvous info without DET_RESOURCES_ID"

    r_type = os.environ.get("DET_RESOURCES_TYPE")
    assert r_type, "Unable to complete rendezvous info without DET_RESOURCES_TYPE"

    rendezvous_info = None
    if r_type == RESOURCES_TYPE_DOCKER_CONTAINER or r_type == RESOURCES_TYPE_K8S_POD:
        rendezvous_info = do_rendezvous_rm_provided(sess, allocation_id, r_id)
    elif r_type == RESOURCES_TYPE_SLURM_JOB:
        rendezvous_info = do_rendezvous_slurm(sess, allocation_id, r_id)
    else:
        raise ValueError(f"unsupported resources type: {r_type}")

    rendezvous_info._to_file()


def proxy_ifaces() -> List[str]:
    # Manual override, for maximum flexibility.
    proxy_ifaces = os.environ.get("DET_SLURM_PROXY_IFACE")

    if not proxy_ifaces:
        return []

    return proxy_ifaces.split(",")


def set_proxy_address(sess: Session, allocation_id: str) -> None:
    proxy_ip, resolution_error = None, None
    for proxy_iface in proxy_ifaces():
        try:
            proxy_ip = get_ip_from_interface(proxy_iface)
            break
        except ValueError as e:
            resolution_error = e
    else:
        logging.warning(f"falling back to naive proxy ip resolution (error={resolution_error})")
        proxy_ip = socket.gethostbyname(socket.gethostname())

    # Right now this is just used in 'singularity-over-slurm' mode when singularity is using the
    # equivalent of 'host' networking in Docker. When supporting any sort of network virtualization
    # (https://sylabs.io/guides/3.0/user-guide/networking.html) this will need some revision.
    bindings.post_PostAllocationProxyAddress(
        sess,
        allocationId=allocation_id,
        body=bindings.v1PostAllocationProxyAddressRequest(
            proxyAddress=proxy_ip,
        ),
    )


def do_proxy(sess: Session, allocation_id: str) -> None:
    r_type = os.environ.get("DET_RESOURCES_TYPE")
    assert r_type, "Unable to complete rendezvous info without DET_RESOURCES_TYPE"

    if r_type == RESOURCES_TYPE_DOCKER_CONTAINER or r_type == RESOURCES_TYPE_K8S_POD:
        return
    elif r_type == RESOURCES_TYPE_SLURM_JOB:
        set_proxy_address(sess, allocation_id)
    else:
        raise ValueError(f"unsupported resources type: {r_type}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--trial", action="store_true")
    parser.add_argument("--resources", action="store_true")
    parser.add_argument("--rendezvous", action="store_true")
    parser.add_argument("--proxy", action="store_true")
    args = parser.parse_args()

    # Avoid reading det.get_cluster_info(), which might (wrongly) set a singleton to None.
    info = det.ClusterInfo._from_file()
    if info is None:
        info = det.ClusterInfo._from_env()
        info._to_file()

    cert = certs.default_load(info.master_url)
    sess = Session(
        info.master_url,
        util.get_container_user_name(),
        None,
        cert,
        max_retries=get_max_retries_config(),
    )

    if args.trial:
        trial_prep(info, cert)

    if args.resources:
        det.ResourcesInfo._by_inspection()._to_file()

    if args.rendezvous:
        do_rendezvous(sess, info.allocation_id)

    if args.proxy:
        do_proxy(sess, info.allocation_id)
