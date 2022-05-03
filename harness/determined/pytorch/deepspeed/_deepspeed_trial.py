import abc
import contextlib
import logging
import os
import pathlib
import pickle
import random
import time
import uuid
from typing import Any, Callable, Dict, Iterator, List, Optional, Type, Union, cast

import deepspeed
import numpy as np
import torch
from deepspeed.runtime import dataloader as ds_loader

import determined as det
from determined import layers, pytorch, util, workload
from determined.common import storage
from determined.pytorch import deepspeed as det_ds


# In most cases in which a user disables data reproducibility checks and chooses to return
# their own data loader, it will most likely be one created as part of DeepSpeed model engine
# initialization.  For the PipelineEngine, a RepeatingLoader is returned that does not have a
# __len__ method.  We patch in a length method here to make sure we can compute epoch length
# and validation length.
def get_length(self: ds_loader.RepeatingLoader) -> int:
    return len(self.loader)


ds_loader.RepeatingLoader.__len__ = get_length


class DeepSpeedTrialController(det.TrialController):
    def __init__(self, trial_inst: det.Trial, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        assert isinstance(
            trial_inst, DeepSpeedTrial
        ), "DeepSpeedTrialController needs a DeepSpeedTrial"
        self.trial = trial_inst
        self.context = cast(det_ds.DeepSpeedTrialContext, self.context)
        self.context._set_determined_profiler(self.prof)
        if torch.cuda.is_available():
            self.prof._set_sync_device(self._sync_device)
        self.callbacks = self.trial.build_callbacks()

        if len(self.context.models) == 0:
            raise det.errors.InvalidExperimentException(
                "Must have at least one model engine. "
                "This might be caused by not wrapping your model with wrap_model_engine()"
            )

        self.wlsq = None  # type: Optional[layers.WorkloadSequencer]
        if self.workloads is None:
            self.workloads, self.wlsq = layers.make_compatibility_workloads(
                self.context._core, self.env, self.context.models[0].train_batch_size()
            )

        self.steps_completed = self.env.steps_completed

    @classmethod
    def pre_execute_hook(
        cls: Type["DeepSpeedTrialController"],
        env: det.EnvContext,
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
        random.seed(env.trial_seed)
        np.random.seed(env.trial_seed)
        torch.random.manual_seed(env.trial_seed)

    @classmethod
    def from_trial(
        cls: Type["DeepSpeedTrialController"], *args: Any, **kwargs: Any
    ) -> det.TrialController:
        return cls(*args, **kwargs)

    @classmethod
    def supports_averaging_training_metrics(cls: Type["DeepSpeedTrialController"]) -> bool:
        return True

    def _set_data_loaders(self) -> None:
        skip_batches = self.env.steps_completed

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
                logging.warning("Please make sure custom data loader repeats indefinitely.")

            validation_data = self.trial.build_validation_data_loader()
            if isinstance(validation_data, pytorch.DataLoader):
                # For pipeline parallel models, we may evaluate on slightly fewer micro batches
                # than there would be in a full pass through the dataset due to automated
                # micro batch interleaving.
                self.validation_loader = validation_data.get_data_loader(
                    repeat=False, skip=0, num_replicas=nreplicas, rank=rank
                )

                if self.context.use_pipeline_parallel:
                    if len(self.validation_loader) < self.context.num_micro_batches_per_slot:
                        raise det.errors.InvalidExperimentException(
                            "Number of train micro batches in validation data loader should not be "
                            "less than the number of gradient accumulation steps when using "
                            "pipeline parallelism."
                        )
                    excluded_micro_batches = (
                        len(validation_data) % self.context.num_micro_batches_per_slot
                    )
                    if excluded_micro_batches:
                        logging.warning(
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
                    logging.warning(
                        "Using custom data loader, please make sure len(validation loader) is "
                        "divisible by gradient accumulation steps."
                    )
                self.validation_loader = validation_data

            # We use cast here instead of assert because the user can return an object that behaves
            # like a DataLoader but is not.
            self.validation_loader = cast(torch.utils.data.DataLoader, self.validation_loader)
            self.num_validation_batches = len(self.validation_loader)
            self.validation_batch_size = len(next(iter(self.validation_loader)))

            if self.context.use_pipeline_parallel:
                self.num_validation_batches = (
                    self.num_validation_batches // self.context.num_micro_batches_per_slot
                )
                self.validation_batch_size *= self.context.num_micro_batches_per_slot

        # We will do a gather on to get train and val loader lengths and broadcast to all slots.
        self.context._epoch_len = (
            len(self.training_loader) if self.training_loader is not None else None
        )
        all_epoch_lens = self.context.distributed.gather(self.context._epoch_len)
        if self.is_chief:
            all_epoch_lens = [le for le in all_epoch_lens if le is not None]
            if min(all_epoch_lens) < max(all_epoch_lens):
                logging.warning(
                    "Training data loader length inconsistent across ranks. "
                    "Using the minimum for epoch length."
                )
            self.context._epoch_len = min(all_epoch_lens) // self.context.num_micro_batches_per_slot
        self.context._epoch_len = self.context.distributed.broadcast(self.context._epoch_len)

        all_tuples = self.context.distributed.gather(
            (self.num_validation_batches, self.validation_batch_size)
        )
        if self.is_chief:
            all_num_validation_batches, all_validation_batch_size = zip(*all_tuples)
            all_num_validation_batches = [le for le in all_num_validation_batches if le is not None]
            if min(all_num_validation_batches) < max(all_num_validation_batches):
                logging.warning(
                    "Validation data loader length inconsistent across ranks. "
                    "Using the minimum for validation length."
                )
            self.num_validation_batches = min(all_num_validation_batches)
            all_validation_batch_size = [le for le in all_validation_batch_size if le is not None]
            if min(all_validation_batch_size) < max(all_validation_batch_size):
                logging.warning(
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
        # don't bind a the loop iteration variable `callback`, which would likely cause us to call
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

            logging.info(self.context._mpu)

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
                    response = {
                        "metrics": self._train_for_step(
                            w.step_id,
                            w.num_batches,
                            w.total_batches_processed,
                        ),
                        "stop_requested": self.context.get_stop_requested(),
                    }  # type: workload.Response

                elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                    action = "validation"
                    response = {
                        "metrics": self._compute_validation_metrics(),
                        "stop_requested": self.context.get_stop_requested(),
                    }

                elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                    action = "checkpointing"
                    # The checkpointing api would have been sufficient if the base_path for the
                    # storage manager is guaranteed to be a shared file system.
                    #
                    # Since we can't guarantee that, we use the base storage_manager instead for
                    # more flexibility.  Since checkpoints can be distributed across multiple
                    # nodes, we will use the same uuid and separate path but each node
                    # will upload its checkpoints to the storage manager individually.
                    storage_manager = self.context._core.checkpoint._storage_manager
                    if self.is_chief:
                        metadata = {
                            "steps_completed": self.steps_completed,
                            "framework": f"torch-{torch.__version__}",
                            "format": "pickle",
                        }
                        storage_id = str(uuid.uuid4())
                        with storage_manager.store_path(storage_id) as path:
                            # Broadcast checkpoint path to all ranks.
                            self.context.distributed.broadcast((storage_id, path))
                            self._save(path)
                            # Gather resources across nodes.
                            all_resources = self.context.distributed.gather(
                                storage.StorageManager._list_directory(path)
                            )
                        resources = {k: v for d in all_resources for k, v in d.items()}

                        self.context._core.checkpoint._report_checkpoint(
                            storage_id, resources, metadata
                        )
                        response = {"uuid": storage_id}
                    else:
                        storage_id, path = self.context.distributed.broadcast(None)
                        self._save(path)
                        # Gather resources across nodes.
                        _ = self.context.distributed.gather(
                            storage.StorageManager._list_directory(path)
                        )
                        if self.context.distributed.local_rank == 0:
                            storage_manager.post_store_path(str(path), storage_id)
                        response = {}

                else:
                    raise AssertionError("Unexpected workload: {}".format(w.kind))

            except det.InvalidHP as e:
                logging.info(f"Invalid hyperparameter exception during {action}: {e}")
                response = workload.InvalidHP()
            response_func(response)

    def get_epoch_idx(self, batch_id: int) -> int:
        return batch_id // cast(int, self.context._epoch_len)

    def _train_for_step(
        self, step_id: int, num_batches: int, total_batches_processed: int
    ) -> workload.Response:
        """
        DeepSpeed allows specifying train_batch_size, train_micro_batch_size_per_gpu, and
        gradient_accumulation_steps. The three are related as follows:
        train_batch_size = train_micro_batch_size * gradient_accumulation_steps.
        Hence, if two are specified, the third can be inferred.

        For pipeline parallel training, DeepSpeed will automatically interleave
        gradient_accumulation_steps worth of micro batches in one train_batch/eval_batch call.

        With the default DeepSpeed model engine (no pipeline parallel training), the backward
        and optimizer step calls track micro batches and will automatically update model weights
        and lr scheduler if micro batches % gradient_accumulation_steps == 0.

        Comparing training with and without pipeline parallel is a common goal.  Since DeepSpeed's
        PipelineEngine trains on a number of micro batches equal to gradient accumulation steps,
        we automatically perform gradient accumulation by default when pipeline parallelism is not
        enabled.  This makes it fair to compare training with and without pipeline parallelism
        at a given batch idx. This can be turned off by setting
        context.disable_auto_grad_accumulation.
        """
        self.prof.set_training(True)
        assert step_id > 0, "step_id should be greater than 0"
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
            self.prof.update_batch_idx(batch_idx)
            batch_start_time = time.time()
            self.context._current_batch_idx = batch_idx
            if self.context.is_epoch_start():
                for callback in self.callbacks.values():
                    with self.prof.record_timing(
                        f"callbacks.{callback.__class__.__name__}.on_training_epoch_start"
                    ):
                        callback.on_training_epoch_start(self.get_epoch_idx(batch_idx))
            # This can be inaccurate if the user's data loader does not return batches with
            # the micro batch size.  It is also slightly inaccurate if the data loader can return
            # partial batches.  The same sort of assumptions are made in the DeepSpeed
            # model engine's accounting and profiling computations.
            batch_inputs = (
                self.context.train_micro_batch_size_per_gpu
                * self.context.num_micro_batches_per_slot
            )
            num_inputs += batch_inputs
            num_train_batch_calls = self.context.num_micro_batches_per_slot
            if self.context.use_pipeline_parallel or self.context._manual_grad_accumulation:
                num_train_batch_calls = 1
            self.context._loss_ids = {}
            for _ in range(num_train_batch_calls):
                with self.prof.record_timing("train_batch", requires_sync=False, accumulate=True):
                    tr_metrics = self.trial.train_batch(
                        self.training_iterator,
                        self.get_epoch_idx(batch_idx),
                        batch_idx,
                    )
                if self.context._mpu.should_report_metrics:
                    if isinstance(tr_metrics, torch.Tensor):
                        tr_metrics = {"loss": tr_metrics}
                    if not isinstance(tr_metrics, dict):
                        raise det.errors.InvalidExperimentException(
                            "train_batch must return a dictionary "
                            f"mapping string names to Tensor metrics, got {type(tr_metrics)}",
                        )

                    for name, metric in tr_metrics.items():
                        # Convert PyTorch metric values to NumPy, so that
                        # `det.util.encode_json` handles them properly without
                        # needing a dependency on PyTorch.
                        if isinstance(metric, torch.Tensor):
                            metric = metric.cpu().detach().numpy()
                        tr_metrics[name] = metric
                    per_batch_metrics.append(tr_metrics)
            # We do a check here to make sure that we do indeed process `num_micro_batches_per_slot`
            # micro batches when training a batch for models that do not use pipeline parallelism.
            model0 = self.context.models[0]
            if not isinstance(model0, deepspeed.PipelineEngine):
                assert (
                    model0.micro_steps % self.context.num_micro_batches_per_slot == 0
                ), "did not train for gradient accumulation steps"

            batch_dur = time.time() - batch_start_time
            samples_per_second = batch_inputs / batch_dur
            samples_per_second *= self.context._mpu.data_parallel_world_size
            self.prof.record_metric("samples_per_second", samples_per_second)

            if self.context.is_epoch_end():
                for callback in self.callbacks.values():
                    with self.prof.record_timing(
                        f"callbacks.{callback.__class__.__name__}.on_training_epoch_end"
                    ):
                        callback.on_training_epoch_end(self.get_epoch_idx(batch_idx))

        # Aggregate and reduce training metrics from all the training processes.
        if self.context.distributed.size > 1 and self.context._average_training_metrics:
            with self.prof.record_timing("average_training_metrics"):
                per_batch_metrics = pytorch._combine_and_average_training_metrics(
                    self.context.distributed, per_batch_metrics
                )
        num_inputs *= self.context._mpu.data_parallel_world_size
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
        self.prof.set_training(False)

        return metrics

    @torch.no_grad()  # type: ignore
    def _compute_validation_metrics(self) -> workload.Response:
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
        keys = None
        batch_metrics = []

        for callback in self.callbacks.values():
            callback.on_validation_epoch_start()

        validation_iterator = iter(self.validation_loader) if self.validation_loader else None
        for idx in range(cast(int, self.num_validation_batches)):
            num_inputs += cast(int, self.validation_batch_size)
            # Note that when using pipeline parallelism, each call to evaluate_batch will request
            # self.context.num_micro_batches_per_slot batches from the validation iterator.
            # This is why we set self.num_validation_batches differently for pipeline parallel
            # and no pipeline parallel when building the data laoders.
            vld_metrics = self.trial.evaluate_batch(validation_iterator, idx)
            if self.context._mpu.should_report_metrics:
                if not isinstance(vld_metrics, dict):
                    raise det.errors.InvalidExperimentException(
                        "evaluate_batch must return a dictionary of string names "
                        "to Tensor metrics",
                    )
                # Verify validation metric names are the same across batches.
                if keys is None:
                    keys = vld_metrics.keys()
                else:
                    if keys != vld_metrics.keys():
                        raise det.errors.InvalidExperimentException(
                            "Validation metric names must match across all batches of data.",
                        )
                # TODO: For performance perform -> cpu() only at the end of validation.
                batch_metrics.append(pytorch._convert_metrics_to_numpy(vld_metrics))
            if self.env.test_mode:
                break

        # keys and list(keys) does not satisfy all cases because it will return dict_keys type if
        # keys is an empty dict. this will then break when passed to zmq_broadcast since it does
        # not know how to serialize dict_keys type.
        all_keys = self.context.distributed.gather(keys if keys is None else list(keys))
        if self.is_chief:
            all_keys = [k for k in all_keys if k is not None]
            keys = all_keys[0]
        keys = self.context.distributed.broadcast(keys)

        for callback in self.callbacks.values():
            callback.on_validation_epoch_end(batch_metrics)

        metrics = pytorch._reduce_metrics(
            self.context.distributed,
            batch_metrics=batch_metrics,
            keys=keys,
            metrics_reducers=pytorch._prepare_metrics_reducers(pytorch.Reducer.AVG, keys=keys),
        )
        metrics.update(
            pytorch._convert_metrics_to_numpy(self.context.reduce_metrics(for_training=False))
        )

        if self.context.distributed.size > 1 and any(
            map(
                lambda c: util.is_overridden(c.on_validation_end, pytorch.PyTorchCallback)
                or util.is_overridden(c.on_validation_step_end, pytorch.PyTorchCallback),
                self.callbacks.values(),
            )
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

        num_inputs *= self.context._mpu.data_parallel_world_size
        step_duration = time.time() - step_start_time
        logging.info(
            det.util.make_timing_log(
                "validated", step_duration, num_inputs, cast(int, self.num_validation_batches)
            )
        )

        return {"num_inputs": num_inputs, "validation_metrics": metrics}

    def _load(self, load_path: pathlib.Path) -> None:
        # Right now we will load all checkpoint shards on each node regardless of which
        # checkpoints are needed.
        # TODO (Liam): revisit later to optimize sharded checkpoint loading.

        # Load stateful things tracked by Determined on all slots.
        ckpt_path = f"det_state_dict_rank{self.context.distributed.rank}.pth"
        maybe_ckpt = load_path.joinpath(ckpt_path)

        if not maybe_ckpt.exists():
            return

        checkpoint = torch.load(str(maybe_ckpt), map_location="cpu")  # type: ignore
        if not isinstance(checkpoint, dict):
            raise det.errors.InvalidExperimentException(
                f"Expected checkpoint at {maybe_ckpt} to be a dict "
                f"but got {type(checkpoint).__name__}."
            )

        for callback in self.callbacks.values():
            callback.on_checkpoint_load_start(checkpoint)

        # We allow users to override load behavior if needed but we default to using
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
                    "Callback '{}' implements load_state_dict(), but no callback state "
                    "was found for that name when restoring from checkpoint. This "
                    "callback will be initialized from scratch"
                )

        # Load workload sequencer state.
        wlsq_path = load_path.joinpath("workload_sequencer.pkl")
        if self.wlsq is not None and wlsq_path.exists():
            with wlsq_path.open("rb") as f:
                self.wlsq.load_state(pickle.load(f))

    def _save(self, path: pathlib.Path) -> None:
        if self.context.distributed.local_rank == 0:
            path.mkdir(parents=True, exist_ok=True)
        _ = self.context.distributed.gather_local(None)  # sync

        if self.is_chief:
            # We assume these stateful objects should be the same across slots and only have
            # the chief save them.
            util.write_user_code(path, self.env.on_cluster)

            if self.wlsq is not None:
                with path.joinpath("workload_sequencer.pkl").open("wb") as f:
                    pickle.dump(self.wlsq.get_state(), f)

        # Save per rank Determined checkpoint.
        rng_state = {
            "cpu_rng_state": torch.random.get_rng_state(),
            "np_rng_state": np.random.get_state(),
            "random_rng_state": random.getstate(),
        }

        if torch.cuda.device_count():
            rng_state["gpu_rng_state"] = torch.cuda.get_rng_state(
                self.context.distributed.get_local_rank()
            )
        checkpoint = {"rng_state": rng_state}

        # PyTorch uses optimizer objects that take the model parameters to
        # optimize on construction, so we store and reload the `state_dict()`
        # of the model and optimizer explicitly (instead of dumping the entire
        # objects) to avoid breaking the connection between the model and the
        # optimizer.
        checkpoint["callbacks"] = {
            name: callback.state_dict() for name, callback in self.callbacks.items()
        }

        for callback in self.callbacks.values():
            callback.on_checkpoint_save_start(checkpoint)

        ckpt_name = f"det_state_dict_rank{self.context.distributed.rank}.pth"
        torch.save(checkpoint, str(path.joinpath(ckpt_name)))

        # We allow users to override save behavior if needed but we default to using
        # the save method provided by DeepSpeed.
        self.trial.save(self.context, path)

        for callback in self.callbacks.values():
            callback.on_checkpoint_end(str(path))

    def _sync_device(self) -> None:
        torch.cuda.synchronize(self.context.device)


class DeepSpeedTrial(det.Trial):
    """
    DeepSpeed trials are created by subclassing this abstract class.

    We can do the following things in this trial class:

    * **Define the DeepSpeed model engine which includes the model, optimizer, and lr_scheduler**.

       In the :meth:`__init__` method, initialize models and, optionally, optimizers and
       LR schedulers and pass them to deepspeed.initialize to build the model engine.  Then
       pass the created model engine to ``wrap_model_engine`` provided by
       :class:`~determined.pytorch.deepspeed.DeepSpeedTrialContext`.
       We support multiple DeepSpeed model engines if they only use data parallelism or if
       they use the same model parallel unit.

    * **Run forward and backward passes**.

       In :meth:`train_batch`, use the methods provided by the DeepSpeed model engine to perform
       the backward pass and optimizer step.  These methods will differ depending on whether
       you are using pipeline parallelism or not.

    """

    trial_controller_class = DeepSpeedTrialController
    trial_context_class = det_ds.DeepSpeedTrialContext

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
        Train one full batch (i.e. train on ``train_batch_size`` samples, perhaps consisting of
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
