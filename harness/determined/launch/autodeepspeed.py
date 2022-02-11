"""
autodeepspeed.py is the launch layer for DeepSpeedTrial in Determined.

It launches the entrypoint script using DeepSpeed's launch process.
"""
import argparse
import logging
import os
import pathlib
import socket
import subprocess
import sys
import time
from typing import Dict, List, cast

from deepspeed.launcher.runner import DEEPSPEED_ENVIRONMENT_NAME, DLTS_HOSTFILE

import determined as det
from determined import constants
from determined.common import api
from determined.common.api import certs


def create_hostlist_file(num_proc_per_machine: int, ip_addresses: List[str]) -> str:
    trial_runner_hosts = ip_addresses.copy()
    if len(ip_addresses) == 1:
        trial_runner_hosts[0] = "localhost"

    hostfile_path = pathlib.Path(DLTS_HOSTFILE)
    os.makedirs(hostfile_path.parent, exist_ok=True)
    with open(hostfile_path, "w") as hostfile:
        lines = [f"{host} slots={num_proc_per_machine}\n" for host in trial_runner_hosts]
        hostfile.writelines(lines)
    return trial_runner_hosts[0]


def create_deepspeed_env_file() -> None:
    """Create an env var export file to pass Determined vars to the deepspeed launcher.

    By default, the deepspeed launcher only keeps env vars that start with one of the following
    ["NCCL", "PYTHON", "MV2", "UCX"].

    There are certain variables that we need to be set that we can pass to deepspeed using
    a custom env vars file.
    """
    INCLUDE = ["PATH", "USE_DEEPSPEED", "DET_CHIEF_IP"]
    with open(DEEPSPEED_ENVIRONMENT_NAME, "w") as f:
        environ = cast(Dict, os.environ.copy())
        for k, v in environ.items():
            if k in INCLUDE:
                f.write(f"{k}={v}\n")


def create_run_command(
    num_proc_per_machine: int,
    ip_addresses: List[str],
) -> List[str]:
    # Construct the deepspeed command.
    master_address = create_hostlist_file(num_proc_per_machine, ip_addresses)
    deepspeed_process_cmd = [
        "deepspeed",
        "-H",
        DLTS_HOSTFILE,
        "--master_addr",
        master_address,
        "--no_python",
        "--no_local_rank",
        "--",
    ]
    return deepspeed_process_cmd


def main(train_entrypoint: str) -> int:
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # Hack: get the container id from the environment.
    container_id = os.environ.get("DET_CONTAINER_ID")
    assert container_id is not None, "Unable to run with DET_CONTAINER_ID unset"

    # TODO: refactor websocket, data_layer, and profiling to to not use the cli_cert.
    cert = certs.default_load(info.master_url)
    certs.cli_cert = cert

    # The launch layer should provide the chief_ip to the training code, so that the training code
    # can function with a different launch layer in a different environment.  Inside Determined, the
    # easiest way to get the chief_ip is with container_addrs.
    chief_ip = info.container_addrs[0]

    # Chief IP is set as an environment variable to support nested launch layers
    os.environ["DET_CHIEF_IP"] = chief_ip

    # All ranks will need to run sshd.
    run_sshd_command = [
        "/usr/sbin/sshd",
        "-p",
        str(constants.DTRAIN_SSH_PORT),
        "-f",
        "/run/determined/ssh/sshd_config",
        "-D",
    ]

    if info.container_rank > 0:
        # Non-chief machines just run sshd.

        # Mark sshd containers as daemon containers that the master should kill when all non-daemon
        # containers (deepspeed launcher, in this case) have exited.
        api.post(
            info.master_url,
            path=f"/api/v1/allocations/{info.allocation_id}/containers/{container_id}/daemon",
            cert=cert,
        )

        # Wrap it in a pid_server to ensure that we can't hang if a worker fails.
        # This is useful for deepspeed which does not have good error handling for remote processes
        # spun up by pdsh.
        pid_server_cmd = [
            "python3",
            "-m",
            "determined.exec.pid_server",
            "--on-fail",
            "SIGTERM",
            "--on-exit",
            "SIGTERM",
            "--grace-period",
            "3",
            "--return-one-on-worker-error",
            f"/tmp/pid_server-{info.allocation_id}",
            str(len(info.slot_ids)),
            "--",
        ]

        logging.debug(
            f"Non-chief [{info.container_rank}] training process launch "
            f"command: {run_sshd_command}."
        )
        return subprocess.Popen(pid_server_cmd + run_sshd_command).wait()

    # Chief machine waits for every worker's sshd to be available.  All machines should be pretty
    # close to in-step by now because all machines just finished synchronizing rendezvous info.
    deadline = time.time() + 20
    for peer in info.container_addrs[1:]:
        while True:
            with socket.socket() as sock:
                sock.settimeout(1)
                try:
                    # Connect to a socket to ensure sshd is listening.
                    sock.connect((peer, constants.DTRAIN_SSH_PORT))
                    # The ssh protocol requires the server to serve an initial greeting.
                    # Receive part of that greeting to know that sshd is accepting/responding.
                    data = sock.recv(1)
                    if not data:
                        raise ValueError("no sshd greeting")
                    # This peer is ready.
                    break
                except Exception:
                    if time.time() > deadline:
                        raise ValueError(
                            f"Chief machine was unable to connect to sshd on peer machine at "
                            f"{peer}:{constants.DTRAIN_SSH_PORT}"
                        )
                    time.sleep(0.1)

    # The chief has several layers of wrapper processes:
    # - a top-level pid_server, which causes the whole container to exit if any local worker dies.
    # - deepspeed, which launches $slots_per_trial copies of the following layers:
    #     - a pid_client process to contact the local pid_server
    #     - worker_process_wrapper, which redirects stdin/stdout to the local container
    #     - harness.py, which actually does the training for the worker

    pid_server_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_server",
        "--on-fail",
        "SIGTERM",
        "--on-exit",
        "SIGTERM",
        "--grace-period",
        "5",
        "--return-one-on-worker-error",
        f"/tmp/pid_server-{info.allocation_id}",
        str(len(info.slot_ids)),
        "--",
    ]

    cmd = create_run_command(
        num_proc_per_machine=len(info.slot_ids), ip_addresses=info.container_addrs
    )

    pid_client_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_client",
        f"/tmp/pid_server-{info.allocation_id}",
        "--",
    ]

    log_redirect_cmd = [
        "python3",
        "-m",
        "determined.exec.worker_process_wrapper",
    ]

    harness_cmd = [
        "python3",
        "-m",
        "determined.exec.harness",
        "--train-entrypoint",
        train_entrypoint,
    ]

    logging.debug(f"chief worker calling deepspeed with args: {cmd[1:]} ...")

    full_cmd = pid_server_cmd + cmd + pid_client_cmd + log_redirect_cmd + harness_cmd

    os.environ["USE_DEEPSPEED"] = "1"
    if len(info.container_addrs) > 1:
        # Create the environment file that will be passed by deepspeed to individual ranks.
        create_deepspeed_env_file()
        # Set custom PDSH args:
        # * bypass strict host checking
        # * -p our custom port
        # * other args are default ssh args for pdsh
        os.environ["PDSH_SSH_ARGS"] = (
            "-o PasswordAuthentication=no -o StrictHostKeyChecking=no "
            f"-p {constants.DTRAIN_SSH_PORT} -2 -a -x %h"
        )
        subprocess.Popen(run_sshd_command)

    return subprocess.Popen(full_cmd).wait()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("train_entrypoint", type=str)
    args = parser.parse_args()
    sys.exit(main(args.train_entrypoint))
