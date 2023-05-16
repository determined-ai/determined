#!/usr/bin/env python3
# To run:
# python -m torch.distributed.run --nnodes=1 --nproc_per_node=2 \
#   --master_addr 127.0.0.1 --master_port 29400 --max_restarts 0 \
#   unmanaged_3_torch_distributed.py

import logging
import random

import torch.distributed as dist

import determined as det
import determined.experimental.unmanaged

config_text = """
name: unmanaged-mode-stage-3

checkpoint_storage:
  host_path: /tmp
  storage_path: determined-cp
  type: shared_fs

searcher:
   name: single
   metric: loss
   max_length: 1
"""


def main():
    logging.basicConfig(format=det.LOG_FORMAT)
    logging.getLogger("determined").setLevel(logging.INFO)

    dist.init_process_group("gloo")

    client = det.experimental.Determined()
    distributed = det.core.DistributedContext.from_torch_distributed()

    unmanaged_info = det.experimental.unmanaged.create_unmanaged_cluster_info(
        client, distributed=distributed, config_text=config_text
    )

    with det.experimental.unmanaged.init(
        distributed=distributed, unmanaged_info=unmanaged_info, client=client
    ) as core_context:
        size = dist.get_world_size()
        for i in range(100):
            if i % size == dist.get_rank():
                core_context.train.report_training_metrics(
                    steps_completed=i,
                    metrics={"loss": random.random(), "rank": dist.get_rank() + 0.01},
                )
                if (i + 1) % 10 == 0:
                    core_context.train.report_validation_metrics(
                        steps_completed=i,
                        metrics={"loss": random.random(), "rank": dist.get_rank() + 0.01},
                    )

            ckpt_metadata = {"steps_completed": i, f"rank_{dist.get_rank()}": "ok"}
            with core_context.checkpoint.store_path(ckpt_metadata, shard=True) as (path, uuid):
                with (path / f"state_{dist.get_rank()}").open("w") as fout:
                    fout.write(f"{i},{unmanaged_info.trial.trial_id},{dist.get_rank()}")

    if dist.get_rank() == 0:
        exp_id = unmanaged_info._trial_info.experiment_id
        print(
            "See the experiment at:",
            det.experimental.unmanaged.url_reverse_webui_exp_view(client, exp_id),
        )


if __name__ == "__main__":
    main()
