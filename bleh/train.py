import logging

import determined as det
from determined import core

logging.basicConfig(level=logging.INFO)

if __name__ == "__main__":
    with core.init() as core_context:
        for i in range(100000):
            core_context.train.report_training_metrics(
                steps_completed=i, metrics={"loss": 1}
            )
