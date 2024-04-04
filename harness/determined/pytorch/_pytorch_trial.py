import abc
import contextlib
import enum
import inspect
import json
import logging
import pathlib
import pickle
import random
import sys
import time
import warnings
from collections import abc as col_abc
from typing import Any, Callable, Dict, Iterator, List, Optional, Tuple, Type, Union

import numpy as np
import torch
import torch.utils.data
from torch import distributed as dist

import determined as det
from determined import core, horovod, pytorch, tensorboard, util

logger = logging.getLogger("determined.pytorch")

# Apex is included only for GPU trials.
try:
    import apex
except ImportError:  # pragma: no cover
    apex = None
    pass


def dataloader_next(dataloader_iter: Iterator) -> Iterator:
    while True:
        try:
            batch = next(dataloader_iter)
        except StopIteration:
            return
        yield batch


class TrainUnit:
    """
    TrainUnit is the base class for the supported training units (Batch, Epoch) containing
    the value of unit, where the value can be an int or an implementable collections.abc.Container.

    TrainUnits are used to define periodic training behavior such as checkpointing and validating.

    int values are treated as periods, e.g. Batch(100) will checkpoint/validate every 100 batches.
    collections.abc.Container values are treated as schedules, e.g. Batch(1,5,10) will
    checkpoint/validate on batches 1, 5, and 10.
    """

    def __init__(self, value: Union[int, col_abc.Container]):
        self.value = value

    @staticmethod
    def _from_searcher_unit(
        length: int, unit: Optional[core.Unit], global_batch_size: Optional[int] = None
    ) -> "TrainUnit":
        if unit == core.Unit.EPOCHS:
            return Epoch(length)
        elif unit == core.Unit.RECORDS:
            if global_batch_size is None:
                raise ValueError("global_batch_size required for searcher unit Records.")
            return Batch._from_records(length, global_batch_size)
        elif unit == core.Unit.BATCHES:
            return Batch(length)
        else:
            raise ValueError(f"unrecognized searcher unit {unit}")

    def _to_searcher_unit(self) -> core.Unit:
        if isinstance(self, Batch):
            return core.Unit.BATCHES
        return core.Unit.EPOCHS

    @staticmethod
    def _from_values(
        batches: Optional[int] = None,
        records: Optional[int] = None,
        epochs: Optional[int] = None,
        global_batch_size: Optional[int] = None,
    ) -> "TrainUnit":
        if sum((batches is not None, records is not None, epochs is not None)) != 1:
            raise ValueError(f"invalid config: batches={batches} records={records} epochs={epochs}")
        if batches is not None:
            if batches < 1:
                batches = sys.maxsize
            return Batch(batches)
        if records is not None:
            assert global_batch_size, "global_batch_size is required for RECORD units."
            if records < 1:
                records = sys.maxsize
            return Batch._from_records(records, global_batch_size)
        if epochs is not None:
            if epochs < 1:
                epochs = sys.maxsize
            return Epoch(epochs)

        # Make mypy happy
        raise ValueError("invalid values")

    def should_stop(self, step_num: int) -> bool:
        if isinstance(self.value, int):
            return self._divides(step_num)
        assert isinstance(self.value, col_abc.Container)
        return step_num in self.value

    def _divides(self, steps: int) -> bool:
        assert isinstance(steps, int) and isinstance(
            self.value, int
        ), "_divides can only be called on int types."
        # Treat <= 0 values as always step
        if self.value < 1:
            return True
        if steps == 0:
            return False
        return steps % self.value == 0


class Epoch(TrainUnit):
    """
    Epoch step type (e.g. Epoch(1) defines 1 epoch)
    """

    pass


