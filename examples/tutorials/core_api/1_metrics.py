#!/usr/bin/env python3

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


def main(core_context, increment_by):
    x = 0
    for batch in range(100):
        x += increment_by
        time.sleep(.1)
        print("x is now", x)
        # NEW: report training metrics to Determined.
        if batch % 10 == 9:
            core_context.train.report_training_metrics(latest_batch=batch, metrics={"x": x})
    # NEW: report a "validation" metric at the end.
    core_context.train.report_validation_metrics(latest_batch=batch, metrics={"x": x})


if __name__ == "__main__":
    logging.basicConfig(stream=sys.stdout, level=logging.INFO)

    # NEW: create a context, and pass it to the main function.
    with det.core.init() as core_context:
        main(core_context=core_context, increment_by=1)
