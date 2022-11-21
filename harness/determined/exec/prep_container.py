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
import urllib3

import determined as det
from determined import constants, gpu
from determined.common import api, util
from determined.common.api import bindings, certs
from determined.util import force_create_symlink


def trial_prep(sess: api.Session, info: det.ClusterInfo) -> None:
    trial_info = det.TrialInfo._from_env()
    trial_info._to_file()

    model_def_resp = None
    try:
        model_def_resp = bindings.get_GetModelDef(sess, experimentId=trial_info.experiment_id)
    except Exception as e:
        # Since this is the very first api call in the entrypoint script, and the call is made
        # before you can debug with a startup hook, we offer an overly-detailed explanation to help
        # sysadmins debug their cluster.
        resp_content = str(e)
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
            f"    endpoint: api/v1/experiments/{trial_info.experiment_id}/model_def\n"
            f"    tls_verify_name: {info.master_cert_name}\n"
            f"    tls_noverify: {noverify}\n"
            f"    tls_cert: {cert_content}\n"
            f"    response content: {resp_content}\n"
        )
        raise

    tgz = base64.b64decode(model_def_resp.to_json()["b64Tgz"])

    with tarfile.open(fileobj=io.BytesIO(tgz), mode="r:gz") as model_def:
        # Ensure all members of the tarball resolve to subdirectories.
        for path in model_def.getnames():
            if os.path.relpath(path).startswith("../"):
                raise ValueError(f"'{path}' in tarball would expand to a parent directory")
        model_def.extractall(path=constants.MANAGED_TRAINING_MODEL_COPY)
        model_def.extractall(path=".")

    # pre-0.18.3 code wrote tensorboard stuff under /tmp/tensorboard
    force_create_symlink(f"/tmp/tensorboard-{info.allocation_id}-0", "/tmp/tensorboard")


def do_rendezvous_rm_provided(
    sess: api.Session, allocation_id: str, resources_id: str
) -> "det.RendezvousInfo":
    resp = bindings.get_AllocationRendezvousInfo(
        sess, allocationId=allocation_id, resourcesId=resources_id
    )
    return det.RendezvousInfo(
        container_addrs=list(resp.rendezvousInfo.addresses), container_rank=resp.rendezvousInfo.rank
    )


def do_rendezvous_slurm(
    sess: api.Session, allocation_id: str, resources_id: str
) -> "det.RendezvousInfo":
    rank_str = os.environ.get("SLURM_PROCID")
    assert rank_str, "Unable to complete rendezvous without SLURM_PROCID"
    rank = int(rank_str)

    num_peers_str = os.environ.get("SLURM_NPROCS")
    assert num_peers_str, "Unable to complete rendezvous without SLURM_NPROCS"
    num_peers = int(num_peers_str)

    rendezvous_ip = socket.gethostbyname(socket.gethostname())
    for rendezvous_iface in rendezvous_ifaces():
        try:
            rendezvous_ip = get_ip_from_interface(rendezvous_iface)
            break
        except ValueError as e:
            logging.warning(f"Unable to resolve ip for {rendezvous_iface}: {str(e)}")

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


# On HPC, the "launcher" tells the Determined Master that the job is "Running"
# as soon as the workload manager (e.g., Slurm, PBS, etc) starts running the job.
# However, if the container is not already cached on the compute node, it will
# first need to be pulled down from the Internet by Singularity or Podman. From
# the Determined Master's point of view, the experiment is not running until all
# the containers are pulled and each container's entry point is executed. The
# Determined Master will show a state of "Pulling" until it receives notification
# from all the containers that they are running.  Therefore, notify the
# Determined Master that the container is running, so that once all the
# containers that are part of the job report they are running, the Determined
# Master can change the state from "Pulling" to "Running".
def send_container_running_notification(sess: api.Session, allocation_id: str) -> None:
    # Tells the Determined Master this container's unique ID.
    rank_str = os.environ.get("SLURM_PROCID")
    assert rank_str, "Unable to send container running notification without SLURM_PROCID"
    rank = int(rank_str)

    # Tells the Determined Master how many containers are part of the job so
    # that it knows how many unique IDs it should expect notifications from
    # in order to change the experiment's state from "Pulling" to "Running".
    num_peers_str = os.environ.get("SLURM_NPROCS")
    assert num_peers_str, "Unable to send container running notification without SLURM_NPROCS"
    num_peers = int(num_peers_str)

    bindings.post_NotifyContainerRunning(
        sess,
        allocationId=allocation_id,
        body=bindings.v1NotifyContainerRunningRequest(
            allocationId=allocation_id,
            requestUuid=str(uuid.uuid4()),
            numPeers=num_peers,
            rank=rank,
            nodeName=socket.gethostname(),
            data={},
        ),
    )


def rendezvous_ifaces() -> List[str]:
    # First case is a manual override. For maximum flexibility, this can be a comma-delimited list.
    rendezvous_iface = os.environ.get("DET_SLURM_RENDEZVOUS_IFACE")

    # If it doesn't work, fallback to just eth. Rendezvous over eth is fine since horovod will
    # still use DET_INTER_NODE_NETWORK_INTERFACE for everything important, and SSH over IB mostly
    # won't work. On systems where we need this, 'eth' will need to be the proper name.
    if not rendezvous_iface:
        rendezvous_iface = get_eth_interface_name()

    # If none of these resolved, we can fall back to something naive.
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


def do_rendezvous(sess: api.Session, allocation_id: str) -> None:
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


def set_proxy_address(sess: api.Session, allocation_id: str) -> None:
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


def do_proxy(sess: api.Session, allocation_id: str) -> None:
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
    parser.add_argument("--notify_container_running", action="store_true")
    args = parser.parse_args()

    # Avoid reading det.get_cluster_info(), which might (wrongly) set a singleton to None.
    info = det.ClusterInfo._from_file()
    if info is None:
        info = det.ClusterInfo._from_env()
        info._to_file()

    try:
        # See the ClusterInfo.trial property for explanation
        debug = info.trial._debug
    except (AssertionError, RuntimeError):
        debug = False

    logging.basicConfig(
        level=logging.DEBUG if debug else logging.INFO,
        format=det.LOG_FORMAT,
    )
    logging.debug("running prep_container")

    cert = certs.default_load(info.master_url)
    sess = api.Session(
        info.master_url,
        util.get_det_username_from_env(),
        None,
        cert,
        max_retries=urllib3.util.retry.Retry(
            total=6,  # With backoff retries for 64 seconds
            backoff_factor=0.5,
        ),
    )

    # Notify the Determined Master that the container is running.
    # This should only be used on HPC clusters.
    if args.notify_container_running:
        send_container_running_notification(sess, info.allocation_id)

    if args.trial:
        trial_prep(sess, info)

    if args.resources:
        det.ResourcesInfo._by_inspection()._to_file()
        for process in gpu.get_gpu_processes():
            logging.warning(
                f"process {process.process_name} "
                f"with pid {process.pid} "
                f"is using {process.used_memory} memory "
                f"of the GPU with uuid {process.gpu_uuid}. "
                "This process is not related to Determined tasks but may interfere with tasks' "
                "ability to use the full GPU."
            )

    if args.rendezvous:
        do_rendezvous(sess, info.allocation_id)

    if args.proxy:
        do_proxy(sess, info.allocation_id)
