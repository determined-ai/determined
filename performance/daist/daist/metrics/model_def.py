import queue
import subprocess
import sys
import threading
import time
from typing import NewType

import numpy as np

import determined as det
from determined.core import Context


class MetricKey:
    type_ = NewType('MetricKey', str)
    READ: type_ = 'read'
    WRITE: type_ = 'write'

    all_ = (READ, WRITE)


class Param:
    type_ = NewType('MetricKey', str)
    BATCH_METRIC_COUNT: type_ = 'batch_metric_count'
    CONCURRENCY: type_ = 'concurrency'
    METRIC_COUNT: type_ = 'metric_count'


def worker_main_in_context(core_context: Context, num_batches: int, metric_count: int) -> int:
    write_latencies = list()

    for batch in range(num_batches):
        if core_context.distributed.rank == 0:
            test_metrics = dict(zip((str(ii) for ii in range(metric_count)),
                                    (float(jj) for jj in np.random.normal(size=metric_count))))
            start_seconds = time.time()
            core_context.train.report_training_metrics(steps_completed=batch,
                                                       metrics=test_metrics)
            write_latencies.append(time.time() - start_seconds)

        if core_context.preempt.should_preempt():
            return 1
        batch += 1

    core_context.train.report_validation_metrics(steps_completed=num_batches,
                                                 metrics={MetricKey.WRITE: write_latencies})

    return 0


def launcher_main(info: det.ClusterInfo):
    # Use subprocess to start one worker process per node.
    slots_per_node = len(info.slot_ids)
    cross_rank = info.container_rank

    procs = []
    for local_rank in range(slots_per_node):
        rank = cross_rank * slots_per_node + local_rank
        cmd = [
            # Use the determined.launch.wrap_rank to wrap the worker process.
            # This ensures logs from each worker can be filtered by rank in the WebUI.
            "python3",
            "-m",
            "determined.launch.wrap_rank",
            str(rank),
            "--",
            # Re-invoke this script but as a worker.
            "python3",
            __file__,
            "worker",
            str(rank),
            str(local_rank),
        ]
        procs.append(subprocess.Popen(cmd))

    # A good launcher normally waits for all workers to finish, but cleans up and exits
    # nonzero immediately if any worker fails to prevent distributed training jobs from
    # hanging.  One way to do this by managing each worker process in a thread and sending
    # exit codes over a Queue as workers complete.
    q = queue.Queue()

    def wait_for_worker(proc):
        exit_code = proc.wait()
        q.put((proc, exit_code))

    threads = [threading.Thread(target=wait_for_worker, args=(proc,)) for proc in procs]

    for t in threads:
        t.start()

    first_failed_exit = 0
    for i in range(slots_per_node):
        proc, worker_exit = q.get()
        procs.remove(proc)
        if worker_exit != 0 and first_failed_exit == 0:
            # When the first worker crashes, preempt the others.
            first_failed_exit = worker_exit
            for proc in procs:
                proc.kill()

    for t in threads:
        t.join()

    return first_failed_exit


def worker_main(info: det.ClusterInfo):
    chief_ip = info.container_addrs[0]
    cross_rank = info.container_rank
    num_nodes = len(info.container_addrs)
    slots_per_node = len(info.slot_ids)
    rank = int(sys.argv[2])
    local_rank = int(sys.argv[3])

    # In the absence of a distributed training framework that might define the
    # rank/local_rank/cross_rank, you can derive them from the ClusterInfo API.
    distributed = det.core.DistributedContext(
        rank=rank,
        size=num_nodes * slots_per_node,
        local_rank=local_rank,
        local_size=slots_per_node,
        cross_rank=cross_rank,
        cross_size=num_nodes,
        chief_ip=chief_ip,
    )

    with det.core.init(distributed=distributed) as core_context:
        exit_code = worker_main_in_context(
            core_context=core_context,
            num_batches=info.trial.hparams['num_batches'],
            metric_count=info.trial.hparams['metric_count'],
        )
    return exit_code


def main():
    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"

    if sys.argv[1] == "launcher":
        exitcode = launcher_main(info)
        sys.exit(exitcode)

    if sys.argv[1] == "worker":
        exitcode = worker_main(info)
        sys.exit(exitcode)

    raise ValueError(f"unrecognized first argument: {sys.argv[1]}")


if __name__ == "__main__":
    main()
