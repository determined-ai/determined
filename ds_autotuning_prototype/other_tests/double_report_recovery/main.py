import logging
import time

import determined as det


def main(core_context) -> None:
    steps_completed = 0
    core_context.train.report_validation_metrics(
        steps_completed=steps_completed, metrics={"steps_completed": steps_completed}
    )
    try:
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics={"steps_completed": steps_completed}
        )
    except det.common.api.errors.APIException:
        steps_completed += 1
        core_context.train.report_validation_metrics(
            steps_completed=steps_completed, metrics={"steps_completed": steps_completed}
        )


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    try:
        distributed = det.core.DistributedContext.from_torch_distributed()
    except KeyError:
        distributed = None
    with det.core.init(distributed=distributed) as core_context:
        main(core_context)
