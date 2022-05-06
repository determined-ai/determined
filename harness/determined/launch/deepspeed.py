"""
deepspeed.py is the launch layer for DeepSpeedTrial in Determined.

It launches the entrypoint script using DeepSpeed's launch process.
"""
import argparse
import logging
import os
import pathlib
import subprocess
import sys
import time
from typing import List

from deepspeed.launcher.runner import DEEPSPEED_ENVIRONMENT_NAME

import determined as det
from determined import constants, util
from determined.common import api
from determined.common.api import certs

hostfile_path = "/tmp/hostfile.txt"


def create_hostlist_file(
    hostfile_path: pathlib.Path, num_proc_per_machine: int, ip_addresses: List[str]
) -> str:
    trial_runner_hosts = ip_addresses.copy()
    # In the single node case, deepspeed doesn't use pdsh so we don't need to launch sshd.
    # Instead, the deepspeed launcher will use localhost as the chief worker ip.
    if len(ip_addresses) == 1:
        trial_runner_hosts[0] = "localhost"

    os.makedirs(hostfile_path.parent, exist_ok=True)
    with open(hostfile_path, "w") as hostfile:
        lines = [f"{host} slots={num_proc_per_machine}\n" for host in trial_runner_hosts]
        hostfile.writelines(lines)
    return trial_runner_hosts[0]


def create_pid_server_cmd(allocation_id: str, num_workers: int) -> List[str]:
    return [
        "python3",
        "-m",
        "determined.exec.pid_server",
        "--on-fail",
        "SIGTERM",
        "--on-exit",
        "SIGTERM",
        "--grace-period",
        "5",
        f"/tmp/pid_server-{allocation_id}",
        str(num_workers),
        "--",
    ]


def create_pid_client_cmd(allocation_id: str) -> List[str]:
    return [
        "python3",
        "-m",
        "determined.exec.pid_client",
        f"/tmp/pid_server-{allocation_id}",
        "--",
    ]


def create_log_redirect_cmd() -> List[str]:
    return [
        "python3",
        "-m",
        "determined.launch.wrap_rank",
        "RANK",
        "--",
    ]


def create_sshd_cmd() -> List[str]:
    return [
        "/usr/sbin/sshd",
        "-p",
        str(constants.DTRAIN_SSH_PORT),
        "-f",
        "/run/determined/ssh/sshd_config",
        "-D",
    ]


def create_deepspeed_env_file() -> None:
    """Create an env var export file to pass Determined vars to the deepspeed launcher.

    By default, the deepspeed launcher only keeps env vars that start with one of the following
    ["NCCL", "PYTHON", "MV2", "UCX"].

    There are certain variables that we need to be set that we can pass to deepspeed using
    a custom env vars file.
    """
    INCLUDE = ["PATH", "USE_DEEPSPEED", "DET_CHIEF_IP", "DET_MANUAL_INIT_DISTRIBUTED"]
    with open(DEEPSPEED_ENVIRONMENT_NAME, "w") as f:
        environ = os.environ.copy()
        for k, v in environ.items():
            if k in INCLUDE:
                f.write(f"{k}={v}\n")


def create_run_command(master_address: str, hostfile_path: str) -> List[str]:
    # Construct the deepspeed command.
    deepspeed_process_cmd = [
        "deepspeed",
        "-H",
        hostfile_path,
        "--master_addr",
        master_address,
        "--no_python",
        "--no_local_rank",
        "--",
    ]
    return deepspeed_process_cmd


