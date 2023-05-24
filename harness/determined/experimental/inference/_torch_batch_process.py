import abc
import logging
import math
import json
import os
import pathlib
import warnings
import uuid

import torch
import torch.distributed as dist

from torch import nn
from torch.utils.data import Dataset
from typing import Any, Dict, Iterator, Optional, Set, Type

import determined as det
from determined import core, pytorch
from determined.common import set_logger
from determined.pytorch import DataLoader

set_logger(False)

DEFAULT_BATCH_SIZE = 1


class TorchBatchProcessorContext(pytorch._PyTorchReducerContext):
    def __init__(self, core_context: core.Context, storage_path: str):
        super().__init__()
        self._core_context = core_context
        self._device = get_default_device(core_context)
        self._tensorboard_path = core_context.train.get_tensorboard_path()
        self._storage_path = storage_path
        self._use_default_storage = False

    def to_device(self, data, warned_types: Optional[Set[Type]] = None) -> pytorch.TorchData:
        """Map generated data to the device allocated by the Determined cluster.

        All the data in the data loader and the models are automatically moved to the
        allocated device. This method aims at providing a function for the data generated
        on the fly.
        """
        return pytorch.to_device(data, self._device, warned_types)

    def get_tensorboard_path(self) -> pathlib.Path:
        """
        Tensorboard files should be written to the path returned to be shown properly in the UI.
        For example, the path should be passed to PyTorch profiler as shown below:

        torch.profiler.profile(
            activities=...,
            schedule=...,
            on_trace_ready=torch.profiler.tensorboard_trace_handler(<tensorboard_path>),
        )
        """
        return self._tensorboard_path

    def get_device(self):
        """
        Get default device associated with this worker
        """
        return self._device

    def get_distributed_context(self):
        return self._core_context.distributed

    def prepare_model_for_inference(self, model: nn.Module) -> nn.Module:
        model.eval()
        model.to(self._device)
        return model

    def get_default_storage_path(self) -> Iterator[pathlib.Path]:
        """
        Should use with "with" syntax
        """
        self._use_default_storage = True
        return self._core_context.checkpoint._storage_manager.store_path(self._storage_path)


def get_default_device(core_context: core.Context) -> torch.device:
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
    # Use WorkerAskChief mode to ensure synchronize correctly across worker
    # Using WorkerAskMaster mode could lead to some workers exiting when others
    # are waiting for synchronization.
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
    def __init__(self, context: TorchBatchProcessorContext) -> None:
        pass

    @abc.abstractmethod
    def process_batch(self, batch, batch_idx) -> None:
        pass

    def run_before_checkpoint(self) -> None:
        """
        Overwrite this function to run certain logic before checkpointing
        This function will be called right before each checkpoint.
        """
        pass

    def clean_up(self) -> None:
        """
        This function will be called right before exiting after completing iteration
        over dataset
        """
        pass


def _load_state(checkpoint_directory):
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("metadata.json").open("r") as f:
        metadata = json.load(f)
        return metadata


def _synchronize_and_checkpoint(
    core_context: core.Context, steps_completed: int, rank: int, default_output_uuid: str
):
    """
    Synchronize the workers and create checkpoint to record steps completed
    """
    if rank == 0:
        steps_completed_list = core_context.distributed.gather(steps_completed)
        min_steps_completed = min(steps_completed_list)

        checkpoint_metadata = {
            "steps_completed": min_steps_completed,
            "default_output_uuid": default_output_uuid,
        }
        with core_context.checkpoint.store_path(checkpoint_metadata) as (path, uuid):
            with open(os.path.join(path, "batch_completed.json"), "w") as file_obj:
                json.dump({"batch_completed": min_steps_completed}, file_obj)
    else:
        core_context.distributed.gather(steps_completed)


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


def _get_storage_information(checkpoint_config, default_uuid_path, core_context) -> str:
    storage_type = checkpoint_config["type"]

    if storage_type == "s3":
        bucket = checkpoint_config["bucket"]
        return f"s3://{bucket}/{default_uuid_path}"
    # TODO test
    elif storage_type == "gcs":
        bucket = checkpoint_config["bucket"]
        return f"gs://{bucket}/{default_uuid_path}"
    # TODO test
    elif storage_type == "azure":
        container = checkpoint_config["container"]
        return f"Azure container: {container} Directory:{default_uuid_path}"
    # TODO test
    elif storage_type == "shared_fs":
        base_path = core_context.checkpoint._storage_manager._base_path
        return f"{base_path}/{default_uuid_path}"
    else:
        raise NotImplementedError(f"Storage type {storage_type} support is not implemented")


