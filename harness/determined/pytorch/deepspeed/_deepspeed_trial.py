import abc
import contextlib
import inspect
import json
import logging
import os
import pathlib
import pickle
import random
import time
import warnings
from typing import Any, Callable, Dict, Iterator, List, Optional, Tuple, Type, Union, cast

import deepspeed
import numpy as np
import torch
from deepspeed.runtime import dataloader as ds_loader

import determined as det
from determined import core, pytorch, tensorboard, util
from determined.pytorch import deepspeed as det_ds

logger = logging.getLogger("determined.pytorch")


# In most cases in which a user disables data reproducibility checks and chooses to return
# their own data loader, it will most likely be one created as part of DeepSpeed model engine
# initialization.  For the PipelineEngine, a RepeatingLoader is returned that does not have a
# __len__ method.  We patch in a length method here to make sure we can compute epoch length
# and validation length.
def get_length(self: ds_loader.RepeatingLoader) -> int:
    return len(self.loader)


def dataloader_next(dataloader_iter: Optional[Iterator]) -> Iterator:
    if dataloader_iter is None:
        return None
    while True:
        try:
            batch = next(dataloader_iter)
        except StopIteration:
            return
        yield batch


ds_loader.RepeatingLoader.__len__ = get_length


class DeepSpeedTrialController:
    def __init__(
        self,
        trial_inst: det.LegacyTrial,
        context: det_ds.DeepSpeedTrialContext,
        checkpoint_period: pytorch.TrainUnit,
        validation_period: pytorch.TrainUnit,
        reporting_period: pytorch.TrainUnit,
        smaller_is_better: bool,
        steps_completed: int,
        latest_checkpoint: Optional[str],
        local_training: bool,
        test_mode: bool,
        searcher_metric_name: Optional[str],
        checkpoint_policy: str,
        step_zero_validation: bool,
        max_length: pytorch.TrainUnit,
        global_batch_size: Optional[int],
        profiling_enabled: Optional[bool],
    ) -> None:
        assert isinstance(
            trial_inst, DeepSpeedTrial
        ), "DeepSpeedTrialController needs a DeepSpeedTrial"
        self.trial = trial_inst
        self.context = context
        self.core_context = self.context._core

        self.is_chief = self.context.distributed.rank == 0

        self.callbacks = self.trial.build_callbacks()
        for callback in self.callbacks.values():
            if util.is_overridden(callback.on_checkpoint_end, pytorch.PyTorchCallback):
                warnings.warn(
                    "The on_checkpoint_end callback is deprecated, please use "
                    "on_checkpoint_write_end instead",
                    FutureWarning,
                    stacklevel=2,
                )

        if len(self.context.models) == 0:
            raise det.errors.InvalidExperimentException(
                "Must have at least one model engine. "
                "This might be caused by not wrapping your model with wrap_model_engine()"
            )

        # Don't initialize the state here because it will be invalid until we load a checkpoint.
        self.state = None  # type: Optional[pytorch._TrialState]
        self.start_from_batch = steps_completed
        self.val_from_previous_run = self.core_context.train._get_last_validation()
        self.step_zero_validation = step_zero_validation

        # Training configs
        self.latest_checkpoint = latest_checkpoint
        self.test_mode = test_mode
        self.searcher_metric_name = searcher_metric_name
        self.checkpoint_policy = checkpoint_policy
        self.smaller_is_better = smaller_is_better
        self.global_batch_size = global_batch_size
        self.profiling_enabled = profiling_enabled

        # Training loop variables
        self.max_length = max_length
        self.checkpoint_period = checkpoint_period
        self.validation_period = validation_period
        self.reporting_period = reporting_period

        # Training loop state
        self.local_training = local_training
        self.trial_id = 0 if local_training else self.core_context.train._trial_id

    @classmethod
    def pre_execute_hook(
        cls: Type["DeepSpeedTrialController"],
        trial_seed: int,
        distributed_backend: det._DistributedBackend,
    ) -> None:
        # We use an environment variable to allow users to enable custom initialization routine for
        # distributed training since the pre_execute_hook runs before trial initialization.
        manual_dist_init = os.environ.get("DET_MANUAL_INIT_DISTRIBUTED")
        if not manual_dist_init:
            # DeepSpeed's init_distributed handles situations in which only 1 gpu is used and
            # also handles multiple calls to init in one process.
            deepspeed.init_distributed(auto_mpi_discovery=False)

        # Set identical random seeds on all training processes.
        # When data parallel world size > 1, each data parallel rank will start at a unique
        # offset in the dataset, ensuring it's processing a unique
        # training batch.
        # TODO (Liam): seed data loading workers so that we can configure different seeds for
        # data augmentations per slot per worker.
        random.seed(trial_seed)
        np.random.seed(trial_seed)
        torch.random.manual_seed(trial_seed)

    def _upload_tb_files(self) -> None:
        self.context._maybe_reset_tbd_writer()
        self.core_context.train.upload_tensorboard_files(
            (lambda _: True) if self.is_chief else (lambda p: not p.match("*tfevents*")),
            tensorboard.util.get_rank_aware_path,
        )

    def _set_data_loaders(self) -> None:
        skip_batches = self.start_from_batch

        # Training and validation data loaders are not built for every slot when model parallelism
        # is used.
        self.training_loader = None  # type: Optional[torch.utils.data.DataLoader]
        self.validation_loader = None  # type: Optional[torch.utils.data.DataLoader]
        self.num_validation_batches = None  # type: Optional[int]
        self.validation_batch_size = None  # type: Optional[int]

        # We currently only allow one model parallel strategy per DeepSpeedTrial.
        # We also assume that the data loader is tied to this one parallelization strategy.
        nreplicas = self.context._mpu.data_parallel_world_size
        rank = self.context._mpu.data_parallel_rank

        # The data loader is only required on ranks that take the data as input or require
        # the data to compute the loss.  There could be intermediate model parallel ranks
        # that do not need a data loader at all.
        if self.context._mpu.should_build_data_loader:
            train_data = self.trial.build_training_data_loader()
            if isinstance(train_data, pytorch.DataLoader):
                # Repeating the data loader is the default behavior for DeepSpeed data loaders when
                # using pipeline parallel.
                self.training_loader = train_data.get_data_loader(
                    repeat=True, skip=skip_batches, num_replicas=nreplicas, rank=rank
                )
            else:
                # Non-determined DataLoader; ensure the user meant to do this.
                if not self.context._data_repro_checks_disabled:
                    raise RuntimeError(
                        pytorch._dataset_repro_warning(
                            "build_training_data_loader", train_data, is_deepspeed_trial=True
                        )
                    )
                self.training_loader = train_data
                logger.warning("Please make sure custom data loader repeats indefinitely.")

            validation_data = self.trial.build_validation_data_loader()
            if isinstance(validation_data, pytorch.DataLoader):
                # For pipeline parallel models, we may evaluate on slightly fewer micro batches
                # than there would be in a full pass through the dataset due to automated
                # micro batch interleaving.
                self.validation_loader = validation_data.get_data_loader(
                    repeat=False, skip=0, num_replicas=nreplicas, rank=rank
                )

                if self.context.use_pipeline_parallel:
                    if len(self.validation_loader) < self.context.get_num_micro_batches_per_slot():
                        raise det.errors.InvalidExperimentException(
                            "Number of train micro batches in validation data loader should not be "
                            "less than the number of gradient accumulation steps when using "
                            "pipeline parallelism."
                        )
                    excluded_micro_batches = (
                        len(validation_data) % self.context.get_num_micro_batches_per_slot()
                    )
                    if excluded_micro_batches:
                        logger.warning(
                            "We will compute validation metrics over "
                            f"{excluded_micro_batches} fewer micro batches on rank "
                            f"{self.context.distributed.get_rank()}"
                        )
            else:
                # Non-determined DataLoader; ensure the user meant to do this.
                if not self.context._data_repro_checks_disabled:
                    raise RuntimeError(
                        pytorch._dataset_repro_warning(
                            "build_validation_data_loader", validation_data, is_deepspeed_trial=True
                        )
                    )
                if self.context.use_pipeline_parallel:
                    logger.warning(
                        "Using custom data loader, please make sure len(validation loader) is "
                        "divisible by gradient accumulation steps."
                    )
                self.validation_loader = validation_data

            # We use cast here instead of assert because the user can return an object that behaves
            # like a DataLoader but is not.
            self.validation_loader = cast(torch.utils.data.DataLoader, self.validation_loader)
            self.num_validation_batches = len(self.validation_loader)
            self.validation_batch_size = pytorch.data_length(next(iter(self.validation_loader)))

            if self.context.use_pipeline_parallel:
                self.num_validation_batches = (
                    self.num_validation_batches // self.context.get_num_micro_batches_per_slot()
                )
                self.validation_batch_size *= self.context.get_num_micro_batches_per_slot()

        # We will do a gather on to get train and val loader lengths and broadcast to all slots.
        self.context._epoch_len = (
            len(self.training_loader) if self.training_loader is not None else None
        )
        all_epoch_lens = self.context.distributed.gather(self.context._epoch_len)
        if self.is_chief:
            all_epoch_lens = [le for le in all_epoch_lens if le is not None]  # type: ignore
            if min(all_epoch_lens) < max(all_epoch_lens):
                logger.warning(
                    "Training data loader length inconsistent across ranks. "
                    "Using the minimum for epoch length."
                )
            self.context._epoch_len = (
                min(all_epoch_lens) // self.context.get_num_micro_batches_per_slot()
            )
        self.context._epoch_len = self.context.distributed.broadcast(self.context._epoch_len)

        all_tuples = self.context.distributed.gather(
            (self.num_validation_batches, self.validation_batch_size)
        )
        if self.is_chief:
            all_num_validation_batches, all_validation_batch_size = zip(*all_tuples)  # type: ignore
            all_num_validation_batches = [
                le for le in all_num_validation_batches if le is not None
            ]  # type: ignore
            if min(all_num_validation_batches) < max(all_num_validation_batches):
                logger.warning(
                    "Validation data loader length inconsistent across ranks. "
                    "Using the minimum for validation length."
                )
            self.num_validation_batches = min(all_num_validation_batches)
            all_validation_batch_size = [
                le for le in all_validation_batch_size if le is not None
            ]  # type: ignore
            if min(all_validation_batch_size) < max(all_validation_batch_size):
                logger.warning(
                    "Validation batch size inconsistent across ranks. "
                    "Num inputs tracking for validation will be incorrect."
                )
            self.validation_batch_size = min(all_validation_batch_size)

        (
            self.num_validation_batches,
            self.validation_batch_size,
        ) = self.context.distributed.broadcast(
            (self.num_validation_batches, self.validation_batch_size)
        )

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

            logger.info(self.context._mpu)

            self._set_data_loaders()

            # We create the training_iterator here rather than in __init__ because we have to be
            # careful to trigger its shutdown explicitly, to avoid hangs in when the user is using
            # multiprocessing-based parallelism for their data loader.
            #
            # We create it before loading state because we don't want the training_iterator
            # shuffling values after we load state.
            self.training_iterator = (
                iter(self.training_loader) if self.training_loader is not None else None
            )

            def cleanup_iterator() -> None:
                # Explicitly trigger the iterator's shutdown (which happens in __del__).
                # See the rather long note in pytorch/torch/utils/data/dataloader.py.
                del self.training_iterator

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
                self.state = pytorch._TrialState(trial_id=self.trial_id)

            for callback in self.callbacks.values():
                callback.on_training_start()

            # Start Determined system metrics profiling if enabled.
            if self.profiling_enabled:
                self.context._core.profiler.on()

            self._run()

    def _run(self) -> None:
        assert self.state

        try:
            if (
                self.step_zero_validation
                and self.val_from_previous_run is None
                and self.state.batches_trained == 0
            ):
                self._validate()

            self._train(
                length=pytorch.Batch(1) if self.test_mode else self.max_length,
                train_boundaries=[
                    pytorch._TrainBoundary(
                        step_type=pytorch._TrainBoundaryType.TRAIN,
                        unit=self.max_length,
                    ),
                    pytorch._TrainBoundary(
                        step_type=pytorch._TrainBoundaryType.VALIDATE, unit=self.validation_period
                    ),
                    pytorch._TrainBoundary(
                        step_type=pytorch._TrainBoundaryType.CHECKPOINT,
                        unit=self.checkpoint_period,
                    ),
                    # Scheduling unit is always configured in batches
                    pytorch._TrainBoundary(
                        step_type=pytorch._TrainBoundaryType.REPORT, unit=self.reporting_period
                    ),
                ],
            )
        except pytorch._ShouldExit as e:
            # Checkpoint unsaved work and exit.
            if not e.skip_exit_checkpoint and not self._checkpoint_is_current():
                self._checkpoint(already_exiting=True)

        except det.InvalidHP as e:
            # Catch InvalidHP to checkpoint before exiting and re-raise for cleanup by core.init()
            if not self._checkpoint_is_current():
                self._checkpoint(already_exiting=True)
            raise e

        return

    def _get_epoch_idx(self, batch_id: int) -> int:
        return batch_id // cast(int, self.context._epoch_len)

    def _train(
        self, length: pytorch.TrainUnit, train_boundaries: List[pytorch._TrainBoundary]
    ) -> None:
        while self._steps_until_complete(length) > 0:
            train_boundaries, training_metrics = self._train_with_boundaries(train_boundaries)

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
                if train_boundary.step_type == pytorch._TrainBoundaryType.TRAIN:
                    if self.is_chief and not step_reported:
                        self._report_training_progress()
                elif train_boundary.step_type == pytorch._TrainBoundaryType.REPORT:
                    if self.is_chief and not step_reported:
                        self._report_training_progress()
                elif train_boundary.step_type == pytorch._TrainBoundaryType.VALIDATE:
                    if not self._validation_is_current():
                        self._validate()
                elif train_boundary.step_type == pytorch._TrainBoundaryType.CHECKPOINT:
                    if not self._checkpoint_is_current():
                        self._checkpoint(already_exiting=False)

                # Reset train step limit
                train_boundary.limit_reached = False

                # After checkpoint/validation steps, check preemption and upload to tensorboard
                if self.context.get_enable_tensorboard_logging():
                    self._upload_tb_files()
                self._stop_requested()

        # Finished training. Perform final checkpoint/validation if necessary.
        if not self._validation_is_current():
            self._validate()
        if not self._checkpoint_is_current():
            self._checkpoint(already_exiting=False)

    def _train_with_boundaries(
        self, train_boundaries: List[pytorch._TrainBoundary]
    ) -> Tuple[List[pytorch._TrainBoundary], List]:
        training_metrics = []

        # Start of train step: tell core API and set model mode
        if self.is_chief:
            self.core_context.train.set_status("training")

        for model in self.context.models:
            model.train()

        self.context.reset_reducers()

        epoch_len = self.context._epoch_len
        assert epoch_len, "Training dataloader uninitialized."

        for batch_idx in range(epoch_len):
            epoch_idx, batch_in_epoch_idx = divmod(batch_idx, epoch_len)

            # Set the batch index on the trial context used by step_optimizer.
            self.context._current_batch_idx = batch_idx

            # Call epoch start callbacks before training first batch in epoch.
            if batch_in_epoch_idx == 0:
                self._on_epoch_start(epoch_idx)

            batch_metrics = self._train_batch(batch_idx=batch_idx, epoch_idx=epoch_idx)
            training_metrics.extend(batch_metrics)
            self._step_batch()

            # Batch complete: check if any training periods have been reached and exit if any
            for step in train_boundaries:
                if isinstance(step.unit, pytorch.Batch):
                    if step.unit.should_stop(batch_idx + 1):
                        step.limit_reached = True

                # True epoch based training not supported, detect last batch of epoch to calculate
                # fully-trained epochs
                if isinstance(step.unit, pytorch.Epoch):
                    if step.unit.should_stop(epoch_idx + 1):
                        if batch_in_epoch_idx == epoch_len - 1:
                            step.limit_reached = True

                # Break early after one batch for test mode
                if step.step_type == pytorch._TrainBoundaryType.TRAIN and self.test_mode:
                    step.limit_reached = True

            # Exit if any train step limits have been reached
            if any(step.limit_reached for step in train_boundaries):
                return train_boundaries, training_metrics

        # True epoch end
        return train_boundaries, training_metrics

    def _train_batch(self, epoch_idx: int, batch_idx: int) -> List[dict]:
        num_micro_batches = self.context.get_num_micro_batches_per_slot()
        if self.context.use_pipeline_parallel or self.context._manual_grad_accumulation:
            num_micro_batches = 1

        # Reset loss IDs for AMP
        self.context._loss_ids = {}

        batch_start_time = time.time()
        per_batch_metrics = []  # type: List[Dict]

        for _ in range(num_micro_batches):
            with contextlib.ExitStack() as exit_stack:
                if self.context.profiler:
                    exit_stack.enter_context(self.context.profiler)

                training_metrics = self.trial.train_batch(
                    self.training_iterator,
                    epoch_idx,
                    batch_idx,
                )

                if self.context.profiler:
                    self.context.profiler.step()

            if self.context._mpu.should_report_metrics:
                if isinstance(training_metrics, torch.Tensor):
                    training_metrics = {"loss": training_metrics}
                if not isinstance(training_metrics, dict):
                    raise det.errors.InvalidExperimentException(
                        "train_batch must return a dictionary "
                        f"mapping string names to Tensor metrics, got {type(training_metrics)}",
                    )

                for name, metric in training_metrics.items():
                    # Convert PyTorch metric values to NumPy, so that
                    # `det.util.encode_json` handles them properly without
                    # needing a dependency on PyTorch.
                    if isinstance(metric, torch.Tensor):
                        metric = metric.cpu().detach().numpy()
                    training_metrics[name] = metric
                per_batch_metrics.append(training_metrics)
        # We do a check here to make sure that we do indeed process `num_micro_batches_per_slot`
        # micro batches when training a batch for models that do not use pipeline parallelism.
        model0 = self.context.models[0]
        if not isinstance(model0, deepspeed.PipelineEngine):
            assert (
                model0.micro_steps % self.context.get_num_micro_batches_per_slot() == 0
            ), "did not train for gradient accumulation steps"

        batch_dur = time.time() - batch_start_time
        batch_inputs = (
            self.context.get_train_micro_batch_size_per_gpu()
            * self.context.get_num_micro_batches_per_slot()
        )
        samples_per_second = batch_inputs / batch_dur
        samples_per_second *= self.context.distributed.size

        # Aggregate and reduce training metrics from all the training processes.
        if self.context.distributed.size > 1:
            metrics = pytorch._combine_and_average_training_metrics(
                self.context.distributed, per_batch_metrics
            )
        else:
            metrics = per_batch_metrics

        return metrics

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
            metadata = {
                "determined_version": det.__version__,
                "steps_completed": self.state.batches_trained,
                "framework": f"torch-{torch.__version__}",
                "format": "pickle",
            }
            with self.context._core.checkpoint.store_path(metadata, shard=True) as (
                path,
                storage_id,
            ):
                self._save(path)
                uuid = storage_id
            for callback in self.callbacks.values():
                callback.on_checkpoint_upload_end(uuid=uuid)
        except det.InvalidHP:
            if not already_exiting:
                self.core_context.train.report_early_exit(core.EarlyExitReason.INVALID_HP)
                raise pytorch._ShouldExit(skip_exit_checkpoint=True)
            raise

    def _stop_requested(self) -> None:
        if self.core_context.preempt.should_preempt():
            raise pytorch._ShouldExit()
        if self.context.get_stop_requested():
            raise pytorch._ShouldExit()

    def _report_training_progress(self) -> None:
        assert self.state
        assert isinstance(self.max_length.value, int)

        if isinstance(self.max_length, pytorch.Batch):
            progress = self.state.batches_trained / self.max_length.value
        elif isinstance(self.max_length, pytorch.Epoch):
            progress = self.state.epochs_trained / self.max_length.value
        else:
            raise ValueError(f"unexpected train unit type {type(self.max_length)}")

        self.core_context.train.report_progress(progress=progress)

    def _checkpoint_is_current(self) -> bool:
        assert self.state
        # State always persists checkpoint step in batches
        return self.state.last_ckpt == self.state.batches_trained

    def _validation_is_current(self) -> bool:
        assert self.state
        # State persists validation step in batches
        return self.state.last_val == self.state.batches_trained

    def _steps_until_complete(self, train_unit: pytorch.TrainUnit) -> int:
        assert isinstance(train_unit.value, int), "invalid length type"
        assert self.state
        if isinstance(train_unit, pytorch.Batch):
            return train_unit.value - self.state.batches_trained
        elif isinstance(train_unit, pytorch.Epoch):
            return train_unit.value - self.state.epochs_trained
        else:
            raise ValueError(f"Unrecognized train unit {train_unit}")

    @torch.no_grad()
    def _validate(self) -> Dict[str, Any]:
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

        batches_evaluated = -1

        if self._evaluate_batch_defined():
            keys = None
            batch_metrics = []

            for callback in self.callbacks.values():
                callback.on_validation_epoch_start()

            validation_iterator = iter(self.validation_loader) if self.validation_loader else None
            for idx in range(cast(int, self.num_validation_batches)):
                batches_evaluated += 1
                num_inputs += cast(int, self.validation_batch_size)
                # Note that when using pipeline parallelism, each call to evaluate_batch will
                # request self.context.num_micro_batches_per_slot batches from the validation
                # iterator. This is why we set self.num_validation_batches differently for
                # pipeline parallel and no pipeline parallel when building the data loaders.
                if util.has_param(self.trial.evaluate_batch, "batch_idx", 2):
                    vld_metrics = self.trial.evaluate_batch(validation_iterator, idx)
                else:
                    vld_metrics = self.trial.evaluate_batch(validation_iterator)  # type: ignore
                if self.context._mpu.should_report_metrics:
                    if not isinstance(vld_metrics, dict):
                        raise det.errors.InvalidExperimentException(
                            "evaluate_batch must return a dictionary "
                            f"mapping string names to Tensor metrics, got {type(vld_metrics)}",
                        )
                    for name, metric in vld_metrics.items():
                        # Convert PyTorch metric values to NumPy, so that
                        # `det.util.encode_json` handles them properly without
                        # needing a dependency on PyTorch.
                        if isinstance(metric, torch.Tensor):
                            metric = metric.cpu().detach().numpy()
                        vld_metrics[name] = metric
                    # Verify validation metric names are the same across batches.
                    if keys is None:
                        keys = vld_metrics.keys()
                    else:
                        if keys != vld_metrics.keys():
                            raise ValueError(
                                "Validation metric names must match across all batches of data: "
                                f"{keys} != {vld_metrics.keys()}.",
                            )
                    batch_metrics.append(pytorch._convert_metrics_to_numpy(vld_metrics))
                if self.test_mode:
                    break

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
            input_counts = self.context.distributed.gather((num_inputs, batches_evaluated + 1))

        else:
            assert self._evaluate_full_dataset_defined(), "evaluate_full_dataset not defined."
            if self.is_chief:
                assert self.validation_loader is not None
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

            # We report "batch" and "epoch" only if these keys are not already reported in user
            # metrics.
            metrics["batches"] = metrics.get("batches", self.state.batches_trained)
            metrics["epochs"] = metrics.get("epochs", self.state.epochs_trained)

            self.core_context.train.report_validation_metrics(
                steps_completed=self.state.batches_trained, metrics=metrics
            )
        should_checkpoint = False

        # Checkpoint according to policy.
        if self.is_chief:
            if not self._checkpoint_is_current():
                if self.checkpoint_policy == "all":
                    should_checkpoint = True
                elif self.checkpoint_policy == "best":
                    assert (
                        self.searcher_metric_name
                    ), "checkpoint policy 'best' but searcher metric name not defined"
                    searcher_metric = self._check_searcher_metric(metrics)
                    assert searcher_metric is not None

                    if self._is_best_validation(now=searcher_metric, before=best_validation_before):
                        should_checkpoint = True
        should_checkpoint = self.context.distributed.broadcast(should_checkpoint)
        if should_checkpoint:
            self._checkpoint(already_exiting=False)
        return metrics

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

    def _evaluate_batch_defined(self) -> bool:
        return util.is_overridden(self.trial.evaluate_batch, DeepSpeedTrial)

    def _evaluate_full_dataset_defined(self) -> bool:
        return util.is_overridden(self.trial.evaluate_full_dataset, DeepSpeedTrial)

    def _load(self, load_path: pathlib.Path) -> None:
        # Right now we will load all checkpoint shards on each node regardless of which
        # checkpoints are needed.
        # TODO (Liam): revisit later to optimize sharded checkpoint loading.
        potential_paths = [
            ["state_dict.pth"],
            ["determined", "state_dict.pth"],
            ["pedl", "state_dict.pth"],
            ["checkpoint.pt"],
            [f"det_state_dict_rank{self.context.distributed.rank}.pth"],
        ]

        # Load stateful things tracked by Determined on all slots.
        checkpoint: Optional[Dict[str, Any]] = None
        for ckpt_path in potential_paths:
            maybe_ckpt = load_path.joinpath(*ckpt_path)
            if maybe_ckpt.exists():
                checkpoint = torch.load(str(maybe_ckpt), map_location="cpu")
                break

        if checkpoint is None or not isinstance(checkpoint, dict):
            return

        if not isinstance(checkpoint, dict):
            raise det.errors.InvalidExperimentException(
                f"Expected checkpoint at {maybe_ckpt} to be a dict "
                f"but got {type(checkpoint).__name__}."
            )

        for callback in self.callbacks.values():
            callback.on_checkpoint_load_start(checkpoint)

        # We allow users to override load behavior if needed, but we default to using
        # the load method provided by DeepSpeed.
        self.trial.load(self.context, load_path)

        if "rng_state" in checkpoint:
            rng_state = checkpoint["rng_state"]
            np.random.set_state(rng_state["np_rng_state"])
            random.setstate(rng_state["random_rng_state"])
            torch.random.set_rng_state(rng_state["cpu_rng_state"])

            if torch.cuda.device_count():
                if "gpu_rng_state" in rng_state:
                    torch.cuda.set_rng_state(
                        rng_state["gpu_rng_state"], device=self.context.distributed.get_local_rank()
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
                    "Callback '{}' implements load_state_dict(), but no callback state "
                    "was found for that name when restoring from checkpoint. This "
                    "callback will be initialized from scratch"
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
            self.state = pytorch._TrialState(trial_id=self.trial_id)
            return

        self.state = pytorch._TrialState(**state)
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
            self.state = pytorch._TrialState(trial_id=self.trial_id)
            return

        self.state = pytorch._TrialState(
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

        if self.is_chief:
            # We assume these stateful objects should be the same across slots and only have
            # the chief save them.
            util.write_user_code(path, not self.local_training)
            assert self.state
            with path.joinpath("trial_state.pkl").open("wb") as f:
                pickle.dump(vars(self.state), f)

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
            "callbacks": {name: callback.state_dict() for name, callback in self.callbacks.items()},
            "rng_state": rng_state,
        }

        for callback in self.callbacks.values():
            callback.on_checkpoint_save_start(checkpoint)
        ckpt_name = f"det_state_dict_rank{self.context.distributed.rank}.pth"
        torch.save(checkpoint, str(path.joinpath(ckpt_name)))

        # We allow users to override save behavior if needed, but we default to using
        # the save method provided by DeepSpeed.
        self.trial.save(self.context, path)

        with open(path.joinpath("load_data.json"), "w") as f2:
            try:
                exp_conf = self.context.get_experiment_config()  # type: Optional[Dict[str, Any]]
                hparams = self.context.get_hparams()  # type: Optional[Dict[str, Any]]
            except ValueError:
                exp_conf = None
                hparams = None

            load_data = {
                "trial_type": "DeepSpeedTrial",
                "experiment_config": exp_conf,
                "hparams": hparams,
            }

            json.dump(load_data, f2)

        for callback in self.callbacks.values():
            # TODO(DET-7912): remove on_checkpoint_end once it has been deprecated long enough.
            callback.on_checkpoint_end(str(path))
            callback.on_checkpoint_write_end(str(path))

    def _sync_device(self) -> None:
        torch.cuda.synchronize(self.context.device)


class DeepSpeedTrial(det.LegacyTrial):
    """
    DeepSpeed trials are created by subclassing this abstract class.

    We can do the following things in this trial class:

    * **Define the DeepSpeed model engine which includes the model, optimizer, and lr_scheduler**.

       In the :meth:`__init__` method, initialize models and, optionally, optimizers and
       LR schedulers and pass them to ``deepspeed.initialize`` to build the model engine.  Then
       pass the created model engine to ``wrap_model_engine`` provided by
       :class:`~determined.pytorch.deepspeed.DeepSpeedTrialContext`.
       We support multiple DeepSpeed model engines if they only use data parallelism or if
       they use the same model parallel unit.

    * **Run forward and backward passes**.

       In :meth:`train_batch`, use the methods provided by the DeepSpeed model engine to perform
       the backward pass and optimizer step.  These methods will differ depending on whether
       you are using pipeline parallelism or not.

    """

    trial_controller_class = DeepSpeedTrialController  # type: ignore
    trial_context_class = det_ds.DeepSpeedTrialContext  # type: ignore

    @abc.abstractmethod
    def __init__(self, context: det_ds.DeepSpeedTrialContext) -> None:
        """
        Initializes a trial using the provided ``context``. The general steps are:

        #. Initialize the model(s) and, optionally, the optimizer and lr_scheduler.  The latter
           two can also be configured using the DeepSpeed config.
        #. Build the DeepSpeed model engine by calling ``deepspeed.initialize`` with the model
           (optionally optimizer and lr scheduler) and a DeepSpeed config.  Wrap it with
           ``context.wrap_model_engine``.
        #. If you want, use a custom model parallel unit by calling ``context.set_mpu``.
        #. If you want, disable automatic gradient accumulation by calling
           ``context.disable_auto_grad_accumulation``.
        #. If you want, use a custom data loader by calling
           ``context.disable_dataset_reproducibility_checks``.

        Here is a code example.

        .. code-block:: python

            self.context = context
            self.args = AttrDict(self.context.get_hparams())

            # Build deepspeed model engine.
            model = ... # build model
            model_engine, optimizer, lr_scheduler, _ = deepspeed.initialize(
                args=self.args,
                model=model,
            )

            self.model_engine = self.context.wrap_model_engine(model_engine)
        """
        pass

    @abc.abstractmethod
    def train_batch(
        self,
        dataloader_iter: Optional[Iterator[pytorch.TorchData]],
        epoch_idx: int,
        batch_idx: int,
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        """
        Train one full batch (i.e. train on ``train_batch_size`` samples, perhaps consisting
        of multiple micro-batches).

        If training without pipeline parallelism, users should implement this function by doing
        the following things:

        #. Get a batch from the ``dataloader_iter`` and pass it to the GPU.
        #. Compute the loss in the forward pass.
        #. Perform the backward pass.
        #. Perform an optimizer step.
        #. Return training metrics in a dictionary.

        Here is a code example.

        .. code-block:: python

            # Assume one model_engine wrapped in ``__init__``.

            batch = self.context.to_device(next(dataloader_iter))
            loss = self.model_engine(batch)
            self.model_engine.backward(loss)
            self.model_engine.step()
            return {"loss": loss}

        If using gradient accumulation over multiple micro-batches, Determined will automatically
        call ``train_batch`` multiple times according to ``gradient_accumulation_steps`` in the
        DeepSpeed config.

        With pipeline parallelism there is no need to manually get a batch from the
        ``dataloader_iter`` and the forward, backward, optimizer steps are combined in the
        model engine's ``train_batch`` method.

        .. code-block:: python

            # Assume one model_engine wrapped in ``__init__``.

            loss = self.model_engine.train_batch(dataloader_iter)
            return {"loss": loss}

        Arguments:
            dataloader_iter (Iterator[torch.utils.data.DataLoader], optional): iterator over
                the train DataLoader.
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
    def build_training_data_loader(self) -> Optional[pytorch.DataLoader]:
        """
        Defines the data loader to use during training.

        Must return an instance of :py:class:`determined.pytorch.DataLoader` unless
        ``context.disable_dataset_reproducibility_checks`` is called.

        If using data parallel training, the batch size should be per GPU batch size.
        If using gradient aggregation, the data loader should return batches with
        ``train_micro_batch_size_per_gpu`` samples each.
        """
        pass

    @abc.abstractmethod
    def build_validation_data_loader(self) -> Optional[pytorch.DataLoader]:
        """
        Defines the data loader to use during validation.

        Must return an instance of :py:class:`determined.pytorch.DataLoader` unless
        ``context.disable_dataset_reproducibility_checks`` is called.

        If using data parallel training, the batch size should be per GPU batch size.
        If using gradient aggregation, the data loader should return batches with a desired
        micro batch size (most of the time this is the same as ``train_micro_batch_size_per_gpu``).
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

    @abc.abstractmethod
    def evaluate_batch(
        self, dataloader_iter: Optional[Iterator[pytorch.TorchData]], batch_idx: int
    ) -> Dict[str, Any]:
        """
        Calculate validation metrics for a batch and return them as a
        dictionary mapping metric names to metric values. Per-batch validation metrics
        are averaged to produce a single set of validation metrics for the entire
        validation set by default.

        The metrics returned from this function must be JSON-serializable.

        DeepSpeedTrial supports more flexible metrics computation via our custom reducer API,
        see :class:`~determined.pytorch.MetricReducer` for more details.

        Arguments:
            dataloader_iter (Iterator[torch.utils.data.DataLoader], optional): iterator over the
                validation DataLoader.
        """
        pass

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

    def evaluation_reducer(self) -> Union[pytorch.Reducer, Dict[str, pytorch.Reducer]]:
        """
        Return a reducer for all evaluation metrics, or a dict mapping metric
        names to individual reducers. Defaults to :obj:`determined.pytorch.Reducer.AVG`.
        """
        return pytorch.Reducer.AVG

    def save(self, context: det_ds.DeepSpeedTrialContext, path: pathlib.Path) -> None:
        """
        Save is called on every GPU to make sure all checkpoint shards are saved.

        By default, we loop through the wrapped model engines and call DeepSpeed's save:

        .. code-block:: python

            for i, m in enumerate(context.models):
                m.save_checkpoint(path, tag=f"model{i}")

        This method can be overwritten for more custom save behavior.
        """
        for i, m in enumerate(context.models):
            m.save_checkpoint(path, tag=f"model{i}")

    def load(
        self,
        context: det_ds.DeepSpeedTrialContext,
        load_path: pathlib.Path,
    ) -> None:
        """
        By default, we loop through the wrapped model engines and call DeepSpeed's load.

        .. code-block:: python

            for i, m in enumerate(context.models):
                m.load_checkpoint(path, tag=f"model{i}")

        This method can be overwritten for more custom load behavior.
        """
        for i, m in enumerate(context.models):
            try:
                m.load_checkpoint(load_path, tag=f"model{i}")
            except AssertionError:
                # DeepSpeed does not provide an error message with many assertion errors in the
                # checkpoint load module.
                raise AssertionError("Failed to load deepspeed checkpoint.")

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
