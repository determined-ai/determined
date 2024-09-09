"""
Stage 3: Let's add hyperparameter search to our model.
"""

import logging
import pathlib
import sys
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
    max_length = 100

    starting_batch = 0
    if latest_checkpoint is not None:
        with core_context.checkpoint.restore_path(latest_checkpoint) as path:
            x, starting_batch = load_state(trial_id, path)

    for batch in range(starting_batch, max_length):
        x += increment_by
        steps_completed = batch + 1
        time.sleep(0.1)
        logging.info(f"x is now {x}")
        if steps_completed % 10 == 0:
            core_context.train.report_training_metrics(
                steps_completed=steps_completed, metrics={"x": x}
            )
            core_context.train.report_progress(steps_completed / float(max_length))

            # NEW: periodically report validation metrics, which the searcher
            # may monitor for the purpose of early-stopping.
            # XXX: need a time metric too
            core_context.train.report_validation_metrics(
                steps_completed=steps_completed, metrics={"x": x}
            )

            checkpoint_metadata = {"steps_completed": steps_completed}
            with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
                save_state(x, steps_completed, trial_id, path)
            if core_context.preempt.should_preempt():
                return


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    latest_checkpoint = info.latest_checkpoint
    trial_id = info.trial.trial_id

    # NEW: get the hyperaparameter values chosen for this trial.
    hparams = info.trial.hparams

    with det.core.init() as core_context:
        main(
            core_context=core_context,
            latest_checkpoint=latest_checkpoint,
            trial_id=trial_id,
            # NEW: configure the "model" using hparams.
            increment_by=hparams["increment_by"],
        )
