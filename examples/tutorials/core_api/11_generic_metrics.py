"""
Stage 1: Let's create the core_context, and use it to start logging metrics.
This is just a few lines of code, and we'll be able to see the results in the
Determined WebUI.
"""

import logging
import time

# NEW: import determined
import determined as det
from determined.common import util as det_util


def main(core_context: det.core.Context, increment_by):
    x = 0
    steps_completed = 1
    for batch in range(100):
        x += increment_by
        steps_completed = batch + 1
        time.sleep(0.1)
        logging.info(f"x is now {x}")
        # NEW: report training metrics.
        if steps_completed % 10 == 0:
            core_context.train._report_trial_metrics(
                group=det_util._LEGACY_TRAINING, total_batches=steps_completed, metrics={"x": x}
            )
    # NEW: report a "validation" metric at the end.
    core_context.train._report_trial_metrics(
        group=det_util._LEGACY_VALIDATION, total_batches=steps_completed, metrics={"x": x}
    )


if __name__ == "__main__":
    # NEW: enable logging, using the det.LOG_FORMAT.  Enabling
    # logging enables useful log messages from the determined library,
    # and det.LOG_FORMAT enables filter-by-level in the WebUI.
    logging.basicConfig(level=logging.DEBUG, format=det.LOG_FORMAT)
    # Log at different levels to demonstrate filter-by-level in the WebUI.
    logging.debug("debug-level message")
    logging.info("info-level message")
    logging.warning("warning-level message")
    logging.error("error-level message")

    # NEW: create a context, and pass it to the main function.
    with det.core.init() as core_context:
        main(core_context=core_context, increment_by=1)
