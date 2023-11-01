import abc
import json
import logging
import math
import os
import pathlib
import uuid
import warnings
from typing import TYPE_CHECKING, Any, ContextManager, Dict, Optional, Set, Sized, Type

import torch
import torch.distributed as dist
from torch import nn
from torch.utils import data

import determined as det
from determined import common, core, pytorch

if TYPE_CHECKING:
    # These modules are only needed for type checking and
    # cause a circular dependency issue. This bypasses it.
    from determined.experimental import checkpoint, model

logger = logging.getLogger("determined.pytorch")

common.set_logger(False)

DEFAULT_BATCH_SIZE = 1


class TorchBatchProcessorContext(pytorch._PyTorchReducerContext):
    def __init__(self, core_context: core.Context, storage_path: str) -> None:
        super().__init__()
        self._core_context = core_context
        self._distributed = core_context.distributed
        self.device = get_default_device(core_context)
        self._tensorboard_path = core_context.train.get_tensorboard_path()
        self._storage_path = storage_path
        self._use_default_storage = False
        self._hparams = None  # type: Optional[Dict[str, Any]]

    def get_hparams(self) -> Dict[str, Any]:
        if self._hparams is None:
            info = det.get_cluster_info()
            assert info, "Must run TorchBatchProcessor in a cluster to run get_hparams()"
            assert info.task_type == "TRIAL", "TorchBatchProcessor must be run inside of a Trial"
            self._hparams = info.trial.hparams
        assert self._hparams is not None
        return self._hparams

    def to_device(
        self, data: pytorch._Data, warned_types: Optional[Set[Type]] = None
    ) -> pytorch.TorchData:
        """
        Accept np.ndarray, torch.Tensor, list, or dictionary. Recursively convert any ndarrays to
        tensors and call .to() on any tensors or data types that have custom serialization logic
        defined via a callable to() attribute.

        If the data cannot be moved to device, log a warning (only once per type) and return the
        original data.
        """
        return pytorch.to_device(data, self.device, warned_types)

    def get_tensorboard_path(self) -> pathlib.Path:
        """
        Tensorboard files should be written to the path returned to be shown properly in the UI.

        For example, the path should be passed to PyTorch profiler as shown below:

        .. code-block:: python

            torch.profiler.profile(
                activities=...,
                schedule=...,
                on_trace_ready=torch.profiler.tensorboard_trace_handler(<tensorboard_path>),
            )

        """
        return self._tensorboard_path

    def prepare_model_for_inference(self, model: nn.Module) -> nn.Module:
        """
        Set model to eval mode and send model to device
        Arguments:
            model: a nn.Module
        """
        model.eval()
        model.to(self.device)
        return model

    def upload_path(self) -> ContextManager[pathlib.Path]:
        """
        Returns a context that uploads files to default storage path on exit.
        """
        self._use_default_storage = True
        return self._core_context.checkpoint._storage_manager.store_path(self._storage_path)

    def report_metrics(self, group: str, steps_completed: int, metrics: Dict[str, Any]) -> None:
        """
        Report metrics data to the master.

        Arguments:
            group (string): metrics group name. Can be used to partition metrics
                into different logical groups or time series.
                "training" and "validation" group names map to built-in training
                and validation time series. Note: Group cannot contain ``.`` character.
            steps_completed (int): global step number, e.g. the number of batches processed.
            metrics (Dict[str, Any]): metrics data dictionary. Must be JSON-serializable.
                When reporting metrics with the same ``group`` and ``steps_completed`` values,
                the dictionary keys must not overlap.
        """
        self._core_context.train.report_metrics(
            group=group,
            steps_completed=steps_completed,
            metrics=metrics,
        )

    def report_task_using_model_version(self, model_version: "model.ModelVersion") -> None:
        """
        Associate ``model_version`` with the current task. This links together the metrics
        reporting so that any metrics which are reported to the current task will be
        visible when querying for metrics associated with this model version

        Args:
            model_Version (model.ModelVersion): The model version to associate with this task
        """
        self._core_context.experimental.report_task_using_model_version(model_version)

    def report_task_using_checkpoint(self, checkpoint: "checkpoint.Checkpoint") -> None:
        """
        Associate ``checkpoint`` with the current task. This links together the metrics
        reporting so that any metrics which are reported to the current task will be
        visible when querying for metrics associated with this checkpoint

        Args:
            checkpoint (checkpoint.Checkpoint): The checkpoint to associate with this task
        """
        self._core_context.experimental.report_task_using_checkpoint(checkpoint)

    def get_distributed_rank(self) -> int:
        """
        The rank of this current process in a trial
        """
        return self._core_context.distributed.get_rank()

    def get_distributed_size(self) -> int:
        """
        The number of slots this trial is running on
        """
        return self._core_context.distributed.get_size()