class Batch(TrainUnit):
    """
    Batch step type (e.g. Batch(1) defines 1 batch)
    """

    @staticmethod
    def _from_records(records: int, global_batch_size: int) -> "Batch":
        return Batch(max(records // global_batch_size, 1))


class _TrainBoundaryType(enum.Enum):
    CHECKPOINT = "CHECKPOINT"
    REPORT = "REPORT"
    VALIDATE = "VALIDATE"
    TRAIN = "TRAIN"


class _TrainBoundary:
    def __init__(self, step_type: _TrainBoundaryType, unit: TrainUnit):
        self.step_type = step_type
        self.unit = unit
        self.limit_reached = False


class ShouldExit(Exception):
    """
    ShouldExit breaks out of the top-level train loop from inside function calls.
    """

    def __init__(self, skip_exit_checkpoint: bool = False):
        self.skip_exit_checkpoint = skip_exit_checkpoint


class _TrialState:
    def __init__(
        self,
        trial_id: int = 0,
        last_ckpt: int = 0,
        step_id: int = 0,
        last_val: int = 0,
        batches_trained: int = 0,
        epochs_trained: int = 0,
    ) -> None:
        # Store TrialID to distinguish between e.g. pause/restart and continue training.
        self.trial_id = trial_id
        self.last_ckpt = last_ckpt
        self.step_id = step_id
        self.last_val = last_val
        self.batches_trained = batches_trained
        self.epochs_trained = epochs_trained


class _PyTorchTrialController:
    def __init__(
        self,
        trial_inst: det.LegacyTrial,
        context: pytorch.PyTorchTrialContext,
        checkpoint_period: TrainUnit,
        validation_period: TrainUnit,
        reporting_period: TrainUnit,
        smaller_is_better: bool,
        steps_completed: int,
        latest_checkpoint: Optional[str],
        local_training: bool,
        test_mode: bool,
        searcher_metric_name: Optional[str],
        checkpoint_policy: str,
        step_zero_validation: bool,
        max_length: Optional[TrainUnit],
        global_batch_size: Optional[int],
        profiling_enabled: Optional[bool],
    ) -> None:
        if not isinstance(trial_inst, PyTorchTrial):
            raise TypeError("PyTorchTrialController requires a PyTorchTrial.")
        self.trial = trial_inst
        self.context = context
        self.core_context = self.context._core

        self.local_training = local_training

        distributed_backend = det._DistributedBackend()
        self.use_horovod = distributed_backend.use_horovod()
        self.use_torch = distributed_backend.use_torch()
        self.is_chief = self.context.distributed.rank == 0

        # Training loop variables
        self.max_length = max_length
        self.checkpoint_period = checkpoint_period
        self.validation_period = validation_period
        self.reporting_period = reporting_period

        # Training loop state
        if local_training:
            self.trial_id = 0
            assert self.max_length, "max_length must be specified for local-training mode."
            self.searcher_unit = self.max_length._to_searcher_unit()
        else:
            self.trial_id = self.core_context.train._trial_id
            configured_units = self.core_context.searcher.get_configured_units()
            if configured_units is None:
                raise ValueError(
                    "Searcher units must be configured for training with PyTorchTrial."
                )
            self.searcher_unit = configured_units

        # Don't initialize the state here because it will be invalid until we load a checkpoint.
        self.state = None  # type: Optional[_TrialState]
        self.start_from_batch = steps_completed
        self.val_from_previous_run = self.core_context.train._get_last_validation()
        self.step_zero_validation = step_zero_validation

        # Training configs
        self.latest_checkpoint = latest_checkpoint
        self.test_mode = test_mode
        self.searcher_metric_name = searcher_metric_name
        self.ckpt_policy = checkpoint_policy
        self.smaller_is_better = smaller_is_better
        self.global_batch_size = global_batch_size
        self.profiling_enabled = profiling_enabled

        if self.searcher_unit == core.Unit.RECORDS:
            if self.global_batch_size is None:
                raise ValueError("global_batch_size required for searcher unit RECORDS.")

        self.callbacks = self.trial.build_callbacks()
        for callback in self.callbacks.values():
            if util.is_overridden(callback.on_checkpoint_end, pytorch.PyTorchCallback):
                warnings.warn(
                    "The on_checkpoint_end callback is deprecated, please use "
                    "on_checkpoint_write_end instead.",
                    FutureWarning,
                    stacklevel=2,
                )

        if len(self.context.models) == 0:
            raise det.errors.InvalidExperimentException(
                "Must have at least one model. "
                "This might be caused by not wrapping your model with wrap_model().",
            )
        if len(self.context.optimizers) == 0:
            raise det.errors.InvalidExperimentException(
                "Must have at least one optimizer. "
                "This might be caused by not wrapping your optimizer with wrap_optimizer().",
            )
        self._check_evaluate_implementation()

        # Currently only horovod and torch backends are supported for distributed training
        if self.context.distributed.size > 1:
            assert (
                self.use_horovod or self.use_torch
            ), "Must use horovod or torch for distributed training."

    @classmethod
    def pre_execute_hook(
        cls: Type["_PyTorchTrialController"],
        trial_seed: int,
        distributed_backend: det._DistributedBackend,
    ) -> None:
        # Initialize the correct horovod.
        if distributed_backend.use_horovod():
            hvd = horovod.hvd
            hvd.require_horovod_type("torch", "PyTorchTrial is in use.")
            hvd.init()
        if distributed_backend.use_torch():
            if torch.cuda.is_available():
                dist.init_process_group(backend="nccl")  # type: ignore
            else:
                dist.init_process_group(backend="gloo")  # type: ignore

        cls._set_random_seeds(trial_seed)

    def _upload_tb_files(self) -> None:
        self.context._maybe_reset_tbd_writer()
        self.core_context.train.upload_tensorboard_files(
            (lambda _: True) if self.is_chief else (lambda p: not p.match("*tfevents*")),
            tensorboard.util.get_rank_aware_path,
        )

    @classmethod
    def _set_random_seeds(cls: Type["_PyTorchTrialController"], seed: int) -> None:
        # Set identical random seeds on all training processes.
        # When using horovod, each worker will start at a unique
        # offset in the dataset, ensuring that it is processing a unique
        # training batch.
        random.seed(seed)
        np.random.seed(seed)
        torch.random.manual_seed(seed)
        # TODO(Aaron): Add flag to enable determinism.
        # torch.backends.cudnn.deterministic = True
        # torch.backends.cudnn.benchmark = False

    def _aggregate_training_metrics(self, training_metrics: List[Dict]) -> Dict:
        # Aggregate and reduce training metrics from all the training processes.
        if self.context.distributed.size > 1:
            batch_metrics = pytorch._combine_and_average_training_metrics(
                self.context.distributed, training_metrics
            )
        else:
            batch_metrics = training_metrics

        metrics = det.util.make_metrics(None, batch_metrics)

        # Ignore batch_metrics entirely for custom reducers; there's no guarantee that per-batch
        # metrics are even logical for a custom reducer.
        metrics["avg_metrics"].update(
            pytorch._convert_metrics_to_numpy(self.context.reduce_metrics(for_training=True))
        )

        if not self.is_chief:
            return {}

        # Only report on the chief worker
        avg_metrics = metrics.get("avg_metrics", {})
        batch_metrics = metrics.get("batch_metrics", [])

        assert self.state
        if self.context.get_enable_tensorboard_logging():
            pytorch._log_tb_metrics(
                self.context.get_tensorboard_writer(),
                "train",
                self.state.batches_trained,
                avg_metrics,
                batch_metrics,
            )

        self.core_context.train.report_training_metrics(
            steps_completed=self.state.batches_trained,
            metrics=avg_metrics,
            batch_metrics=batch_metrics,
        )
        return metrics

    def _is_best_validation(self, now: float, before: Optional[float]) -> bool:
        if before is None:
            return True

        return (now < before) if self.smaller_is_better else (now > before)

    def _on_epoch_start(self, epoch_idx: int) -> None:
        for callback in self.callbacks.values():
            sig = inspect.signature(callback.on_training_epoch_start)
            if sig.parameters:
                callback.on_training_epoch_start(epoch_idx)
            else:
                logger.warning(
                    "on_training_epoch_start() without parameters is deprecated"
                    " since 0.17.8. Please add epoch_idx parameter."
                )
                callback.on_training_epoch_start()  # type: ignore[call-arg]

    def _on_epoch_end(self, epoch_idx: int) -> None:
        for callback in self.callbacks.values():
            callback.on_training_epoch_end(epoch_idx)

    def _checkpoint(self, already_exiting: bool) -> None:
        if self.is_chief:
            self.core_context.train.set_status("checkpointing")

        assert self.state
        self.state.last_ckpt = self.state.batches_trained

        try:
            uuid = ""
            if self.is_chief:
                metadata = {
                    "determined_version": det.__version__,
                    "steps_completed": self.state.batches_trained,
                    "framework": f"torch-{torch.__version__}",
                    "format": "pickle",
                }
                with self.context._core.checkpoint.store_path(metadata) as (
                    path,
                    storage_id,
                ):
                    self._save(path)
                    uuid = storage_id
            uuid = self.context.distributed.broadcast(uuid)
            for callback in self.callbacks.values():
                callback.on_checkpoint_upload_end(uuid=uuid)
        except det.InvalidHP:
            if not already_exiting:
                self.core_context.train.report_early_exit(core.EarlyExitReason.INVALID_HP)
                raise ShouldExit(skip_exit_checkpoint=True)
            raise

    def _check_evaluate_implementation(self) -> None:
        """
        Check if the user has implemented evaluate_batch
        or evaluate_full_dataset.
        """
        logger.debug(f"Evaluate_batch_defined: {self._evaluate_batch_defined()}.")
        logger.debug(f"Evaluate full dataset defined: {self._evaluate_full_dataset_defined()}.")
        if self._evaluate_batch_defined() == self._evaluate_full_dataset_defined():
            raise det.errors.InvalidExperimentException(
                "Please define exactly one of: `evaluate_batch()` or `evaluate_full_dataset()`. "
                "For most use cases `evaluate_batch()` is recommended because "
                "it can be parallelized across all devices.",
            )

    def _evaluate_batch_defined(self) -> bool:
        return util.is_overridden(self.trial.evaluate_batch, PyTorchTrial)

    def _evaluate_full_dataset_defined(self) -> bool:
        return util.is_overridden(self.trial.evaluate_full_dataset, PyTorchTrial)

    def _set_data_loaders(self) -> None:
        skip_batches = self.start_from_batch

        num_replicas = self.context.distributed.size
        rank = self.context.distributed.rank

        train_data = self.trial.build_training_data_loader()
        if isinstance(train_data, pytorch.DataLoader):
            self.training_loader = train_data.get_data_loader(
                repeat=True, skip=skip_batches, num_replicas=num_replicas, rank=rank
            )
        else:
            # Non-determined DataLoader; ensure the user meant to do this.
            if not self.context.experimental._data_repro_checks_disabled:
                raise RuntimeError(
                    pytorch._dataset_repro_warning("build_training_data_loader", train_data)
                )
            self.training_loader = train_data

        # All workers use the chief's definition of epoch lengths, which is based on the training
        # loader's len. If this len does not exist, epoch durations cannot be deduced, and they
        # default to max int.
        try:
            epoch_len = len(self.training_loader)
        except TypeError:
            epoch_len = sys.maxsize
        self.context._epoch_len = self.context.distributed.broadcast(epoch_len)

        # Validation loader will be undefined on process ranks > 0
        # when the user defines `validate_full_dataset()`.
        self.validation_loader = None  # type: Optional[torch.utils.data.DataLoader]
        validation_data = self.trial.build_validation_data_loader()
        if self._evaluate_batch_defined():
            if isinstance(validation_data, pytorch.DataLoader):
                self.validation_loader = validation_data.get_data_loader(
                    repeat=False, skip=0, num_replicas=num_replicas, rank=rank
                )
            else:
                # Non-determined DataLoader; ensure the user meant to do this.
                if not self.context.experimental._data_repro_checks_disabled:
                    raise RuntimeError(
                        pytorch._dataset_repro_warning(
                            "build_validation_data_loader", validation_data
                        )
                    )
                self.validation_loader = validation_data
        elif self.is_chief:
            if isinstance(validation_data, pytorch.DataLoader):
                self.validation_loader = validation_data.get_data_loader(
                    repeat=False, skip=0, num_replicas=1, rank=0
                )
            else:
                # Non-determined DataLoader; ensure the user meant to do this.
                if not self.context.experimental._data_repro_checks_disabled:
                    raise RuntimeError(
                        pytorch._dataset_repro_warning(
                            "build_validation_data_loader", validation_data
                        )
                    )
                self.validation_loader = validation_data

    def _step_batch(self) -> None:
        assert self.state
        self.state.batches_trained += 1

        epoch_len = self.context._epoch_len
        assert epoch_len, "Training dataloader not initialized."

        # True epoch-based training is not supported. Epoch end is calculated with batch.
        epoch_idx, batch_in_epoch_idx = divmod(self.state.batches_trained - 1, epoch_len)

        if batch_in_epoch_idx == epoch_len - 1:
            self._on_epoch_end(epoch_idx)
            self.state.epochs_trained += 1

    def _stop_requested(self) -> None:
        if self.core_context.preempt.should_preempt():
            raise ShouldExit()
        if self.context.get_stop_requested():
            raise ShouldExit()

    def _report_searcher_progress(
        self, op: core.SearcherOperation, unit: Optional[core.Unit]
    ) -> None:
        assert self.state
        if unit == core.Unit.BATCHES:
            op.report_progress(self.state.batches_trained)
        elif unit == core.Unit.RECORDS:
            assert self.global_batch_size, "global_batch_size must be specified for RECORDS"
            op.report_progress(self.global_batch_size * self.state.batches_trained)
        elif unit == core.Unit.EPOCHS:
            op.report_progress(self.state.epochs_trained)

    def _checkpoint_is_current(self) -> bool:
        assert self.state
        # State always persists checkpoint step in batches
        return self.state.last_ckpt == self.state.batches_trained

    def _validation_is_current(self) -> bool:
        assert self.state
        # State persists validation step in batches
        return self.state.last_val == self.state.batches_trained

    def _steps_until_complete(self, train_unit: TrainUnit) -> int:
        assert isinstance(train_unit.value, int), "invalid length type"
        assert self.state
        if isinstance(train_unit, Batch):
            return train_unit.value - self.state.batches_trained
        elif isinstance(train_unit, Epoch):
            return train_unit.value - self.state.epochs_trained
        else:
            raise ValueError(f"Unrecognized train unit {train_unit}")

    def run(self) -> None:
        @contextlib.contextmanager
        def defer(fn: Callable, *args: Any) -> Iterator[None]:
            try:
                yield
            finally:
                fn(*args)

        # We define on_shutdown here instead of inside the `for callback in...` loop to ensure we
        # don't bind the loop iteration variable `callback`, which would likely cause us to call
        # on_trial_shutdown() multiple times for the final callback, and not at all for the others.
        def on_shutdown(callback_name: str, on_trial_shutdown: Callable) -> None:
            on_trial_shutdown()

        with contextlib.ExitStack() as exit_stack:
            for callback in self.callbacks.values():
                callback.on_trial_startup(self.start_from_batch, self.latest_checkpoint)
                exit_stack.enter_context(
                    defer(on_shutdown, callback.__class__.__name__, callback.on_trial_shutdown)
                )

            self._set_data_loaders()

            # We create the training_iterator (and training enumerator) here rather than in
            # __init__ because we have to be careful to trigger its shutdown explicitly, to avoid
            # hangs in when the user is using multiprocessing-based parallelism for their
            # dataloader.
            #
            # We create it before loading state because we don't want the training_iterator
            # shuffling values after we load state.
            self.training_iterator = iter(self.training_loader)
            self.training_enumerator = enumerate(
                dataloader_next(self.training_iterator), start=self.start_from_batch
            )

            def cleanup_iterator() -> None:
                # Explicitly trigger the training iterator's shutdown (which happens in __del__).
                # See the rather long note in pytorch/torch/utils/data/dataloader.py.
                del self.training_iterator
                del self.training_enumerator

            exit_stack.enter_context(defer(cleanup_iterator))

            # If a load path is provided load weights and restore the data location.
            if self.latest_checkpoint is not None:
                logger.info(f"Restoring trial from checkpoint {self.latest_checkpoint}")
                with self.context._core.checkpoint.restore_path(
                    self.latest_checkpoint
                ) as load_path:
                    self._load(load_path)
            else:
                # If we are not loading, initialize a fresh state.
                self.state = _TrialState(trial_id=self.trial_id)

            if self.context.distributed.size > 1 and self.use_horovod:
                hvd = horovod.hvd
                hvd.broadcast_parameters(self.context._main_model.state_dict(), root_rank=0)
                for optimizer in self.context.optimizers:
                    hvd.broadcast_optimizer_state(optimizer, root_rank=0)

            for callback in self.callbacks.values():
                callback.on_training_start()

            # Start the Determined system metrics profiler if enabled.
            if self.profiling_enabled:
                self.core_context.profiler.on()

            self._run()

    def _run(self) -> None:
        ops: Iterator[det.core.SearcherOperation]
        assert self.state

        try:
            if (
                self.step_zero_validation
                and self.val_from_previous_run is None
                and self.state.batches_trained == 0
            ):
                self._validate()

            if self.local_training:
                assert self.max_length and isinstance(self.max_length.value, int)
                ops = iter(
                    [
                        det.core.DummySearcherOperation(
                            length=self.max_length.value, is_chief=self.is_chief
                        )
                    ]
                )
            else:
                ops = self.core_context.searcher.operations()

            for op in ops:
                if self.local_training:
                    train_unit = self.max_length
                else:
                    train_unit = TrainUnit._from_searcher_unit(
                        op.length, self.searcher_unit, self.global_batch_size
                    )
                assert train_unit

                self._train_for_op(
                    op=op,
                    train_boundaries=[
                        _TrainBoundary(
                            step_type=_TrainBoundaryType.TRAIN,
                            unit=train_unit,
                        ),
                        _TrainBoundary(
                            step_type=_TrainBoundaryType.VALIDATE, unit=self.validation_period
                        ),
                        _TrainBoundary(
                            step_type=_TrainBoundaryType.CHECKPOINT,
                            unit=self.checkpoint_period,
                        ),
                        # Scheduling unit is always configured in batches
                        _TrainBoundary(
                            step_type=_TrainBoundaryType.REPORT, unit=self.reporting_period
                        ),
                    ],
                )
        except ShouldExit as e:
            # Checkpoint unsaved work and exit.
            if not e.skip_exit_checkpoint and not self._checkpoint_is_current():
                self._checkpoint(already_exiting=True)
        except det.InvalidHP as e:
            # Catch InvalidHP to checkpoint before exiting and re-raise for cleanup by core.init()
            if not self._checkpoint_is_current():
                self._checkpoint(already_exiting=True)
            raise e
        return

    def _train_with_boundaries(
        self, training_enumerator: Iterator, train_boundaries: List[_TrainBoundary]
    ) -> Tuple[List[_TrainBoundary], List]:
        training_metrics = []

        # Start of train step: tell core API and set model mode
        if self.is_chief:
            self.core_context.train.set_status("training")

        for model in self.context.models:
            model.train()

        self.context.reset_reducers()

        epoch_len = self.context._epoch_len
        assert epoch_len, "Training dataloader uninitialized."

        for batch_idx, batch in training_enumerator:
            epoch_idx, batch_in_epoch_idx = divmod(batch_idx, epoch_len)

            # Set the batch index on the trial context used by step_optimizer.
            self.context._current_batch_idx = batch_idx

            # Call epoch start callbacks before training first batch in epoch.
            if batch_in_epoch_idx == 0:
                self._on_epoch_start(epoch_idx)

            batch_metrics = self._train_batch(batch=batch, batch_idx=batch_idx, epoch_idx=epoch_idx)
            training_metrics.append(batch_metrics)
            self._step_batch()

            # Batch complete: check if any training periods have been reached and exit if any
            for step in train_boundaries:
                if isinstance(step.unit, Batch):
                    if step.unit.should_stop(batch_idx + 1):
                        step.limit_reached = True

                # True epoch based training not supported, detect last batch of epoch to calculate
                # fully-trained epochs
                if isinstance(step.unit, Epoch):
                    if step.unit.should_stop(epoch_idx + 1):
                        if batch_in_epoch_idx == epoch_len - 1:
                            step.limit_reached = True

                # Break early after one batch for test mode
                if step.step_type == _TrainBoundaryType.TRAIN and self.test_mode:
                    step.limit_reached = True

            # Exit if any train step limits have been reached
            if any(step.limit_reached for step in train_boundaries):
                return train_boundaries, training_metrics

        # True epoch end
        return train_boundaries, training_metrics

    def _train_for_op(
        self, op: core.SearcherOperation, train_boundaries: List[_TrainBoundary]
    ) -> None:
        if self.test_mode:
            train_length = Batch(1)
        elif self.local_training:
            train_length = self.max_length  # type: ignore
        else:
            train_length = TrainUnit._from_searcher_unit(
                op.length, self.searcher_unit, self.global_batch_size
            )  # type: ignore
        assert train_length

        while self._steps_until_complete(train_length) > 0:
            train_boundaries, training_metrics = self._train_with_boundaries(
                self.training_enumerator, train_boundaries
            )

            metrics = self._aggregate_training_metrics(training_metrics)
            metrics = self.context.distributed.broadcast(metrics)
            for callback in self.callbacks.values():
                callback.on_training_workload_end(
                    avg_metrics=metrics["avg_metrics"],
                    batch_metrics=metrics["batch_metrics"],
                )

            step_reported = False

            for train_boundary in train_boundaries:
                if not train_boundary.limit_reached:
                    continue

                # Train step limits reached, proceed accordingly.
                if train_boundary.step_type == _TrainBoundaryType.TRAIN:
                    if not op._completed and self.is_chief and not step_reported:
                        self._report_searcher_progress(op, self.searcher_unit)
                        step_reported = True
                elif train_boundary.step_type == _TrainBoundaryType.REPORT:
                    if not op._completed and self.is_chief and not step_reported:
                        self._report_searcher_progress(op, self.searcher_unit)
                        step_reported = True
                elif train_boundary.step_type == _TrainBoundaryType.VALIDATE:
                    if not self._validation_is_current():
                        self._validate(op)
                elif train_boundary.step_type == _TrainBoundaryType.CHECKPOINT:
                    if not self._checkpoint_is_current():
                        self._checkpoint(already_exiting=False)

                # Reset train step limit
                train_boundary.limit_reached = False

                # After checkpoint/validation steps, check preemption and upload to tensorboard
                if self.context.get_enable_tensorboard_logging():
                    self._upload_tb_files()
                self._stop_requested()

        # Finished training for op. Perform final checkpoint/validation if necessary.
        if not self._validation_is_current():
            self._validate(op)
        if not self._checkpoint_is_current():
            self._checkpoint(already_exiting=False)

        # Test mode will break after one batch despite not completing op.
        if self.is_chief and not self.test_mode:
            # The only case where op isn't reported as completed is if we restarted but
            # op.length was already trained for and validated on; in that case just raise
            # ShouldExit; we have nothing to do.
            if not op._completed:
                raise ShouldExit(skip_exit_checkpoint=True)

    def _check_searcher_metric(self, val_metrics: Dict) -> Any:
        if self.searcher_metric_name not in val_metrics:
            raise RuntimeError(
                f"Search method is configured to use metric '{self.searcher_metric_name}' but "
                f"model definition returned validation metrics {list(val_metrics.keys())}. The "
                f"metric used by the search method must be one of the validation "
                "metrics returned by the model definition."
            )

        # Check that the searcher metric has a scalar value so that it can be compared for
        # search purposes. Other metrics don't have to be scalars.
        searcher_metric = val_metrics[self.searcher_metric_name]
        if not util.is_numerical_scalar(searcher_metric):
            raise RuntimeError(
                f"Searcher validation metric '{self.searcher_metric_name}' returned "
                f"a non-scalar value: {searcher_metric}."
            )
        return searcher_metric

    def _get_epoch_idx(self, batch_id: int) -> int:
        assert self.context._epoch_len, "Training dataloader uninitialized."
        return batch_id // self.context._epoch_len

    def _auto_step_lr_scheduler_per_batch(
        self, batch_idx: int, lr_scheduler: pytorch.LRScheduler
    ) -> None:
        """
        This function automatically steps an LR scheduler. It should be called per batch.
        """
        # Never step lr when we do not step optimizer.
        if not self.context._should_communicate_and_update():
            return

        if lr_scheduler._step_mode == pytorch.LRScheduler.StepMode.STEP_EVERY_BATCH:
            start_idx = batch_idx - self.context._aggregation_frequency + 1
            for i in range(start_idx, batch_idx + 1):
                if (i + 1) % lr_scheduler._frequency == 0:
                    lr_scheduler.step()
        elif lr_scheduler._step_mode == pytorch.LRScheduler.StepMode.STEP_EVERY_OPTIMIZER_STEP:
            if (batch_idx + 1) % lr_scheduler._frequency == 0:
                lr_scheduler.step()
        elif lr_scheduler._step_mode == pytorch.LRScheduler.StepMode.STEP_EVERY_EPOCH:
            # We will step if the next optimizer step will land in the next epoch.
            epoch_idx = self._get_epoch_idx(batch_idx)
            next_steppable_batch = batch_idx + self.context._aggregation_frequency
            next_batch_epoch_idx = self._get_epoch_idx(next_steppable_batch)
            for e in range(epoch_idx, next_batch_epoch_idx):
                if (e + 1) % lr_scheduler._frequency == 0:
                    lr_scheduler.step()

    def _should_update_scaler(self) -> bool:
        if not self.context._scaler or not self.context.experimental._auto_amp:
            return False
        return self.context._should_communicate_and_update()

    def _train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, Any]:
        # Reset loss IDs for AMP
        self.context._loss_ids = {}

        batch_start_time = time.time()

        if self.context.experimental._auto_to_device:
            batch = self.context.to_device(batch)  # type: ignore

        with contextlib.ExitStack() as exit_stack:
            if self.context.profiler:
                exit_stack.enter_context(self.context.profiler)

            training_metrics = self.trial.train_batch(
                batch=batch,
                epoch_idx=epoch_idx,
                batch_idx=batch_idx,
            )

            if self.context.profiler:
                self.context.profiler.step()

        if self._should_update_scaler():
            # We update the scaler once after train_batch is done because the GradScaler is
            # expected to be one-per-training-loop, with one .update() call after all .step(opt)
            # calls for that batch are completed [1].
            #
            # [1] pytorch.org/docs/master/notes/amp_examples.html
            #         #working-with-multiple-models-losses-and-optimizers
            self.context._scaler.update()  # type: ignore

        if isinstance(training_metrics, torch.Tensor):
            training_metrics = {"loss": training_metrics}

        # Step learning rate of a pytorch.LRScheduler.
        for lr_scheduler in self.context.lr_schedulers:
            self._auto_step_lr_scheduler_per_batch(batch_idx, lr_scheduler)

        for name, metric in training_metrics.items():
            # Convert PyTorch metric values to NumPy, so that
            # `det.util.encode_json` handles them properly without
            # needing a dependency on PyTorch.
            if isinstance(metric, torch.Tensor):
                metric = metric.cpu().detach().numpy()
            training_metrics[name] = metric

        batch_dur = time.time() - batch_start_time
        samples_per_second = self.trial.get_batch_length(batch) / batch_dur
        samples_per_second *= self.context.distributed.size

        return training_metrics

    @torch.no_grad()  # type: ignore
    def _validate(self, searcher_op: Optional[core.SearcherOperation] = None) -> Dict[str, Any]:
        # Report a validation step is starting.
        if self.is_chief:
            self.core_context.train.set_status("validating")

        self.context.reset_reducers()

        # Set the behavior of certain layers (e.g., dropout) that are
        # different between training and inference.
        for model in self.context.models:
            model.eval()

        step_start_time = time.time()

        for callback in self.callbacks.values():
            callback.on_validation_start()

        num_inputs = 0
        metrics = {}  # type: Dict[str, Any]

        if self._evaluate_batch_defined():
            keys = None
            batch_metrics = []

            assert isinstance(self.validation_loader, torch.utils.data.DataLoader)
            for callback in self.callbacks.values():
                callback.on_validation_epoch_start()

            idx = -1  # Later, we'll use this default to see if we've iterated at all.
            for idx, batch in enumerate(iter(self.validation_loader)):
                if self.context.experimental._auto_to_device:
                    batch = self.context.to_device(batch)
                num_inputs += self.trial.get_batch_length(batch)

                if util.has_param(self.trial.evaluate_batch, "batch_idx", 2):
                    vld_metrics = self.trial.evaluate_batch(batch=batch, batch_idx=idx)
                else:
                    vld_metrics = self.trial.evaluate_batch(batch=batch)  # type: ignore
                # Verify validation metric names are the same across batches.
                if keys is None:
                    keys = vld_metrics.keys()
                else:
                    if keys != vld_metrics.keys():
                        raise ValueError(
                            "Validation metric names must match across all batches of data: "
                            f"{keys} != {vld_metrics.keys()}.",
                        )
                if not isinstance(vld_metrics, dict):
                    raise TypeError(
                        "validation_metrics() must return a "
                        "dictionary of string names to Tensor "
                        "metrics; "
                        f"got {vld_metrics}.",
                    )
                # TODO: For performance perform -> cpu() only at the end of validation.
                batch_metrics.append(pytorch._convert_metrics_to_numpy(vld_metrics))
                if self.test_mode:
                    break

            if idx == -1:
                raise RuntimeError("validation_loader is empty.")

            for callback in self.callbacks.values():
                callback.on_validation_epoch_end(batch_metrics)

            metrics = pytorch._reduce_metrics(
                self.context.distributed,
                batch_metrics=batch_metrics,
                keys=keys,
                metrics_reducers=pytorch._prepare_metrics_reducers(
                    self.trial.evaluation_reducer(), keys=keys
                ),
            )

            # Gather a list of per-worker (num_inputs, num_batches) tuples.
            input_counts = self.context.distributed.gather((num_inputs, idx + 1))

        else:
            assert self._evaluate_full_dataset_defined(), "evaluate_full_dataset not defined."
            assert self.validation_loader is not None
            if self.is_chief:
                metrics = self.trial.evaluate_full_dataset(data_loader=self.validation_loader)

                if not isinstance(metrics, dict):
                    raise TypeError(
                        f"eval() must return a dictionary, got {type(metrics).__name__}."
                    )

                metrics = pytorch._convert_metrics_to_numpy(metrics)

        metrics.update(
            pytorch._convert_metrics_to_numpy(self.context.reduce_metrics(for_training=False))
        )

        if self.context.distributed.size > 1 and any(
            util.is_overridden(c.on_validation_end, pytorch.PyTorchCallback)
            for c in self.callbacks.values()
        ):
            logger.debug(
                "Broadcasting metrics to all worker processes to execute a "
                "validation step end callback."
            )
            metrics = self.context.distributed.broadcast(metrics)

        for callback in self.callbacks.values():
            callback.on_validation_end(metrics)

        assert self.state
        self.state.last_val = self.state.batches_trained

        # Report metrics.
        if self.is_chief:
            # Skip reporting timings if evaluate_full_dataset() was defined.  This is far less
            # common than evaluate_batch() and we can't know how the user processed their
            # validation data.
            if self._evaluate_batch_defined():
                # Reshape and sum.
                # TODO: remove the type directive once we upgrade to mypy >= 1.7.0
                inputs_total, batches_total = [sum(n) for n in zip(*input_counts)]  # type: ignore
                step_duration = time.time() - step_start_time
                logger.info(
                    det.util.make_timing_log(
                        "validated", step_duration, inputs_total, batches_total
                    )
                )
            if self.context.get_enable_tensorboard_logging():
                pytorch._log_tb_metrics(
                    self.context.get_tensorboard_writer(),
                    "val",
                    self.state.batches_trained,
                    metrics,
                )

            # Get best validation before reporting metrics.
            best_validation_before = self.core_context.train.get_experiment_best_validation()

            self.core_context.train.report_validation_metrics(self.state.batches_trained, metrics)

        searcher_metric = None

        # Report searcher status.
        if self.is_chief and searcher_op:
            if self.local_training:
                searcher_length = self.max_length
            else:
                searcher_length = TrainUnit._from_searcher_unit(
                    searcher_op.length, self.searcher_unit, self.global_batch_size
                )
            if self.searcher_metric_name:
                searcher_metric = self._check_searcher_metric(metrics)

            assert searcher_length
            if self._steps_until_complete(searcher_length) < 1 and not searcher_op._completed:
                searcher_op.report_completed(searcher_metric)

        should_checkpoint = False

        # Checkpoint according to policy.
        if self.is_chief:
            if not self._checkpoint_is_current():
                if self.ckpt_policy == "all":
                    should_checkpoint = True
                elif self.ckpt_policy == "best":
                    assert (
                        self.searcher_metric_name
                    ), "checkpoint policy 'best' but searcher metric name not defined"
                    assert searcher_metric is not None

                    if self._is_best_validation(now=searcher_metric, before=best_validation_before):
                        should_checkpoint = True

        should_checkpoint = self.context.distributed.broadcast(should_checkpoint)
        if should_checkpoint:
            self._checkpoint(already_exiting=False)
        return metrics

    def _load(self, load_path: pathlib.Path) -> None:
        # Backwards compat with older checkpoint formats. List is of the newest to
        # the oldest known state_dict locations.
        potential_paths = [
            ["state_dict.pth"],
            ["determined", "state_dict.pth"],
            ["pedl", "state_dict.pth"],
            ["checkpoint.pt"],
        ]

        checkpoint: Optional[Dict[str, Any]] = None
        for ckpt_path in potential_paths:
            maybe_ckpt = load_path.joinpath(*ckpt_path)
            if maybe_ckpt.exists():
                checkpoint = torch.load(str(maybe_ckpt), map_location="cpu")  # type: ignore
                break

        if checkpoint is None or not isinstance(checkpoint, dict):
            return

        for callback in self.callbacks.values():
            callback.on_checkpoint_load_start(checkpoint)

        if "model_state_dict" in checkpoint:
            # Backward compatible with older checkpoint format.
            if "models_state_dict" in checkpoint:
                raise RuntimeError("Both model_state_dict and models_state_dict in checkpoint.")
            if len(self.context.models) > 1:
                raise RuntimeError(
                    "Old-format checkpoint cannot be loaded into a context with more than one "
                    "model."
                )
            self.context.models[0].load_state_dict(checkpoint["model_state_dict"])
        else:
            for idx, model in enumerate(self.context.models):
                model_state_dict = checkpoint["models_state_dict"][idx]
                try:
                    model.load_state_dict(model_state_dict)
                except Exception:
                    # If the checkpointed model is non-DDP and the current model is DDP, append
                    # module prefix to the checkpointed data
                    if isinstance(model, torch.nn.parallel.DistributedDataParallel):
                        logger.debug("Loading non-DDP checkpoint into a DDP model.")
                        self._add_prefix_in_state_dict_if_not_present(model_state_dict, "module.")
                    else:
                        # If the checkpointed model is DDP and if we are currently running in
                        # single-slot mode, remove the module prefix from checkpointed data
                        logger.debug("Loading DDP checkpoint into a non-DDP model.")
                        torch.nn.modules.utils.consume_prefix_in_state_dict_if_present(
                            model_state_dict, "module."
                        )
                    model.load_state_dict(model_state_dict)

        if "optimizer_state_dict" in checkpoint:
            # Backward compatible with older checkpoint format.
            if "optimizers_state_dict" in checkpoint:
                raise RuntimeError(
                    "Both optimizer_state_dict and optimizers_state_dict in checkpoint."
                )
            if len(self.context.optimizers) > 1:
                raise RuntimeError(
                    "Old-format checkpoint cannot be loaded into a context with more than one "
                    "optimizer."
                )
            self.context.optimizers[0].load_state_dict(checkpoint["optimizer_state_dict"])
        else:
            for idx, optimizer in enumerate(self.context.optimizers):
                optimizer.load_state_dict(checkpoint["optimizers_state_dict"][idx])

        if "lr_scheduler" in checkpoint:
            # Backward compatible with older checkpoint format.
            if "lr_schedulers_state_dict" in checkpoint:
                raise RuntimeError("Both lr_scheduler and lr_schedulers_state_dict in checkpoint.")
            if len(self.context.lr_schedulers) > 1:
                raise RuntimeError(
                    "Old-format checkpoint cannot be loaded into a context with more than one LR "
                    "scheduler."
                )
            self.context.lr_schedulers[0].load_state_dict(checkpoint["lr_scheduler"])
        else:
            for idx, lr_scheduler in enumerate(self.context.lr_schedulers):
                lr_scheduler.load_state_dict(checkpoint["lr_schedulers_state_dict"][idx])

        if "scaler_state_dict" in checkpoint:
            if self.context._scaler:
                self.context._scaler.load_state_dict(checkpoint["scaler_state_dict"])
            else:
                logger.warning(
                    "There exists scaler_state_dict in checkpoint but the experiment is not using "
                    "AMP."
                )
        else:
            if self.context._scaler:
                logger.warning(
                    "The experiment is using AMP but scaler_state_dict does not exist in the "
                    "checkpoint."
                )

        if "amp_state" in checkpoint:
            if self.context._use_apex:
                apex.amp.load_state_dict(checkpoint["amp_state"])
            else:
                logger.warning(
                    "There exists amp_state in checkpoint but the experiment is not using Apex."
                )
        else:
            if self.context._use_apex:
                logger.warning(
                    "The experiment is using Apex but amp_state does not exist in the checkpoint."
                )

        if "rng_state" in checkpoint:
            rng_state = checkpoint["rng_state"]
            np.random.set_state(rng_state["np_rng_state"])
            random.setstate(rng_state["random_rng_state"])
            torch.random.set_rng_state(rng_state["cpu_rng_state"])

            if torch.cuda.device_count():
                if "gpu_rng_state" in rng_state:
                    torch.cuda.set_rng_state(
                        rng_state["gpu_rng_state"], device=self.context.distributed.local_rank
                    )
                else:
                    logger.warning(
                        "The system has a gpu but no gpu_rng_state exists in the checkpoint."
                    )
            else:
                if "gpu_rng_state" in rng_state:
                    logger.warning(
                        "There exists gpu_rng_state in checkpoint but the system has no gpu."
                    )
        else:
            logger.warning("The checkpoint has no random state to restore.")

        callback_state = checkpoint.get("callbacks", {})
        for name in self.callbacks:
            if name in callback_state:
                self.callbacks[name].load_state_dict(callback_state[name])
            elif util.is_overridden(self.callbacks[name].load_state_dict, pytorch.PyTorchCallback):
                logger.warning(
                    f"Callback '{name}' implements load_state_dict(), but no callback state "
                    "was found for that name when restoring from checkpoint. This "
                    "callback will be initialized from scratch."
                )

        save_path = load_path.joinpath("trial_state.pkl")
        if save_path.exists():
            with save_path.open("rb") as f:
                self._load_state(pickle.load(f))
        else:
            # Support legacy save states.
            wlsq_path = load_path.joinpath("workload_sequencer.pkl")
            if wlsq_path.exists():
                with wlsq_path.open("rb") as f:
                    self._load_wlsq_state(pickle.load(f))

    def _load_state(self, state: Any) -> None:
        # Load our state from the checkpoint if we are continuing training after a pause or restart.
        # If the trial_id doesn't match our current trial id, we're continuing training a previous
        # trial and should start from a fresh state.
        if state.get("trial_id") != self.trial_id:
            self.state = _TrialState(trial_id=self.trial_id)
            return

        self.state = _TrialState(**state)
        assert self.state

        # Detect the case where the final validation we made was against this exact checkpoint.  In
        # that case, the master will know about the validation, but it would not appear in the
        # checkpoint state.  If the validation was before the last checkpoint, the checkpoint state
        # is already correct, while any validations after the last checkpoint aren't valid anymore
        # and can be safely ignored.
        if self.state.batches_trained == self.val_from_previous_run:
            self.state.last_val = self.state.batches_trained

    def _load_wlsq_state(self, state: Any) -> None:
        if state.get("trial_id") != self.trial_id:
            self.state = _TrialState(trial_id=self.trial_id)
            return

        self.state = _TrialState(
            trial_id=state.get("trial_id"),
            last_ckpt=state.get("last_ckpt"),
            last_val=state.get("last_val"),
            step_id=state.get("step_id"),
            # steps_completed is a legacy field kept to support loading from older checkpoints.
            # checkpoints should only persist batches_trained and epochs_trained
            batches_trained=state.get("steps_completed"),
            epochs_trained=self._get_epoch_idx(state.get("steps_completed")),
        )

        assert self.state
        if self.state.batches_trained == self.val_from_previous_run:
            self.state.last_val = self.state.batches_trained

    def _save(self, path: pathlib.Path) -> None:
        path.mkdir(parents=True, exist_ok=True)

        util.write_user_code(path, not self.local_training)

        rng_state = {
            "cpu_rng_state": torch.random.get_rng_state(),
            "np_rng_state": np.random.get_state(),
            "random_rng_state": random.getstate(),
        }

        if torch.cuda.device_count():
            rng_state["gpu_rng_state"] = torch.cuda.get_rng_state(
                self.context.distributed.local_rank
            )

        # PyTorch uses optimizer objects that take the model parameters to
        # optimize on construction, so we store and reload the `state_dict()`
        # of the model and optimizer explicitly (instead of dumping the entire
        # objects) to avoid breaking the connection between the model and the
        # optimizer.
        checkpoint = {
            "models_state_dict": [model.state_dict() for model in self.context.models],
            "optimizers_state_dict": [
                optimizer.state_dict() for optimizer in self.context.optimizers
            ],
            "lr_schedulers_state_dict": [
                lr_scheduler.state_dict() for lr_scheduler in self.context.lr_schedulers
            ],
            "callbacks": {name: callback.state_dict() for name, callback in self.callbacks.items()},
            "rng_state": rng_state,
        }

        if self.context._scaler:
            checkpoint["scaler_state_dict"] = self.context._scaler.state_dict()

        if self.context._use_apex:
            checkpoint["amp_state"] = apex.amp.state_dict()

        for callback in self.callbacks.values():
            callback.on_checkpoint_save_start(checkpoint)

        torch.save(checkpoint, str(path.joinpath("state_dict.pth")))

        assert self.state
        with path.joinpath("trial_state.pkl").open("wb") as f:
            pickle.dump(vars(self.state), f)

        trial_cls = type(self.trial)
        with open(path.joinpath("load_data.json"), "w") as f2:
            try:
                exp_conf = self.context.get_experiment_config()  # type: Optional[Dict[str, Any]]
                hparams = self.context.get_hparams()  # type: Optional[Dict[str, Any]]
            except ValueError:
                exp_conf = None
                hparams = None

            load_data = {
                "trial_type": "PyTorchTrial",
                "experiment_config": exp_conf,
                "hparams": hparams,
                "trial_cls_spec": f"{trial_cls.__module__}:{trial_cls.__qualname__}",
                "is_trainer": True,
            }

            if self.context._is_pre_trainer:
                load_data.pop("is_trainer")

            json.dump(load_data, f2)

        for callback in self.callbacks.values():
            # TODO(DET-7912): remove on_checkpoint_end once it has been deprecated long enough.
            callback.on_checkpoint_end(str(path))
            callback.on_checkpoint_write_end(str(path))

    def _sync_device(self) -> None:
        torch.cuda.synchronize(self.context.device)

    @staticmethod
    def _add_prefix_in_state_dict_if_not_present(state_dict: Dict[str, Any], prefix: str) -> None:
        """Adds the prefix in state_dict in place, if it does not exist.
        ..note::
            Given a `state_dict` from a non-DDP model, a DDP model can load it by applying
            `_add_prefix_in_state_dict_if_present(state_dict, "module.")` before calling
            :meth:`torch.nn.Module.load_state_dict`.
        Args:
            state_dict (OrderedDict): a state-dict to be loaded to the model.
            prefix (str): prefix.
        """
        keys = sorted(state_dict.keys())
        for key in keys:
            if not key.startswith(prefix):
                newkey = prefix + key
                state_dict[newkey] = state_dict.pop(key)

        # also add the prefix to metadata if not exists.
        if "_metadata" in state_dict:
            metadata = state_dict["_metadata"]
            for key in list(metadata.keys()):
                if not key.startswith(prefix):
                    newkey = prefix + key
                    metadata[newkey] = metadata.pop(key)


class PyTorchTrial(det.LegacyTrial):
    """
    PyTorch trials are created by subclassing this abstract class.

    We can do the following things in this trial class:

    * **Define models, optimizers, and LR schedulers**.

      In the :meth:`__init__` method, initialize models, optimizers, and LR schedulers
      and wrap them with ``wrap_model``, ``wrap_optimizer``, ``wrap_lr_scheduler``
      provided by :class:`~determined.pytorch.PyTorchTrialContext`.

    * **Run forward and backward passes**.

      In :meth:`train_batch`, call ``backward`` and ``step_optimizer`` provided by
      :class:`~determined.pytorch.PyTorchTrialContext`.
      We support arbitrary numbers of models, optimizers, and LR schedulers
      and arbitrary orders of running forward and backward passes.

    * **Configure automatic mixed precision**.

      In the :meth:`__init__` method, call ``configure_apex_amp`` provided by
      :class:`~determined.pytorch.PyTorchTrialContext`.

    * **Clip gradients**.

      In :meth:`train_batch`, pass a function into
      ``step_optimizer(optimizer, clip_grads=...)`` provided by
      :class:`~determined.pytorch.PyTorchTrialContext`.
    """

    trial_controller_class = _PyTorchTrialController  # type: ignore
    trial_context_class = pytorch.PyTorchTrialContext  # type: ignore

    @abc.abstractmethod
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        """
        Initializes a trial using the provided ``context``. The general steps are:

        #. Initialize model(s) and wrap them with ``context.wrap_model``.
        #. Initialize optimizer(s) and wrap them with ``context.wrap_optimizer``.
        #. Initialize learning rate schedulers and wrap them with ``context.wrap_lr_scheduler``.
        #. If desired, wrap models and optimizer with ``context.configure_apex_amp``
           to use ``apex.amp`` for automatic mixed precision.
        #. Define custom loss function and metric functions.

        .. warning::

           You may see metrics for trials that are paused and later continued that are significantly
           different from trials that are not paused if some of your models, optimizers, and
           learning rate schedulers are not wrapped. The reason is that the model's state may not be
           restored accurately or completely from the checkpoint, which is saved to a checkpoint and
           then later loaded into the trial during resumed training. When using PyTorch, this can
           sometimes happen if the PyTorch API is not used correctly.

        Here is a code example.

        .. code-block:: python

            self.context = context

            self.a = self.context.wrap_model(MyModelA())
            self.b = self.context.wrap_model(MyModelB())
            self.opt1 = self.context.wrap_optimizer(torch.optm.Adam(self.a))
            self.opt2 = self.context.wrap_optimizer(torch.optm.Adam(self.b))

            (self.a, self.b), (self.opt1, self.opt2) = self.context.configure_apex_amp(
                models=[self.a, self.b],
                optimizers=[self.opt1, self.opt2],
                num_losses=2,
            )

            self.lrs1 = self.context.wrap_lr_scheduler(
                lr_scheduler=LambdaLR(self.opt1, lr_lambda=lambda epoch: 0.95 ** epoch),
                step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
            ))
        """
        pass

    @abc.abstractmethod
    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        """
        Train on one batch.

        Users should implement this function by doing the following things:

        1. Run forward passes on the models.

        2. Calculate the gradients with the losses with ``context.backward``.

        3. Call an optimization step for the optimizers with ``context.step_optimizer``.
           You can clip gradients by specifying the argument ``clip_grads``.

        4. Step LR schedulers if using manual step mode.

        5. Return training metrics in a dictionary.

        Here is a code example.

        .. code-block:: python

            # Assume two models, two optimizers, and two LR schedulers were initialized
            # in ``__init__``.

            # Calculate the losses using the models.
            loss1 = self.model1(batch)
            loss2 = self.model2(batch)

            # Run backward passes on losses and step optimizers. These can happen
            # in arbitrary orders.
            self.context.backward(loss1)
            self.context.backward(loss2)
            self.context.step_optimizer(
                self.opt1,
                clip_grads=lambda params: torch.nn.utils.clip_grad_norm_(params, 0.0001),
            )
            self.context.step_optimizer(self.opt2)

            # Step the learning rate.
            self.lrs1.step()
            self.lrs2.step()

            return {"loss1": loss1, "loss2": loss2}

        Arguments:
            batch (Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor):
                batch of data for training.
            epoch_idx (integer): index of the current epoch among all the batches processed
                per device (slot) since the start of training.
            batch_idx (integer): index of the current batch among all the epochs processed
                per device (slot) since the start of training.
        Returns:
            torch.Tensor or Dict[str, Any]:
                training metrics to return.
        """
        pass

    @abc.abstractmethod
    def build_training_data_loader(self) -> Union[pytorch.DataLoader, torch.utils.data.DataLoader]:
        """
        Defines the data loader to use during training.

        Most implementations of :class:`determined.pytorch.PyTorchTrial` will return a
        :class:`determined.pytorch.DataLoader` here. Some use cases may not fit the assumptions of
        :class:`determined.pytorch.DataLoader`. In that event, a bare
        ``torch.utils.data.DataLoader`` may be returned if steps in the note atop
        :ref:`pytorch-reproducible-dataset` are followed.
        """
        pass

    @abc.abstractmethod
    def build_validation_data_loader(
        self,
    ) -> Union[pytorch.DataLoader, torch.utils.data.DataLoader]:
        """
        Defines the data loader to use during validation.

        Users with a MapDataset will normally return a :class:`determined.pytorch.DataLoader`, but
        users with an IterableDataset or with other advanced needs may sacrifice some
        Determined-managed functionality (ex: automatic data sharding) to return a bare
        :class:`torch.utils.data.DataLoader` following the best-practices described in
        :ref:`pytorch-reproducible-dataset`.
        """
        pass

    def build_callbacks(self) -> Dict[str, pytorch.PyTorchCallback]:
        """
        Defines a dictionary of string names to callbacks to be used during
        training and/or validation.

        The string name will be used as the key to save and restore callback
        state for any callback that defines :meth:`load_state_dict` and :meth:`state_dict`.
        """
        return {}

    def evaluate_batch(self, batch: pytorch.TorchData, batch_idx: int) -> Dict[str, Any]:
        """
        Calculate validation metrics for a batch and return them as a
        dictionary mapping metric names to metric values. Per-batch validation metrics
        are reduced (aggregated) to produce a single set of validation metrics for the
        entire validation set (see :meth:`evaluation_reducer`).

        There are two ways to specify evaluation metrics. Either override
        :meth:`evaluate_batch` or :meth:`evaluate_full_dataset`. While
        :meth:`evaluate_full_dataset` is more flexible,
        :meth:`evaluate_batch` should be preferred, since it can be
        parallelized in distributed environments, whereas
        :meth:`evaluate_full_dataset` cannot. Only one of
        :meth:`evaluate_full_dataset` and :meth:`evaluate_batch` should be
        overridden by a trial.

        The metrics returned from this function must be JSON-serializable.

        Arguments:
            batch (Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor):
                batch of data for evaluating.
            batch_idx (integer): index of the current batch among all the epochs processed
                per device (slot) since the start of training.
        """
        pass

    def evaluation_reducer(self) -> Union[pytorch.Reducer, Dict[str, pytorch.Reducer]]:
        """
        Return a reducer for all evaluation metrics, or a dict mapping metric
        names to individual reducers. Defaults to :obj:`determined.pytorch.Reducer.AVG`.
        """
        return pytorch.Reducer.AVG

    def evaluate_full_dataset(self, data_loader: torch.utils.data.DataLoader) -> Dict[str, Any]:
        """
        Calculate validation metrics on the entire validation dataset and
        return them as a dictionary mapping metric names to reduced metric
        values (i.e., each returned metric is the average or sum of that metric
        across the entire validation set).

        This validation cannot be distributed and is performed on a single
        device, even when multiple devices (slots) are used for training. Only
        one of :meth:`evaluate_full_dataset` and :meth:`evaluate_batch` should
        be overridden by a trial.

        The metrics returned from this function must be JSON-serializable.

        Arguments:
            data_loader (torch.utils.data.DataLoader): data loader for evaluating.
        """
        pass

    def get_batch_length(self, batch: Any) -> int:
        """Count the number of records in a given batch.

        Override this method when you are using custom batch types, as produced
        when iterating over the class:`determined.pytorch.DataLoader`.
        For example, when using ``pytorch_geometric``:

        .. code-block:: python

            # Extra imports:
            from determined.pytorch import DataLoader
            from torch_geometric.data.dataloader import Collater

            # Trial methods:
            def build_training_data_loader(self):
                return DataLoader(
                    self.train_subset,
                    batch_size=self.context.get_per_slot_batch_size(),
                    collate_fn=Collater([], []),
                )

            def get_batch_length(self, batch):
                # `batch` is `torch_geometric.data.batch.Batch`.
                return batch.num_graphs

        Arguments:
            batch (Any): input training or validation data batch object.
        """
        return pytorch.data_length(batch)
