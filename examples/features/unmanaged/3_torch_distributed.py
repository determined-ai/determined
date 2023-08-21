#!/usr/bin/env python3
# To run:
# python -m torch.distributed.run --nnodes=1 --nproc_per_node=2 \
#  --master_addr 127.0.0.1 --master_port 29400 --max_restarts 0 \
#  3_torch_distributed.py

import logging
import random

import torch.distributed as dist

import determined
import determined.core
from determined.experimental import core_v2


def main():
    dist.init_process_group("gloo")

    logging.basicConfig(format=determined.LOG_FORMAT)
    logging.getLogger("determined").setLevel(logging.INFO)
    distributed = core_v2.DistributedContext.from_torch_distributed()
    core_v2.init(
        defaults=core_v2.DefaultConfig(
            name="unmanaged-3-torch-distributed",
        ),
        distributed=distributed,
    )

    # Use framework-native dtrain utilities, as normal.
    size = dist.get_world_size()
    for i in range(100):
        if i % size == dist.get_rank():
            core_v2.train.report_training_metrics(
                steps_completed=i,
                metrics={"loss": random.random(), "rank": dist.get_rank() + 0.01},
            )
            if (i + 1) % 10 == 0:
                core_v2.train.report_validation_metrics(
                    steps_completed=i,
                    metrics={"loss": random.random(), "rank": dist.get_rank() + 0.01},
                )

        ckpt_metadata = {"steps_completed": i, f"rank_{dist.get_rank()}": "ok"}
        with core_v2.checkpoint.store_path(ckpt_metadata, shard=True) as (path, uuid):
            with (path / f"state_{dist.get_rank()}").open("w") as fout:
                fout.write(f"{i},{dist.get_rank()}")

    if dist.get_rank() == 0:
        print(
            "See the experiment at:",
            core_v2.url_reverse_webui_exp_view(),
        )

    core_v2.close()


if __name__ == "__main__":
    main()
