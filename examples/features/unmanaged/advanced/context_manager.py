#!/usr/bin/env python3

import random

from determined.experimental import core_v2


def main():
    with core_v2.init_context(
        defaults=core_v2.DefaultConfig(
            name="unmanaged-context-manager",
        ),
    ) as core_context:
        for i in range(100):
            print(f"training loss: {random.random()}")

            core_context.train.report_training_metrics(
                steps_completed=i, metrics={"loss": random.random()}
            )

            if (i + 1) % 10 == 0:
                print(f"validation loss: {random.random()}")

                core_context.train.report_validation_metrics(
                    steps_completed=i, metrics={"loss": random.random()}
                )


if __name__ == "__main__":
    main()
