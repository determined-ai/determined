#!/usr/bin/env python3

import logging
import random

import determined as det
import determined.experimental.unmanaged

config_text = """
name: unmanaged-mode-stage-2

checkpoint_storage:
  host_path: /tmp
  storage_path: determined-cp
  type: shared_fs

searcher:
   name: custom
   metric: loss
"""


def runner(client: det.experimental.Determined, exp_id: int, hparams: dict = {}):
    unmanaged_info = det.experimental.unmanaged.create_unmanaged_trial_cluster_info(
        client, config_text, exp_id, hparams=hparams
    )

    with det.experimental.unmanaged.init(
        unmanaged_info=unmanaged_info, client=client
    ) as core_context:
        for i in range(100):
            core_context.train.report_training_metrics(
                steps_completed=i, metrics={"loss": random.random()}
            )
            if (i + 1) % 10 == 0:
                core_context.train.report_validation_metrics(
                    steps_completed=i, metrics={"loss": random.random()}
                )

                with core_context.checkpoint.store_path({"steps_completed": i}) as (path, uuid):
                    with (path / "state").open("w") as fout:
                        fout.write(f"{i},{unmanaged_info.trial.trial_id}")


def main():
    logging.basicConfig(format=det.LOG_FORMAT)
    logging.getLogger("determined").setLevel(logging.INFO)
    client = det.experimental.Determined()

    exp_id = det.experimental.unmanaged.create_unmanaged_experiment(client, config_text=config_text)
    print(f"Created experiment {exp_id}")

    # Grid search.
    for i in range(4):
        runner(client, exp_id, {"i": i})

    print(
        "See the experiment at:",
        det.experimental.unmanaged.url_reverse_webui_exp_view(client, exp_id),
    )


if __name__ == "__main__":
    main()
