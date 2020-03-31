import enum
import logging
import os
import pathlib
import random
import shutil
from abc import abstractmethod
from typing import Any, Dict, List, Optional, Tuple, Union, cast

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
    Reducer,
    TorchData,
    _Data,
    _LRHelper,
    _reduce_metrics,
    data_length,
    to_device,
)
from determined_common import check

# Apex is included only for GPU trials.
try:
    import apex
except ImportError:
    if torch.cuda.is_available():
        logging.warning("Failed to import apex.")
    pass


class _WarningLogs(enum.Enum):
    FAILED_MOVING_TO_DEVICE = 1


class PyTorchTrialController(det.LoopTrialController):
    def __init__(self, trial_inst: det.Trial, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        check.is_instance(trial_inst, PyTorchTrial, "PyTorchTrialController needs an PyTorchTrial")
        self.trial = cast(PyTorchTrial, trial_inst)
        self._check_evaluate_implementation()

        self.model = self.trial.build_model()

        # Validation loader will be undefined on process ranks > 0
        # when the user defines `validate_full_dataset()`.
        self.validation_loader = None  # type: Optional[torch.utils.data.DataLoader]

        self._set_data_loaders()

        # Track whether a warning logging category has already been issued to the user.
        self.warning_logged = {_WarningLogs.FAILED_MOVING_TO_DEVICE: False}

        self._init_model()

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
        raise NotImplementedError("PyTorchTrial only supports the Native API")

    @staticmethod
    def support_determined_native() -> bool:
        return True

    def _init_device(self) -> None:
        self.n_gpus = len(self.env.container_gpus)
        if self.hvd_config.use:
            check.gt(self.n_gpus, 0)
            # We launch a horovod process per GPU. Each process
            # needs to bind to a unique GPU.
            self.device = torch.device(hvd.local_rank())
            torch.cuda.set_device(self.device)
        elif self.n_gpus > 0:
            self.device = torch.device("cuda", 0)
        else:
            self.device = torch.device("cpu")
        check.is_not_none(self.device)

    def _init_model(self) -> None:
        self.optimizer = self.trial.optimizer(self.model)
        # TODO: Check that optimizer is not an amp optimizer.

        self.lr_helper = _LRHelper(self.trial.create_lr_scheduler(self.optimizer))

        self._init_device()
        self.model = self.model.to(self.device)

        if self.hvd_config.use:
            use_compression = self.hvd_config.fp16_compression
            self.optimizer = hvd.DistributedOptimizer(
                self.optimizer,
                named_parameters=self.model.named_parameters(),
                backward_passes_per_step=self.hvd_config.aggregation_frequency,
                compression=hvd.Compression.fp16 if use_compression else hvd.Compression.none,
            )
            logging.debug("Initialized optimizer for distributed and optimized parallel training.")
        elif self.n_gpus > 1:
            check.eq(
                self.hvd_config.aggregation_frequency,
                1,
                "Please enable `optimized_parallel` to use aggregation "
                "frequency greater than 1 for single machine multi-GPU "
                "training.",
            )
            self.model = nn.DataParallel(self.model)
            logging.debug("Initialized mode for native parallel training.")

        # If a load path is provided load weights and restore the data location.
        self._load()

        self._configure_amp()

        if self.hvd_config.use:
            hvd.broadcast_parameters(self.model.state_dict(), root_rank=0)
            hvd.broadcast_optimizer_state(self.optimizer, root_rank=0)

        # Initialize training and validation iterators.
        self.training_iterator = iter(self.training_loader)

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

    def _get_amp_setting(self) -> str:
        amp_setting = self.env.experiment_config.get("optimizations", {}).get(
            "mixed_precision", None
        )
        check.is_not_none(amp_setting)
        check.not_in(
            "amp",
            self.env.hparams,
            "Please move `amp` setting from `hyperparameters` "
            "to `optimizations[`mixed_precision`]`.",
        )

        return cast(str, amp_setting)

    def use_amp(self) -> bool:
        return self._get_amp_setting() != "O0"

    def _configure_amp(self) -> None:
        if self.use_amp():
            check.true(
                torch.cuda.is_available(),
                "Mixed precision training (AMP) is supported only on GPU slots.",
            )
            check.false(
                not self.hvd_config.use and self.n_gpus > 1,
                "To enable mixed precision training (AMP) for parallel training, "
                'please set `resources["optimized_parallel"] = True`.',
            )

            logging.info(
                f"Enabling mixed precision training with opt_level: {self._get_amp_setting()}."
            )
            self.model, self.optimizer = apex.amp.initialize(
                self.model,
                self.optimizer,
                opt_level=self._get_amp_setting(),
                verbosity=1 if self.is_chief or self.env.experiment_config.debug_enabled() else 0,
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

    def _set_data_loaders(self) -> None:
        skip_batches = (self.env.first_step() - 1) * self.batches_per_step

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
                check.eq(len(args), 1)
                num_batches = cast(int, args[0])
                response_func(self._train_for_step(w.step_id, num_batches))
            elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                response_func(self._compute_validation_metrics())
            elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.eq(len(args), 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                response_func(self._save(path))
            elif w.kind == workload.Workload.Kind.TERMINATE:
                break
            else:
                raise AssertionError("Unexpected workload: {}".format(w.kind))

    def get_epoch_idx(self, batch_id: int) -> int:
        return batch_id // len(self.training_loader)

    def _to_device(self, data: _Data) -> TorchData:
        return to_device(
            data, self.device, self.warning_logged[_WarningLogs.FAILED_MOVING_TO_DEVICE],
        )

    @staticmethod
    def _average_gradients(parameters: Any, divisor: int) -> None:
        check.gt_eq(divisor, 1)
        if divisor == 1:
            return

        divisor_value = float(divisor)
        for p in filter(lambda param: param.grad is not None, parameters):
            p.grad.data.div_(divisor_value)

    def _clip_grads(self, parameters: Any) -> None:
        # TODO: Support clip by norm other than L2.
        clip_grad_l2_norm = self.env.hparams.get("clip_grad_l2_norm", None)
        clip_by_val = self.env.hparams.get("clip_grad_val", None)
        check.false(
            clip_grad_l2_norm is not None and clip_by_val is not None,
            "Please specify either `clip_grad_l2_norm` or `clip_by_val` "
            "in your hparams, not both.",
        )
        if clip_grad_l2_norm is not None:
            logging.debug(f"Clipping gradients by L2 norm of: {clip_grad_l2_norm}.")
            torch.nn.utils.clip_grad_norm_(parameters, clip_grad_l2_norm)  # type: ignore
        elif clip_by_val is not None:
            logging.debug(f"Clipping gradients by value of: {clip_by_val}.")
            torch.nn.utils.clip_grad_value_(parameters, clip_by_val)  # type: ignore
        else:
            logging.debug("No gradient clipping enabled.")

    def _train_for_step(self, step_id: int, batches_per_step: int) -> workload.Response:
        check.gt(step_id, 0)

        step_idx = step_id - 1
        start = step_idx * batches_per_step
        end = start + batches_per_step

        # Set the behavior of certain layers (e.g., dropout) that are different
        # between training and inference.
        self.model.train()

        metrics = []  # type: List[Dict]
        num_inputs = 0

        for batch_idx in range(start, end):
            batch = next(self.training_iterator)
            num_inputs += data_length(batch)

            batch = self._to_device(batch)
            # Forward pass.
            tr_metrics = self.trial.train_batch(
                batch=batch,
                model=self.model,
                epoch_idx=self.get_epoch_idx(batch_idx),
                batch_idx=batch_idx,
            )

            if isinstance(tr_metrics, torch.Tensor):
                tr_metrics = {"loss": tr_metrics}

            check.is_instance(
                tr_metrics,
                dict,
                "train_batch() must return a dictionary "
                "mapping string names to Tensor metrics, got {type(tr_metrics)}",
            )
            check.is_in("loss", tr_metrics.keys(), 'Please include "loss" in you training metrics.')

            # Backwards pass.
            loss = tr_metrics["loss"]
            communicate_and_update = (batch_idx + 1) % self.hvd_config.aggregation_frequency == 0
            if self.use_amp():
                with apex.amp.scale_loss(loss, self.optimizer) as scaled_loss:
                    scaled_loss.backward()
                    if self.hvd_config.use and communicate_and_update:
                        self.optimizer.synchronize()
            else:
                loss.backward()

            if communicate_and_update:
                parameters = (
                    self.model.parameters()
                    if not self.use_amp()
                    else apex.amp.master_params(self.optimizer)
                )

                if self.hvd_config.average_aggregated_gradients:
                    self._average_gradients(
                        parameters=parameters, divisor=self.hvd_config.aggregation_frequency
                    )

                self._clip_grads(parameters)

                if self.hvd_config.use and self.use_amp():
                    with self.optimizer.skip_synchronize():
                        self.optimizer.step()
                else:
                    self.optimizer.step()
                self.optimizer.zero_grad()

                if self.lr_helper.should_step_lr(
                    batch_idx, len(self.training_loader), self.hvd_config.aggregation_frequency
                ):
                    self.lr_helper.step()

            for name, metric in tr_metrics.items():
                # Convert PyTorch metric values to NumPy, so that
                # `det.util.encode_json` handles them properly without
                # needing a dependency on PyTorch.
                if isinstance(metric, torch.Tensor):
                    metric = metric.cpu().detach().numpy()
                tr_metrics[name] = metric

            check.is_in("loss", tr_metrics, 'Please include "loss" in your training metrics.')
            metrics.append(tr_metrics)

        if not self.is_chief:
            return workload.Skipped()

        if self.hvd_config.use:
            num_inputs *= hvd.size()

        logging.debug(f"Done training step: {num_inputs} records in {batches_per_step} batches.")
        return det.util.make_metrics(num_inputs, metrics)

    @staticmethod
    def _convert_metrics_to_numpy(metrics: Dict[str, Any]) -> Dict[str, Any]:
        for metric_name, metric_val in metrics.items():
            logging.debug(metric_name, metric_val)
            if isinstance(metric_val, torch.Tensor):
                logging.debug(f"Converting {metric_name} to CPU.")
                metrics[metric_name] = metric_val.cpu().numpy()
        return metrics

    @torch.no_grad()
    def _compute_validation_metrics(self) -> workload.Response:
        # Set the behavior of certain layers (e.g., dropout) that are
        # different between training and inference.
        self.model.eval()
        num_inputs = 0
        metrics = {}  # type: Optional[Dict[str, Any]]

        if self._evaluate_batch_defined():
            keys = None
            batch_metrics = []

            self.validation_loader = cast(torch.utils.data.DataLoader, self.validation_loader)
            check.gt(len(self.validation_loader), 0)
            for batch in self.validation_loader:
                batch = self._to_device(batch)
                num_inputs += data_length(batch)

                vld_metrics = self.trial.evaluate_batch(batch=batch, model=self.model)
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

            keys = cast(Any, keys)
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
                    data_loader=self.validation_loader, model=self.model
                )

                check.is_instance(
                    metrics, dict, f"eval() must return a dictionary, got {type(metrics)}."
                )

                metrics = self._convert_metrics_to_numpy(metrics)
                num_inputs = self.context.get_per_slot_batch_size() * len(self.validation_loader)

        if not self.is_chief:
            return workload.Skipped()

        return {"num_inputs": num_inputs, "validation_metrics": metrics}

    def _prepare_metrics_reducers(self, keys: Any) -> Dict[str, Reducer]:
        metrics_reducers = {}  # type: Dict[str, Reducer]
        if isinstance(self.trial.evaluation_reducer(), Dict):
            check.eq(
                metrics_reducers.keys(),
                keys,
                "Please provide a single evaluation reducer or "
                "provide a reducer for every validation metric.",
            )
        elif isinstance(self.trial.evaluation_reducer(), Reducer):
            for key in keys:
                metrics_reducers[key] = cast(Reducer, self.trial.evaluation_reducer())

        for key in keys:
            check.true(
                isinstance(metrics_reducers[key], Reducer),
                "Please select `det.pytorch.Reducer` " "for reducing validation metrics.",
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
            combined_metrics, batches_per_process = self._combine_metrics_across_processes(metrics)
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
                return None

        return metrics

    def _combine_metrics_across_processes(
        self, metrics: Dict[str, Any]
    ) -> Tuple[Optional[Dict[str, Any]], Optional[List[int]]]:
        # The chief receives the metric from every other training process.
        check.true(self.hvd_config.use)
        self.validation_loader = cast(torch.utils.data.DataLoader, self.validation_loader)

        metrics_lists = {}  # type: Dict[str, Any]
        batches_per_process = []  # type: List[int]
        if self.is_chief:
            self.train_process_comm_chief = cast(ipc.ZMQServer, self.train_process_comm_chief)
            worker_metrics = self.train_process_comm_chief.barrier(num_connections=hvd.size() - 1)
            worker_metrics = cast(List[ipc.MetricsInfo], worker_metrics)

            for metric_name in metrics.keys():
                metrics_lists[metric_name] = [metrics[metric_name]]
                for worker_metric in worker_metrics:
                    metrics_lists[metric_name].append(worker_metric.metrics[metric_name])

            batches_per_process.append(len(self.validation_loader))
            for worker_metric in worker_metrics:
                batches_per_process.append(worker_metric.num_batches)

            return metrics_lists, batches_per_process
        else:
            self.train_process_comm_worker = cast(ipc.ZMQClient, self.train_process_comm_worker)
            self.train_process_comm_worker.barrier(
                message=ipc.MetricsInfo(metrics=metrics, num_batches=len(self.validation_loader))
            )
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

        self.model.load_state_dict(checkpoint["model_state_dict"])
        self.optimizer.load_state_dict(checkpoint["optimizer_state_dict"])
        self.lr_helper.load_state_dict(checkpoint.get("lr_scheduler"))

    def _save(self, path: pathlib.Path) -> workload.Response:
        if not self.is_chief:
            return workload.Skipped()

        path.mkdir(parents=True, exist_ok=True)

        # The pickled_model_path is the entire nn.Module saved to a file via
        # pickle. This model can be recovered using the det.pytorch.checkpoint.load
        # method so long as the code is saved along with the pickled nn.Module.
        # https://pytorch.org/docs/stable/notes/serialization.html#recommend-saving-models
        pickled_model_path = path.joinpath("model.pth")
        code_path = path.joinpath("code")

        torch.save(self.model, pickled_model_path, pickle_module=cloudpickle)  # type: ignore

        # The model code is the current working directory.
        shutil.copytree(os.getcwd(), code_path, ignore=shutil.ignore_patterns("__pycache__"))

        util.write_checkpoint_metadata(
            path,
            self.env,
            {
                "torch_version": torch.__version__,  # type: ignore
            },
        )

        # PyTorch uses optimizer objects that take the model parameters to
        # optimize on construction, so we store and reload the `state_dict()`
        # of the model and optimizer explicitly (instead of dumping the entire
        # objects) to avoid breaking the connection between the model and the
        # optimizer.
        checkpoint = {
            "model_state_dict": self.model.state_dict(),
            "optimizer_state_dict": self.optimizer.state_dict(),
        }

        if self.lr_helper:
            checkpoint["lr_scheduler"] = self.lr_helper.state_dict()

        torch.save(  # type: ignore
            checkpoint, str(path.joinpath("state_dict.pth")), pickle_module=cloudpickle,
        )

        return {}


class PyTorchTrial(det.Trial):
    """
    PyTorch trials are created by subclassing the abstract class
    :class:`PyTorchTrial`.  Users must define all abstract methods to create the
    deep learning model associated with a specific trial, and to subsequently
    train and evaluate it.
    """

    trial_controller_class = PyTorchTrialController

    @abstractmethod
    def build_model(self) -> nn.Module:
        """
        Defines the deep learning architecture associated with a trial, which
        typically depends on the trial's specific hyperparameter settings
        stored in the :py:attr:`hparams` dictionary. This method returns the model as an
        an instance or subclass of :py:class:`nn.Module`.
        """
        pass

    @abstractmethod
    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:  # type: ignore
        """
        Describes the optimizer to be used during training of the given model,
        an instance of :py:class:`torch.optim.Optimizer`.
        """
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
        pass

    @abstractmethod
    def build_training_data_loader(self) -> DataLoader:
        """
        Defines the data loader to use during training.

        Must return an instance of ``determined.pytorch.DataLoader``.
        """
        pass

    @abstractmethod
    def build_validation_data_loader(self) -> DataLoader:
        """
        Defines the data loader to use during validation.

        Must return an instance of ``determined.pytorch.DataLoader``.
        """
        pass

    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        """
        Calculate evaluation metrics for a batch and return them as a
        dictionary mapping metric names to metric values.

        There are two ways to specify evaluation metrics. Either override
        :meth:`evaluate_batch` or :meth:`evaluate_full_dataset`. While
        :meth:`evaluate_full_dataset()` is more flexible,
        :meth:`evaluate_batch()` should be preferred, since it can be
        parallelized in distributed environments, whereas
        :meth:`evaluate_full_dataset()` cannot. Only one of
        :meth:`evaluate_full_dataset` and :meth:`evaluate_batch` should be
        overridden by a trial.
        """
        pass

    def evaluation_reducer(self) -> Union[Reducer, Dict[str, Reducer]]:
        """
        Return a reducer for all evaluation metrics, or a dict mapping metric
        names to individual reducers. Defaults to
        :obj:`det.pytorch.Reducer.AVG`.
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
        """
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
        pass


def reset_parameters(model: torch.nn.Module) -> None:
    """
    Recursively calls `reset_parameters()` for all modules.

    Important: Call this prior to loading any backbone weights,
    otherwise those weights will be overwritten.
    """
    logging.info("Resetting model parameters.")
    for _, module in model.named_modules():
        reset_params = getattr(module, "reset_parameters", None)
        if callable(reset_params):
            reset_params()
