from __future__ import print_function

import json
import logging
import math
import os
import pathlib
from typing import Any, Optional

import filelock
import model
import torch
import torch.distributed as dist
import torchvision as tv
import torchvision.transforms as transforms
from torch import nn

import determined as det
from determined import core, pytorch


def run_inference(
    model: nn.module,
    data_loader: Any,
    context: core.Context,
    rank: int,
    skip: int,
    per_worker_iterate_length: int,
) -> None:
    model.eval()
    records_processed = 0

    # I set up my AWS cluster with file system, this is where the fs is mounted to my container
    inference_output_dir = "/run/determined/workdir/shared_fs/inference_out/"
    # The first worker will create it, and exist_ok option makes sure subsequent workers
    # do not run into error
    pathlib.Path.mkdir(pathlib.Path(inference_output_dir), parents=True, exist_ok=True)

    dataloader_iterator = iter(data_loader)

    with torch.no_grad():
        for batch_idx in range(skip, per_worker_iterate_length):
            X = next(dataloader_iterator, None)
            logging.info(f"Working on batch is {batch_idx}")
            if X is not None:
                data, label = X
                output = model(data)
                preds = output.argmax(dim=1, keepdim=True)

                file_name = f"inference_out_{rank}_{batch_idx}.json"

                output = []

                for pred in preds:
                    output.append(pred[0].item())

                with open(
                    os.path.join(inference_output_dir, f"{file_name}"),
                    "w",
                ) as f:
                    json.dump({"predictions": output}, f)

                # After each batch, synchronize and update number of catches completed
                if context.distributed.rank == 0:
                    work_completed_this_round = sum(context.distributed.gather(len(data)))
                    records_processed += work_completed_this_round
                    checkpoint_metadata = {
                        "steps_completed": batch_idx,
                        "records_processed": records_processed,
                    }
                    with context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
                        with open(os.path.join(path, "batch_completed.json"), "w") as file_obj:
                            json.dump({"batch_completed": batch_idx}, file_obj)
                else:
                    context.distributed.gather(len(data))

                if context.preempt.should_preempt():
                    return


def _get_data_loader(batch_size: int, total_worker: int, rank: int, skip: int) -> [Any, int]:
    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
    )
    with filelock.FileLock(os.path.join("/tmp", "inference.lock")):
        inference_data = tv.datasets.CIFAR10(
            root="/data", train=False, download=True, transform=transform
        )
    dataloader = pytorch.DataLoader(
        dataset=inference_data,
        batch_size=batch_size,
        shuffle=False,
    ).get_data_loader(repeat=False, skip=skip, num_replicas=total_worker, rank=rank)

    # Enumerate over dataloader directly may cause some workers to iterate for 1 more time
    # than others when drop_last = False. If those workers synchronize on the last batch_idx,
    # they would hang forever as other workers never hit that last batch_idx.
    # To avoid the issue, we calculate and take the ceiling of the iteration count to ensure
    # all workers iterate for the same number of times.
    per_worker_iterate_length = math.ceil(len(inference_data) / batch_size / total_worker)

    return dataloader, per_worker_iterate_length


def _load_state(checkpoint_directory: str):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("metadata.json").open("r") as f:
        metadata = json.load(f)
        return metadata


def _initialize_distributed_backend() -> Optional[core.DistributedContext]:
    # Pytorch specific initialization
    if torch.cuda.is_available():
        dist.init_process_group(
            backend="nccl",
        )  # type: ignore
        return core.DistributedContext.from_torch_distributed()
    else:
        dist.init_process_group(backend="gloo")  # type: ignore
    return core.DistributedContext.from_torch_distributed()


def main(context: core.Context):
    batch_size = 200
    info = det.get_cluster_info()
    total_worker = len(info.container_addrs)
    rank = context.distributed.get_rank()
    latest_checkpoint = info.latest_checkpoint
    skip = 0
    if latest_checkpoint is not None:
        logging.info("Checkpoint is not none")
        with context.checkpoint.restore_path(latest_checkpoint) as path:
            metadata = _load_state(path)
            steps_completed = metadata["steps_completed"]
            skip = steps_completed
            logging.info(f"Steps completed {steps_completed}")

    data_loader, per_worker_iterate_length = _get_data_loader(
        batch_size, total_worker, rank, skip=skip
    )
    run_inference(model.get_model(), data_loader, context, rank, skip, per_worker_iterate_length)


if __name__ == "__main__":
    distributed = _initialize_distributed_backend()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context)
