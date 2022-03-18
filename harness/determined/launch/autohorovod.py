"""
autohorovod.py is the default launch layer for Determined.

It launches the entrypoint script under horovodrun when slots_per_trial>1, or as a regular
subprocess otherwise.
"""
import argparse
import logging
import os
import socket
import subprocess
import sys
import time

import determined as det
from determined import horovod
from determined.common import api
from determined.common.api import certs
from determined.constants import HOROVOD_SSH_PORT


def main(train_entrypoint: str) -> int:
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # Hack: get the resources id from the environment.
    resources_id = os.environ.get("DET_RESOURCES_ID")
    assert resources_id is not None, "Unable to run with DET_RESOURCES_ID unset"

    # Hack: read the full config.  The experiment config is not a stable API!
    experiment_config = info.trial._config

    debug = experiment_config.get("debug", False)

    # TODO: refactor websocket, data_layer, and profiling to to not use the cli_cert.
    cert = certs.default_load(info.master_url)
    certs.cli_cert = cert

    # The launch layer should provide the chief_ip to the training code, so that the training code
    # can function with a different launch layer in a different environment.  Inside Determined, the
    # easiest way to get the chief_ip is with container_addrs.
    chief_ip = info.container_addrs[0]

    # Chief IP is set as an environment variable to support nested launch layers
    os.environ["DET_CHIEF_IP"] = chief_ip

    if info.container_rank > 0:
        # Non-chief machines just run sshd.

        # Mark sshd containers as daemon resources that the master should kill when all non-daemon
        # contiainers (horovodrun, in this case) have exited.
        api.post(
            info.master_url,
            path=f"/api/v1/allocations/{info.allocation_id}/resources/{resources_id}/daemon",
            cert=cert,
        )

        # Wrap it in a pid_server to ensure that we can't hang if a worker fails.
        # TODO: After the upstream horovod bugfix (github.com/horovod/horovod/pull/3060) is in a
        # widely-used release of horovod, we should remove this pid_server layer, which just adds
        # unnecessary complexity.
        pid_server_cmd = [
            "python3",
            "-m",
            "determined.exec.pid_server",
            "--on-fail",
            "SIGTERM",
            "--on-exit",
            "WAIT",
            f"/tmp/pid_server-{info.allocation_id}",
            str(len(info.slot_ids)),
            "--",
        ]

        run_sshd_command = [
            "/usr/sbin/sshd",
            "-p",
            str(HOROVOD_SSH_PORT),
            "-f",
            "/run/determined/ssh/sshd_config",
            "-D",
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
                    sock.connect((peer, HOROVOD_SSH_PORT))
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
                            f"{peer}:{HOROVOD_SSH_PORT}"
                        )
                    time.sleep(0.1)

    # The chief has several layers of wrapper processes:
    # - a top-level pid_server, which causes the whole container to exit if any local worker dies.
    # - horovodrun, which launches $slots_per_trial copies of the following layers:
    #     - a pid_client process to contact the local pid_server
    #     - worker_process_wrapper, which redirects stdin/stdout to the local container
    #     - harness.py, which actually does the training for the worker
    #
    # It is a bug in horovod that causes us to have this pid_server/pid_client pair of layers.
    # We can remove these layers when the upstream fix has been around for long enough that we can
    # reasonably require user images to have patched horovod installations.

    pid_server_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_server",
        # Use SIGINT on horovod, because it freaks out with SIGTERM.
        "--on-fail",
        "SIGINT",
        "--on-exit",
        "WAIT",
        f"/tmp/pid_server-{info.allocation_id}",
        str(len(info.slot_ids)),
        "--",
    ]

    # TODO: remove this (very old) hack when we have a configurable launch layer.
    hvd_optional_args = experiment_config.get("data", {}).get("__det_dtrain_args", [])

    hvd_cmd = horovod.create_run_command(
        num_proc_per_machine=len(info.slot_ids),
        ip_addresses=info.container_addrs,
        inter_node_network_interface=info.trial._inter_node_network_interface,
        optimizations=experiment_config["optimizations"],
        debug=debug,
        optional_args=hvd_optional_args,
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

    logging.debug(f"chief worker calling horovodrun with args: {hvd_cmd[1:]} ...")

    os.environ["USE_HOROVOD"] = "True"

    return subprocess.Popen(
        pid_server_cmd + hvd_cmd + pid_client_cmd + log_redirect_cmd + harness_cmd
    ).wait()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("train_entrypoint", type=str)
    args = parser.parse_args()
    sys.exit(main(args.train_entrypoint))
