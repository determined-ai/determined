#!/usr/bin/env python3

import random

from determined.experimental import core_v2


def main():
    core_v2.init(
        # For managed experiments, will be overridden by the yaml config.
        config=core_v2.Config(
            name="unmanaged-1-singleton",
            # labels=["some", "set", "of", "labels"],
            # description="some description",
            # workspace="...",
            # project="...",
        ),
    )
    max_length = 100
    for i in range(max_length):
        print(f"training loss: {random.random()}")

        core_v2.train.report_training_metrics(steps_completed=i, metrics={"loss": random.random()})

        if (i + 1) % 10 == 0:
            print(f"validation loss: {random.random()}")

            core_v2.train.report_validation_metrics(
                steps_completed=i, metrics={"loss": random.random()}
            )
            core_v2.train.report_progress(i / float(max_length))

    core_v2.close()


if __name__ == "__main__":
    main()
