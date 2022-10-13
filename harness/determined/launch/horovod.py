"""
autohorovod.py is the default launch layer for Determined.

It launches the entrypoint script under horovodrun when slots_per_trial>1, or as a regular
subprocess otherwise.
"""
import argparse
import logging
import os
import subprocess
import sys
import time
from typing import List, Tuple

import determined as det
from determined import horovod, util
from determined.common import api
from determined.common.api import certs
from determined.constants import DTRAIN_SSH_PORT


def create_sshd_worker_cmd(
    allocation_id: str, num_slot_ids: int, debug: bool = False
) -> Tuple[List[str], List[str]]:
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
        "SIGTERM",
        f"/tmp/pid_server-{allocation_id}",
        str(num_slot_ids),
        "--",
    ]

    run_sshd_command = [
        "/usr/sbin/sshd",
        "-p",
        str(DTRAIN_SSH_PORT),
        "-f",
        "/run/determined/ssh/sshd_config",
        "-D",
    ]
    if debug:
        run_sshd_command.append("-e")
    return pid_server_cmd, run_sshd_command


def create_hvd_pid_server_cmd(allocation_id: str, num_slot_ids: int) -> List[str]:
    return [
        "python3",
        "-m",
        "determined.exec.pid_server",
        # Use SIGINT on horovod, because it freaks out with SIGTERM.
        "--on-fail",
        "SIGINT",
        "--on-exit",
        "WAIT",
        f"/tmp/pid_server-{allocation_id}",
        str(num_slot_ids),
        "--",
    ]


def create_worker_wrapper_cmd(allocation_id: str) -> List[str]:
    pid_client_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_client",
        f"/tmp/pid_server-{allocation_id}",
        "--",
    ]

    log_redirect_cmd = [
        "python3",
        "-m",
        "determined.launch.wrap_rank",
        "HOROVOD_RANK,OMPI_COMM_WORLD_RANK,PMI_RANK",
        "--",
    ]

    return pid_client_cmd + log_redirect_cmd


def main(hvd_args: List[str], script: List[str], autohorovod: bool) -> int:
    hvd_args = hvd_args or []

    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"
    assert info.task_type == "TRIAL", f'must be run with task_type="TRIAL", not "{info.task_type}"'

    # When --autohorovod was set, detect single-slot and zero-slot trials.
    if autohorovod and len(info.container_addrs) == 1 and len(info.slot_ids) <= 1:
        p = subprocess.Popen(script)
        with det.util.forward_signals(p):
            return p.wait()

    # Hack: get the resources id from the environment.
    resources_id = os.environ.get("DET_RESOURCES_ID")
    assert resources_id is not None, "Unable to run with DET_RESOURCES_ID unset"

    # Hack: read the full config.  The experiment config is not a stable API!
    experiment_config = info.trial._config

    debug = experiment_config.get("debug", False)
    if debug:
        logging.getLogger().setLevel(logging.DEBUG)

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
        # containers (horovodrun, in this case) have exited.
        api.post(
            info.master_url,
            path=f"/api/v1/allocations/{info.allocation_id}/resources/{resources_id}/daemon",
            cert=cert,
        )

        pid_server_cmd, run_sshd_command = create_sshd_worker_cmd(
            info.allocation_id, len(info.slot_ids), debug=debug
        )

        logging.debug(
            f"Non-chief [{info.container_rank}] training process launch "
            f"command: {run_sshd_command}."
        )
        p = subprocess.Popen(pid_server_cmd + run_sshd_command)
        with det.util.forward_signals(p):
            return p.wait()

    # Chief machine waits for every worker's sshd to be available.  All machines should be pretty
    # close to in-step by now because all machines just finished synchronizing rendezvous info.
    deadline = time.time() + 20
    for peer_addr in info.container_addrs[1:]:
        util.check_sshd(peer_addr, deadline, DTRAIN_SSH_PORT)

    # The chief has several layers of wrapper processes:
    # - a top-level pid_server, which causes the whole container to exit if any local worker dies.
    # - horovodrun, which launches $slots_per_trial copies of the following layers:
    #     - a pid_client process to contact the local pid_server
    #     - wrap_rank, which redirects stdin/stdout to the local container
    #     - harness.py, which actually does the training for the worker
    #
    # It is a bug in horovod that causes us to have this pid_server/pid_client pair of layers.
    # We can remove these layers when the upstream fix has been around for long enough that we can
    # reasonably require user images to have patched horovod installations.

    pid_server_cmd = create_hvd_pid_server_cmd(info.allocation_id, len(info.slot_ids))

    # TODO: remove this (very old) hack when we have a configurable launch layer.
    hvd_optional_args = experiment_config.get("data", {}).get("__det_dtrain_args", [])
    hvd_optional_args += hvd_args
    if debug:
        hvd_optional_args += ["--mpi-args=-v --display-map"]

    hvd_cmd = horovod.create_run_command(
        num_proc_per_machine=len(info.slot_ids),
        ip_addresses=info.container_addrs,
        inter_node_network_interface=info.trial._inter_node_network_interface,
        optimizations=experiment_config["optimizations"],
        debug=debug,
        optional_args=hvd_optional_args,
    )

    worker_wrapper_cmd = create_worker_wrapper_cmd(info.allocation_id)

    logging.debug(f"chief worker calling horovodrun with args: {hvd_cmd[1:]} ...")

    os.environ["USE_HOROVOD"] = "1"

    # We now have environment images with built-in OpenMPI.   When invoked the
    # SLURM_JOBID variable triggers integration with SLURM, however, we are
    # running in a singularity container and SLURM may or may not have
    # compatible configuration enabled.  We therefore clear the SLURM_JOBID variable
    # before invoking mpi so that mpirun will honor the args passed via horvod
    # run to it describing the hosts and process topology, otherwise mpi ends
    # up wanting to launch all -np# processes on the local causing an oversubscription
    # error ("There are not enough slots available in the system").
    os.environ.pop("SLURM_JOBID", None)
    p = subprocess.Popen(pid_server_cmd + hvd_cmd + worker_wrapper_cmd + script)
    with det.util.forward_signals(p):
        return p.wait()


