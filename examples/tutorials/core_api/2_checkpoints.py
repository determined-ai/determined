#!/usr/bin/env python3

"""
Stage 2: Let's add checkpointing and preemption support to our "training" code.  After this, we will
be able to stop and continue training in two different ways: either by pausing and reactivating
training via the Determined WebUI, or by clicking the "Continue Trial" button after the experiment
has completed.

Note that these two forms of continuing have different behaviors.  While we always want to preserve
the value we are incrementing (our "model weight"), we don't always want to preserve the batch
index.  When we pause and reactive we want training to continue from the same batch index, but
when starting a fresh experiment we need training to start with a fresh batch index as well.  We
save the trial ID in the checkpoint and use that to distinguish the two types of continues.
"""

import logging
import pathlib
import sys
import time

import determined as det


# NEW: given a checkpoint_directory of type pathlib.Path, save our state to a file.
# You can save multiple files, and use any file names or directory structures.
# All files nested under `checkpoint_directory` path fill be included into the checkpoint.
def save_state(x, latest_batch, trial_id, checkpoint_directory):
    with checkpoint_directory.joinpath("state").open("w") as f:
        f.write(f"{x},{latest_batch},{trial_id}")


# NEW: given a checkpoint_directory, load our state from a file.
def load_state(trial_id, checkpoint_directory):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("state").open("r") as f:
        x, latest_batch, ckpt_trial_id = [int(field) for field in f.read().split(",")]
    if ckpt_trial_id == trial_id:
        return x, latest_batch
    else:
        # This is a new trial; load the "model weight" but not the batch count.
        return x, 0


def main(core_context, latest_checkpoint, trial_id, increment_by):
    x = 0

    # NEW: load a checkpoint if one was provided.
    starting_batch = 0
    if latest_checkpoint is not None:
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            x, starting_batch = load_state(trial_id, path)

    for batch in range(starting_batch, 100):
        x += increment_by
        time.sleep(.1)
        print("x is now", x)
        if batch % 10 == 9:
            core_context.train.report_training_metrics(latest_batch=batch, metrics={"x": x})

            # NEW: write checkpoints at regular intervals to limit lost progress in case of a crash
            # during training.
            checkpoint_metadata = {"latest_batch": batch}
            with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
                save_state(x, batch, trial_id, path)

            # NEW: check for a preemption signal.  This could originate from a higher-priority task
            # bumping us off the cluster, or for a user pausing the experiment via the WebUI or CLI.
            if core_context.preempt.should_preempt():
                # In this case, we just saved a checkpoint, so we can just shut down early and
                # pick up training again when we are reactivated.
                return

    core_context.train.report_validation_metrics(latest_batch=batch, metrics={"x": x})


if __name__ == "__main__":
    logging.basicConfig(stream=sys.stdout, level=logging.INFO)

    # NEW: use the ClusterInfo API to access information about the current running task.  We
    # choose to extract the information we need from the ClusterInfo API here and pass it into
    # main() so that you could eventually write your main() to run on- or off-cluster.
    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    latest_checkpoint = info.latest_checkpoint
    trial_id = info.trial.trial_id

    with det.core.init() as core_context:
        main(
            core_context=core_context,
            latest_checkpoint=latest_checkpoint,
            trial_id=trial_id,
            increment_by=1
        )
