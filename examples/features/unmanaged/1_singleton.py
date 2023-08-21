#!/usr/bin/env python3

import random

from determined.experimental import core_v2


def main():
    core_v2.init(
        # For managed experiments, will be overridden by the yaml config.
        # Future: merge this and yaml configs field-by-field at runtime.
        defaults=core_v2.DefaultConfig(
            name="unmanaged-1-singleton",
            # labels=["some", "set", "of", "labels"],
            # description="some description",
        ),
        # `UnmanagedConfig` values will not get merged, and will only be used in the unmanaged mode.
        # unmanaged=core_v2.UnmanagedConfig(
        #   workspace="...",
        #   project="...",
        # )
    )

    for i in range(100):
        print(f"training loss: {random.random()}")

        core_v2.train.report_training_metrics(steps_completed=i, metrics={"loss": random.random()})

        if (i + 1) % 10 == 0:
            print(f"validation loss: {random.random()}")

            core_v2.train.report_validation_metrics(
                steps_completed=i, metrics={"loss": random.random()}
            )

    core_v2.close()


if __name__ == "__main__":
    main()
