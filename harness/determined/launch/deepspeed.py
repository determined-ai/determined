"""
deepspeed.py is the launch layer for DeepSpeedTrial in Determined.

It launches the entrypoint script using DeepSpeed's launch process.
"""

import argparse
import logging
import os
import re
import shlex
import subprocess
import sys
import time
from typing import Dict, List, Mapping, Optional

import deepspeed
import filelock
from deepspeed.launcher import runner
from packaging import version

import determined as det
from determined import common, constants, util
from determined.common.api import authentication, certs

hostfile_path = None
deepspeed_version = version.parse(deepspeed.__version__)

logger = logging.getLogger("determined.launch.deepspeed")


def is_using_cuda() -> bool:
    val = os.getenv("CUDA_VISIBLE_DEVICES")

    if val is None or len(val.strip()) == 0:
        return False
    else:
        return True


def is_nccl_socket_ifname_env_var_set() -> bool:
    val = os.getenv("NCCL_SOCKET_IFNAME")

    if val is None or len(val.strip()) == 0:
        return False
    else:
        return True


def get_hostfile_path(multi_machine: bool, allocation_id: str) -> Optional[str]:
    if not multi_machine:
        return None

    # When the task container uses "/tmp" from the host, having a file with
    # a well-known name in a world writable directory is not only a security
    # issue, but it can also cause a user's experiment to fail due to the
    # file being owned by another user. Hence we suffix the hostfile by
    # the allocation_id so it should be unique per trial launch.
    hostfile_path = f"/tmp/hostfile-{allocation_id}.txt"
    os.environ["DET_DEEPSPEED_HOSTFILE_PATH"] = hostfile_path
    return hostfile_path


def create_hostlist_file(
    hostfile_path: Optional[str], host_slot_counts: List[int], ip_addresses: List[str]
) -> str:
    assert len(host_slot_counts) == len(ip_addresses), "don't know slots for each node"

    trial_runner_hosts = ip_addresses.copy()
    # In the single node case, deepspeed doesn't use pdsh so we don't need to launch sshd.
    # Instead, the deepspeed launcher will use localhost as the chief worker ip.
    if len(ip_addresses) == 1:
        trial_runner_hosts[0] = "localhost"

    if hostfile_path is not None and not os.path.exists(hostfile_path):
        os.makedirs(os.path.dirname(hostfile_path), exist_ok=True)
        with open(hostfile_path, "w") as hostfile:
            lines = [
                f"{host} slots={slots}\n"
                for host, slots in zip(trial_runner_hosts, host_slot_counts)
            ]
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
        "--signal-children",
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


def filter_env_vars(env: Mapping[str, str]) -> Dict[str, str]:
    """
    Calculate all non-dangerous environment variables that we can pass to training workers.

    By default, the deepspeed launcher only keeps env vars that start with one of the following
    ["NCCL", "PYTHON", "MV2", "UCX"].

    We modify this behavior to include all environment variables except for ones we know to be
    problematic.  This is the same strategy taken by horovodrun.
    """

    EXCLUDE_REGEX = [
        "BASH_FUNC_.*",
        "OLDPWD",
        "HOSTNAME",
        ".*CUDA_VISIBLE_DEVICES",
        "SLURM_PROCID",
        "DET_SLOT_IDS",
        "DET_AGENT_ID",
    ]

    excludes = [re.compile(x) for x in EXCLUDE_REGEX]

    def should_keep(k: str, v: str) -> bool:
        if not any(x.match(k) for x in excludes):
            return True
        logger.debug(
            f"Excluding environment variable {k}={v} from training script environment, "
            "since it is likely unsafe to share between workers."
        )
        return False

    return {k: v for k, v in env.items() if should_keep(k, v)}


def create_deepspeed_env_file() -> None:
    """Create an env var export file to pass Determined vars to the deepspeed launcher."""
    with open(runner.DEEPSPEED_ENVIRONMENT_NAME, "w") as f:
        for k, v in filter_env_vars(os.environ).items():
            # We need to turn our envvars into shell-escaped strings to export them correctly
            # since values may contain spaces and quotes.  shlex.quote was removed from the
            # deepspeed launcher in 0.6.2 so we add it here for this version onwards.
            if deepspeed_version >= version.parse("0.6.2"):
                line = f"{k}={shlex.quote(v)}"
            else:
                line = f"{k}={v}"

            # By default, IFS is set to a space, a tab, and a newline character.
            # Therefore, the IFS environment variable will be written as:
            #
            # IFS='
            # '
            #
            # The single quote on its own line will cause the
            # "key, val = var.split('=', maxsplit=1)" in
            # "deepspeed/launcher/runner.py" to fail with the error:
            #
            # ValueError: not enough values to unpack (expected 2, got 1)
            #
            # Therefore, avoid writing any environment variables containing a
            # newline character.
            if "\n" not in line and "\r" not in line:
                f.write(f"{line}\n")
            else:
                logger.warning(
                    f"Excluding environment variable {k} because it contains a newline character."
                )


