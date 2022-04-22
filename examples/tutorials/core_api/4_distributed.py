#!/usr/bin/env python3

"""
Stage 4: Let's do all the same things, but across multiple workers.

Multi-slot tasks in Determined get the following features out-of-the box:
- batch scheduling
- IP address coordination between workers
- distributed communication primitives
- coordinated cross-worker preemption support
- checkpoint download sharing between workers on the same node
"""

import logging
import pathlib
import sys
import time
import multiprocessing

import determined as det


def save_state(x, latest_batch, trial_id, checkpoint_directory):
    with checkpoint_directory.joinpath("state").open("w") as f:
        f.write(f"{x},{latest_batch},{trial_id}")


def load_state(trial_id, checkpoint_directory):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("state").open("r") as f:
        x, latest_batch, ckpt_trial_id = [int(field) for field in f.read().split(",")]
    if ckpt_trial_id == trial_id:
        return x, latest_batch
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
            # NEW: we're going to increment by the sum of every worker's increment_by value.
            # In reality, that sum is just increment_by*num_workers, but the point is to show how we
            # can use the communication primitives.
            all_increment_bys = core_context.distributed.allgather(increment_by)
            x += sum(all_increment_bys)
            time.sleep(.1)
            # NEW: some logs are easier to read if we only print diagnostics from the chief.
            if core_context.distributed.rank == 0:
                print("x is now", x)
            batch += 1
            if batch % 10 == 0:
                # NEW: only the chief reports training metrics and progress, and uploads checkpoints.
                if core_context.distributed.rank == 0:
                    core_context.train.report_training_metrics(latest_batch=batch, metrics={"x": x})
                    op.report_progress(batch)
                    checkpoint_metadata = {"latest_batch": batch}
                    with core_context.checkpoint.store_path(checkpoint_metadata) as (checkpoint_directory, uuid):
                        save_state(x, batch, trial_id, path)
                    last_checkpoint_batch = batch
                if core_context.preempt.should_preempt():
                    return
        # NEW: only the chief is able to report_validation_metrics and report_completed.
        if core_context.distributed.rank == 0:
            core_context.train.report_validation_metrics(latest_batch=batch, metrics={"x": x})
            op.report_completed(x)

    # NEW: again, only the chief uploads checkpoints.
    if core_context.distributed.rank == 0 and last_checkpoint_batch != batch:
        checkpoint_metadata = {"latest_batch": batch}
        with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
            save_state(x, batch, trial_id, path)


if __name__ == "__main__":
    logging.basicConfig(stream=sys.stdout, level=logging.INFO)

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    latest_checkpoint = info.latest_checkpoint
    trial_id = info.trial.trial_id
    hparams = info.trial.hparams

    # NEW: we are going to launch one process per slot.  In many distributed training frameworks,
    # like horovod, torch.distributed, or deepspeed, there is a launch script of some sort.  But in
    # our case, we'll just use multiprocessing.Process.
    #
    # Ultimately, we'll need to create a DistributedContext on each worker to pass into core.init().
    # In the absence of a distributed training framework that might decide
    # rank/local_rank/cross_rank for us, we can choose values from the ClusterInfo API.
    slots_per_node = len(info.slot_ids)
    num_nodes = len(info.container_addrs)
    cross_rank = info.container_rank

    def worker_fn(local_rank):
        rank = cross_rank * slots_per_node + local_rank
        print(f"worker starting with rank={rank}")
        distributed = det.core.DistributedContext(
            rank=rank,
            size=num_nodes * slots_per_node,
            local_rank=local_rank,
            local_size=slots_per_node,
            cross_rank=cross_rank,
            cross_size=num_nodes,
            chief_ip=info.container_addrs[0],
        )
        with det.core.init(distributed=distributed) as core_context:
            main(
                core_context=core_context,
                latest_checkpoint=latest_checkpoint,
                trial_id=trial_id,
                increment_by=hparams["increment_by"],
            )

    procs = [
        multiprocessing.Process(target=worker_fn, args=(local_rank,))
        for local_rank in range(slots_per_node)
    ]

    for p in procs:
        p.start()

    for p in procs:
        p.join()
