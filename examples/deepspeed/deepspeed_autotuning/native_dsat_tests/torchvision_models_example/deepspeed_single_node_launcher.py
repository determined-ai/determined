import argparse
import os
import subprocess
import sys
from typing import List, Tuple

import determined as det
from determined import constants

C10D_PORT = 29400


def create_launch_cmd(
    num_nodes: int, proc_per_node: int, node_rank: int, master_addr: str, override_args: List[str]
) -> List[str]:
    cmd = [
        "deepspeed",
        "--num_nodes",
        str(num_nodes),
        "--num_gpus",
        str(proc_per_node),
        "--master_addr",
        master_addr,
        "--master_port",
        str(C10D_PORT),
        "--no_local_rank",
    ]

    cmd.extend(override_args)
    return cmd


def create_log_redirect_cmd() -> List[str]:
    return [
        "python3",
        "-m",
        "determined.launch.wrap_rank",
        "RANK",
        "--",
    ]


def create_pid_server_cmd(allocation_id: str, num_slot_ids: int) -> List[str]:
    return [
        "python3",
        "-m",
        "determined.exec.pid_server",
        "--on-fail",
        "SIGTERM",
        "--on-exit",
        "WAIT",
        f"/tmp/pid_server-{allocation_id}",
        str(num_slot_ids),
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


def create_sshd_cmd() -> List[str]:
    return [
        "/usr/sbin/sshd",
        "-p",
        str(constants.DTRAIN_SSH_PORT),
        "-f",
        "/run/determined/ssh/sshd_config",
        "-D",
    ]


def main(override_args: List[str], script: List[str]) -> int:
    override_args = override_args or []

    info = det.get_cluster_info()
    assert info is not None, "must be run on-cluster"

    # GG: USE_DEEPSPEED env var is used when determining the backend in _trial_controller.py
    os.environ["USE_DEEPSPEED"] = "1"

    chief_ip = info.container_addrs[0]
    os.environ["DET_CHIEF_IP"] = chief_ip

    launch_cmd = create_launch_cmd(
        len(info.container_addrs),
        len(info.slot_ids),
        info.container_rank,
        "localhost" if len(info.container_addrs) == 1 else chief_ip,
        override_args,
    )

    log_redirect_cmd = create_log_redirect_cmd()

    # Due to a bug in PyTorch, we need to wrap the launcher in pid_server/pid_client to correctly
    # handle errors and ensure workers don't hang when a process fails
    pid_server_cmd = create_pid_server_cmd(info.allocation_id, len(info.slot_ids))
    pid_client_cmd = create_pid_client_cmd(info.allocation_id)

    # The pid_xxx_cmd and log_redirect_cmd interfere with autotuning, for some reason.
    full_cmd = pid_server_cmd + launch_cmd + pid_client_cmd + log_redirect_cmd + script
    full_cmd = launch_cmd + script

    # All ranks will need to run sshd.
    run_sshd_command = create_sshd_cmd()
    # Set custom PDSH args:
    # * bypass strict host checking
    # * -p our custom port
    # * other args are default ssh args for pdsh
    os.environ["PDSH_SSH_ARGS"] = (
        "-o PasswordAuthentication=no -o StrictHostKeyChecking=no "
        f"-p {constants.DTRAIN_SSH_PORT} -2 -a -x %h"
    )
    os.environ["PDSH_SSH_ARGS_APPEND"] = "-i /run/determined/ssh/id_rsa"
    # Chief worker also needs to run sshd for auto-tuning.
    subprocess.Popen(run_sshd_command)

    p = subprocess.Popen(full_cmd)
    with det.util.forward_signals(p):
        return p.wait()


def parse_args(args: List[str]) -> Tuple[List[str], List[str]]:
    if "--" in args:
        split = args.index("--")
        override_args = args[:split]
        args = args[split + 1 :]
    else:
        override_args = []

    parser = argparse.ArgumentParser(
        usage="%(prog)s [[TORCH_OVERRIDES...] --] (--trial TRIAL)|(SCRIPT...)",
        description=("Launch a script under pytorch distributed on a Determined cluster"),
        epilog=(
            "TORCH_OVERRIDES may be a list of arguments to pass directly to "
            "torch.distributed.launch to override the values set by Determined automatically.  "
            "When provided, the list of override arguments must be terminated by a `--` argument."
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

    return override_args, script


if __name__ == "__main__":
    override_args, script = parse_args(sys.argv[1:])
    sys.exit(main(override_args, script))
