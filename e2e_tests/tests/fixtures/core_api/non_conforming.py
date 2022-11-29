import determined as det


# NEW: given a checkpoint_directory of type pathlib.Path, save our state to a file.
# You can save multiple files, and use any file names or directory structures.
# All files nested under `checkpoint_directory` path will be included into the checkpoint.
def save_state(x, steps_completed, trial_id, checkpoint_directory):
    with checkpoint_directory.joinpath("state").open("w") as f:
        f.write(f"{x},{steps_completed},{trial_id}")


with det.core.init() as core_context:
    core_context.train.report_validation_metrics(
        steps_completed=2, metrics={"x": 0}
    )
    core_context.train.report_training_metrics(
        steps_completed=3, metrics={"x": 1}
    )

    checkpoint_metadata = {"steps_completed": 4}
    with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
        with path.joinpath("state").open("w") as f:
            f.write(f"hello, world")