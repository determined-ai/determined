"""
Stage 4: Let's do all the same things, but across multiple workers.

Multi-slot tasks in Determined get the following features out-of-the box:
- batch scheduling
- IP address coordination between workers
- distributed communication primitives
- coordinated cross-worker preemption support
- checkpoint download sharing between workers on the same node
- filter logs by rank in the WebUI
"""

import logging
import pathlib
import queue
import subprocess
import sys
import threading
import time

import determined as det


def save_state(x, steps_completed, trial_id, checkpoint_directory):
    with checkpoint_directory.joinpath("state").open("w") as f:
        f.write(f"{x},{steps_completed},{trial_id}")


def load_state(trial_id, checkpoint_directory):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("state").open("r") as f:
        x, steps_completed, ckpt_trial_id = [int(field) for field in f.read().split(",")]
    if ckpt_trial_id == trial_id:
        return x, steps_completed
    else:
        return x, 0


def main(core_context, latest_checkpoint, trial_id, increment_by):
    x = 0

    starting_batch = 0
    if latest_checkpoint is not None:
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            x, starting_batch = load_state(trial_id, path)

    batch = starting_batch
    last_checkpoint_batch = None
    for op in core_context.searcher.operations():
        while batch < op.length:
            # NEW: Increment by the sum of every worker's increment_by value.
            # In reality, it is just increment_by*num_workers, but the point is
            # to show how to use the communication primitives.
            all_increment_bys = core_context.distributed.allgather(increment_by)
            x += sum(all_increment_bys)
            steps_completed = batch + 1
            time.sleep(0.1)
            # NEW: some logs are easier to read if you only log from the chief.
            if core_context.distributed.rank == 0:
                logging.info(f"x is now {x}")
            if steps_completed % 10 == 0:
                # NEW: only the chief may report training metrics and progress,
                # or upload checkpoints.
                if core_context.distributed.rank == 0:
                    core_context.train.report_training_metrics(
                        steps_completed=steps_completed, metrics={"x": x}
                    )
                    op.report_progress(steps_completed)
                    checkpoint_metadata = {"steps_completed": steps_completed}
                    with core_context.checkpoint.store_path(checkpoint_metadata) as (
                        checkpoint_directory,
                        uuid,
                    ):
                        save_state(x, steps_completed, trial_id, checkpoint_directory)
                    last_checkpoint_batch = steps_completed
                if core_context.preempt.should_preempt():
                    return
            batch += 1

        # NEW: only the chief may report validation metrics and completed operations.
        if core_context.distributed.rank == 0:
            core_context.train.report_validation_metrics(
                steps_completed=steps_completed, metrics={"x": x}
            )
            op.report_completed(x)

    # NEW: again, only the chief may upload checkpoints.
    if core_context.distributed.rank == 0 and last_checkpoint_batch != steps_completed:
        checkpoint_metadata = {"steps_completed": steps_completed}
        with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
            save_state(x, steps_completed, trial_id, path)


# NEW: Launch one process per slot.  In many distributed training frameworks, like horovod,
# torch.distributed, or deepspeed, there is a launcher of some sort provided by the framework.
# This example implements a launcher from scratch using subprocess and threading.
def launcher_main(slots_per_node, num_nodes, cross_rank):
    # Use subprocess to start one worker process per node.
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
        worker_exit = proc.wait()
        q.put((proc, worker_exit))

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


# NEW: every worker needs to create a DistributedContext to pass into core.init().
def worker_main(slots_per_node, num_nodes, cross_rank, chief_ip, rank, local_rank):
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
        main(
            core_context=core_context,
            latest_checkpoint=latest_checkpoint,
            trial_id=trial_id,
            increment_by=hparams["increment_by"],
        )


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    latest_checkpoint = info.latest_checkpoint
    trial_id = info.trial.trial_id
    hparams = info.trial.hparams

    # NEW: gather rank information from the ClusterInfo API.
    slots_per_node = len(info.slot_ids)
    num_nodes = len(info.container_addrs)
    cross_rank = info.container_rank
    chief_ip = info.container_addrs[0]

    # NEW: This script is invoked both as a launcher-of-workers, and again as each worker.
    if sys.argv[1] == "launcher":
        # Usage: SCRIPT launcher
        exitcode = launcher_main(slots_per_node, num_nodes, cross_rank)
        sys.exit(exitcode)

    if sys.argv[1] == "worker":
        # Usage: SCRIPT worker $RANK $LOCAL_RANK
        logging.info(f"worker starting")
        rank = int(sys.argv[2])
        local_rank = int(sys.argv[3])
        exitcode = worker_main(slots_per_node, num_nodes, cross_rank, chief_ip, rank, local_rank)
        sys.exit(exitcode)

    raise ValueError(f"unrecognized first argument: {sys.argv[1]}")
