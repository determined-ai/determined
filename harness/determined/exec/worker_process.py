import faulthandler
import logging
import os
import pathlib
import sys

import determined as det
from determined import ipc, layers, load, profiler


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

    # API code expects credential to be available as an environment variable
    os.environ["DET_TASK_TOKEN"] = worker_process_env.env.det_task_token

    if worker_process_env.env.experiment_config.debug_enabled():
        faulthandler.dump_traceback_later(30, repeat=True)

    # Establish the connection to the ZMQBroadcastServer in this container.
    pub_url = f"tcp://localhost:{worker_process_env.broadcast_pub_port}"
    sub_url = f"tcp://localhost:{worker_process_env.broadcast_pull_port}"
    with ipc.ZMQBroadcastClient(pub_url, sub_url) as broadcast_client:

        # Wrap the communication layer in a workload.Stream.
        subrec = layers.SubprocessReceiver(broadcast_client)
        workloads = iter(subrec)

        (
            prof_start_on_batch,
            prof_end_after_batch,
        ) = worker_process_env.env.experiment_config.profiling_interval()
        local_rank = int(os.getenv("HOROVOD_LOCAL_RANK", 0))
        with profiler.ProfilerAgent(
            trial_id=worker_process_env.env.det_trial_id,
            agent_id=worker_process_env.env.det_agent_id,
            master_url=worker_process_env.env.master_url,
            profiling_is_enabled=worker_process_env.env.experiment_config.profiling_enabled(),
            global_rank=worker_process_env.rendezvous_info.get_rank(),
            local_rank=local_rank,
            start_on_batch=prof_start_on_batch,
            end_after_batch=prof_end_after_batch,
        ) as prof:

            with det._catch_sys_exit():
                with det._catch_init_invalid_hp(workloads):
                    controller = load.prepare_controller(
                        worker_process_env.env,
                        workloads,
                        worker_process_env.load_path,
                        worker_process_env.rendezvous_info,
                        worker_process_env.hvd_config,
                        prof,
                    )

                try:
                    controller.run()

                except Exception as e:
                    broadcast_client.send_exception_message()
                    raise e


if __name__ == "__main__":
    try:
        main()
    except det.InvalidHP:
        logging.info("InvalidHP detected, worker is exiting")
        pass