def get_default_device(core_context: core.Context) -> torch.device:
    local_rank = core_context.distributed.local_rank
    local_num_gpu = torch.cuda.device_count()
    if local_num_gpu == 0:
        return torch.device("cpu")
    else:
        # Assuming there would not be more process than CUDA devices
        return torch.device("cuda", local_rank)


def _initialize_distributed_backend() -> Optional[core.DistributedContext]:
    distributed_backend = det._DistributedBackend()
    if distributed_backend.use_torch():
        if torch.cuda.is_available():
            dist.init_process_group(backend="nccl")  # type: ignore
        else:
            dist.init_process_group(backend="gloo")  # type: ignore
        return core.DistributedContext.from_torch_distributed()

    info = det.get_cluster_info()
    if info and (len(info.container_addrs) > 1 or len(info.slot_ids) > 1):
        raise ValueError(
            "In multi-slot managed cluster training, you must wrap your training script with a "
            "distributed launch layer such as determined.launch.torch_distributed"
        )
    return None


def _initialize_default_inference_context(
    distributed_context: Optional[core.DistributedContext],
) -> core.Context:
    if distributed_context is None:
        distributed_context = _initialize_distributed_backend()
    # Use WorkerAskChief mode to ensure synchronize correctly across worker
    # Using WorkerAskMaster mode could lead to some workers exiting when others
    # are waiting for synchronization.
    return det.core.init(
        distributed=distributed_context, preempt_mode=core.PreemptMode.WorkersAskChief
    )


class TorchBatchProcessor(metaclass=abc.ABCMeta):
    @abc.abstractmethod
    def __init__(self, context: TorchBatchProcessorContext) -> None:
        """
        User can initialize necessary resources in the init function, such as
        model for prediction

        Arguments:
            context: an TorchBatchProcessorContext instance
        """
        pass

    @abc.abstractmethod
    def process_batch(self, batch: Any, batch_idx: int) -> None:
        """
        This function will be called with every batch of data in the dataset

        Arguments:
            batch: a batch of data of the dataset passed into torch_batch_process
            batch_idx: index of the batch. Note that index is per worker. For example, if there are
                8 batches of data to process and 4 workers, each worker would get two batches of
                data (batch_idx = 0 and batch_idx = 1)
        """
        pass

    def on_checkpoint_start(self) -> None:  # noqa: B027
        """
        This function will be called right before each checkpoint
        """
        pass

    def on_finish(self) -> None:  # noqa: B027
        """
        This function will be called right before exiting after completing iteration
        over dataset
        """
        pass


def _load_state(checkpoint_directory: pathlib.Path) -> Any:
    checkpoint_directory = pathlib.Path(checkpoint_directory)
    with checkpoint_directory.joinpath("metadata.json").open("r") as f:
        metadata = json.load(f)
        return metadata


