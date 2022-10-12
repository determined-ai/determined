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

def main(core_context):
    with core_context.profiler.init(enabled, begin_on_batch, end_after_batch) as profiler:
        for batch in range(100):
            with profiler.record_timing("batch"):
                print(batch)
            profiler.step_batch()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    # NEW: create a context, and pass it to the main function.
    with det.core.init() as core_context:
        main(core_context=core_context)
