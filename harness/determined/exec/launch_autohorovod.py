"""
launch_autohorovod.py is the default launch layer for Determined.

It launches the entrypoint script under horovodrun when slots_per_trial>1, or as a regular
subprocess otherwise.
"""

import distutils.util
import json
import logging
import os
import pathlib
import socket
import subprocess
import sys
import time

import simplejson

import determined as det
import determined.common
from determined import gpu, horovod, layers
from determined.common import api, constants, storage
from determined.common.api import certs
from determined.constants import HOROVOD_SSH_PORT

ENVIRONMENT_VARIABLE_KEYS = {
    "DET_MASTER_ADDR",
    "DET_MASTER_PORT",
    "DET_USE_TLS",
    "DET_AGENT_ID",
    "DET_SLOT_IDS",
    "DET_CONTAINER_ID",
    "DET_USE_GPU",
    "DET_EXPERIMENT_ID",
    "DET_TRIAL_ID",
    "DET_TRIAL_SEED",
    "DET_EXPERIMENT_CONFIG",
    "DET_HPARAMS",
    "DET_LATEST_BATCH",
    "DET_RENDEZVOUS_PORT",
    "DET_TRIAL_RUNNER_NETWORK_INTERFACE",
    "DET_ALLOCATION_SESSION_TOKEN",
    "DET_TRIAL_RUN_ID",
    "DET_ALLOCATION_ID",
    "DET_RENDEZVOUS_INFO",
}


