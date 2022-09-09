import contextlib
from enum import Enum

import determined as det
from determined import core, horovod
from determined import pytorch
from typing import Any, Callable, Dict, Iterator, List, Optional
import torch
import torch.nn as nn
from determined.common import check
from determined.horovod import hvd
import torch.distributed as dist
import logging

from determined.pytorch import PyTorchTrialContext


class TrainStepType(Enum):
    EPOCH = "EPOCH"
    BATCH = "BATCH"


class Trainer:
    def __init__(self, trial: pytorch.PyTorchTrial, context: PyTorchTrialContext):
        self.trial = trial
        self.context = context
        self.core_context = self.context._core
        self.train_loader = None
        self.distributed_backend = det._DistributedBackend()
        self.is_chief = self.core_context.distributed.rank == 0

    def _build_training_loader(self):
        train_data = self.trial.build_training_data_loader()
        num_replicas = self.core_context.distributed.size
        rank = self.core_context.distributed.rank

        training_loader = train_data.get_data_loader(
            repeat=False, num_replicas=num_replicas, rank=rank
        )
        return training_loader

    def _build_validation_loader(self):
        val_data = self.trial.build_validation_data_loader()
        if self.is_chief:
            return val_data.get_data_loader(
                repeat=False, skip=0, num_replicas=1, rank=0
            )

        num_replicas = self.core_context.distributed.size
        rank = self.core_context.distributed.rank

        val_loader = val_data.get_data_loader(
            repeat=False, num_replicas=num_replicas, rank=rank
        )
        return val_loader

    def train(
        self,
        max_epochs: Optional[int] = None,
        # OR
        max_batches: Optional[int] = None,
        min_checkpoint_period: int = 1,
        min_validation_period: int = 1,
        average_training_metrics: bool = True,
        average_aggregated_gradients: bool = True,
        aggregation_frequency: int = 2,
        searcher_metric="validation_loss",
        profiling=True,
        profiling_start=0,
        profiling_end=10,
        sync_timings=None,
        checkpoint_policy="best|all|none"
    ):
        assert (max_epochs is None) ^ (max_batches is None), "Either max_batches or max_epochs must be defined"

        # TODO: figure out a better way to do this.
        self.context.aggregation_frequency = aggregation_frequency

        if self.context.distributed.size > 1 and self.distributed_backend.use_horovod():
            hvd.broadcast_parameters(self.context._main_model.state_dict(), root_rank=0)
            for optimizer in self.context.optimizers:
                hvd.broadcast_optimizer_state(optimizer, root_rank=0)

        training_loader = self._build_training_loader()
        self.train_loader = training_loader

        # Set models to training mode
        for model in self.context.models:
            model.train()

        # Report training has started
        self.core_context.train.set_status("training")

        train_loop = TrainLoop(
            self,
            max_epochs,
            max_batches,
            min_validation_period,
            min_checkpoint_period,
        )

        train_loop.run()

        return

    def report_training_metrics(self, records_completed, steps_completed, metrics):
        metrics = self.context.distributed.broadcast(metrics)
        # Only report on the chief worker
        if self.is_chief:
            metrics = self._prepare_metrics(num_inputs=records_completed, batch_metrics=metrics)
            self.core_context.train.report_training_metrics(
                steps_completed=steps_completed, metrics=metrics
            )

    def validate_and_report_metrics(self, steps_completed):
        val_metrics = self.validate()
        if self.is_chief:
            self.core_context.train.report_validation_metrics(steps_completed=steps_completed, metrics=val_metrics)

    def validate(self):
        val_loader = self._build_validation_loader()

        # Set models to evaluation mode
        for model in self.context.models:
            model.eval()

        # Report training has started
        self.core_context.train.set_status("validating")

        keys = None
        batch_metrics = []

        for batch_idx, batch in enumerate(val_loader):
            val_metrics = self.trial.evaluate_batch(batch)
            batch_metrics.append(pytorch._convert_metrics_to_numpy(val_metrics))

            if keys is None:
                keys = val_metrics.keys()

        metrics = pytorch._reduce_metrics(
            self.context.distributed,
            batch_metrics=batch_metrics,
            keys=keys,
            metrics_reducers=pytorch._prepare_metrics_reducers(
                self.trial.evaluation_reducer(), keys=keys
            ),
        )

        metrics.update(
            pytorch._convert_metrics_to_numpy(self.context.reduce_metrics(for_training=False))
        )

        # Set models back to training mode
        for model in self.context.models:
            model.train()
        return metrics

    def _prepare_metrics(self, num_inputs: int, batch_metrics: List):
        for metrics in batch_metrics:
            for name, metric in metrics.items():
                # Convert PyTorch metric values to NumPy, so that
                # `det.util.encode_json` handles them properly without
                # needing a dependency on PyTorch.
                if isinstance(metric, torch.Tensor):
                    metric = metric.cpu().detach().numpy()
                metrics[name] = metric
        metrics = det.util.make_metrics(num_inputs, batch_metrics)
        metrics["avg_metrics"].update(
            pytorch._convert_metrics_to_numpy(self.context.reduce_metrics(for_training=True))
        )
        return metrics