def _synchronize_and_checkpoint(
    core_context: core.Context, steps_completed: int, default_output_uuid: str
) -> None:
    """
    Synchronize the workers and create checkpoint to record steps completed
    """
    if core_context.distributed.get_rank() == 0:
        steps_completed_list = core_context.distributed.gather(steps_completed)
        if steps_completed_list is None:
            return
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
    # TODO: update after searcher context removal to report to master
    records_processed = batch_idx * total_worker * batch_size
    completion_rate = records_processed / dataset_len
    searcher_op.report_progress(completion_rate)


def _validate_dataloader_kwargs(
    dataloader_kwargs: Dict[str, Any], batch_size: Optional[int]
) -> None:
    if "shuffle" in dataloader_kwargs:
        if dataloader_kwargs["shuffle"]:
            raise ValueError("'shuffle' must be false for accurate sharding and checkpointing")
    if "sampler" in dataloader_kwargs:
        raise ValueError(
            "Please remove 'sampler' arg as we will initialize a sampler automatically."
        )
    if "batch_sampler" in dataloader_kwargs:
        raise ValueError(
            "Please remove 'batch_sampler' arg as we will initialize "
            "a batch_sampler automatically."
        )
    if batch_size is not None:
        if "batch_size" in dataloader_kwargs:
            raise ValueError(
                "batch_size is passed into torch_batch_process " "and dataloader_kwargs"
            )


def _validate_iterate_length(iterate_length: Optional[int], times_iterate: int) -> int:
    if iterate_length is None:
        return times_iterate

    if iterate_length <= 0:
        warnings.warn(
            f"iterate_length {iterate_length} is not valid. "
            f"Ignoring this argument and iterate over entire dataset once",
            stacklevel=2,
        )
        return times_iterate

    if iterate_length > times_iterate:
        warnings.warn(
            f"iterate_length {iterate_length} exceeds sharded dataset length. "
            f"Ignoring this argument and iterate over entire dataset once",
            stacklevel=2,
        )
        return times_iterate
    return iterate_length


def _get_storage_information(
    checkpoint_config: Dict[str, Any], default_uuid_path: str, core_context: core.Context
) -> str:
    storage_type = checkpoint_config["type"]

    if storage_type == "s3":
        bucket = checkpoint_config["bucket"]
        return f"s3://{bucket}/{default_uuid_path}"
    elif storage_type == "gcs":
        bucket = checkpoint_config["bucket"]
        return f"gs://{bucket}/{default_uuid_path}"
    elif storage_type == "azure":
        container = checkpoint_config["container"]
        return f"Azure container: {container} Directory:{default_uuid_path}"
    elif storage_type == "shared_fs":
        base_path = core_context.checkpoint._storage_manager._base_path
        return f"{base_path}/{default_uuid_path}"
    else:
        raise NotImplementedError(f"Storage type {storage_type} support is not implemented")


