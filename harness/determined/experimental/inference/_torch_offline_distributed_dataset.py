import abc
import logging
import json
import os
import pathlib
from dataclasses import dataclass

import torch
import torch.distributed as dist

from torch.utils.data import BatchSampler, Dataset, DataLoader, SequentialSampler
from typing import Any, List, Optional

import determined as det
from determined import core
from determined.common import set_logger
from determined.pytorch import adapt_batch_sampler

set_logger(False)


@dataclass
class TorchPerBatchProcessInfo:
    batch_idx: int
    worker_rank: int
    torch_profiler: Optional[torch.profiler.profile]


class TorchPerBatchProcessor(metaclass=abc.ABCMeta):
    """
    User can initialize necessary resources in the init function, such as
    - model for prediction
    - storage client (e.g. s3 client)
    """

    @abc.abstractmethod
    def process_batch(self, batch, additional_info: TorchPerBatchProcessInfo) -> None:
        pass


def _load_state(checkpoint_directory):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("metadata.json").open("r") as f:
        metadata = json.load(f)
        return metadata


class TorchDistributedDatasetProcessor:
    def __init__(
        self,
        core_context,
        per_batch_processor: TorchPerBatchProcessor,
        dataset: Dataset,
        batch_size: int = 64,
        checkpoint_interval: int = 5,
        dataloader_num_workers: int = 4,
    ):
        self._core_context = core_context
        self._per_batch_processor = per_batch_processor
        self._dataset = dataset
        self._dataset_len = len(dataset)
        self._batch_size = batch_size
        self._checkpoint_interval = checkpoint_interval
        self._dataloader_num_workers = dataloader_num_workers
        self._torch_profiler = None

        info = det.get_cluster_info()
        slots_per_node = len(info.slot_ids)
        num_nodes = len(info.container_addrs)
        self._total_worker = num_nodes * slots_per_node
        # container_rank is cross rank
        self._rank = info.container_rank
        latest_checkpoint = info.latest_checkpoint
        self._skip = 0

        # Check if previous checkpoint exists
        if latest_checkpoint is not None:
            logging.info("Checkpoint is not none")
            with self._core_context.checkpoint.restore_path(latest_checkpoint) as path:
                metadata = _load_state(path)
                self._skip = metadata["steps_completed"]
                logging.info(f"Previous run completed {self._skip} steps")

    # TODO: Switch to determined's dataloader class in data utils and add collate_fn as
    #  well as worker start fn args
    def _create_dataloader(self) -> DataLoader:
        """
        Create sharded deterministic dataloader from dataset
        """
        if isinstance(self._dataset, torch.utils.data.IterableDataset):
            raise Exception("Only map style dataset with __getitem__ method is supported.")
        sampler = SequentialSampler(self._dataset)
        batch_sampler = BatchSampler(sampler, self._batch_size, drop_last=False)
        # Adapt batch_sampler for distributed inference and trial resumption if applicable
        batch_sampler = adapt_batch_sampler(
            batch_sampler,
            repeat=False,
            skip=self._skip,
            num_replicas=self._total_worker,
            rank=self._rank,
        )

        return torch.utils.data.DataLoader(
            self._dataset, batch_sampler=batch_sampler, num_workers=self._dataloader_num_workers
        )

    def _synchronize_and_checkpoint(self, batch_idx: int):
        """
        Synchronize the workers and create checkpoint to record steps completed
        """
        if self._rank == 0:
            self._core_context.distributed.gather(batch_idx)
            checkpoint_metadata = {
                "steps_completed": batch_idx + 1,
            }
            with self._core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
                with open(os.path.join(path, "batch_completed.json"), "w") as file_obj:
                    json.dump({"batch_completed": batch_idx}, file_obj)
        else:
            self._core_context.distributed.gather(batch_idx)

    def _report_progress_to_master(
        self, searcher_op: core.DummySearcherOperation, batch_idx: int
    ) -> None:
        completion_rate = self._calculate_progress(batch_idx)
        searcher_op.report_progress(completion_rate)

    def _calculate_progress(self, batch_idx: int):
        records_processed = batch_idx * self._total_worker * self._batch_size
        return records_processed / self._dataset_len

    def _get_tensorboard_path(self) -> pathlib.Path:
        """
        Get the path where files for consumption by TensorBoard should be written
        """
        return self._core_context.train.get_tensorboard_path()

    def set_torch_profiler(self, *args: List[str], **kwargs: Any) -> None:
        self._torch_profiler = torch.profiler.profile(
            on_trace_ready=torch.profiler.tensorboard_trace_handler(
                str(self._get_tensorboard_path())
            ),
            *args,
            **kwargs,
        )

    def run(self):
        """
        Apply per_batch_processor to the dataset
        """
        dataloader = self._create_dataloader()
        batch_idx = 0

        # Create dummy searcher op to report progress to master
        dummy_searcher_op = None
        # Initialize dummy searcher for progress report
        if self._rank == 0:
            dummy_searcher_op = core.DummySearcherOperation(1, True)

        for idx, X in enumerate(dataloader):
            batch_idx = idx + self._skip
            logging.info(f"Currently processing batch {batch_idx}")
            self._per_batch_processor.process_batch(
                batch=X,
                additional_info=TorchPerBatchProcessInfo(
                    batch_idx=batch_idx + self._skip,
                    worker_rank=self._rank,
                    torch_profiler=self._torch_profiler,
                ),
            )

            if (idx + 1) % self._checkpoint_interval == 0:
                self._synchronize_and_checkpoint(batch_idx)

                if self._rank == 0:
                    self._report_progress_to_master(dummy_searcher_op, batch_idx)

                if self._core_context.preempt.should_preempt():
                    return

        self._synchronize_and_checkpoint(batch_idx)
        if self._rank == 0:
            # Report to master the run has completed
            self._report_progress_to_master(
                dummy_searcher_op, self._dataset_len / (self._total_worker * self._batch_size)
            )


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
            "distributed launch layer such as determined.launch.torch_distributed or "
            "determined.launch.horovod."
        )
    return None