class TrainLoop:
    def __init__(
        self,
        trainer: Trainer,
        max_epochs: int = None,
        max_batches: int = None,
        min_validation_period: int = 1,
        min_checkpoint_period: int = 1,
    ):
        self.trainer = trainer
        self.step_type = TrainStepType.EPOCH if max_epochs else TrainStepType.BATCH
        self.max_steps = max_epochs if self.step_type == TrainStepType.EPOCH else max_batches
        self.batches_trained = 0
        self.epochs_trained = 0
        self.records_completed = 0
        self.steps_completed = 0
        self.checkpoint_period = min_checkpoint_period
        self.validation_period = min_validation_period
        self.reporting_period = min(min_checkpoint_period, min_validation_period)
        self.training_metrics = []

    def _steps_remaining(self):
        return self.max_steps - self.steps_completed

    def run(self):
        while self._steps_remaining() > 0:
            for batch_idx, batch in enumerate(self.trainer.train_loader):
                self._train_batch(batch, self.epochs_trained, batch_idx)
                if not self._steps_remaining() > 0:
                    return
            self._step_epoch()

    def _train_batch(self, batch, epochs, batch_idx):
        self.trainer.context._current_batch_idx = self.batches_trained
        self.records_completed += self.trainer.trial.get_batch_length(batch)
        batch_metrics = self.trainer.trial.train_batch(batch, epochs, batch_idx)
        self.training_metrics.append(batch_metrics)
        self._step_batch()

    def _train_step(self):
        self.steps_completed += 1
        if self.steps_completed % self.reporting_period == 0:
            self.trainer.report_training_metrics(self.records_completed, self.steps_completed, self.training_metrics)
            self.training_metrics = []
        if self.steps_completed % self.validation_period == 0:
            self.trainer.validate_and_report_metrics(self.steps_completed)

    def _step_batch(self):
        self.batches_trained += 1
        if self.step_type == TrainStepType.BATCH:
            self._train_step()

    def _step_epoch(self):
        self.epochs_trained += 1
        if self.step_type == TrainStepType.EPOCH:
            self._train_step()


def initialize_distributed_backend():
    distributed_backend = det._DistributedBackend()
    if distributed_backend.use_horovod():
        hvd.require_horovod_type("torch", "PyTorchTrial is in use.")
        hvd.init()
        return core.DistributedContext.from_horovod(horovod.hvd)
    elif distributed_backend.use_torch():
        if torch.cuda.is_available():
            dist.init_process_group(backend="nccl")  # type: ignore
        else:
            dist.init_process_group(backend="gloo")  # type: ignore
        return core.DistributedContext.from_torch_distributed()
    else:
        print(f"Backend {distributed_backend} not supported")


@contextlib.contextmanager
def init():
    distributed_context = initialize_distributed_backend()
    core_context = core.init(distributed=distributed_context)
    context = PyTorchTrialContext(core_context, hparams={"global_batch_size": 32})
    yield context

