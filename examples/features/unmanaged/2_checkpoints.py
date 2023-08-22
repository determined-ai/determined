#!/usr/bin/env python3

import random

from determined.experimental import core_v2


def main():
    core_v2.init(
        defaults=core_v2.DefaultConfig(
            name="unmanaged-2-checkpoints",
            # We allow configuring the local checkpoint storage directory.
            # checkpoint_storage="/tmp/determined-cp",
        ),
        unmanaged=core_v2.UnmanagedConfig(
            external_experiment_id="test-unmanaged-2-checkpoints",
            external_trial_id="test-unmanaged-2-checkpoints",
            # e.g., requeued jobs on slurm:
            # external_experiment_id=f"some-prefix-{os.environ[SLURM_JOB_ID}",
            # external_trial_id=f"some-prefix-{os.environ[SLURM_JOB_ID}",
        ),
    )

    latest_checkpoint = core_v2.info.latest_checkpoint
    initial_i = 0
    if latest_checkpoint is not None:
        with core_v2.checkpoint.restore_path(latest_checkpoint) as path:
            with (path / "state").open() as fin:
                ckpt = fin.read()
                print("Checkpoint contents:", ckpt)

                i_str, _ = ckpt.split(",")
                initial_i = int(i_str)

    print("determined experiment id: ", core_v2.info._trial_info.experiment_id)
    print("initial step:", initial_i)
    for i in range(initial_i, initial_i + 100):
        core_v2.train.report_training_metrics(steps_completed=i, metrics={"loss": random.random()})
        if (i + 1) % 10 == 0:
            loss = random.random()
            core_v2.train.report_validation_metrics(steps_completed=i, metrics={"loss": loss})

            with core_v2.checkpoint.store_path({"steps_completed": i}) as (path, uuid):
                with (path / "state").open("w") as fout:
                    fout.write(f"{i},{loss}")

    core_v2.close()


if __name__ == "__main__":
    main()
