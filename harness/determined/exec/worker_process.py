import faulthandler
import logging
import pathlib
import sys

import determined as det
from determined import ipc, layers, load
from determined.event_trail import create_event_trail_thread, TrialInfoEventV1
from determined.pytorch import PyTorchTrialController
from determined.estimator import EstimatorTrialController
from determined.keras import TFKerasTrialController


def config_logging(worker_process_env: layers.WorkerProcessContext) -> None:
    log_level = logging.DEBUG if worker_process_env.debug else logging.INFO
    logging.basicConfig(
        level=log_level, format="%(asctime)s:%(levelname)s [%(process)s]: %(message)s"
    )
    logging.getLogger().setLevel(log_level)
    logging.debug("Starting training process initialization.")


def main() -> None:
    if len(sys.argv) != 2:
        print("worker_process_env_path must be provided as a commandline argument", file=sys.stderr)
        sys.exit(1)

    # Load the worker process env.
    worker_process_env_path = pathlib.Path(sys.argv[1])
    worker_process_env = layers.WorkerProcessContext.from_file(worker_process_env_path)

    config_logging(worker_process_env)

    if worker_process_env.env.experiment_config.debug_enabled():
        faulthandler.dump_traceback_later(30, repeat=True)

    master_addr = worker_process_env.env.master_addr
    master_port = worker_process_env.env.master_port
    use_tls = worker_process_env.env.use_tls
    with create_event_trail_thread(master_addr, master_port, use_tls, noop=False) as event_trail:
        worker_process_env.env.set_event_trail_background_thread(event_trail)
        # Establish the connection to the ZMQBroadcastServer in this container.
        pub_url = f"tcp://localhost:{worker_process_env.broadcast_pub_port}"
        sub_url = f"tcp://localhost:{worker_process_env.broadcast_pull_port}"
        with ipc.ZMQBroadcastClient(pub_url, sub_url) as broadcast_client:

            # Wrap the communication layer in a workload.Stream.
            subrec = layers.SubprocessReceiver(broadcast_client)

            with det._catch_sys_exit():
                controller = load.prepare_controller(
                    worker_process_env.env,
                    iter(subrec),
                    worker_process_env.load_path,
                    worker_process_env.rendezvous_info,
                    worker_process_env.hvd_config,
                )

                experiment_id = worker_process_env.env.det_experiment_id
                trial_id = worker_process_env.env.det_trial_id
                if isinstance(controller, TFKerasTrialController):
                    trial_info_event = TrialInfoEventV1(experiment_id, trial_id, TrialInfoEventV1.TrialFramework.KERAS)
                elif isinstance(controller, PyTorchTrialController):
                    trial_info_event = TrialInfoEventV1(experiment_id, trial_id, TrialInfoEventV1.TrialFramework.PYTORCH)
                elif isinstance(controller, EstimatorTrialController):
                    trial_info_event = TrialInfoEventV1(experiment_id, trial_id, TrialInfoEventV1.TrialFramework.ESTIMATOR)
                else:
                    raise RuntimeError
                worker_process_env.env.event_trail.enqueue_for_async_send(trial_info_event)

                try:
                    controller.run()

                except Exception as e:
                    broadcast_client.send_exception_message()
                    raise e


if __name__ == "__main__":
    main()
