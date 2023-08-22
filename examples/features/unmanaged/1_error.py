#!/usr/bin/env python3
#
# Demonstrate an experiment which errors out.

import random

from determined.experimental import core_v2


def main():
    core_v2.init(
        defaults=core_v2.DefaultConfig(
            name="unmanaged-1-error",
        ),
    )

    for i in range(100):
        core_v2.train.report_training_metrics(steps_completed=i, metrics={"loss": random.random()})
        if i == 15:
            raise ValueError("oops")
        if (i + 1) % 10 == 0:
            core_v2.train.report_validation_metrics(
                steps_completed=i, metrics={"loss": random.random()}
            )

    core_v2.close()


if __name__ == "__main__":
    main()