def main() -> int:
    missing_vars = ENVIRONMENT_VARIABLE_KEYS.difference(set(os.environ))
    if missing_vars:
        for var in missing_vars:
            print(f"missing environment variable: {var}", file=sys.stderr)
        return 1

    experiment_config = simplejson.loads(os.environ["DET_EXPERIMENT_CONFIG"])
    debug = experiment_config.get("debug", False)
    determined.common.set_logger(debug)

    master_addr = os.environ["DET_MASTER_ADDR"]
    master_port = int(os.environ["DET_MASTER_PORT"])
    use_tls = distutils.util.strtobool(os.environ.get("DET_USE_TLS", "false"))
    master_cert_file = os.environ.get("DET_MASTER_CERT_FILE")
    master_cert_name = os.environ.get("DET_MASTER_CERT_NAME")
    agent_id = os.environ["DET_AGENT_ID"]
    container_id = os.environ["DET_CONTAINER_ID"]
    hparams = simplejson.loads(os.environ["DET_HPARAMS"])

    # TODO: refactor websocket, data_layer, and profiling to to not use the cli_cert.
    master_url = f"http{'s' if use_tls else ''}://{master_addr}:{master_port}"
    cert = certs.default_load(master_url)
    certs.cli_cert = cert

    latest_checkpoint = os.environ.get("DET_LATEST_CHECKPOINT")

    latest_batch = int(os.environ["DET_LATEST_BATCH"])

    use_gpu = distutils.util.strtobool(os.environ.get("DET_USE_GPU", "false"))
    slot_ids = json.loads(os.environ["DET_SLOT_IDS"])
    det_rendezvous_port = os.environ["DET_RENDEZVOUS_PORT"]
    det_trial_unique_port_offset = int(os.environ["DET_TRIAL_UNIQUE_PORT_OFFSET"])
    det_trial_runner_network_interface = os.environ["DET_TRIAL_RUNNER_NETWORK_INTERFACE"]
    det_trial_id = os.environ["DET_TRIAL_ID"]
    det_experiment_id = os.environ["DET_EXPERIMENT_ID"]
    det_agent_id = os.environ["DET_AGENT_ID"]
    det_cluster_id = os.environ["DET_CLUSTER_ID"]
    det_allocation_token = os.environ["DET_ALLOCATION_SESSION_TOKEN"]
    trial_seed = int(os.environ["DET_TRIAL_SEED"])
    trial_run_id = int(os.environ["DET_TRIAL_RUN_ID"])
    allocation_id = os.environ["DET_ALLOCATION_ID"]

    container_gpus = gpu.get_gpu_uuids_and_validate(use_gpu, slot_ids)

    env = det.EnvContext(
        master_addr=master_addr,
        master_port=master_port,
        use_tls=use_tls,
        master_cert_file=master_cert_file,
        master_cert_name=master_cert_name,
        container_id=container_id,
        experiment_config=experiment_config,
        hparams=hparams,
        latest_checkpoint=latest_checkpoint,
        latest_batch=latest_batch,
        use_gpu=use_gpu,
        container_gpus=container_gpus,
        slot_ids=slot_ids,
        debug=debug,
        det_rendezvous_port=det_rendezvous_port,
        det_trial_unique_port_offset=det_trial_unique_port_offset,
        det_trial_runner_network_interface=det_trial_runner_network_interface,
        det_trial_id=det_trial_id,
        det_experiment_id=det_experiment_id,
        det_agent_id=det_agent_id,
        det_cluster_id=det_cluster_id,
        det_allocation_token=det_allocation_token,
        trial_seed=trial_seed,
        trial_run_id=trial_run_id,
        allocation_id=allocation_id,
        managed_training=True,
        test_mode=False,
        on_cluster=True,
    )

    logging.info(
        f"New trial runner in (container {container_id}) on agent {agent_id}: {env.__dict__}."
    )

    # TODO: this should go in the chief worker, not in the launch layer.  For now, the
    # DistributedContext is not created soon enough for that to work well.
    try:
        storage.validate_config(
            env.experiment_config["checkpoint_storage"],
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )
    except Exception as e:
        logging.error("Checkpoint storage validation failed: {}".format(e))
        return 1

    jri = json.loads(os.environ["DET_RENDEZVOUS_INFO"])
    rendezvous_info = det.RendezvousInfo(addrs=jri["addresses"], rank=jri["rank"])

    hvd_config = horovod.HorovodContext.from_configs(
        env.experiment_config, rendezvous_info, env.hparams
    )

    if not hvd_config.use:
        # Skip running in subprocesses to improve startup times.
        from determined.exec import harness

        return harness.main(env, rendezvous_info, hvd_config)

    # Horovod-based training will require us to pass the environment config through the filesystem.
    # This is so that machine-specific variables, like DET_CONTAINER_ID, are correct on every
    # worker, even when the worker is on a different machine than where horovodrun was called.
    # Otherwise the horovodrun machine will blindly duplicate its own variables everywhere.
    # TODO: It would be better to do this master-side.
    wpc = layers.WorkerProcessContext(hvd_config, rendezvous_info, env)
    wpc_path = f"/tmp/worker_process_context{env.allocation_id}"
    wpc.to_file(pathlib.Path(wpc_path))

    if rendezvous_info.rank > 0:
        # Non-chief machines just run sshd.

        # Mark sshd containers as daemon containers that the master should kill when all non-daemon
        # contiainers (horovodrun, in this case) have exited.
        api.post(
            master_url,
            path=f"/api/v1/allocations/{env.allocation_id}/containers/{env.container_id}/daemon",
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
            f"/tmp/pid_server-{env.allocation_id}",
            str(len(env.slot_ids)),
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
            f"Non-chief [{rendezvous_info.get_rank()}] training process launch "
            f"command: {run_sshd_command}."
        )
        return subprocess.Popen(pid_server_cmd + run_sshd_command).wait()

    # Chief machine waits for every worker's sshd to be available.  All machines should be pretty
    # close to in-step by now because all machines just finished synchronizing rendezvous info.
    deadline = time.time() + 20
    for peer in rendezvous_info.get_ip_addresses()[1:]:
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
        f"/tmp/pid_server-{env.allocation_id}",
        str(len(env.slot_ids)),
        "--",
    ]

    hvd_cmd = horovod.create_run_command(
        num_proc_per_machine=len(env.slot_ids),
        ip_addresses=rendezvous_info.get_ip_addresses(),
        env=env,
        debug=env.experiment_config.debug_enabled(),
        optional_args=env.experiment_config.horovod_optional_args() + sys.argv[1:],
    )

    pid_client_cmd = [
        "python3",
        "-m",
        "determined.exec.pid_client",
        f"/tmp/pid_server-{env.allocation_id}",
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
        wpc_path,
    ]

    logging.debug(f"chief worker calling horovodrun with args: {hvd_cmd[1:]} ...")

    return subprocess.Popen(
        pid_server_cmd + hvd_cmd + pid_client_cmd + log_redirect_cmd + harness_cmd
    ).wait()


if __name__ == "__main__":
    sys.exit(main())
