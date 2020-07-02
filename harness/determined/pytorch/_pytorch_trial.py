import logging
import pathlib
import random
from abc import abstractmethod
from typing import Any, Dict, Iterator, List, Optional, Tuple, Union, cast

import cloudpickle
import numpy as np
import torch
import torch.nn as nn

import determined as det
from determined import horovod, ipc, util, workload
from determined.horovod import hvd
from determined.pytorch import (
    DataLoader,
    LRScheduler,
    PyTorchTrialContext,
    Reducer,
    TorchData,
    _callback,
    _reduce_metrics,
    data_length,
)
from determined_common import check


class PyTorchTrialController(det.LoopTrialController):
    def __init__(self, trial_inst: det.Trial, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        check.is_instance(trial_inst, PyTorchTrial, "PyTorchTrialController needs an PyTorchTrial")
        self.trial = cast(PyTorchTrial, trial_inst)
        self.context = cast(PyTorchTrialContext, self.context)
        self.callbacks = self.trial.build_callbacks()

        # TODO(DET-3262): remove this backward compatibility of old interface.
        if (
            util.is_overridden(self.trial.build_model, PyTorchTrial)
            or util.is_overridden(self.trial.optimizer, PyTorchTrial)
            or util.is_overridden(self.trial.create_lr_scheduler, PyTorchTrial)
        ):
            check.true(
                util.is_overridden(self.trial.build_model, PyTorchTrial)
                and util.is_overridden(self.trial.optimizer, PyTorchTrial),
                "Both build_model() and optimizer() must be defined "
                "if any of build_model(), optimizer(), and create_lr_scheduler() are defined. "
                "If you want to use the new interface, you should instead instantiate your models, "
                "optimizers, and LR schedulers in __init__ and call context.backward(loss) "
                "and context.step_optimizer(optimizer) in train_batch.",
            )

            model = self.context._Model(self.trial.build_model())
            optim = self.context._Optimizer(self.trial.optimizer(model))

            lr_scheduler = self.trial.create_lr_scheduler(optim)
            if lr_scheduler is not None:
                self.context.lr_schedulers.append(lr_scheduler)

            if det.ExperimentConfig(self.context.get_experiment_config()).mixed_precision_enabled():
                self.context._configure_apex_amp(
                    models=model,
                    optimizers=optim,
                    opt_level=self.context.get_experiment_config()
                    .get("optimizations", {})
                    .get("mixed_precision", "O0"),
                )

            train_batch = self.trial.train_batch

            def new_train_batch(
                batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
            ) -> Union[torch.Tensor, Dict[str, Any]]:
                tr_metrics = train_batch(batch, model, epoch_idx, batch_idx)
                if isinstance(tr_metrics, torch.Tensor):
                    tr_metrics = {"loss": tr_metrics}
                check.is_instance(
                    tr_metrics,
                    dict,
                    "train_batch() must return a dictionary "
                    f"mapping string names to Tensor metrics, got {type(tr_metrics)}",
                )
                check.is_in(
                    "loss", tr_metrics.keys(), 'Please include "loss" in you training metrics.'
                )

                def clip_grads(parameters: Iterator) -> None:
                    for callback in self.callbacks.values():
                        callback.on_before_optimizer_step(parameters)

                self.context._backward(tr_metrics["loss"])
                self.context._step_optimizer(self.context.optimizers[0], clip_grads=clip_grads)
                return tr_metrics

            self.trial.__setattr__("train_batch", new_train_batch)

        check.gt_eq(len(self.context.models), 1, "Must have at least one model")
        check.gt_eq(len(self.context.optimizers), 1, "Must have at least one optimizer")
        self._check_evaluate_implementation()

        # Validation loader will be undefined on process ranks > 0
        # when the user defines `validate_full_dataset()`.
        self.validation_loader = None  # type: Optional[torch.utils.data.DataLoader]
        self._set_data_loaders()

        # If a load path is provided load weights and restore the data location.
        self._load()

        if self.hvd_config.use:
            hvd.broadcast_parameters(self.context._main_model.state_dict(), root_rank=0)
            for optimizer in self.context.optimizers:
                hvd.broadcast_optimizer_state(optimizer, root_rank=0)

        self.training_iterator = iter(self.training_loader)

    @staticmethod
    def pre_execute_hook(env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        # Initialize the correct horovod.
        if hvd_config.use:
            hvd.require_horovod_type("torch", "PyTorchTrial is in use.")
            hvd.init()

        PyTorchTrialController._set_random_seeds(env.trial_seed)

    @staticmethod
    def _set_random_seeds(seed: int) -> None:
        # Set identical random seeds on all training processes.
        # When using horovod, each worker will start at a unique
        # offset in the dataset, ensuring it's processing a unique
        # training batch.
        random.seed(seed)
        np.random.seed(seed)
        torch.random.manual_seed(seed)  # type: ignore
        # TODO(Aaron): Add flag to enable determinism.
        # torch.backends.cudnn.deterministic = True
        # torch.backends.cudnn.benchmark = False

    @staticmethod
    def from_trial(*args: Any, **kwargs: Any) -> det.TrialController:
        return PyTorchTrialController(*args, **kwargs)

    @staticmethod
    def from_native(*args: Any, **kwargs: Any) -> det.TrialController:
        raise NotImplementedError("PyTorchTrial only supports the Trial API")

    @staticmethod
    def support_determined_native() -> bool:
        return True

    def _check_evaluate_implementation(self) -> None:
        """
        Check if the user has implemented evaluate_batch
        or evaluate_full_dataset.
        """
        logging.debug(f"Evaluate_batch_defined: {self._evaluate_batch_defined()}.")
        logging.debug(f"Evaluate full dataset defined: {self._evaluate_full_dataset_defined()}.")
        check.not_eq(
            self._evaluate_batch_defined(),
            self._evaluate_full_dataset_defined(),
            "Please define exactly one of: `evaluate_batch()` or `evaluate_full_dataset()`. "
            "For most use cases `evaluate_batch()` is recommended is recommended because "
            "it can be parallelized across all devices.",
        )

    def _evaluate_batch_defined(self) -> bool:
        return util.is_overridden(self.trial.evaluate_batch, PyTorchTrial)

    def _evaluate_full_dataset_defined(self) -> bool:
        return util.is_overridden(self.trial.evaluate_full_dataset, PyTorchTrial)

    @staticmethod
    def supports_multi_gpu_training() -> bool:
        return True

    @staticmethod
    def supports_mixed_precision() -> bool:
        return True

    @staticmethod
    def supports_averaging_training_metrics() -> bool:
        return True

    def _set_data_loaders(self) -> None:
        skip_batches = self.env.initial_workload.total_batches_processed

        nreplicas = hvd.size() if self.hvd_config.use else 1
        rank = hvd.rank() if self.hvd_config.use else 0

        self.training_loader = self.trial.build_training_data_loader().get_data_loader(
            repeat=True, skip=skip_batches, num_replicas=nreplicas, rank=rank
        )

        validation_dataset = self.trial.build_validation_data_loader()
        if self._evaluate_batch_defined():
            self.validation_loader = validation_dataset.get_data_loader(
                repeat=False, skip=0, num_replicas=nreplicas, rank=rank
            )
        elif self.is_chief:
            self.validation_loader = validation_dataset.get_data_loader(
                repeat=False, skip=0, num_replicas=1, rank=0
            )

    def run(self) -> None:
        for w, args, response_func in self.workloads:
            if w.kind == workload.Workload.Kind.RUN_STEP:
                response_func(
                    util.wrap_metrics(
                        self._train_for_step(w.step_id, w.num_batches, w.total_batches_processed),
                        self.context.get_stop_requested(),
                    )
                )
            elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                response_func(
                    util.wrap_metrics(
                        self._compute_validation_metrics(), self.context.get_stop_requested()
                    )
                )
            elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.eq(len(args), 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                response_func(self._save(path))
            elif w.kind == workload.Workload.Kind.TERMINATE:
                response_func({} if self.is_chief else workload.Skipped())
                break
            else:
                raise AssertionError("Unexpected workload: {}".format(w.kind))

    def get_epoch_idx(self, batch_id: int) -> int:
        return batch_id // len(self.training_loader)

    def _average_training_metrics(
        self, per_batch_metrics: List[Dict[str, Any]]
    ) -> List[Dict[str, Any]]:
        """Average training metrics across GPUs"""
        check.true(self.hvd_config.use, "Can only average training metrics in multi-GPU training.")
        metrics_timeseries = util._list_to_dict(per_batch_metrics)

        # combined_timeseries is: dict[metric_name] -> 2d-array.
        # A measurement is accessed via combined_timeseries[metric_name][process_idx][batch_idx].
        combined_timeseries, _ = self._combine_metrics_across_processes(
            metrics_timeseries, num_batches=len(per_batch_metrics)
        )

        # If the value for a metric is a single-element array, the averaging process will
        # change that into just the element. We record what metrics are single-element arrays
        # so we can wrap them in an array later (for perfect compatibility with non-averaging
        # codepath).
        array_metrics = []
        for metric_name in per_batch_metrics[0].keys():
            if isinstance(per_batch_metrics[0][metric_name], np.ndarray):
                array_metrics.append(metric_name)

        if self.is_chief:
            combined_timeseries_type = Dict[str, List[List[Any]]]
            combined_timeseries = cast(combined_timeseries_type, combined_timeseries)
            num_batches = len(per_batch_metrics)
            num_processes = hvd.size()
            averaged_metrics_timeseries = {}  # type: Dict[str, List]

            for metric_name in combined_timeseries.keys():
                averaged_metrics_timeseries[metric_name] = []
                for batch_idx in range(num_batches):
                    batch = [
                        combined_timeseries[metric_name][process_idx][batch_idx]
                        for process_idx in range(num_processes)
                    ]

                    np_batch = np.array(batch)
                    batch_avg = np.mean(np_batch[np_batch != None])  # noqa: E711
                    if metric_name in array_metrics:
                        batch_avg = np.array(batch_avg)
                    averaged_metrics_timeseries[metric_name].append(batch_avg)
            per_batch_metrics = util._dict_to_list(averaged_metrics_timeseries)
        return per_batch_metrics

    def _auto_step_lr_scheduler_per_batch(self, batch_idx: int, lr_scheduler: LRScheduler) -> None:
        """
        This function aims at automatically step a LR scheduler. It should be called per batch.
        """
        if lr_scheduler._step_mode == LRScheduler.StepMode.STEP_EVERY_BATCH:
            lr_scheduler.step()
        elif lr_scheduler._step_mode == LRScheduler.StepMode.STEP_EVERY_EPOCH:
            mod = (batch_idx + 1) % len(self.training_loader)
            if mod == 0 or mod < self.hvd_config.aggregation_frequency:
                lr_scheduler.step()

    def _train_for_step(
        self, step_id: int, num_batches: int, total_batches_processed: int
    ) -> workload.Response:
        check.gt(step_id, 0)

        # Set the behavior of certain layers (e.g., dropout) that are different
        # between training and inference.
        for model in self.context.models:
            model.train()

        for callback in self.callbacks.values():
            callback.on_train_step_start(step_id)

        start = total_batches_processed
        end = start + num_batches

        per_batch_metrics = []  # type: List[Dict]
        num_inputs = 0

        for batch_idx in range(start, end):
            batch = next(self.training_iterator)
            num_inputs += data_length(batch)
            batch = self.context._to_device(batch)

            self.context._current_batch_idx = batch_idx
            self.context._loss_ids = {}
            tr_metrics = self.trial.train_batch(
                batch=batch,
                model=self.context.models[0],
                epoch_idx=self.get_epoch_idx(batch_idx),
                batch_idx=batch_idx,
            )
            if isinstance(tr_metrics, torch.Tensor):
                tr_metrics = {"loss": tr_metrics}
            check.is_instance(
                tr_metrics,
                dict,
                "train_batch() must return a dictionary "
                f"mapping string names to Tensor metrics, got {type(tr_metrics)}",
            )
            check.is_in("loss", tr_metrics.keys(), 'Please include "loss" in you training metrics.')

            # Step learning rate of a LRScheduler.
            for lr_scheduler in self.context.lr_schedulers:
                self._auto_step_lr_scheduler_per_batch(batch_idx, lr_scheduler)

            for name, metric in tr_metrics.items():
                # Convert PyTorch metric values to NumPy, so that
                # `det.util.encode_json` handles them properly without
                # needing a dependency on PyTorch.
                if isinstance(metric, torch.Tensor):
                    metric = metric.cpu().detach().numpy()
                tr_metrics[name] = metric

            check.is_in("loss", tr_metrics, 'Please include "loss" in your training metrics.')
            per_batch_metrics.append(tr_metrics)

        # Aggregate and reduce training metrics from all the training processes.
        if self.hvd_config.use and self.hvd_config.average_training_metrics:
            per_batch_metrics = self._average_training_metrics(per_batch_metrics)
        if self.hvd_config.use:
            num_inputs *= hvd.size()
        metrics = det.util.make_metrics(num_inputs, per_batch_metrics)

        for callback in self.callbacks.values():
            callback.on_train_step_end(step_id, metrics)

        if not self.is_chief:
            # The training metrics are reported only in the chief process.
            return workload.Skipped()

        logging.debug(f"Done training step: {num_inputs} records in {num_batches} batches.")

        return metrics

    @staticmethod
    def _convert_metrics_to_numpy(metrics: Dict[str, Any]) -> Dict[str, Any]:
        for metric_name, metric_val in metrics.items():
            if isinstance(metric_val, torch.Tensor):
                metrics[metric_name] = metric_val.cpu().numpy()
        return metrics

    @torch.no_grad()
    def _compute_validation_metrics(self) -> workload.Response:
        # Set the behavior of certain layers (e.g., dropout) that are
        # different between training and inference.
        for model in self.context.models:
            model.eval()

        for callback in self.callbacks.values():
            callback.on_validation_step_start()

        num_inputs = 0
        metrics = {}  # type: Optional[Dict[str, Any]]

        if self._evaluate_batch_defined():
            keys = None
            batch_metrics = []

            self.validation_loader = cast(torch.utils.data.DataLoader, self.validation_loader)
            check.gt(len(self.validation_loader), 0)
            for batch in self.validation_loader:
                batch = self.context._to_device(batch)
                num_inputs += data_length(batch)

                vld_metrics = self.trial.evaluate_batch(batch=batch, model=self.context.models[0])
                # Verify validation metric names are the same across batches.
                if keys is None:
                    keys = vld_metrics.keys()
                else:
                    check.eq(
                        keys,
                        vld_metrics.keys(),
                        "Validation metric names must match across all batches of data.",
                    )
                check.is_instance(
                    vld_metrics,
                    dict,
                    "validation_metrics() must return a "
                    "dictionary of string names to Tensor "
                    "metrics",
                )
                # TODO: For performance perform -> cpu() only at the end of validation.
                batch_metrics.append(self._convert_metrics_to_numpy(vld_metrics))

            metrics = self._reduce_metrics(
                batch_metrics=batch_metrics,
                keys=keys,
                metrics_reducers=self._prepare_metrics_reducers(keys=keys),
            )

            if self.hvd_config.use:
                num_inputs *= hvd.size()

        else:
            check.true(self._evaluate_full_dataset_defined())
            self.validation_loader = cast(torch.utils.data.DataLoader, self.validation_loader)
            if self.is_chief:
                metrics = self.trial.evaluate_full_dataset(
                    data_loader=self.validation_loader, model=self.context.models[0]
                )

                check.is_instance(
                    metrics, dict, f"eval() must return a dictionary, got {type(metrics)}."
                )

                metrics = self._convert_metrics_to_numpy(metrics)
                num_inputs = self.context.get_per_slot_batch_size() * len(self.validation_loader)

        if self.hvd_config.use and any(
            map(
                lambda c: util.is_overridden(c.on_validation_step_end, _callback.PyTorchCallback),
                self.callbacks.values(),
            )
        ):
            logging.debug(
                "Broadcasting metrics to all worker processes to execute a "
                "validation step end callback"
            )
            metrics = hvd.broadcast_object(metrics, root_rank=0)

        for callback in self.callbacks.values():
            callback.on_validation_step_end(cast(Dict[str, Any], metrics))

        if not self.is_chief:
            return workload.Skipped()

        return {"num_inputs": num_inputs, "validation_metrics": metrics}

    def _prepare_metrics_reducers(self, keys: Any) -> Dict[str, Reducer]:
        metrics_reducers = {}  # type: Dict[str, Reducer]
        if isinstance(self.trial.evaluation_reducer(), Dict):
            metrics_reducers = cast(Dict[str, Any], self.trial.evaluation_reducer())
            check.eq(
                metrics_reducers.keys(),
                keys,
                "Please provide a single evaluation reducer or "
                "provide a reducer for every validation metric. "
                f"Expected keys: {keys}, provided keys: {metrics_reducers.keys()}.",
            )
        elif isinstance(self.trial.evaluation_reducer(), Reducer):
            for key in keys:
                metrics_reducers[key] = cast(Reducer, self.trial.evaluation_reducer())

        for key in keys:
            check.true(
                isinstance(metrics_reducers[key], Reducer),
                "Please select `det.pytorch.Reducer` for reducing validation metrics.",
            )

        return metrics_reducers

    def _reduce_metrics(
        self, batch_metrics: List, keys: Any, metrics_reducers: Dict[str, Reducer]
    ) -> Optional[Dict[str, Any]]:
        metrics = {
            name: _reduce_metrics(
                reducer=metrics_reducers[name],
                metrics=np.stack([b[name] for b in batch_metrics], axis=0),
                num_batches=None,
            )
            for name in keys or []
        }

        if self.hvd_config.use:
            # If using horovod combine metrics across all processes.
            # Only the chief process will receive all the metrics.
            self.validation_loader = cast(torch.utils.data.DataLoader, self.validation_loader)
            num_batches = len(self.validation_loader)
            combined_metrics, batches_per_process = self._combine_metrics_across_processes(
                metrics, num_batches
            )
            if self.is_chief:
                # Only the chief collects all the metrics.
                combined_metrics = self._convert_metrics_to_numpy(
                    cast(Dict[str, Any], combined_metrics)
                )
                metrics = {
                    name: _reduce_metrics(
                        reducer=metrics_reducers[name],
                        metrics=combined_metrics[name],
                        num_batches=batches_per_process,
                    )
                    for name in keys or []
                }
            else:
                return {}

        return metrics

    def _combine_metrics_across_processes(
        self, metrics: Dict[str, Any], num_batches: int
    ) -> Tuple[Optional[Dict[str, Any]], Optional[List[int]]]:
        # The chief receives the metric from every other training process.
        check.true(self.hvd_config.use)

        metrics_lists = {}  # type: Dict[str, Any]
        batches_per_process = []  # type: List[int]
        if self.is_chief:
            self.train_process_comm_chief = cast(
                ipc.ZMQBroadcastServer, self.train_process_comm_chief
            )
            worker_metrics, _ = self.train_process_comm_chief.gather_with_polling(lambda: None)
            self.train_process_comm_chief.broadcast(None)
            worker_metrics = cast(List[ipc.MetricsInfo], worker_metrics)

            for metric_name in metrics.keys():
                metrics_lists[metric_name] = [metrics[metric_name]]
                for worker_metric in worker_metrics:
                    metrics_lists[metric_name].append(worker_metric.metrics[metric_name])

            batches_per_process.append(num_batches)
            for worker_metric in worker_metrics:
                batches_per_process.append(worker_metric.num_batches)

            return metrics_lists, batches_per_process
        else:
            self.train_process_comm_worker = cast(
                ipc.ZMQBroadcastClient, self.train_process_comm_worker
            )
            self.train_process_comm_worker.send(
                ipc.MetricsInfo(metrics=metrics, num_batches=num_batches)
            )
            # Synchronize with the chief so that there is no risk of accidentally calling send()
            # for a future gather before all workers have called send() on this gather.
            _ = self.train_process_comm_worker.recv()
            return None, None

    def _load(self) -> None:
        if not self.load_path:
            return

        # Backwards compat with older checkpoint formats. List is newest to
        # oldest known state_dict locations.
        potential_paths = [
            ["state_dict.pth"],
            ["determined", "state_dict.pth"],
            ["pedl", "state_dict.pth"],
            ["checkpoint.pt"],
        ]

        for ckpt_path in potential_paths:
            maybe_ckpt = self.load_path.joinpath(*ckpt_path)
            if maybe_ckpt.exists():
                checkpoint = torch.load(maybe_ckpt, map_location="cpu")  # type: ignore
                break

        if "model_state_dict" in checkpoint:
            # Backward compatible with older checkpoint format.
            check.not_in("models_state_dict", checkpoint)
            check.eq(len(self.context.models), 1)
            self.context.models[0].load_state_dict(checkpoint["model_state_dict"])
        else:
            for idx, model in enumerate(self.context.models):
                model.load_state_dict(checkpoint["models_state_dict"][idx])

        if "optimizer_state_dict" in checkpoint:
            # Backward compatible with older checkpoint format.
            check.not_in("optimizers_state_dict", checkpoint)
            check.eq(len(self.context.optimizers), 1)
            self.context.optimizers[0].load_state_dict(checkpoint["optimizer_state_dict"])
        else:
            for idx, optimizer in enumerate(self.context.optimizers):
                optimizer.load_state_dict(checkpoint["optimizers_state_dict"][idx])

        if "lr_scheduler" in checkpoint:
            # Backward compatible with older checkpoint format.
            check.not_in("lr_schedulers_state_dict", checkpoint)
            check.eq(len(self.context.lr_schedulers), 1)
            self.context.lr_schedulers[0].load_state_dict(checkpoint["lr_scheduler"])
        else:
            for idx, lr_scheduler in enumerate(self.context.lr_schedulers):
                lr_scheduler.load_state_dict(checkpoint["lr_schedulers_state_dict"][idx])

        callback_state = checkpoint.get("callbacks", {})
        for name in self.callbacks:
            if name in callback_state:
                self.callbacks[name].load_state_dict(callback_state[name])
            elif util.is_overridden(
                self.callbacks[name].load_state_dict, _callback.PyTorchCallback
            ):
                logging.warning(
                    "Callback '{}' implements load_state_dict(), but no callback state "
                    "was found for that name when restoring from checkpoint. This "
                    "callback will be initialized from scratch"
                )

    def _save(self, path: pathlib.Path) -> workload.Response:
        if not self.is_chief:
            return workload.Skipped()

        path.mkdir(parents=True, exist_ok=True)

        # The model code is the current working directory.
        util.write_user_code(path)

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
        }

        torch.save(  # type: ignore
            checkpoint, str(path.joinpath("state_dict.pth")), pickle_module=cloudpickle
        )

        for callback in self.callbacks.values():
            callback.on_checkpoint_end(str(path))

        return cast(
            workload.Response,
            {
                "framework": f"torch-{torch.__version__}",  # type: ignore
                "format": "cloudpickle",
            },
        )


class PyTorchTrial(det.Trial):
    """
    PyTorch trials are created by subclassing the abstract class
    :class:`PyTorchTrial`.
    """

    # TODO(DET-3204): Add the following comments to the docstring.
    # There are two different methods to define models, optimizers, and LR schedulers.
    #
    # 1. Define the abstract methods for the only model, optimizer, and LR scheduler and
    #    Determine will initialize them for the users. Note this method doesn't support
    #    multiple models, optimizers, and LR schedulers.
    #
    # 2. Initialize models, optimizers, and LR schedulers in the :meth:`__init__`.
    #    If using this way, users should calculate the gradients with the losses and call an
    #    optimization step for the optimizers on your own in :meth:`train_batch`.
    #    Users are able to use arbitrary number of models, optimizers, and LR schedulers
    #    and run the forward and backward passes and step optimizers in arbitrary orders.

    trial_controller_class = PyTorchTrialController
    trial_context_class = PyTorchTrialContext

    @abstractmethod
    def __init__(self, trial_context: PyTorchTrialContext) -> None:
        """
        Initializes a trial using the provided trial context.

        Override this function to initialize any shared state between the
        function implementations.

        Models, optimizers, and LR schedulers can be defined in the abstract methods.
        """

        # TODO(DET-3204): Add the following comments to the docstring.
        # Alternatively, we could initialize them with the methods provided by
        # :py:class:`det.pytorch.PytorchTrialContext`, including ``Model``,
        # ``Optimizer``, ``LRScheduler``, ``_configure_apex_amp``.
        # Here is a code example.
        #
        # .. code-block:: python
        #
        #     self.context = trial_context
        #
        #     self.a = self.context.Model(MyModelA())
        #     self.b = self.context.Model(MyModelB())
        #     self.opt1 = self.context.Optimizer(torch.optm.Adam(self.a))
        #     self.opt2 = self.context.Optimizer(torch.optm.Adam(self.b))
        #
        #     (self.a, self.b), (self.opt1, self.opt2) = self.context._configure_apex_amp(
        #         models=[self.a, self.b],
        #         optimizers=[self.opt1, self.opt2],
        #         num_losses=2,
        #     )
        #
        #     self.lrs1 = context.LRScheduler(
        #         lr_scheduler=LambdaLR(self.opt1, lr_lambda=lambda epoch: 0.95 ** epoch),
        #         step_mode=LRScheduler.StepMode.STEP_EVERY_EPOCH,
        #     ))
        pass

    def build_model(self) -> nn.Module:
        """
        Defines the deep learning architecture associated with a trial. This method
        returns the model as an instance or subclass of :py:class:`nn.Module`.
        """
        # TODO(DET-3267): deprecate this when releasing pytorch flexible primitives.
        pass

    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:  # type: ignore
        """
        Describes the optimizer to be used during training of the given model,
        an instance of :py:class:`torch.optim.Optimizer`.
        """
        # TODO(DET-3267): deprecate this when releasing pytorch flexible primitives.
        pass

    def create_lr_scheduler(
        self, optimizer: torch.optim.Optimizer  # type: ignore
    ) -> Optional[LRScheduler]:
        """
        Create a learning rate scheduler for the trial given an instance of the
        optimizer.

        Arguments:
            optimizer (torch.optim.Optimizer): instance of the optimizer to be
                used for training

        Returns:
            :py:class:`det.pytorch.LRScheduler`:
                Wrapper around a :obj:`torch.optim.lr_scheduler._LRScheduler`.
        """
        # TODO(DET-3267): deprecate this when releasing pytorch flexible primitives.
        pass

    @abstractmethod
    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        """
        Calculate the loss for a batch and return it in a dictionary.
        :py:obj:`batch_idx` represents the total number of batches processed per
        device (slot) since the start of training.
        """

        # TODO(DET-3204): Add the following comments to the docstring.
        # If the models, optimizers, and LR schedulers are initialized in :meth:`__init__`
        # by users, users should calculate the gradients with the losses with :meth:`backward`
        # and call an optimization step for the optimizers with :meth:`step_optimizer`
        # on your own. Here is a code example.
        #
        # .. code-block:: python
        #
        #     # Assume having initialized models, optimizers, and LR schedulers (when
        #     # using manual mode): self.model1, self.model2, self.opt1, self.opt2, and self.lrs1.
        #
        #     # Calculate the losses using the models.
        #     loss1 = self.model1(batch)
        #     loss2 = self.model2(batch)
        #
        #     # Run backward passes on losses and step optimizers. These can happen
        #     # in arbitrary orders.
        #     self.context.backward(loss1)
        #     self.context.backward(loss2)
        #     self.context.step_optimizer(self.opt1)
        #     self.context.step_optimizer(self.opt2)
        #
        #     # Step the learning rate.
        #     self.lrs1.step()
        #
        #     return {"loss1": loss1, "loss2": loss2}

        # TODO(DET-3267): deprecate the model argument when releasing pytorch flexible primitives.
        pass

    @abstractmethod
    def build_training_data_loader(self) -> DataLoader:
        """
        Defines the data loader to use during training.

        Must return an instance of :py:class:`determined.pytorch.DataLoader`.
        """
        pass

    @abstractmethod
    def build_validation_data_loader(self) -> DataLoader:
        """
        Defines the data loader to use during validation.

        Must return an instance of :py:class:`determined.pytorch.DataLoader`.
        """
        pass

    def build_callbacks(self) -> Dict[str, _callback.PyTorchCallback]:
        """
        Defines a dictionary of string names to callbacks (if any) to be used
        during training and/or validation.

        The string name will be used as the key to save and restore callback
        state for any callback that defines :meth:`load_state_dict` and :meth:`state_dict`.
        """
        return {}

    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        """
        Calculate evaluation metrics for a batch and return them as a
        dictionary mapping metric names to metric values.

        There are two ways to specify evaluation metrics. Either override
        :meth:`evaluate_batch` or :meth:`evaluate_full_dataset`. While
        :meth:`evaluate_full_dataset` is more flexible,
        :meth:`evaluate_batch` should be preferred, since it can be
        parallelized in distributed environments, whereas
        :meth:`evaluate_full_dataset` cannot. Only one of
        :meth:`evaluate_full_dataset` and :meth:`evaluate_batch` should be
        overridden by a trial.

        The metrics returned from this function must be JSON-serializable.
        """
        # TODO(DET-3267): deprecate the model argument when releasing pytorch flexible primitives.
        pass

    def evaluation_reducer(self) -> Union[Reducer, Dict[str, Reducer]]:
        """
        Return a reducer for all evaluation metrics, or a dict mapping metric
        names to individual reducers. Defaults to :obj:`det.pytorch.Reducer.AVG`.
        """
        return Reducer.AVG

    def evaluate_full_dataset(
        self, data_loader: torch.utils.data.DataLoader, model: nn.Module
    ) -> Dict[str, Any]:
        """
        Calculate validation metrics on the entire validation dataset and
        return them as a dictionary mapping metric names to reduced metric
        values (i.e., each returned metric is the average or sum of that metric
        across the entire validation set).

        This validation can not be distributed and is performed on a single
        device, even when multiple devices (slots) are used for training. Only
        one of :meth:`evaluate_full_dataset` and :meth:`evaluate_batch` should
        be overridden by a trial.

        The metrics returned from this function must be JSON-serializable.
        """
        # TODO(DET-3267): deprecate the model argument when releasing pytorch flexible primitives.
        pass


def reset_parameters(model: torch.nn.Module) -> None:
    """
    Recursively calls ``reset_parameters()`` for all modules.

    Important: Call this prior to loading any backbone weights,
    otherwise those weights will be overwritten.
    """
    logging.info("Resetting model parameters.")
    for _, module in model.named_modules():
        reset_params = getattr(module, "reset_parameters", None)
        if callable(reset_params):
            reset_params()