def parse_args(args: List[str]) -> Tuple[List[str], List[str], bool]:
    # Manually extract anything before the first "--" to pass through to horovodrun.
    if "--" in args:
        split = args.index("--")
        hvd_args = args[:split]
        args = args[split + 1 :]
    else:
        hvd_args = []

    # Then parse the rest of the commands normally.
    parser = argparse.ArgumentParser(
        usage="%(prog)s [[HVD_OVERRIDES...] --] (--trial TRIAL)|(SCRIPT...)",
        description=(
            "Launch a script under horovodrun on a Determined cluster, with automatic handling of "
            "IP addresses, sshd containers, and shutdown mechanics."
        ),
        epilog=(
            "HVD_OVERRIDES may be a list of arguments to pass directly to horovodrun to override "
            "the values set by Determined automatically.  When provided, the list of override "
            "arguments must be terminated by a `--` argument."
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

    # --autohorovod is an internal-only flag.  What it does is it causes the code skip the
    # horovodrun wrapper when slots_per_trial <= 1.  This has two effects:
    # 1. the execution stack for non-distributed training is simpler, because horovodrun would only
    #    add complexity, and
    # 2. the training code becomes more complex because it has to be aware of multi-vs-single-slot
    #    configuration and avoiding using horovod in the single-slot case.
    # In Determined-owned training loops, we pay the code complexity cost so that the execution of
    # single-slot training stays as simple as possible.  Determined users should not have to pay for
    # dtrain if they don't use it.  However, if a user is writing a custom horovod-based training
    # script and using this det.launch.horovod module as their launch layer, presumably they are
    # already comfortable with horovod and they are not likely to want to pay the code complexity
    # cost.  That is why this flag is internal; we don't really expect any users to ever use it.
    parser.add_argument("--autohorovod", action="store_true", help=argparse.SUPPRESS)

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

    return hvd_args, script, parsed.autohorovod


if __name__ == "__main__":
    hvd_args, script, autohorovod = parse_args(sys.argv[1:])
    sys.exit(main(hvd_args, script, autohorovod))