def main(script: List[str]) -> int:
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # Hack: get the resources id from the environment.
    resources_id = os.environ.get("DET_RESOURCES_ID")
    assert resources_id is not None, "Unable to run with DET_RESOURCES_ID unset"

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
    run_sshd_command = create_sshd_cmd()

    if info.container_rank > 0:
        # Non-chief machines just run sshd.

        # Mark sshd containers as daemon containers that the master should kill when all non-daemon
        # containers (deepspeed launcher, in this case) have exited.
        api.post(
            info.master_url,
            path=f"/api/v1/allocations/{info.allocation_id}/resources/{resources_id}/daemon",
            cert=cert,
        )

        # Wrap it in a pid_server to ensure that we can't hang if a worker fails.
        # This is useful for deepspeed which does not have good error handling for remote processes
        # spun up by pdsh.
        pid_server_cmd = create_pid_server_cmd(info.allocation_id, len(info.slot_ids))

        logging.debug(
            f"Non-chief [{info.container_rank}] training process launch "
            f"command: {run_sshd_command}."
        )
        return subprocess.Popen(pid_server_cmd + run_sshd_command).wait()

    # We always need to set this variable to initialize the context correctly, even in the single
    # slot case.
    os.environ["USE_DEEPSPEED"] = "1"

    # The chief has several layers of wrapper processes:
    # - a top-level pid_server, which causes the whole container to exit if any local worker dies.
    # - deepspeed, which launches $slots_per_trial copies of the following layers:
    #     - a pid_client process to contact the local pid_server
    #     - wrap_rank, which redirects stdin/stdout to the local container
    #     - harness.py, which actually does the training for the worker

    pid_server_cmd = create_pid_server_cmd(info.allocation_id, len(info.slot_ids))

    master_address = create_hostlist_file(
        hostfile_path=pathlib.Path(hostfile_path),
        num_proc_per_machine=len(info.slot_ids),
        ip_addresses=info.container_addrs,
    )
    cmd = create_run_command(master_address, hostfile_path)

    pid_client_cmd = create_pid_client_cmd(info.allocation_id)

    log_redirect_cmd = create_log_redirect_cmd()

    harness_cmd = script

    logging.debug(f"chief worker calling deepspeed with args: {cmd[1:]} ...")

    full_cmd = pid_server_cmd + cmd + pid_client_cmd + log_redirect_cmd + harness_cmd

    multi_machine = len(info.container_addrs) > 1
    if not multi_machine:
        return subprocess.Popen(full_cmd).wait()

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

    # Chief worker also needs to run sshd when using pdsh and multi-machine training.
    sshd_process = subprocess.Popen(run_sshd_command)

    try:
        # Chief machine waits for every worker's sshd to be available.  All machines should be
        # close to in-step by now because all machines just finished synchronizing rendezvous
        # info.
        deadline = time.time() + 20
        for peer_addr in info.container_addrs:
            util.check_sshd(peer_addr, deadline, constants.DTRAIN_SSH_PORT)

        return subprocess.Popen(full_cmd).wait()
    finally:
        sshd_process.kill()
        sshd_process.wait()


def parse_args(args: List[str]) -> List[str]:
    # Then parse the rest of the commands normally.
    parser = argparse.ArgumentParser(
        usage="%(prog)s (--trial TRIAL)|(SCRIPT...)",
        description=(
            "Launch a script under deepspeed on a Determined cluster, with automatic handling of "
            "IP addresses, sshd containers, and shutdown mechanics."
        ),
    )
    # For legacy Trial classes.
    parser.add_argument(
        "--trial",
        help=(
            "use a Trial class as the entrypoint to training.  When --trial is used, the SCRIPT "
            "positional argument must be omitted."
        ),
    )
    # For training scripts.
    parser.add_argument(
        "script",
        metavar="SCRIPT...",
        nargs=argparse.REMAINDER,
        help="script to launch for training",
    )
    parsed = parser.parse_args(args)

    script = parsed.script or []

    if parsed.trial is not None:
        if script:
            # When --trial is set, any other args are an error.
            parser.print_usage()
            print("error: extra arguments to --trial:", script, file=sys.stderr)
            sys.exit(1)
        script = det.util.legacy_trial_entrypoint_to_script(parsed.trial)
    elif not script:
        # There needs to be at least one script argument.
        parser.print_usage()
        print("error: empty script is not allowed", file=sys.stderr)
        sys.exit(1)

    return script


if __name__ == "__main__":
    script = parse_args(sys.argv[1:])
    sys.exit(main(script))
