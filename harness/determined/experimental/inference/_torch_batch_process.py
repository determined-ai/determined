import abc
import logging
import math
import json
import os
import pathlib
import warnings

from dataclasses import dataclass

import torch
import torch.distributed as dist

from torch.utils.data import Dataset
from typing import Any, Dict, Optional, Type

import determined as det
from determined import core
from determined.common import set_logger
from determined.pytorch import DataLoader
from determined.tensorboard.util import get_rank_aware_path

set_logger(False)

DEFAULT_BATCH_SIZE = 1


@dataclass
class TorchBatchInitInfo:
    rank: int
    local_rank: int
    size: int
    num_agents: int
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
        distributed=distributed_context, preempt_mode=core.PreemptMode.WorkersAskChief
    )


class TorchBatchProcessor(metaclass=abc.ABCMeta):
    """
    User can initialize necessary resources in the init function, such as
    - model for prediction
    - storage client (e.g. s3 client)
    """

    @abc.abstractmethod
    def __init__(self, init_info: TorchBatchInitInfo) -> None:
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
        batch_indices = core_context.distributed.gather(batch_idx)
        min_batch_index = min(batch_indices)

        checkpoint_metadata = {
            "steps_completed": batch_idx + 1,
        }
        with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
            print(uuid)
            with open(os.path.join(path, "batch_completed.json"), "w") as file_obj:
                json.dump({"batch_completed": min_batch_index}, file_obj)
    else:
        core_context.distributed.gather(batch_idx)


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


def _validate_dataloader_kwargs(
    dataloader_kwargs: Dict[str, Any], batch_size: Optional[int]
) -> None:
    if "shuffle" in dataloader_kwargs:
        if dataloader_kwargs["shuffle"]:
            raise Exception("'shuffle' must be false for accurate sharding and checkpointing")
    if "sampler" in dataloader_kwargs:
        raise Exception(
            "Please remove 'sampler' arg as we will initialize a sampler automatically."
        )
    if "batch_sampler" in dataloader_kwargs:
        raise Exception(
            "Please remove 'batch_sampler' arg as we will initialize "
            "a batch_sampler automatically."
        )
    if batch_size is not None:
        if "batch_size" in dataloader_kwargs:
            raise Exception(
                "batch_size is passed into torch_batch_process " "and dataloader_kwargs"
            )


def _validate_iterate_length(iterate_length: Optional[int], times_iterate: int):
    if iterate_length is None:
        return times_iterate

    if iterate_length <= 0:
        warnings.warn(
            f"iterate_length {iterate_length} is not valid. "
            f"Ignoring this argument and iterate over entire dataset once"
        )
        return times_iterate

    if iterate_length > times_iterate:
        warnings.warn(
            f"iterate_length {iterate_length} exceeds sharded dataset length. "
            f"Ignoring this argument and iterate over entire dataset once"
        )
        return times_iterate
    return iterate_length


def torch_batch_process(
    batch_processor_cls: Type[TorchBatchProcessor],
    dataset: Dataset,
    batch_size: Optional[int] = None,
    iterate_length: Optional[int] = None,
    checkpoint_interval: int = 5,
    check_preempt_interval: Optional[int] = None,
    dataloader_kwargs: Dict[str, Any] = {},
):
    with initialize_default_inference_context() as core_context:
        _validate_dataloader_kwargs(dataloader_kwargs, batch_size)

        if check_preempt_interval is None:
            check_preempt_interval = checkpoint_interval

        if batch_size is None:
            if "batch_size" in dataloader_kwargs:
                # remove batch_size from dataloader_kwargs
                # and assign to batch_size
                batch_size = dataloader_kwargs.pop("batch_size")
            else:
                batch_size = DEFAULT_BATCH_SIZE
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
            init_info=TorchBatchInitInfo(
                rank=rank,
                local_rank=core_context.distributed.get_local_rank(),
                size=core_context.distributed.get_size(),
                num_agents=core_context.distributed.get_num_agents(),
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
            dataset=dataset, batch_size=batch_size, shuffle=False, **dataloader_kwargs
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
        max_batch = math.ceil(dataset_len / batch_size / total_worker)
        iterate_length = _validate_iterate_length(iterate_length, max_batch)

        for batch_idx in range(skip, iterate_length):
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

            if (batch_idx + 1) % check_preempt_interval == 0:
                if core_context.preempt.should_preempt():
                    _synchronize_and_checkpoint(core_context, batch_idx, rank)
                    return

        _synchronize_and_checkpoint(core_context, batch_idx, rank)
        core_context._tensorboard_manager.sync(mangler=get_rank_aware_path)

        if rank == 0:
            # Report to master the run has completed
            # Metrics reported does not matter
            dummy_searcher_op.report_completed(1)
