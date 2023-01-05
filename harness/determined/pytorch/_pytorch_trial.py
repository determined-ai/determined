import contextlib
import json
import logging
import pathlib
import pickle
import random
import sys
import time
import warnings
from abc import abstractmethod
from inspect import signature
from typing import Any, Callable, Dict, Iterator, List, Optional, Type, Union, cast

import numpy as np
import torch
import torch.distributed as dist

import determined as det
from determined import layers, pytorch, tensorboard, util, workload
from determined.horovod import hvd
from determined.util import has_param

# Apex is included only for GPU trials.
try:
    import apex
except ImportError:  # pragma: no cover
    apex = None
    pass


class PyTorchTrialController(det.TrialController):
    def __init__(self, trial_inst: det.Trial, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        if not isinstance(trial_inst, PyTorchTrial):
            raise TypeError("PyTorchTrialController requires a PyTorchTrial.")
        self.trial = trial_inst
        self.context = cast(pytorch.PyTorchTrialContext, self.context)
        self.context._set_determined_profiler(self.prof)
        if torch.cuda.is_available():
            self.prof._set_sync_device(self._sync_device)
        self.callbacks = self.trial.build_callbacks()
        for callback in self.callbacks.values():
            if util.is_overridden(callback.on_checkpoint_end, pytorch.PyTorchCallback):
                warnings.warn(
                    "The on_checkpoint_end callback is deprecated, please use "
                    "on_checkpoint_write_end instead.",
                    FutureWarning,
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

        self.wlsq = None  # type: Optional[layers.WorkloadSequencer]
        if self.workloads is None:
            self.workloads, self.wlsq = layers.make_compatibility_workloads(
                self.context._core,
                self.env,
                self.context.get_global_batch_size(),
            )

        self.steps_completed = self.env.steps_completed

        # Currently only horovod and torch backends are supported for distributed training
        if self.context.distributed.size > 1:
            assert (
                self.use_horovod or self.use_torch
            ), "Must use horovod or torch for distributed training"

    @classmethod
    def create_metric_writer(
        cls: Type["PyTorchTrialController"],
    ) -> tensorboard.BatchMetricWriter:
        from determined.tensorboard.metric_writers.pytorch import TorchWriter

        writer = TorchWriter()
        return tensorboard.BatchMetricWriter(writer)

    @classmethod
    def pre_execute_hook(
        cls: Type["PyTorchTrialController"],
        env: det.EnvContext,
        distributed_backend: det._DistributedBackend,
    ) -> None:
        # Initialize the correct horovod.
        if distributed_backend.use_horovod():
            hvd.require_horovod_type("torch", "PyTorchTrial is in use.")
            hvd.init()
        if distributed_backend.use_torch():
            if torch.cuda.is_available():
                dist.init_process_group(backend="nccl")  # type: ignore
            else:
                dist.init_process_group(backend="gloo")  # type: ignore

        cls._set_random_seeds(env.trial_seed)

    @classmethod
    def _set_random_seeds(cls: Type["PyTorchTrialController"], seed: int) -> None:
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

    @classmethod
    def from_trial(
        cls: Type["PyTorchTrialController"], *args: Any, **kwargs: Any
    ) -> det.TrialController:
        return cls(*args, **kwargs)

    @classmethod
    def supports_mixed_precision(cls: Type["PyTorchTrialController"]) -> bool:
        return True

    def _check_evaluate_implementation(self) -> None:
        """
        Check if the user has implemented evaluate_batch
        or evaluate_full_dataset.
        """
        logging.debug(f"Evaluate_batch_defined: {self._evaluate_batch_defined()}.")
        logging.debug(f"Evaluate full dataset defined: {self._evaluate_full_dataset_defined()}.")
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
        skip_batches = self.env.steps_completed

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
            with self.prof.record_timing(f"callbacks.{callback_name}.on_trial_shutdown"):
                on_trial_shutdown()

        with contextlib.ExitStack() as exit_stack:
            for callback in self.callbacks.values():
                with self.prof.record_timing(
                    f"callbacks.{callback.__class__.__name__}.on_trial_startup"
                ):
                    callback.on_trial_startup(self.steps_completed, self.env.latest_checkpoint)
                exit_stack.enter_context(
                    defer(on_shutdown, callback.__class__.__name__, callback.on_trial_shutdown)
                )

            self._set_data_loaders()

            # We create the training_iterator here rather than in __init__ because we have to be
            # careful to trigger its shutdown explicitly, to avoid hangs in when the user is using
            # multiprocessing-based parallelism for their dataloader.
            #
            # We create it before loading state because we don't want the training_iterator
            # shuffling values after we load state.
            self.training_iterator = iter(self.training_loader)

            def cleanup_iterator() -> None:
                # Explicitly trigger the training iterator's shutdown (which happens in __del__).
                # See the rather long note in pytorch/torch/utils/data/dataloader.py.
                del self.training_iterator

            exit_stack.enter_context(defer(cleanup_iterator))

            # If a load path is provided load weights and restore the data location.
            if self.env.latest_checkpoint is not None:
                logging.info(f"Restoring trial from checkpoint {self.env.latest_checkpoint}")
                with self.context._core.checkpoint.restore_path(
                    self.env.latest_checkpoint
                ) as load_path:
                    self._load(load_path)

            if self.context.distributed.size > 1 and self.use_horovod:
                hvd.broadcast_parameters(self.context._main_model.state_dict(), root_rank=0)
                for optimizer in self.context.optimizers:
                    hvd.broadcast_optimizer_state(optimizer, root_rank=0)

            with self.prof:
                for callback in self.callbacks.values():
                    with self.prof.record_timing(
                        f"callbacks.{callback.__class__.__name__}.on_training_start"
                    ):
                        callback.on_training_start()
                self._run()

    def _run(self) -> None:
        assert self.workloads is not None
        for w, response_func in self.workloads:
            try:
                if w.kind == workload.Workload.Kind.RUN_STEP:
                    action = "training"
                    metrics = self._train_for_step(
                        w.step_id,
                        w.num_batches,
                        w.total_batches_processed,
                    )
                    response = {
                        "metrics": metrics,
                        "stop_requested": self.context.get_stop_requested(),
                    }  # type: workload.Response
                    metrics = self.context.distributed.broadcast(metrics)
                    for callback in self.callbacks.values():
                        callback.on_training_workload_end(
                            avg_metrics=metrics["avg_metrics"],
                            batch_metrics=metrics["batch_metrics"],
                        )
                    if (
                        self.context.distributed.size > 1
                        and not self.context._average_training_metrics
                    ):
                        warnings.warn(
                            "Only the chief worker's training metrics are being reported, due "
                            "to setting average_training_metrics to False.",
                            UserWarning,
                        )
                elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                    action = "validation"
                    response = {
                        "metrics": self._compute_validation_metrics(),
                        "stop_requested": self.context.get_stop_requested(),
                    }
                elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                    action = "checkpointing"
                    uuid = ""
                    if self.is_chief:
                        metadata = {
                            "determined_version": det.__version__,
                            "steps_completed": self.steps_completed,
                            "framework": f"torch-{torch.__version__}",
                            "format": "pickle",
                        }
                        with self.context._core.checkpoint.store_path(metadata) as (
                            path,
                            storage_id,
                        ):
                            self._save(path)
                            uuid = storage_id
                        response = {"uuid": storage_id}
                    else:
                        response = {}
                    uuid = self.context.distributed.broadcast(uuid)
                    for callback in self.callbacks.values():
                        callback.on_checkpoint_upload_end(uuid=uuid)

                else:
                    raise AssertionError("Unexpected workload: {}".format(w.kind))

            except det.InvalidHP as e:
                logging.info(f"Invalid hyperparameter exception during {action}: {e}")
                response = workload.InvalidHP()
            response_func(response)
            self.upload_tb_files()

    def get_epoch_idx(self, batch_id: int) -> int:
        return batch_id // self.context._epoch_len  # type: ignore

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
            epoch_idx = self.get_epoch_idx(batch_idx)
            next_steppable_batch = batch_idx + self.context._aggregation_frequency
            next_batch_epoch_idx = self.get_epoch_idx(next_steppable_batch)
            for e in range(epoch_idx, next_batch_epoch_idx):
                if (e + 1) % lr_scheduler._frequency == 0:
                    lr_scheduler.step()

    def _should_update_scaler(self) -> bool:
        if not self.context._scaler or not self.context.experimental._auto_amp:
            return False
        return self.context._should_communicate_and_update()  # type: ignore

    def _train_for_step(
        self, step_id: int, num_batches: int, total_batches_processed: int
    ) -> workload.Metrics:
        self.prof.set_training(True)
        step_start_time = time.time()
        self.context.reset_reducers()

        # Set the behavior of certain layers (e.g., dropout) that are different
        # between training and inference.
        for model in self.context.models:
            model.train()

        start = total_batches_processed
        end = start + num_batches

        per_batch_metrics = []  # type: List[Dict]
        num_inputs = 0

        for batch_idx in range(start, end):
            self.steps_completed += 1
            batch_start_time = time.time()
            self.prof.update_batch_idx(batch_idx)
            with self.prof.record_timing("dataloader_next", requires_sync=False):
                batch = next(self.training_iterator)
            batch_inputs = self.trial.get_batch_length(batch)
            num_inputs += batch_inputs

            if self.context.experimental._auto_to_device:
                with self.prof.record_timing("to_device", accumulate=True):
                    batch = self.context.to_device(batch)

            self.context._current_batch_idx = batch_idx
            epoch_idx = self.get_epoch_idx(batch_idx)
            if self.context.is_epoch_start():
                for callback in self.callbacks.values():
                    with self.prof.record_timing(
                        f"callbacks.{callback.__class__.__name__}.on_training_epoch_start"
                    ):
                        sig = signature(callback.on_training_epoch_start)
                        if sig.parameters:
                            callback.on_training_epoch_start(epoch_idx)
                        else:
                            logging.warning(
                                "on_training_epoch_start() without parameters is deprecated"
                                " since 0.17.8. Please add epoch_idx parameter."
                            )
                            callback.on_training_epoch_start()  # type: ignore[call-arg]

            self.context._loss_ids = {}

            with self.prof.record_timing("train_batch", requires_sync=False):
                if self.context.profiler:
                    with self.context.profiler as torch_profiler:
                        tr_metrics = self.trial.train_batch(
                            batch=batch,
                            epoch_idx=epoch_idx,
                            batch_idx=batch_idx,
                        )
                        torch_profiler.step()
                else:
                    tr_metrics = self.trial.train_batch(
                        batch=batch,
                        epoch_idx=epoch_idx,
                        batch_idx=batch_idx,
                    )
            if self._should_update_scaler():
                # We update the scaler once after train_batch is done because the GradScaler is
                # expected to be one-per-training-loop, with one .update() call after all .step(opt)
                # calls for that batch are completed [1].
                #
                # [1] pytorch.org/docs/master/notes/amp_examples.html
                #         #working-with-multiple-models-losses-and-optimizers
                self.context._scaler.update()
            if isinstance(tr_metrics, torch.Tensor):
                tr_metrics = {"loss": tr_metrics}
            if not isinstance(tr_metrics, dict):
                raise TypeError(
                    "train_batch() must return a dictionary "
                    f"mapping string names to Tensor metrics, got {type(tr_metrics)}.",
                )

            # Step learning rate of a pytorch.LRScheduler.
            with self.prof.record_timing("step_lr_schedulers"):
                for lr_scheduler in self.context.lr_schedulers:
                    self._auto_step_lr_scheduler_per_batch(batch_idx, lr_scheduler)

            with self.prof.record_timing("from_device"):
                for name, metric in tr_metrics.items():
                    # Convert PyTorch metric values to NumPy, so that
                    # `det.util.encode_json` handles them properly without
                    # needing a dependency on PyTorch.
                    if isinstance(metric, torch.Tensor):
                        metric = metric.cpu().detach().numpy()
                    tr_metrics[name] = metric

            batch_dur = time.time() - batch_start_time
            samples_per_second = batch_inputs / batch_dur
            samples_per_second *= self.context.distributed.size
            self.prof.record_metric("samples_per_second", samples_per_second)
            per_batch_metrics.append(tr_metrics)

            if self.context.is_epoch_end():
                for callback in self.callbacks.values():
                    with self.prof.record_timing(
                        f"callbacks.{callback.__class__.__name__}.on_training_epoch_end"
                    ):
                        callback.on_training_epoch_end(epoch_idx)

        # Aggregate and reduce training metrics from all the training processes.
        if self.context.distributed.size > 1 and self.context._average_training_metrics:
            with self.prof.record_timing("average_training_metrics"):
                per_batch_metrics = pytorch._combine_and_average_training_metrics(
                    self.context.distributed, per_batch_metrics
                )
        num_inputs *= self.context.distributed.size
        metrics = det.util.make_metrics(num_inputs, per_batch_metrics)

        # Ignore batch_metrics entirely for custom reducers; there's no guarantee that per-batch
        # metrics are even logical for a custom reducer.
        with self.prof.record_timing("reduce_metrics"):
            metrics["avg_metrics"].update(
                pytorch._convert_metrics_to_numpy(self.context.reduce_metrics(for_training=True))
            )

        if not self.is_chief:
            # The training metrics are reported only in the chief process.
            return {}

        step_duration = time.time() - step_start_time
        logging.info(det.util.make_timing_log("trained", step_duration, num_inputs, num_batches))
        self.metric_writer.on_train_step_end(
            self.steps_completed,
            metrics["avg_metrics"],
            metrics["batch_metrics"],
        )
        return metrics

    @torch.no_grad()  # type: ignore
    def _compute_validation_metrics(self) -> workload.Metrics:
        self.context.reset_reducers()
        # Set the behavior of certain layers (e.g., dropout) that are
        # different between training and inference.
        for model in self.context.models:
            model.eval()

        step_start_time = time.time()

        for callback in self.callbacks.values():
            if util.is_overridden(callback.on_validation_step_start, pytorch.PyTorchCallback):
                logging.warning(
                    "on_validation_step_start is now deprecated, "
                    "please use on_validation_start instead"
                )
                callback.on_validation_step_start()

        for callback in self.callbacks.values():
            callback.on_validation_start()

        num_inputs = 0
        metrics = {}  # type: Dict[str, Any]

        if self._evaluate_batch_defined():
            keys = None
            batch_metrics = []

            assert isinstance(self.validation_loader, torch.utils.data.DataLoader)
            if len(self.validation_loader) == 0:
                raise RuntimeError("validation_loader is empty.")
            for callback in self.callbacks.values():
                callback.on_validation_epoch_start()
            for idx, batch in enumerate(self.validation_loader):
                if self.context.experimental._auto_to_device:
                    batch = self.context.to_device(batch)
                num_inputs += self.trial.get_batch_length(batch)

                if has_param(self.trial.evaluate_batch, "batch_idx", 2):
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
                if self.env.test_mode:
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
            input_counts = self.context.distributed.gather((num_inputs, idx + 1))
            if self.context.distributed.rank == 0:
                assert input_counts is not None
                # Reshape and sum.
                num_inputs, num_batches = [sum(n) for n in zip(*input_counts)]

        else:
            assert self._evaluate_full_dataset_defined(), "evaluate_full_dataset not defined."
            self.validation_loader = cast(torch.utils.data.DataLoader, self.validation_loader)
            if self.is_chief:
                metrics = self.trial.evaluate_full_dataset(data_loader=self.validation_loader)

                if not isinstance(metrics, dict):
                    raise TypeError(
                        f"eval() must return a dictionary, got {type(metrics).__name__}."
                    )

                metrics = pytorch._convert_metrics_to_numpy(metrics)
                num_inputs = self.context.get_per_slot_batch_size() * len(self.validation_loader)

        metrics.update(
            pytorch._convert_metrics_to_numpy(self.context.reduce_metrics(for_training=False))
        )

        if self.context.distributed.size > 1 and any(
            util.is_overridden(c.on_validation_end, pytorch.PyTorchCallback)
            or util.is_overridden(c.on_validation_step_end, pytorch.PyTorchCallback)
            for c in self.callbacks.values()
        ):
            logging.debug(
                "Broadcasting metrics to all worker processes to execute a "
                "validation step end callback"
            )
            metrics = self.context.distributed.broadcast(metrics)

        for callback in self.callbacks.values():
            if util.is_overridden(callback.on_validation_step_end, pytorch.PyTorchCallback):
                logging.warning(
                    "on_validation_step_end is now deprecated, please use on_validation_end instead"
                )
                callback.on_validation_step_end(metrics)

        for callback in self.callbacks.values():
            callback.on_validation_end(metrics)

        if not self.is_chief:
            return {}

        # Skip reporting timings if evaluate_full_dataset() was defined.  This is far less common
        # than evaluate_batch() and we can't know how the user processed their validation data.
        if self._evaluate_batch_defined():
            step_duration = time.time() - step_start_time
            logging.info(
                det.util.make_timing_log("validated", step_duration, num_inputs, num_batches)
            )
        self.metric_writer.on_validation_step_end(self.steps_completed, metrics)
        return {"num_inputs": num_inputs, "validation_metrics": metrics}

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
                        logging.debug("Loading non-DDP checkpoint into a DDP model")
                        self._add_prefix_in_state_dict_if_not_present(model_state_dict, "module.")
                    else:
                        # If the checkpointed model is DDP and if we are currently running in
                        # single-slot mode, remove the module prefix from checkpointed data
                        logging.debug("Loading DDP checkpoint into a non-DDP model")
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
                logging.warning(
                    "There exists scaler_state_dict in checkpoint but the experiment is not using "
                    "AMP."
                )
        else:
            if self.context._scaler:
                logging.warning(
                    "The experiment is using AMP but scaler_state_dict does not exist in the "
                    "checkpoint."
                )

        if "amp_state" in checkpoint:
            if self.context._use_apex:
                apex.amp.load_state_dict(checkpoint["amp_state"])
            else:
                logging.warning(
                    "There exists amp_state in checkpoint but the experiment is not using Apex."
                )
        else:
            if self.context._use_apex:
                logging.warning(
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
                    logging.warning(
                        "The system has a gpu but no gpu_rng_state exists in the checkpoint."
                    )
            else:
                if "gpu_rng_state" in rng_state:
                    logging.warning(
                        "There exists gpu_rng_state in checkpoint but the system has no gpu."
                    )
        else:
            logging.warning("The checkpoint has no random state to restore.")

        callback_state = checkpoint.get("callbacks", {})
        for name in self.callbacks:
            if name in callback_state:
                self.callbacks[name].load_state_dict(callback_state[name])
            elif util.is_overridden(self.callbacks[name].load_state_dict, pytorch.PyTorchCallback):
                logging.warning(
                    f"Callback '{name}' implements load_state_dict(), but no callback state "
                    "was found for that name when restoring from checkpoint. This "
                    "callback will be initialized from scratch"
                )

        # Load workload sequencer state.
        wlsq_path = load_path.joinpath("workload_sequencer.pkl")
        if self.wlsq is not None and wlsq_path.exists():
            with wlsq_path.open("rb") as f:
                self.wlsq.load_state(pickle.load(f))

    def _save(self, path: pathlib.Path) -> None:
        path.mkdir(parents=True, exist_ok=True)

        util.write_user_code(path, self.env.on_cluster)

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

        if self.wlsq is not None:
            with path.joinpath("workload_sequencer.pkl").open("wb") as f:
                pickle.dump(self.wlsq.get_state(), f)

        trial_cls = type(self.trial)
        with open(path.joinpath("load_data.json"), "w") as f2:
            json.dump(
                {
                    "trial_type": "PyTorchTrial",
                    "experiment_config": self.context.env.experiment_config,
                    "hparams": self.context.env.hparams,
                    "trial_cls_spec": f"{trial_cls.__module__}:{trial_cls.__qualname__}",
                },
                f2,
            )

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


class PyTorchTrial(det.Trial):
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

    trial_controller_class = PyTorchTrialController
    trial_context_class = pytorch.PyTorchTrialContext

    @abstractmethod
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

    @abstractmethod
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

    @abstractmethod
    def build_training_data_loader(self) -> pytorch.DataLoader:
        """
        Defines the data loader to use during training.

        Must return an instance of :py:class:`determined.pytorch.DataLoader`.
        """
        pass

    @abstractmethod
    def build_validation_data_loader(self) -> pytorch.DataLoader:
        """
        Defines the data loader to use during validation.

        Must return an instance of :py:class:`determined.pytorch.DataLoader`.
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
        when iterating over the `DataLoader`.
        For example, when using `pytorch_geometric`:

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
