"""
Stage 1: Let's create the core_context, and use it to start logging metrics.
This is just a few lines of code, and we'll be able to see the results in the
Determined WebUI.
"""

import logging
import sys
import time

# NEW: import determined
import determined as det
from determined import profiler

def main(core_context, increment_by):
    for batch in range(100):
        with core_context.profiler.record_timing("train_batch.backward"):
            steps_completed = batch + 1
            time.sleep(1)
            logging.info(f"batch is now {batch}")
        core_context.profiler.record_metric("test metric", 1.0)
        # NEW: report training metrics.
        if steps_completed % 10 == 0:
            core_context.train.report_training_metrics(
                steps_completed=steps_completed, metrics={"x": batch}
            )
    # NEW: report a "validation" metric at the end.
    core_context.train.report_validation_metrics(
        steps_completed=steps_completed, metrics={"x": 100}
    )


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    # NEW: create a context, and pass it to the main function.
    with det.core.init() as core_context:
        main(core_context=core_context, increment_by=1)
