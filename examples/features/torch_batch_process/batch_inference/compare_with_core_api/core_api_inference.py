import json
import logging
import math
import os
import pathlib
from typing import Any, Optional

import constants
import filelock
import model
import torch
import torch.distributed as dist
import torchvision as tv
from torch import nn
from torch.utils import data
from torchvision import transforms

import determined as det
from determined import core, pytorch

logging.getLogger().setLevel(logging.INFO)


def run_inference(
    ml_model: nn.Module,
    data_loader: Any,
    context: core.Context,
    rank: int,
    skip: int,
    per_worker_iterate_length: int,
    pred_dir: pathlib.Path,
    checkpoint_interval: int,
) -> None:
    ml_model.eval()
    dataloader_iterator = iter(data_loader)

    with torch.no_grad():
        last_checkpoint_step = skip
        steps_completed = skip
        for batch_idx in range(skip, per_worker_iterate_length):
            X = next(dataloader_iterator, None)
            if X is not None:
                data, label = X
                output = ml_model(data)
                preds = output.argmax(dim=1, keepdim=True)

                file_name = f"inference_out_{rank}_{batch_idx}.json"

                output = []

                for pred in preds:
                    output.append(pred[0].item())

                with open(os.path.join(pred_dir, f"{file_name}"), "w") as f:
                    json.dump({"predictions": output}, f)

                steps_completed = batch_idx + 1
                logging.info(f"Completed step {steps_completed}")

            if steps_completed % checkpoint_interval == 0:
                checkpoint(steps_completed, context)
                last_checkpoint_step = steps_completed
                if context.preempt.should_preempt():
                    return

        if steps_completed > last_checkpoint_step:
            checkpoint(steps_completed, context)


def checkpoint(steps_completed: int, context: core.Context):
    if context.distributed.rank == 0:
        context.distributed.gather(steps_completed)
        checkpoint_metadata = {
            "steps_completed": steps_completed,
        }
        with context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
            with open(os.path.join(path, "steps_completed"), "w") as f:
                f.write(str(steps_completed))
    else:
        context.distributed.gather(steps_completed)


def get_data_loader(
    batch_size: int, total_worker: int, rank: int, data_dir: pathlib.Path, skip: int
) -> [Any, int]:
    transform = transforms.Compose(
        [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
    )

    lock = filelock.FileLock(constants.LOCK_FILE)

    with lock:
        inference_data = tv.datasets.CIFAR10(
            root=data_dir, train=False, download=True, transform=transform
        )

    sampler = data.SequentialSampler(inference_data)
    sampler = data.BatchSampler(sampler, batch_size=batch_size, drop_last=False)
    sampler = pytorch.samplers.DistributedBatchSampler(sampler, total_worker, rank)
    dataloader = data.DataLoader(inference_data, batch_sampler=sampler)

    # Enumerate over dataloader directly may cause some workers to iterate for 1 more time
    # than others when drop_last = False. If those workers synchronize on the last batch_idx,
    # they would hang forever as other workers never hit that last batch_idx.
    # To avoid the issue, we calculate and take the ceiling of the iteration count to ensure
    # all workers iterate for the same number of times.
    per_worker_iterate_length = math.ceil(len(inference_data) / batch_size / total_worker)
    logging.info(f"per_worker_iterate_length is {per_worker_iterate_length}")
    return dataloader, per_worker_iterate_length


def load_state(checkpoint_directory: str):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with open(os.path.join(checkpoint_directory, "steps_completed"), "r") as f:
        return int(f.read())


def initialize_distributed_backend() -> Optional[core.DistributedContext]:
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
    slots_per_node = len(info.slot_ids)
    num_nodes = len(info.container_addrs)
    total_worker = num_nodes * slots_per_node
    rank = context.distributed.get_rank()

    latest_checkpoint = info.latest_checkpoint
    steps_completed = 0
    if latest_checkpoint is not None:
        logging.info("Checkpoint is not none")
        with context.checkpoint.restore_path(latest_checkpoint) as path:
            steps_completed = load_state(path)
            logging.info(f"Steps completed {steps_completed}")

    # The first worker will create these directories is they do not already exist
    pathlib.Path.mkdir(pathlib.Path(constants.PREDICTIONS_DIRECTORY), parents=True, exist_ok=True)
    pathlib.Path.mkdir(pathlib.Path(constants.DATA_DIRECTORY), parents=True, exist_ok=True)

    data_loader, per_worker_iterate_length = get_data_loader(
        batch_size, total_worker, rank, constants.DATA_DIRECTORY, skip=steps_completed
    )
    run_inference(
        model.build_model(),
        data_loader,
        context,
        rank,
        steps_completed,
        per_worker_iterate_length,
        constants.PREDICTIONS_DIRECTORY,
        5,
    )


if __name__ == "__main__":
    distributed = initialize_distributed_backend()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context)
