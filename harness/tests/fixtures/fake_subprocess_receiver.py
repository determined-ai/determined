import pathlib
import sys
import traceback
from typing import Generator

from determined import constants, ipc, layers, workload

NUM_FAKE_WORKLOADS = 10


def fake_workload_gen() -> Generator[workload.Workload, None, None]:
    # Generate some fake workloads.
    batches_completed = 0
    for i in range(NUM_FAKE_WORKLOADS - 1):
        yield workload.train_workload(
            i + 1,
            batches_per_step=constants.DEFAULT_BATCHES_PER_STEP,
            batches_completed=batches_completed,
        )
        batches_completed += constants.DEFAULT_BATCHES_PER_STEP
    yield workload.validation_workload(i)


def main() -> None:
    if len(sys.argv) != 2:
        print("worker_process_env_path must be provided as a commandline argument", file=sys.stderr)
        sys.exit(1)

    # Load the worker process env.
    worker_process_env_path = pathlib.Path(sys.argv[1])
    worker_process_env = layers.WorkerProcessContext.from_file(worker_process_env_path)

    # Establish the connection to the ZMQBroadcastServer.
    pub_url = f"tcp://localhost:{worker_process_env.broadcast_pub_port}"
    sub_url = f"tcp://localhost:{worker_process_env.broadcast_pull_port}"
    with ipc.ZMQBroadcastClient(pub_url, sub_url) as broadcast_client:

        # Wrap the communication layer in a workload.Stream.
        subrec = layers.SubprocessReceiver(broadcast_client)

        # Compare the workloads received against the expected stream of workloads.
        expected = fake_workload_gen()
        for i, (wkld, _, resp_fn) in enumerate(iter(subrec)):
            assert wkld == next(expected)
            resp_fn({"count": i})

        assert i == NUM_FAKE_WORKLOADS


if __name__ == "__main__":
    try:
        main()
    except Exception:
        traceback.print_exc(file=sys.stderr)
        raise
