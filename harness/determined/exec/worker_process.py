import os
import pathlib
import sys

import determined as det
from determined import ipc, layers, load, log
from determined.experimental import debug


def main() -> None:
    if len(sys.argv) != 2:
        print("worker_process_env_path must be provided as a commandline argument", file=sys.stderr)
        sys.exit(1)

    # Load the worker process env.
    worker_process_env_path = pathlib.Path(sys.argv[1])
    worker_process_env = layers.WorkerProcessContext.from_file(worker_process_env_path)

    env = worker_process_env.env
    rendezvous_info = worker_process_env.rendezvous_info
    hvd_config = worker_process_env.hvd_config
    load_path = worker_process_env.load_path

    # Technically we don't need the horovod rank until after the horovod object is initialized in
    # the framework-specific pre_exec_hook(), but because that point occurs buried deep in the
    # trial/native loading code, it's much cleaner to read the environment variable here.
    # When we remove native, we should refactor this to not read the environment variable.
    hvd_rank = int(os.environ.get("HOROVOD_RANK", -2))

    env.dbg.set_loggers()
    log.harness.debug("Starting training process initialization.")

    # Establish the connection to the ZMQBroadcastServer in this container.
    pub_url = f"tcp://localhost:{worker_process_env.broadcast_pub_port}"
    sub_url = f"tcp://localhost:{worker_process_env.broadcast_pull_port}"

    stack_trace_thread = debug.stack_trace_thread(env.dbg.stack_trace_period_sec)
    with stack_trace_thread, ipc.ZMQBroadcastClient(pub_url, sub_url) as broadcast_client:

        # Wrap the communication layer in a workload.Stream.
        subrec = layers.SubprocessReceiver(broadcast_client)

        # Gather metrics for just this process, not the whole system.
        with layers.ProfilingLayer(
            workloads=iter(subrec),
            period=env.dbg.resource_profile_period_sec,
            initial_workload_state=str(env.initial_workload.kind),
            machine_rank=rendezvous_info.get_rank(),
            worker_rank=hvd_rank,
            system_level_metrics=False,
            process_level_metrics=True,
        ) as profiling_layer:

            with det._catch_sys_exit():
                controller = load.prepare_controller(
                    env,
                    iter(profiling_layer),
                    load_path,
                    rendezvous_info,
                    hvd_config,
                )

                try:
                    controller.run()
                except Exception as e:
                    broadcast_client.send_exception_message()
                    raise e


if __name__ == "__main__":
    main()
