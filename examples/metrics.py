import logging
import random

import determined as det


def main(core_context, increment_by):
    N = 5
    losses = [random.random() for i in range(N)]
    factors = [(random.random(), random.random() / 10) for i in range(N)]
    training = {
        'loss': 0,
        'loss2': 1,
        'loss3': 2,
    }
    validation = {
        'loss': 0,
        'loss2': 3,
        'loss3': 4,
    }

    for batch in range(15000):
        steps_completed = batch + 1
        for i in range(N):
            losses[i] = losses[i] * (1 - (1 if random.random() > factors[i][0] else -1) * random.random() * factors[i][1])

        training_dict = {k: losses[training[k]] for k in training}
        core_context.train.report_training_metrics(
                steps_completed=steps_completed, metrics=training_dict
        )

        val_dict = {k: losses[validation[k]] for k in validation}
        core_context.train.report_validation_metrics(
                steps_completed=steps_completed, metrics=val_dict
        )


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    with det.core.init() as core_context:
        main(core_context=core_context, increment_by=1)