import abc
import logging
import pathlib
import pickle
import random
import time
from typing import Any, Dict, Iterator, List, Optional, Type, Union, cast

import deepspeed
import numpy as np
import torch

import determined as det
from determined import layers, pytorch, util, workload
from determined.common import storage
from determined.pytorch import _metric_utils
from determined.pytorch import deepspeed as det_ds


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

        # Training and validation dataloders are not built for every slot when model parallelism
        # is used.
        self.training_loader = None  # type: Optional[torch.utils.data.DataLoader]
        self.validation_loader = None  # type: Optional[torch.utils.data.DataLoader]
        self.num_validation_batches = None  # type: Optional[int]
        self.validation_batch_size = None  # type: Optional[int]
        self._set_data_loaders()

        self.wlsq = None  # type: Optional[layers.WorkloadSequencer]
        if self.workloads is None:
            self.workloads, self.wlsq = layers.make_compatibility_workloads(
                self.context._core, self.env
            )

        self.latest_batch = self.env.latest_batch

    @classmethod
    def pre_execute_hook(
        cls: Type["DeepSpeedTrialController"],
        env: det.EnvContext,
        distributed_backend: det._DistributedBackend,
    ) -> None:
        # DeepSpeed's init_distributed handles situations in which only 1 gpu is used and
        # also handles multiple calls to init in one process.
        deepspeed.init_distributed()
        cls._set_random_seeds(env.trial_seed)

    @classmethod
    def _set_random_seeds(cls: Type["DeepSpeedTrialController"], seed: int) -> None:
        # Set identical random seeds on all training processes.
        # When data parallel world size > 1, each data parallel rank will start at a unique
        # offset in the dataset, ensuring it's processing a unique
        # training batch.
        # TODO (Liam): seed data loading workers so that we can configure different seeds for
        # data augmentations per slot per worker.
        random.seed(seed)
        np.random.seed(seed)
        torch.random.manual_seed(seed)
        # TODO(Aaron): Add flag to enable determinism.
        # torch.backends.cudnn.deterministic = True
        # torch.backends.cudnn.benchmark = False

    @classmethod
    def from_trial(
        cls: Type["DeepSpeedTrialController"], *args: Any, **kwargs: Any
    ) -> det.TrialController:
        return cls(*args, **kwargs)

    @classmethod
    def supports_averaging_training_metrics(cls: Type["DeepSpeedTrialController"]) -> bool:
        return True

    def _set_data_loaders(self) -> None:
        skip_batches = self.env.latest_batch

        # We currently only allow one model parallel strategy per DeepSpeedTrial.
        # We also assume that the dataloader is tied to this one parallelization strategy.
        nreplicas = self.context.mpu.get_data_parallel_world_size()
        rank = self.context.mpu.get_data_parallel_rank()

        # The dataloader is only required on ranks that take the data as input or require
        # the data to compute the loss.  There could be intermediate model parallel ranks
        # that do not need a dataloader at all.
        if self.context.mpu.should_build_data_loader():
            train_data = self.trial.build_training_data_loader()
            if isinstance(train_data, pytorch.DataLoader):
                # Repeating the dataloader is the default behavior for DeepSpeed dataloaders when
                # using pipeline parallel.
                self.training_loader = train_data.get_data_loader(
                    repeat=True, skip=skip_batches, num_replicas=nreplicas, rank=rank
                )
            elif isinstance(train_data, torch.utils.data.DataLoader):
                # Non-determined DataLoader; ensure the user meant to do this.
                if not self.context._data_repro_checks_disabled:
                    raise RuntimeError(
                        pytorch._data._dataset_repro_warning(
                            "build_training_data_loader", train_data
                        )
                    )
                self.training_loader = train_data
                logging.warning("Please make sure custom dataloader repeats indefinitely.")

            validation_data = cast(pytorch.DataLoader, self.trial.build_validation_data_loader())
            if isinstance(validation_data, pytorch.DataLoader):
                # For pipeline parallel models, we may evaluate on slightly fewer micro batches
                # than there would be in a full pass through the dataset due to automated
                # micro batch interleaving.
                self.validation_loader = validation_data.get_data_loader(
                    repeat=False, skip=0, num_replicas=nreplicas, rank=rank
                )
                if self.context.use_pipeline_parallel:
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
                        pytorch._data._dataset_repro_warning(
                            "build_validation_data_loader", validation_data
                        )
                    )
                if self.context.use_pipeline_parallel:
                    logging.warning(
                        "Using custom dataloader, please make sure len(validation loader) is "
                        "divisible by gradient accumulation steps."
                    )
                self.validation_loader = validation_data

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
        all_epoch_lens = self.context.distributed._zmq_gather(self.context._epoch_len)
        if self.is_chief:
            all_epoch_lens = [le for le in all_epoch_lens if le is not None]
            if min(all_epoch_lens) < max(all_epoch_lens):
                logging.warning(
                    "Training dataloader length inconsistent across ranks. "
                    "Using the minimum for epoch length."
                )
            self.context._epoch_len = min(all_epoch_lens) // self.context.num_micro_batches_per_slot
        self.context._epoch_len = self.context.distributed._zmq_broadcast(self.context._epoch_len)

        all_tuples = self.context.distributed._zmq_gather(
            (self.num_validation_batches, self.validation_batch_size)
        )
        if self.is_chief:
            all_num_validation_batches, all_validation_batch_size = zip(*all_tuples)
            all_num_validation_batches = [le for le in all_num_validation_batches if le is not None]
            if min(all_num_validation_batches) < max(all_num_validation_batches):
                logging.warning(
                    "Validation dataloader length inconsistent across ranks. "
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
        ) = self.context.distributed._zmq_broadcast(
            (self.num_validation_batches, self.validation_batch_size)
        )

    def run(self) -> None:
        # We create the dataloading iterators here rather than in __init__ because we have to be
        # careful to trigger its shutdown explicitly, to avoid hangs when the user is using
        # multiprocessing-based parallelism for their dataloader.
        #
        # We create it before loading state because we don't want the training_iterator shuffling
        # values after we load state.
        self.training_iterator = (
            iter(self.training_loader) if self.training_loader is not None else None
        )
        try:
            # If a load path is provided load weights and restore the data location.
            if self.env.latest_checkpoint is not None:
                logging.info(f"Restoring trial from checkpoint {self.env.latest_checkpoint}")
                with self.context._core.checkpointing.restore_path(
                    self.env.latest_checkpoint
                ) as load_path:
                    self._load(pathlib.Path(load_path))

            with self.prof:
                for callback in self.callbacks.values():
                    with self.prof.record_timing(
                        f"callbacks.{callback.__class__.__name__}.on_training_start"
                    ):
                        callback.on_training_start()
                self._run()

        finally:
            # Explicitly trigger the dataloader iterator shutdowns (which happens in __del__).
            # See the rather long note in pytorch/torch/utils/data/dataloader.py.
            if self.training_iterator is not None:
                del self.training_iterator

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
                    storage_manager = self.context._core.checkpointing._storage_manager
                    if self.is_chief:
                        metadata = {
                            "latest_batch": self.latest_batch,
                            "framework": f"torch-{torch.__version__}",
                            "format": "pickle",
                        }
                        with storage_manager.store_path() as (
                            storage_id,
                            path,
                        ):
                            # Broadcast checkpoint path to all ranks.
                            self.context.distributed._zmq_broadcast((storage_id, path))
                            self._save(pathlib.Path(path))
                            # Gather resources across nodes.
                            all_resources = self.context.distributed._zmq_gather(
                                storage.StorageManager._list_directory(path)
                            )
                        resources = {k: v for d in all_resources for k, v in d.items()}

                        self.context._core.checkpointing._report_checkpoint(
                            storage_id, resources, metadata
                        )
                        response = {"uuid": storage_id}
                    else:
                        storage_id, path = self.context.distributed._zmq_broadcast(None)
                        self._save(pathlib.Path(path))
                        # Gather resources across nodes.
                        _ = self.context.distributed._zmq_gather(
                            storage.StorageManager._list_directory(path)
                        )
                        if self.context.distributed._is_local_chief:
                            storage_manager.post_store_path(storage_id, path)
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

        Comparing throughput with and without pipeline parallel is a common goal so we will
        automatically perform gradient accumulation by default when pipeline parallelism is not
        used.  This can be turned off by setting context.disable_auto_grad_accumulation.
        """
        self.prof.set_training(True)
        assert step_id > 0, "step_id should be greater than 0"
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
            self.latest_batch += 1
            self.prof.update_batch_idx(batch_idx)
            batch_start_time = time.time()
            self.context._current_batch_idx = batch_idx
            if self.context.is_epoch_start():
                for callback in self.callbacks.values():
                    with self.prof.record_timing(
                        f"callbacks.{callback.__class__.__name__}.on_training_epoch_start"
                    ):
                        callback.on_training_epoch_start(self.get_epoch_idx(batch_idx))
            # This can be inaccurate if the user's dataloader does not return batches with
            # the micro batch size.  It is also slightly inaccurate if the dataloader can return
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
                    if self.context.profiler:
                        with self.context.profiler as torch_profiler:
                            tr_metrics = self.trial.train_batch(
                                self.training_iterator,
                                self.get_epoch_idx(batch_idx),
                                batch_idx,
                            )
                            torch_profiler.step()
                    else:
                        tr_metrics = self.trial.train_batch(
                            self.training_iterator,
                            self.get_epoch_idx(batch_idx),
                            batch_idx,
                        )
                if self.context.mpu.should_report_metrics():
                    if isinstance(tr_metrics, torch.Tensor):
                        tr_metrics = {"loss": tr_metrics}
                    if not isinstance(tr_metrics, dict):
                        raise det.errors.InvalidExperimentException(
                            "train_batch() must return a dictionary "
                            f"mapping string names to Tensor metrics, got {type(tr_metrics)}",
                        )

                    with self.prof.record_timing("from_device"):
                        for name, metric in tr_metrics.items():
                            # Convert PyTorch metric values to NumPy, so that
                            # `det.util.encode_json` handles them properly without
                            # needing a dependency on PyTorch.
                            if isinstance(metric, torch.Tensor):
                                metric = metric.cpu().detach().numpy()
                            tr_metrics[name] = metric
                    per_batch_metrics.append(tr_metrics)
            # We do a check here to make sure that we do indeed process `num_micro_batches_per_slot`
            # micro batches when train a batch for models that do not use pipeline parallelism.
            # This will add some checking when the user wants manual gradient accumulation.
            for m in self.context.models:
                if not isinstance(m, deepspeed.PipelineEngine):
                    assert m.micro_steps % self.context.num_micro_batches_per_slot == 0

            batch_dur = time.time() - batch_start_time
            samples_per_second = batch_inputs / batch_dur
            samples_per_second *= self.context.mpu.get_data_parallel_world_size()
            self.prof.record_metric("samples_per_second", samples_per_second)

        # Aggregate and reduce training metrics from all the training processes.
        # We need this because only slots in the last stage of the pipeline compute
        # metrics and we need to aggregate on chief slot anyway to report.
        if self.context.distributed.size > 1:
            with self.prof.record_timing("average_training_metrics"):
                per_batch_metrics = _metric_utils._average_training_metrics(
                    self.context.distributed, per_batch_metrics
                )
        num_inputs *= self.context.mpu.get_data_parallel_world_size()
        metrics = det.util.make_metrics(num_inputs, per_batch_metrics)

        # Ignore batch_metrics entirely for custom reducers; there's no guarantee that per-batch
        # metrics are even logical for a custom reducer.
        with self.prof.record_timing("reduce_metrics"):
            metrics["avg_metrics"].update(
                _metric_utils._convert_metrics_to_numpy(
                    self.context.reduce_metrics(for_training=True)
                )
            )

        if not self.is_chief:
            # The training metrics are reported only in the chief process.
            return {}

        logging.debug(f"Done training step: {num_inputs} records in {num_batches} batches.")
        self.prof.set_training(False)

        return metrics

    @torch.no_grad()  # type: ignore
    def _compute_validation_metrics(self) -> workload.Response:
        self.context.reset_reducers()
        # Set the behavior of certain layers (e.g., dropout) that are
        # different between training and inference.
        for model in self.context.models:
            model.eval()

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
            # and no pipeline parallel when building the datalaoders.
            vld_metrics = self.trial.evaluate_batch(validation_iterator, idx)
            if self.context.mpu.should_report_metrics():
                # Verify validation metric names are the same across batches.
                if keys is None:
                    keys = vld_metrics.keys()
                else:
                    if keys != vld_metrics.keys():
                        raise det.errors.InvalidExperimentException(
                            "Validation metric names must match across all batches of data.",
                        )
                if not isinstance(vld_metrics, dict):
                    raise det.errors.InvalidExperimentException(
                        "validation_metrics() must return a "
                        "dictionary of string names to Tensor "
                        "metrics",
                    )
                # TODO: For performance perform -> cpu() only at the end of validation.
                batch_metrics.append(_metric_utils._convert_metrics_to_numpy(vld_metrics))
            if self.env.test_mode:
                break

        all_keys = self.context.distributed._zmq_gather(keys if keys is None else list(keys))
        if self.is_chief:
            all_keys = [k for k in all_keys if k is not None]
            keys = all_keys[0]
        keys = self.context.distributed._zmq_broadcast(keys)

        for callback in self.callbacks.values():
            callback.on_validation_epoch_end(batch_metrics)

        metrics = _metric_utils._reduce_metrics(
            self.context.distributed,
            batch_metrics=batch_metrics,
            keys=keys,
            metrics_reducers=_metric_utils._prepare_metrics_reducers(
                pytorch.Reducer.AVG, keys=keys
            ),
        )
        metrics.update(
            _metric_utils._convert_metrics_to_numpy(self.context.reduce_metrics(for_training=False))
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
            metrics = self.context.distributed._zmq_broadcast(metrics)

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

        num_inputs *= self.context.mpu.get_data_parallel_world_size()
        logging.debug(
            f"Done validating: {num_inputs} records in {self.num_validation_batches} batches."
        )
        return {"num_inputs": num_inputs, "validation_metrics": metrics}

    def _load(self, load_path: pathlib.Path) -> None:
        # Right now we will load all checkpoint shards on each node regardless of which
        # checkpoints are needed.
        # TODO (Liam): revisit later to optimize sharded checkpoint loading.

        # Load stateful things tracked by Determined on all slots.
        checkpoint: Optional[Dict[str, Any]] = None
        ckpt_path = "det_state_dict.pth"
        maybe_ckpt = load_path.joinpath(ckpt_path)
        if maybe_ckpt.exists():
            checkpoint = torch.load(str(maybe_ckpt), map_location="cpu")  # type: ignore
        if checkpoint is None or not isinstance(checkpoint, dict):
            return

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
        if self.context.distributed._is_local_chief:
            path.mkdir(parents=True, exist_ok=True)
        self.context.distributed._zmq_gather(None)  # sync

        if self.is_chief:
            # We assume these stateful objects should be the same across slots and only have
            # the chief save them.
            util.write_user_code(path, self.env.on_cluster)

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

            ckpt_name = "det_state_dict.pth"
            torch.save(checkpoint, str(path.joinpath(ckpt_name)))

            for callback in self.callbacks.values():
                callback.on_checkpoint_end(str(path))

            if self.wlsq is not None:
                with path.joinpath("workload_sequencer.pkl").open("wb") as f:
                    pickle.dump(self.wlsq.get_state(), f)

        # We allow users to override save behavior if needed but we default to using
        # the save method provided by DeepSpeed.
        self.trial.save(self.context, path)

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
       the backward pass and and optimizer step.  These methods will differ depending on whether
       you are using pipeline parallelism or not.

    """

    trial_controller_class = DeepSpeedTrialController
    trial_context_class = det_ds.DeepSpeedTrialContext

    @abc.abstractmethod
    def __init__(self, context: det_ds.DeepSpeedTrialContext) -> None:
        """
        Initializes a trial using the provided ``context``. The general steps are:
        #. Initialize the distributed backend using ``deepspeed.init_distributed``.
        #. Initialize the model(s) and, optionally, the optimizer and lr_scheduler.  The latter
           two can also be configured using the DeepSpeed config.
        #. Build the DeepSpeed model engine by calling ``deepspeed.initialize`` with the model
           (optionally optimizer and lr scheduler) and a DeepSpeed config.  Wrap it with
           ``context.wrap_model_engine``.
        #. If desired, use a custom model parallel unit by calling ``context.wrap_mpu``.

        Here is a code example.

        .. code-block:: python

            self.context = context
            self.args = AttrDict(self.context.get_hparams())

            # Init distributed backend.
            deepspeed.init_distributed()

            # Build deepspeed model engine.  We recommend using the overwrite_deepspeed_config
            # function bellow to make sure determined config and deepspeed config are consistent
            # and to easily support hyperparameter tuning.
            base_ds_config = self.context.overwrite_deepspeed_config(self.args.deepspeed_config)

            model = ... # build model
            model_engine, optimizer, lr_scheduler, _ = deepspeed.initialize(
                args=self.args,
                model=model,
                config=base_ds_config
            )

            self.model_engine = self.context.wrap_model_engine(model_engine)
        """
        pass

    @abc.abstractmethod
    def train_batch(
        self,
        dataloader_iter: Optional[Iterator[torch.utils.data.DataLoader]],
        epoch_idx: int,
        batch_idx: int,
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        """
        Train one full batch (i.e. train on `train_batch_size` samples, perhaps consisting of
        of multiple micro-batches).

        If training without pipeline parallelism, users should implement this function by doing
        the following things:

        #. Get a batch from the dataloader_iter and pass it to the gpu.
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
        call ``train_batch``


        With pipeline parallelism there is no need to manually get a batch from the dataloader_iter
        and the forward, backward, optimizer steps are combined in the model engine's
        ``train_batch`` method.

        .. code-block:: python

            # Assume one model_engine wrapped in ``__init__``.

            loss = self.model_engine.train_batch(dataloader_iter)
            return {"loss": loss}

        Arguments:
            dataloader_iter (Iterator[torch.utils.data.DataLoader], optional): iterator over
                the train DataLoader.
            epoch_idx (integer): index of the current epoch among all the batches processed
                per device (slot) since the start of training.
            batch_idx (integer): index of the current batch among all the epoches processed
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

        Must return an instance of :py:class:`determined.pytorch.DataLoader`.

        If using data parallel training, the batch size should be per gpu batch size.
        If using gradient aggregation, the dataloader should return batches with
        `train_micro_batch_size_per_gpu` samples each.
        """
        pass

    @abc.abstractmethod
    def build_validation_data_loader(self) -> Optional[pytorch.DataLoader]:
        """
        Defines the data loader to use during validation.

        Must return an instance of :py:class:`determined.pytorch.DataLoader`.

        If using data parallel training, the batch size should be per gpu batch size.
        If using gradient aggregation, the dataloader should return batches with
        `train_micro_batch_size_per_gpu` samples each.
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
        self, dataloader_iter: Optional[Iterator[torch.utils.data.DataLoader]], batch_idx: int
    ) -> Dict[str, Any]:
        """
        Calculate validation metrics for a batch and return them as a
        dictionary mapping metric names to metric values. Per-batch validation metrics
        are averaged to produce a single set of validation metrics for the entire
        validation set by default.

        The metrics returned from this function must be JSON-serializable.

        Arguments:
            dataloader_iter (Iterator[torch.utils.data.DataLoader], optional): iterator over the
                validation DataLoader.
        """
        pass

    def save(self, context: det_ds.DeepSpeedTrialContext, path: pathlib.Path) -> None:
        """
        Save is called on every gpu to make sure all checkpoint shards are saved.

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
            m.load_checkpoint(load_path, tag=f"model{i}")
