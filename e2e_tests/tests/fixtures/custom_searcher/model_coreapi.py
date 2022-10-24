import logging
import pathlib

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
            x += increment_by
            steps_completed = batch + 1
            if steps_completed % 100 == 0:
                core_context.train.report_training_metrics(
                    steps_completed=steps_completed, metrics={"validation_error": x}
                )

                op.report_progress(batch)

                checkpoint_metadata = {"steps_completed": steps_completed}
                with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
                    save_state(x, steps_completed, trial_id, path)
                last_checkpoint_batch = steps_completed
                if core_context.preempt.should_preempt():
                    return
            batch += 1

        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics={"validation_error": x}
        )
        op.report_completed(x)

    if last_checkpoint_batch != steps_completed:
        checkpoint_metadata = {"steps_completed": steps_completed}
        with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
            save_state(x, steps_completed, trial_id, path)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    latest_checkpoint = info.latest_checkpoint
    trial_id = info.trial.trial_id
    hparams = info.trial.hparams

    with det.core.init() as core_context:
        main(core_context, latest_checkpoint, trial_id, increment_by=hparams["increment_by"])