def _reduce_metrics(batch_processor_context, core_context, rank, steps_completed):
    reducables = [wrapped for wrapped in batch_processor_context._wrapped_reducers]
    # If the user has set metric reducers
    if len(reducables) > 0:
        # Reduce metrics (blocking as reduce across slots is needed)
        # Report reduced metrics to master
        gatherables = [wrapped.per_slot_reduce() for wrapped in reducables]
        if rank == 0:
            gathered = core_context.distributed.gather(gatherables)
            metrics = batch_processor_context.run_cross_slot_reduction(reducables, gathered)
            core_context.train.report_validation_metrics(
                steps_completed=steps_completed,
                metrics=metrics,
            )
        else:
            # Other ranks sent metrics to chief
            core_context.distributed.gather(gatherables)


def torch_batch_process(
    batch_processor_cls: Type[TorchBatchProcessor],
    dataset: Dataset,
    batch_size: Optional[int] = None,
    iterate_length: Optional[int] = None,
    checkpoint_interval: int = 5,
    dataloader_kwargs: Dict[str, Any] = {},
):
    with initialize_default_inference_context() as core_context:
        """
        (1) Set up necessary variables to run batch processing
        """

        _validate_dataloader_kwargs(dataloader_kwargs, batch_size)

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

        # Synchronize default output uuid
        if rank == 0:
            default_output_uuid = str(uuid.uuid4())
            core_context.distributed.broadcast(default_output_uuid)
        else:
            default_output_uuid = core_context.distributed.broadcast(None)

        # Get previous trial state from checkpoint if available
        if latest_checkpoint is not None:
            logging.info("Checkpoint is not none")
            with core_context.checkpoint.restore_path(latest_checkpoint) as path:
                metadata = _load_state(path)
                skip = metadata["steps_completed"]
                logging.info(f"Previous run completed {skip} steps")
                default_output_uuid = metadata["default_output_uuid"]

        output_uuid_with_rank = default_output_uuid + f"/rank_{rank}"

        batch_processor_context = TorchBatchProcessorContext(core_context, output_uuid_with_rank)

        per_batch_processor = batch_processor_cls(
            context=batch_processor_context,
        )

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

        last_checkpoint_idx = 0
        batch_idx = skip
        steps_completed = skip

        """
        (2) Run batch processing
        """

        for batch_idx in range(skip, iterate_length):
            X = next(dataloader_iterator, None)
            if X is not None:
                per_batch_processor.process_batch(batch=X, batch_idx=batch_idx)
            steps_completed = batch_idx + 1

            # Checkpoint and check preemption
            if (batch_idx + 1) % checkpoint_interval == 0:
                logging.info(f"Completed steps:  {steps_completed} and checkpointing")
                per_batch_processor.run_before_checkpoint()
                _synchronize_and_checkpoint(
                    core_context, steps_completed, rank, default_output_uuid
                )
                last_checkpoint_idx = batch_idx
                # Report progress can only be done accurately with synchronization
                core_context._tensorboard_manager.sync()
                if rank == 0:
                    _report_progress_to_master(
                        dummy_searcher_op, batch_idx, total_worker, batch_size, dataset_len
                    )
                # Check preemption
                if core_context.preempt.should_preempt():
                    # Finish reducing metrics and report to not lose state before preempting
                    _reduce_metrics(batch_processor_context, core_context, rank, steps_completed)
                    return

        """
        (3) Finish up after batch processing
        """
        if batch_idx > last_checkpoint_idx:
            per_batch_processor.run_before_checkpoint()
            logging.info(f"Completed steps:  {steps_completed} and checkpointing")
            _synchronize_and_checkpoint(core_context, iterate_length, rank, default_output_uuid)

        _reduce_metrics(batch_processor_context, core_context, rank, steps_completed)
        # Finish any tensorboard uploads remaining
        core_context._tensorboard_manager.sync()

        per_batch_processor.clean_up()

        # If user has used default storage, print out the default storage path
        if rank == 0 and batch_processor_context._use_default_storage:
            default_storage_path = _get_storage_information(
                info.trial._config["checkpoint_storage"], default_output_uuid, core_context
            )
            logging.info(f"Files stored with default paths are at: {default_storage_path}")

        if rank == 0:
            # Report to master the run has completed
            # Metrics reported does not matter
            dummy_searcher_op.report_completed(1)
