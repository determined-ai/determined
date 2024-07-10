#!/usr/bin/env python3
#
# Demonstrate an experiment which errors out.
import os
import random
import tempfile

from determined.experimental import core_v2


def main():
    assert "DET_TEST_EXTERNAL_EXP_ID" in os.environ
    name = os.environ["DET_TEST_EXTERNAL_EXP_ID"]

    checkpoint_storage = tempfile.mkdtemp()
    core_v2.init(
        config=core_v2.Config(
            name=name,
            external_experiment_id=name,
            external_trial_id=name,
        ),
        checkpoint_storage=checkpoint_storage,
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
