import random
from determined.experimental import core_v2

def main():
    # Initialize the trial / session
    core_v2.init(
        # For managed experiments, the below will be overridden by the yaml config.
        defaults=core_v2.Config(
            name="detached_mode_example",
        ),
        # `UnmanagedConfig` values will not get merged, and will only be used in the unmanaged mode.
        unmanaged=core_v2.UnmanagedConfig(
            workspace="guangqing1",
            project="tt",
            # external_experiment_id="ext_experiment_1",
            # external_trial_id="ext_trial_1"
        )
    )
    for i in range(100, 200):
        # Report training metrics to the trial initialized above
        core_v2.train.report_metrics(group="high_freq", steps_completed=i, metrics={"loss": random.random()})

        # Report validation metrics every 10 steps and then checkpoint
        if (i + 1) % 10 == 0:
            loss = random.random()
            print(f"loss is: {loss}")
            core_v2.train.report_metrics(
                group="low_freq", steps_completed=i, metrics={"loss": random.random()}
            )
            ckpt_metadata = {"steps_completed": i}
            with core_v2.checkpoint.store_path(ckpt_metadata, shard=False) as (path, uuid):
                print(f"path: {path} uuid: {uuid}")
                with (path/uuid).open("w") as fout:
                    fout.write(f"{i}")

    core_v2.close()


if __name__ == "__main__":
    main()