def _reduce_metrics(
    batch_processor_context: TorchBatchProcessorContext,
    core_context: core.Context,
    rank: int,
    steps_completed: int,
) -> None:
    reducables = list(batch_processor_context._wrapped_reducers)
    # If the user has set metric reducers
    if len(reducables) > 0:
        # Reduce metrics (blocking as reduce across slots is needed)
        # Report reduced metrics to master
        gatherables = [wrapped.per_slot_reduce() for wrapped in reducables]
        if rank == 0:
            gathered = core_context.distributed.gather(gatherables)
            if gathered is not None:
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
    dataset: data.Dataset,
    batch_size: Optional[int] = None,
    max_batches: Optional[int] = None,
    checkpoint_interval: int = 5,
    dataloader_kwargs: Optional[Dict[str, Any]] = None,
    distributed_context: Optional[core.DistributedContext] = None,
) -> None:
    """
    ```torch_batch_process``` shard and iterate through the provided dataset and process the dataset
    with user-defined logic in ```batch_processor_cls```.

    Arguments:
        batch_processor_cls: A user-defined class extending ```TorchBatchProcessor```
        dataset: A torch dataset class implementing __len__() and __getitem__()
        batch_size: The number of items to in each batch
        max_batches: The maximum number of batches to iterate over per worker
        checkpoint_interval: Interval to checkpoint progress (i.e. record number
            of batches processed)
        dataloader_kwargs: Kwargs to pass to PyTorch dataloader
        distributed_context: Distributed context to initialize core context
    """
    with _initialize_default_inference_context(distributed_context) as core_context:
        """
        (1) Set up necessary variables to run batch processing
        """

        # Validate argument inputs
        if checkpoint_interval <= 0:
            raise ValueError("checkpoint_interval should be a positive integer")

        if dataloader_kwargs is None:
            dataloader_kwargs = {}
        _validate_dataloader_kwargs(dataloader_kwargs, batch_size)

        if batch_size is None:
            if "batch_size" in dataloader_kwargs:
                # remove batch_size from dataloader_kwargs
                # and assign to batch_size
                batch_size = dataloader_kwargs.pop("batch_size")
            else:
                batch_size = DEFAULT_BATCH_SIZE
        if not isinstance(dataset, Sized):
            raise Exception("Dataset must implement __len__()")

        dataset_len = len(dataset)

        info = det.get_cluster_info()

        if info is None:
            raise Exception("torch_batch_process only runs on-cluster.")

        slots_per_node = len(info.slot_ids)

        num_nodes = len(info.container_addrs)
        total_worker = num_nodes
        if slots_per_node > 0:
            total_worker *= slots_per_node
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
            logger.info("Checkpoint is not none")
            with core_context.checkpoint.restore_path(latest_checkpoint) as path:
                metadata = _load_state(path)
                skip = metadata["steps_completed"]
                logger.info(f"Previous run completed {skip} steps")
                default_output_uuid = metadata["default_output_uuid"]

        output_uuid_with_rank = default_output_uuid + f"/rank_{rank}"

        batch_processor_context = TorchBatchProcessorContext(core_context, output_uuid_with_rank)

        per_batch_processor = batch_processor_cls(
            context=batch_processor_context,
        )

        dataloader = pytorch.DataLoader(
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
        dist_dataset_batch_count = math.ceil(dataset_len / batch_size / total_worker)
        iterate_length = _validate_iterate_length(max_batches, dist_dataset_batch_count)

        last_checkpoint_idx = -1
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
                logger.info(f"Completed steps:  {steps_completed} and checkpointing")

                per_batch_processor.on_checkpoint_start()
                if core_context._tensorboard_manager is not None:
                    core_context._tensorboard_manager.sync()
                _synchronize_and_checkpoint(core_context, steps_completed, default_output_uuid)
                last_checkpoint_idx = batch_idx

                # Report progress can only be done accurately with synchronization
                # when rank == 0, dummy_searcher_op will be initialized, but lint is complaining
                # therefore, adding additional check here
                if rank == 0 and dummy_searcher_op is not None:
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
            per_batch_processor.on_checkpoint_start()
            logger.info(f"Completed steps:  {steps_completed} and checkpointing")
            _synchronize_and_checkpoint(core_context, iterate_length, default_output_uuid)

        _reduce_metrics(batch_processor_context, core_context, rank, steps_completed)
        # Finish any tensorboard uploads remaining
        if core_context._tensorboard_manager is not None:
            core_context._tensorboard_manager.sync()

        per_batch_processor.on_finish()

        # If user has used default storage, print out the default storage path
        if rank == 0 and batch_processor_context._use_default_storage:
            default_storage_path = _get_storage_information(
                info.trial._config["checkpoint_storage"], default_output_uuid, core_context
            )
            logger.info(f"Files stored with default paths are at: {default_storage_path}")

        # Perform gather here to ensure we only report progress to master when all workers finish
        core_context.distributed.gather(None)

        # when rank == 0, dummy_searcher_op will be initialized, but lint is complaining
        # therefore, adding additional check here
        if rank == 0 and dummy_searcher_op is not None:
            # Report to master the run has completed
            # Metrics reported does not matter
            dummy_searcher_op.report_completed(1)
