"""
launch_autohorovod.py is the default launch layer for Determined.

It launches the entrypoint script under horovodrun when slots_per_trial>1, or as a regular
subprocess otherwise.
"""

import distutils.util
import faulthandler
import json
import logging
import os
import sys

import simplejson

import determined as det
import determined.common
from determined import gpu, horovod, layers, load, workload
from determined.common import constants, storage
from determined.common.api import certs

ENVIRONMENT_VARIABLE_KEYS = {
    "DET_MASTER_ADDR",
    "DET_MASTER_PORT",
    "DET_AGENT_ID",
    "DET_SLOT_IDS",
    "DET_CONTAINER_ID",
    "DET_USE_GPU",
    "DET_EXPERIMENT_ID",
    "DET_TRIAL_ID",
    "DET_TRIAL_SEED",
    "DET_EXPERIMENT_CONFIG",
    "DET_HPARAMS",
    "DET_LATEST_CHECKPOINT",
    "DET_RENDEZVOUS_PORT",
    "DET_TRIAL_RUNNER_NETWORK_INTERFACE",
    "DET_TASK_TOKEN",
}


# def build_and_run_training_pipeline(env: det.EnvContext) -> None:
#
#     # Create the socket manager. The socket manager will connect to the master and read messages
#     # until it receives the rendezvous_info.
#     #
#     # TODO(ryan): Pull profiler hooks out of SocketManager and into their own layer.
#     with layers.SocketManager(env) as socket_mgr:
#         workloads = iter(socket_mgr)
#         hvd_config = horovod.HorovodContext.from_configs(
#             env.experiment_config, socket_mgr.get_rendezvous_info(), env.hparams
#         )
#         logging.info(f"Horovod config: {hvd_config.__dict__}.")
#
#         # Horovod distributed training is done inside subprocesses.
#         if hvd_config.use:
#             layers.SubprocessLauncher(
#                 env, workloads, socket_mgr.get_rendezvous_info(), hvd_config
#             ).run()
#         else:
#             if env.experiment_config.debug_enabled():
#                 faulthandler.dump_traceback_later(30, repeat=True)
#
#             with det._catch_sys_exit():
#                 with det._catch_init_invalid_hp(workloads):
#                     controller = load.prepare_controller(
#                         env,
#                         workloads,
#                         socket_mgr.get_rendezvous_info(),
#                         hvd_config,
#                     )
#                 controller.run()
#
#
# def do_training(env: det.EnvContext) -> None:
#     # XXX get real rendezvous info
#     rendezvous_info = det.RendezvousInfo(["192.168.0.4:1735"], 0)
#     hvd_config = horovod.HorovodContext.from_configs(
#         env.experiment_config, rendezvous_info, env.hparams
#     )
#     logging.info(f"Horovod config: {hvd_config.__dict__}.")
#     controller = load.prepare_controller(
#         env,
#         rendezvous_info,
#         hvd_config,
#     )
#     controller.run()


def main() -> None:
    for k in ENVIRONMENT_VARIABLE_KEYS:
        if k not in os.environ:
            sys.exit("Environment not set: missing " + k)

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
    certs.cli_cert = certs.default_load(
        master_url=f"http{'s' if use_tls else ''}://{master_addr}:{master_port}"
    )

    with open(os.environ["DET_LATEST_CHECKPOINT"], "r") as f:
        latest_checkpoint = json.load(f)

    use_gpu = distutils.util.strtobool(os.environ.get("DET_USE_GPU", "false"))
    slot_ids = json.loads(os.environ["DET_SLOT_IDS"])
    det_rendezvous_port = os.environ["DET_RENDEZVOUS_PORT"]
    det_trial_unique_port_offset = int(os.environ["DET_TRIAL_UNIQUE_PORT_OFFSET"])
    det_trial_runner_network_interface = os.environ["DET_TRIAL_RUNNER_NETWORK_INTERFACE"]
    det_trial_id = os.environ["DET_TRIAL_ID"]
    det_experiment_id = os.environ["DET_EXPERIMENT_ID"]
    det_agent_id = os.environ["DET_AGENT_ID"]
    det_cluster_id = os.environ["DET_CLUSTER_ID"]
    det_task_token = os.environ["DET_TASK_TOKEN"]
    trial_seed = int(os.environ["DET_TRIAL_SEED"])

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
        det_task_token=det_task_token,
        trial_seed=trial_seed,
        managed_training=True,
        test_mode=False,
        on_cluster=True,
    )

    logging.info(
        f"New trial runner in (container {container_id}) on agent {agent_id}: {env.__dict__}."
    )

    try:
        storage.validate_config(
            env.experiment_config["checkpoint_storage"],
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )
    except Exception as e:
        logging.error("Checkpoint storage validation failed: {}".format(e))
        sys.exit(1)

    jri = json.loads(os.environ["DET_RENDEZVOUS_INFO"])
    rendezvous_info = det.RendezvousInfo(
        addrs=jri["addresses"],
        rank=jri["rank"],
    )

    hvd_config = horovod.HorovodContext.from_configs(
        env.experiment_config, rendezvous_info, env.hparams
    )

    if hvd_config.use:
        print("launching with horovodrun")
        layers.SubprocessLauncher(env, rendezvous_info, hvd_config).run()
    else:
        print("launching with Popen")
        layers.SubprocessLauncher(
            env,
            rendezvous_info,
            hvd_config,
            python_subprocess_entrypoint="determined.exec.worker_process",
        ).run()


if __name__ == "__main__":
    main()
