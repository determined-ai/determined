import logging
import random
import time

import determined as det
from determined.common import util as det_util

metric_groups = [
    det_util._LEGACY_TRAINING,
    det_util._LEGACY_VALIDATION,
    "group_b",
    "grou%p_c",
    "infer ence",
    "inf%er en/ce",
]


def main(core_context: det.core.Context, increment_by: float):
    x = 0
    total_batches = 1
    for batch in range(100):
        x += increment_by
        total_batches = batch + 1
        time.sleep(0.1)
        logging.info(f"x is now {x}")
        idx = batch % len(metric_groups)
        group = metric_groups[idx]
        noise = random.random() * x
        metrics = {f"z{group}/me.t r%i]\\c_{i}": x * (i + 1) + noise for i in range(3)}
        metrics.update({f"m{i}": x * (i + 2) + noise for i in range(3)})
        core_context.train._report_trial_metrics(
            group=group, total_batches=total_batches, metrics=metrics
        )


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG, format=det.LOG_FORMAT)
    with det.core.init() as core_context:
        main(core_context=core_context, increment_by=1)