def create_run_command(master_address: str, hostfile_path: Optional[str]) -> List[str]:
    # Construct the deepspeed command.
    deepspeed_process_cmd = ["deepspeed"]
    if hostfile_path is not None:
        deepspeed_process_cmd += ["-H", hostfile_path]
    deepspeed_process_cmd += ["--master_addr", master_address, "--no_python", "--no_local_rank"]
    if deepspeed_version > version.parse("0.6.4"):
        deepspeed_process_cmd.append("--no_ssh_check")  # Bypass deepspeed's ssh check.
    deepspeed_process_cmd.append("--")
    return deepspeed_process_cmd


def check_deepspeed_version(multi_machine: bool) -> None:
    if not multi_machine:
        return
    # Upstream deepspeed added an ssh check from 0.6.1 onwards that does not have the
    # StrictHostKeyChecking=no arg and fails with our agents.  A PR adding a no_ssh_check arg
    # to bypass this should land for versions 0.6.5 and onwards.
    if deepspeed_version <= version.parse("0.6.0"):
        return
    if deepspeed_version > version.parse("0.6.4"):
        return
    raise ValueError(
        "This launcher is incompatible with deepspeed versions 0.6.1 to 0.6.4 due to an ssh check "
        "in the upstream launcher that fails with our setup.  We perform our own ssh check by "
        "default and can bypass this ssh check for deepspeed versions >= 0.6.5."
    )


def main(script: List[str]) -> int:
    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'
    experiment_config = det.ExperimentConfig(info.trial._config)
    common.set_logger(experiment_config.debug_enabled())

    multi_machine = len(info.container_addrs) > 1
    check_deepspeed_version(multi_machine)

    # Hack: get the resources id from the environment.
    resources_id = os.environ.get("DET_RESOURCES_ID")
    assert resources_id is not None, "Unable to run with DET_RESOURCES_ID unset"

    # The launch layer should provide the chief_ip to the training code, so that the training code
    # can function with a different launch layer in a different environment.  Inside Determined, the
    # easiest way to get the chief_ip is with container_addrs.
    chief_ip = info.container_addrs[0]

    # Chief IP is set as an environment variable to support nested launch layers
    os.environ["DET_CHIEF_IP"] = chief_ip

    # If the NCCL_SOCKET_IFNAME environment variable wasn't explicitly set by
    # the user in the experiment's YAML file, then set it to the distributed
    # network interface, if the value of "dtrain_network_interface" under
    # "task_container_defaults" has been set in the "master.yaml".
    if is_using_cuda() and not is_nccl_socket_ifname_env_var_set():
        dtrain_network_interface = os.environ.get("DET_INTER_NODE_NETWORK_INTERFACE", None)

        if dtrain_network_interface is not None and len(dtrain_network_interface) > 0:
            os.environ["NCCL_SOCKET_IFNAME"] = dtrain_network_interface

    # In some downstream training code, the hostfile is expected on all nodes so we create the
    # hostfile on all nodes here before opening the non-chief node subprocess.  Since we
    # use the allocation id to create the hostfile path, we register that path to an environment
    # variable for access downstream.  A filelock is used to ensure we do not have clashing writes
    # if the hostfile_path is on a shared filesystem.
    hostfile_path = get_hostfile_path(multi_machine, info.allocation_id)

    lock = str(hostfile_path) + ".lock"
    with filelock.FileLock(lock):
        master_address = create_hostlist_file(
            hostfile_path=hostfile_path,
            host_slot_counts=info.container_slot_counts,
            ip_addresses=info.container_addrs,
        )

    # All ranks will need to run sshd.
    run_sshd_command = create_sshd_cmd()

    if info.container_rank > 0:
        # Non-chief machines just run sshd.

        # Mark sshd containers as daemon containers that the master should kill when all non-daemon
        # containers (deepspeed launcher, in this case) have exited.
        cert = certs.default_load(info.master_url)
        sess = authentication.login_with_cache(info.master_url, cert=cert)
        sess.post(f"/api/v1/allocations/{info.allocation_id}/resources/{resources_id}/daemon")

        # Wrap it in a pid_server to ensure that we can't hang if a worker fails.
        # This is useful for deepspeed which does not have good error handling for remote processes
        # spun up by pdsh.
        pid_server_cmd = create_pid_server_cmd(info.allocation_id, len(info.slot_ids))

        logger.debug(
            f"Non-chief [{info.container_rank}] training process launch "
            f"command: {run_sshd_command}."
        )
        p = subprocess.Popen(pid_server_cmd + run_sshd_command)
        with det.util.forward_signals(p):
            return p.wait()

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

    cmd = create_run_command(master_address, hostfile_path)

    pid_client_cmd = create_pid_client_cmd(info.allocation_id)

    log_redirect_cmd = create_log_redirect_cmd()

    harness_cmd = script

    logger.debug(f"chief worker calling deepspeed with args: {cmd[1:]} ...")

    full_cmd = pid_server_cmd + cmd + pid_client_cmd + log_redirect_cmd + harness_cmd

    if not multi_machine:
        p = subprocess.Popen(full_cmd)
        with det.util.forward_signals(p):
            return p.wait()

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

        p = subprocess.Popen(full_cmd)
        with det.util.forward_signals(p):
            return p.wait()
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
