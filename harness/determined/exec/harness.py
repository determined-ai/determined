"""
The entrypoint for the isolated environment we use to run trials.

Basic workflow is:
  * Agent launches a new container that has this script as its
    entrypoint. The agent passes along various parameters (e.g., master
    address, workload) via environment variables.

  * The script establishes a WebSocket connection back to the master,
    and sends a TRIAL_RUNNER_STARTUP message including the container's
    initial workload. We then start running the specified workload.

  * When the initial workload is complete, the trial runner notifies the
    master via a WORKLOAD_COMPLETED message.

  * The master sends a RUN_WORKLOAD message to the trial runner to ask
    it to do some work, e.g., run a step of the current trial,
    checkpoint the model to persistent storage, or compute the model's
    current validation metrics. This can happen many times to run multiple
    steps of the same trial in a row.

  * Eventually, the master asks the trial runner to exit via a TERMINATE
    message.

"""
import contextlib
import distutils.util
import faulthandler
import json
import logging
import os
import pathlib
import sys
from typing import Any, Dict, Iterator, Optional

import simplejson

import determined as det
import determined.common
from determined import gpu, horovod, layers, load, workload
from determined.common import constants, storage

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
    "DET_INITIAL_WORKLOAD",
    "DET_LATEST_CHECKPOINT",
    "DET_WORKLOAD_MANAGER_TYPE",
    "DET_RENDEZVOUS_PORT",
    "DET_TRIAL_RUNNER_NETWORK_INTERFACE",
    "DET_TASK_TOKEN",
}


@contextlib.contextmanager
def maybe_load_checkpoint(
    storage_mgr: storage.StorageManager, checkpoint: Optional[Dict[str, Any]]
) -> Iterator[Optional[pathlib.Path]]:
    """
    Either wrap a storage_mgr.restore_path() context manager, or be a noop
    context manager if there is no checkpoint to load.
    """

    if checkpoint is None:
        yield None

    else:
        metadata = storage.StorageMetadata.from_json(checkpoint)
        logging.info("Restoring trial from checkpoint {}".format(metadata.storage_id))

        with storage_mgr.restore_path(metadata) as path:
            yield pathlib.Path(path)


def build_and_run_training_pipeline(env: det.EnvContext) -> None:

    # Create the socket manager. The socket manager will connect to the master and read messages
    # until it receives the rendezvous_info.
    #
    # TODO(ryan): Pull profiler hooks out of SocketManager and into their own layer.
    with layers.SocketManager(env) as socket_mgr:

        # Create the storage manager. This is used to download the initial checkpoint here in
        # build_training_pipeline and also used by the workload manager to create and store
        # checkpoints during training.
        storage_mgr = storage.build(
            env.experiment_config["checkpoint_storage"],
            container_path=constants.SHARED_FS_CONTAINER_PATH,
        )

        [tensorboard_mgr, tensorboard_writer] = load.prepare_tensorboard(
            env, constants.SHARED_FS_CONTAINER_PATH
        )

        # Create the workload manager. The workload manager will receive workloads from the
        # socket_mgr, and augment them with some additional arguments. Additionally, the
        # workload manager is responsible for some generic workload hooks for things like timing
        # workloads, preparing checkpoints, and uploading completed checkpoints.  Finally, the
        # workload manager does some sanity checks on response messages that originate from the
        # trial.
        #
        # TODO(ryan): Refactor WorkloadManager into separate layers that do each separate task.
        workload_mgr = layers.build_workload_manager(
            env,
            iter(socket_mgr),
            socket_mgr.get_rendezvous_info(),
            storage_mgr,
            tensorboard_mgr,
            tensorboard_writer,
        )

        workloads = iter(workload_mgr)
        hvd_config = horovod.HorovodContext.from_configs(
            env.experiment_config, socket_mgr.get_rendezvous_info(), env.hparams
        )
        logging.info(f"Horovod config: {hvd_config.__dict__}.")

        # Load the checkpoint, if necessary. Any possible sinks to this pipeline will need access
        # to this checkpoint.
        with maybe_load_checkpoint(storage_mgr, env.latest_checkpoint) as load_path:

            # Horovod distributed training is done inside subprocesses.
            if hvd_config.use:
                subproc = layers.SubprocessLauncher(
                    env, workloads, load_path, socket_mgr.get_rendezvous_info(), hvd_config
                )
                subproc.run()
            else:
                if env.experiment_config.debug_enabled():
                    faulthandler.dump_traceback_later(30, repeat=True)

                with det._catch_sys_exit():
                    with det._catch_init_invalid_hp(workloads):
                        controller = load.prepare_controller(
                            env,
                            workloads,
                            load_path,
                            socket_mgr.get_rendezvous_info(),
                            hvd_config,
                        )
                    controller.run()


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
    initial_work = workload.Workload.from_json(simplejson.loads(os.environ["DET_INITIAL_WORKLOAD"]))

    with open(os.environ["DET_LATEST_CHECKPOINT"], "r") as f:
        latest_checkpoint = json.load(f)

    use_gpu = distutils.util.strtobool(os.environ.get("DET_USE_GPU", "false"))
    slot_ids = json.loads(os.environ["DET_SLOT_IDS"])
    workload_manager_type = os.environ["DET_WORKLOAD_MANAGER_TYPE"]
    det_rendezvous_port = os.environ["DET_RENDEZVOUS_PORT"]
    det_trial_unique_port_offset = int(os.environ["DET_TRIAL_UNIQUE_PORT_OFFSET"])
    det_trial_runner_network_interface = os.environ["DET_TRIAL_RUNNER_NETWORK_INTERFACE"]
    det_trial_id = os.environ["DET_TRIAL_ID"]
    det_experiment_id = os.environ["DET_EXPERIMENT_ID"]
    det_agent_id = os.environ["DET_AGENT_ID"]
    det_cluster_id = os.environ["DET_CLUSTER_ID"]
    det_task_token = os.environ["DET_TASK_TOKEN"]
    trial_seed = int(os.environ["DET_TRIAL_SEED"])

    gpu_uuids = gpu.get_gpu_uuids_and_validate(use_gpu, slot_ids)

    env = det.EnvContext(
        master_addr,
        master_port,
        use_tls,
        master_cert_file,
        master_cert_name,
        container_id,
        experiment_config,
        hparams,
        initial_work,
        latest_checkpoint,
        use_gpu,
        gpu_uuids,
        slot_ids,
        debug,
        workload_manager_type,
        det_rendezvous_port,
        det_trial_unique_port_offset,
        det_trial_runner_network_interface,
        det_trial_id,
        det_experiment_id,
        det_agent_id,
        det_cluster_id,
        det_task_token,
        trial_seed,
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

    try:
        build_and_run_training_pipeline(env)
    except det.InvalidHP:
        logging.info("InvalidHP detected, gracefully exiting trial")
        pass


if __name__ == "__main__":
    main()
