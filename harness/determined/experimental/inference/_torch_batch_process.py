import abc
import logging
import math
import json
import os
import pathlib
from dataclasses import dataclass

import torch
import torch.distributed as dist

from torch.utils.data import Dataset
from typing import Callable, Optional, Type

import determined as det
from determined import core
from determined.common import set_logger
from determined.pytorch import DataLoader
from determined.tensorboard.util import get_rank_aware_path

set_logger(False)


@dataclass
class TorchBatchInitInfo:
    worker_rank: int
    default_device: torch.device
    tensorboard_path: str


def get_default_device(core_context) -> torch.device:
    local_rank = core_context.distributed.local_rank
    local_num_gpu = torch.cuda.device_count()
    if local_rank >= local_num_gpu:
        return torch.device("cpu")
    else:
        return torch.device("cuda", local_rank)


def initialize_distributed_backend() -> Optional[core.DistributedContext]:
    info = det.get_cluster_info()

    distributed_backend = det._DistributedBackend()
    if distributed_backend.use_torch():
        if torch.cuda.is_available():
            dist.init_process_group(backend="nccl")  # type: ignore
        else:
            dist.init_process_group(backend="gloo")  # type: ignore
        return core.DistributedContext.from_torch_distributed()
    elif info and (len(info.container_addrs) > 1 or len(info.slot_ids) > 1):
        raise ValueError(
            "In multi-slot managed cluster training, you must wrap your training script with a "
            "distributed launch layer such as determined.launch.torch_distributed"
        )
    return None


def initialize_default_inference_context() -> core.Context:
    distributed_context = initialize_distributed_backend()
    # Setting preempt mode to WorkerAskMaster makes the call non-blocking
    # We are also ok if workers are preempted at different batch idx
    return det.core.init(
        distributed=distributed_context, preempt_mode=core.PreemptMode.WorkersAskMaster
    )


class TorchBatchProcessor(metaclass=abc.ABCMeta):
    """
    User can initialize necessary resources in the init function, such as
    - model for prediction
    - storage client (e.g. s3 client)
    """

    @abc.abstractmethod
    def __init__(self, core_context: core.Context, init_info: TorchBatchInitInfo) -> None:
        pass

    @abc.abstractmethod
    def process_batch(self, batch, batch_idx) -> None:
        pass


def _load_state(checkpoint_directory):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("metadata.json").open("r") as f:
        metadata = json.load(f)
        return metadata


def _synchronize_and_checkpoint(core_context: core.Context, batch_idx: int, rank: int):
    """
    Synchronize the workers and create checkpoint to record steps completed
    """
    if rank == 0:
        # Simply need to gather, no need to pass information around
        core_context.distributed.gather(None)
        checkpoint_metadata = {
            "steps_completed": batch_idx + 1,
        }
        with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
            with open(os.path.join(path, "batch_completed.json"), "w") as file_obj:
                json.dump({"batch_completed": batch_idx}, file_obj)
    else:
        core_context.distributed.gather(None)


def _report_progress_to_master(
    searcher_op: core.DummySearcherOperation,
    batch_idx: int,
    total_worker: int,
    batch_size: int,
    dataset_len: int,
) -> None:
    records_processed = batch_idx * total_worker * batch_size
    completion_rate = records_processed / dataset_len
    searcher_op.report_progress(completion_rate)


def torch_batch_process(
    batch_processor_cls: Type[TorchBatchProcessor],
    dataset: Dataset,
    batch_size: int = 64,
    checkpoint_interval: int = 5,
    dataloader_num_workers: int = 2,
    dataloader_collate_fn: Callable = None,
    dataloader_worker_init_fn: Callable = None,
    dataloader_drop_last=False,
):
    with initialize_default_inference_context() as core_context:
        dataset_len = len(dataset)

        info = det.get_cluster_info()
        slots_per_node = len(info.slot_ids)
        num_nodes = len(info.container_addrs)
        total_worker = num_nodes * slots_per_node
        # Get global rank
        rank = core_context.distributed.get_rank()
        latest_checkpoint = info.latest_checkpoint
        skip = 0

        per_batch_processor = batch_processor_cls(
            core_context=core_context,
            init_info=TorchBatchInitInfo(
                worker_rank=rank,
                default_device=get_default_device(core_context),
                tensorboard_path=core_context.train.get_tensorboard_path(),
            ),
        )

        # Check if previous checkpoint exists
        if latest_checkpoint is not None:
            logging.info("Checkpoint is not none")
            with core_context.checkpoint.restore_path(latest_checkpoint) as path:
                metadata = _load_state(path)
                skip = metadata["steps_completed"]
                logging.info(f"Previous run completed {skip} steps")

        dataloader = DataLoader(
            dataset=dataset,
            batch_size=batch_size,
            shuffle=False,
            num_workers=dataloader_num_workers,
            collate_fn=dataloader_collate_fn,
            worker_init_fn=dataloader_worker_init_fn,
            drop_last=dataloader_drop_last,
        ).get_data_loader(repeat=False, skip=skip, num_replicas=total_worker, rank=rank)

        # Create dummy searcher op to report progress to master
        dummy_searcher_op = None
        # Initialize dummy searcher for progress report
        if rank == 0:
            dummy_searcher_op = core.DummySearcherOperation(1, True)

        dataloader_iterator = iter(dataloader)

        # Enumerate over dataloader directly may cause some workers to iterate for 1 more time
        # than others when drop_last = False. If those workers synchronize on the last batch_idx,
        # they would hang forever as other workers never hit that last batch_idx.
        # To avoid the issue, we calculate and take the ceiling of the iteration count to ensure
        # all workers iterate for the same number of times.
        times_iterate = math.ceil(dataset_len / batch_size / total_worker)
        for batch_idx in range(skip, times_iterate):
            logging.info(f"Currently processing batch {batch_idx}")
            X = next(dataloader_iterator, None)
            if X is not None:
                per_batch_processor.process_batch(batch=X, batch_idx=batch_idx)

            if (batch_idx + 1) % checkpoint_interval == 0:
                _synchronize_and_checkpoint(core_context, batch_idx, rank)
                # Report progress can only be done accurately with synchronization
                core_context._tensorboard_manager.sync(mangler=get_rank_aware_path)
                if rank == 0:
                    _report_progress_to_master(
                        dummy_searcher_op, batch_idx, total_worker, batch_size, dataset_len
                    )

            # If preempt mode is set to WorkerAskMaster, checking should preempt is cheap
            # Calling preempt without synchronization means workers can be on different
            # batch idx when preempted. It is ok since when we resume, we will resume from
            # last checkpoint idx
            if core_context.preempt.should_preempt():
                return

        _synchronize_and_checkpoint(core_context, batch_idx, rank)
        core_context._tensorboard_manager.sync(mangler=get_rank_aware_path)

        if rank == 0:
            # Report to master the run has completed
            # Metrics reported does not matter
            dummy_searcher_op.report_completed(1)
