import contextlib
import logging
import pathlib
import time
import warnings
from typing import Any, Callable, Dict, Iterator, List, Optional, Set, Tuple, Type, Union

import torch
from torch import nn

import determined as det
from determined import horovod, pytorch, util

logger = logging.getLogger("determined.pytorch")

# Apex is included only for GPU trials.
try:
    import apex
except ImportError:  # pragma: no cover
    if torch.cuda.is_available():
        logger.warning("Failed to import apex.")
    pass

# AMP is only available in PyTorch 1.6+
try:
    from torch.cuda import amp

    HAVE_AMP = True
except ImportError:  # pragma: no cover
    HAVE_AMP = False
    if torch.cuda.is_available():
        logger.warning("PyTorch AMP is unavailable.")
    pass


class PyTorchTrialContext(pytorch._PyTorchReducerContext):
    """Contains runtime information for any Determined workflow that uses the ``PyTorch`` API.

    With this class, users can do the following things:

    1. Wrap PyTorch models, optimizers, and LR schedulers with their Determined-compatible
       counterparts using :meth:`wrap_model`, :meth:`wrap_optimizer`, :meth:`wrap_lr_scheduler`,
       respectively. The Determined-compatible objects are capable of transparent
       distributed training, checkpointing and exporting, mixed-precision training,
       and gradient aggregation.
    2. Configure apex amp by calling :meth:`configure_apex_amp` (optional).
    3. Calculate the gradients with :meth:`backward` on a specified loss.
    4. Run an optimization step with :meth:`step_optimizer`.
    5. Functionalities inherited from :class:`determined.TrialContext`, including getting
       the runtime information and properly handling training data in distributed training.
    """

    def __init__(
        self,
        core_context: det.core.Context,
        trial_seed: Optional[int],
        hparams: Optional[Dict],
        slots_per_trial: int,
        num_gpus: int,
        exp_conf: Optional[Dict[str, Any]],
        aggregation_frequency: int,
        steps_completed: int,
        managed_training: bool,
        debug_enabled: bool,
        enable_tensorboard_logging: bool = True,
    ) -> None:
        self._core = core_context
        self.distributed = self._core.distributed
        pytorch._PyTorchReducerContext.__init__(self, self.distributed.allgather)
        self._per_slot_batch_size, self._global_batch_size = (
            util.calculate_batch_sizes(
                hparams=hparams,
                slots_per_trial=slots_per_trial,
                trialname="PyTorchTrial",
            )
            if hparams and hparams.get("global_batch_size", None)
            else (None, None)
        )
        self._hparams = hparams
        self._num_gpus = num_gpus
        self._debug_enabled = debug_enabled
        self._exp_conf = exp_conf

        self._trial_seed = trial_seed
        self._distributed_backend = det._DistributedBackend()
        self._steps_completed = steps_completed
        self.device = self._init_device()

        # Track which types we have issued warnings for in to_device().
        self._to_device_warned_types = set()  # type: Set[Type]

        # The following attributes are initialized during the lifetime of
        # a PyTorchTrialContext.
        self.models = []  # type: List[nn.Module]
        self.optimizers = []  # type: List[torch.optim.Optimizer]
        self.profiler = None  # type: Any
        self.lr_schedulers = []  # type: List[pytorch.LRScheduler]
        self._epoch_len = None  # type: Optional[int]

        # Keep a map of wrapped models to their original input forms, which is needed
        # by torch DDP and apex to initialize in the correct order
        self._wrapped_models = {}  # type: Dict[nn.Module, nn.Module]

        # Keep a map of optimizer configs set in wrap_optimizer which can differ per-optimizer
        self._optimizer_configs = {}  # type: Dict[Any, Any]

        # Use a main model to contain all the models because when using horovod
        # to broadcast the states of models we want to avoid name conflicts for these
        # states, so we set all the models to be submodule of the main model with
        # different names using __setattr__ and use the state_dict of the main model
        # for broadcasting. Note that broadcast_parameters only accepts state_dict()
        # although its doc says it also accepts named_parameters()
        self._main_model = nn.Module()
        self._scaler = None
        self._use_apex = False
        self._loss_ids = {}  # type: Dict[torch.Tensor, int]
        self._last_backward_batch_idx = None  # type: Optional[int]
        self._current_batch_idx = None  # type: Optional[int]

        self.experimental = pytorch.PyTorchExperimentalContext(self)
        self._reducers = pytorch._PyTorchReducerContext()

        self._managed_training = managed_training

        self._aggregation_frequency = aggregation_frequency

        self._fp16_compression_default = False
        self._average_aggregated_gradients_default = True
        self._is_pre_trainer = False

        self._stop_requested = False

        self._tbd_writer = None  # type: Optional[Any]
        self._enable_tensorboard_logging = enable_tensorboard_logging
        # Timestamp for batching TensorBoard uploads
        self._last_tb_reset_ts: Optional[float] = None

    def get_global_batch_size(self) -> int:
        """
        Return the global batch size.
        """
        if self._global_batch_size is None:
            raise ValueError(
                "global_batch_size is undefined in this Trial because hparams was not "
                "configured. Please check the init() call to Trainer API."
            )
        return self._global_batch_size

    def get_per_slot_batch_size(self) -> int:
        """
        Return the per-slot batch size. When a model is trained with a single GPU, this is equal to
        the global batch size. When multi-GPU training is used, this is equal to the global batch
        size divided by the number of GPUs used to train the model.
        """
        if self._per_slot_batch_size is None:
            raise ValueError(
                "per_slot_batch_size is undefined in this Trial because hparams was not "
                "configured. Please check the init() call to Trainer API."
            )

        return self._per_slot_batch_size

    def get_experiment_config(self) -> Dict[str, Any]:
        if self._exp_conf is None:
            raise ValueError(
                "exp_conf is undefined in this Trial. Please check the init() call to Trainer API."
            )
        return self._exp_conf

    def get_hparam(self, name: str) -> Any:
        """
        Return the current value of the hyperparameter with the given name.
        """
        if self._hparams is None:
            raise ValueError(
                "hparams is undefined in this Trial because hparams was not "
                "configured. Please check the init() call to Trainer API."
            )
        if name not in self.get_hparams():
            raise ValueError(
                "Could not find name '{}' in experiment "
                "hyperparameters. Please check your experiment "
                "configuration 'hyperparameters' section.".format(name)
            )
        if name == "global_batch_size":
            logger.warning(
                "Please use `context.get_per_slot_batch_size()` and "
                "`context.get_global_batch_size()` instead of accessing "
                "`global_batch_size` directly."
            )
        return self.get_hparams()[name]

    def get_hparams(self) -> Dict[str, Any]:
        if self._hparams is None:
            raise ValueError(
                "hparams is undefined in this Trial because hparams was not "
                "configured. Please check the init() call to Trainer API."
            )
        return self._hparams

    def get_stop_requested(self) -> bool:
        """
        Return whether a trial stoppage has been requested.
        """
        return self._stop_requested

    def set_stop_requested(self, stop_requested: bool) -> None:
        """
        Set a flag to request a trial stoppage. When this flag is set to True,
        we finish the step, checkpoint, then exit.
        """
        if not isinstance(stop_requested, bool):
            raise AssertionError("stop_requested must be a boolean")

        logger.info(
            "A trial stoppage has requested. The trial will be stopped "
            "at the end of the current step."
        )
        self._stop_requested = stop_requested

    def set_enable_tensorboard_logging(self, enable_tensorboard_logging: bool) -> None:
        """
        Set a flag to indicate whether automatic upload to tensorboard is enabled.
        """
        if not isinstance(enable_tensorboard_logging, bool):
            raise AssertionError("enable_tensorboard_logging must be a boolean")

        self._enable_tensorboard_logging = enable_tensorboard_logging

    def get_enable_tensorboard_logging(self) -> bool:
        """
        Return whether automatic tensorboard logging is enabled
        """
        return self._enable_tensorboard_logging

    def autocast_forward_pass(self, to_wrap: torch.nn.Module) -> torch.nn.Module:
        # First, ensure the forward pass is wrapped in an autocast context:
        class _AutocastForwardPassModel(type(to_wrap)):  # type: ignore
            def __init__(wrapper) -> None:
                self.model = to_wrap

            def __getattr__(wrapper, name):  # type: ignore
                return getattr(to_wrap, name)

            def __setattr__(wrapper, name, value):  # type: ignore
                return setattr(to_wrap, name, value)

            def __delattr__(wrapper, name):  # type: ignore
                return delattr(to_wrap, name)

            def forward(wrapper, *arg, **kwarg):  # type: ignore
                with amp.autocast():
                    return to_wrap.forward(*arg, **kwarg)

        wrapped = _AutocastForwardPassModel()

        # To not print errors recursively or every forward pass
        warned_types = set()

        # Second, eliminate any need for the loss functions to be in that context:
        def end_fp16(module: torch.nn.Module, input: Any, output: Any) -> Any:  # noqa: A002
            if isinstance(output, torch.Tensor):
                return output.float() if output.dtype == torch.float16 else output
            if isinstance(output, dict):
                return {k: end_fp16(module, input, v) for k, v in output.items()}
            if isinstance(output, list):
                return [end_fp16(module, input, d) for d in output]
            if isinstance(output, tuple):
                return tuple(end_fp16(module, input, d) for d in output)
            # If there are other types that embed Tensors still using fp16 and loss computation
            # subsequently fails, then experimental.use_amp should not be used and the forward pass
            # should be manually wrapped in an autocast context.
            if type(output) not in warned_types:
                warned_types.add(type(output))
                logger.warn(
                    f"Unexpected type '{type(output).__name__}' outputted by model in experimental "
                    "AMP mode."
                )
            return output

        wrapped.register_forward_hook(end_fp16)

        return wrapped

    def wrap_model(self, model: torch.nn.Module) -> torch.nn.Module:
        """Returns a wrapped model."""

        if self._managed_training:
            if self._use_apex:
                raise det.errors.InvalidExperimentException(
                    "Must call wrap_model() before configure_apex_amp.",
                )

            model = model.to(self.device)

            if self.distributed.size > 1 and self._distributed_backend.use_torch():
                wrapped_model = self._PyTorchDistributedDataParallel(model)
            else:
                wrapped_model = model

            self._wrapped_models[wrapped_model] = model
        else:
            wrapped_model = model

        model_id = len(self.models)
        self._main_model.__setattr__(f"model_{model_id}", wrapped_model)

        if self.experimental._auto_amp:
            wrapped_model = self.autocast_forward_pass(wrapped_model)

        self.models.append(wrapped_model)
        return wrapped_model

    def wrap_optimizer(
        self,
        optimizer: torch.optim.Optimizer,
        backward_passes_per_step: int = 1,
        fp16_compression: Optional[bool] = None,
        average_aggregated_gradients: Optional[bool] = None,
    ) -> torch.optim.Optimizer:
        """Returns a wrapped optimizer.

        The optimizer must use the models wrapped by :meth:`wrap_model`. This function
        creates a ``horovod.DistributedOptimizer`` if using parallel/distributed training.

        ``backward_passes_per_step`` can be used to specify how many gradient aggregation
        steps will be performed in a single ``train_batch`` call per optimizer step.
        In most cases, this will just be the default value 1.  However, this advanced functionality
        can be used to support training loops like the one shown below:

        .. code-block:: python

            def train_batch(
                self, batch: TorchData, epoch_idx: int, batch_idx: int
            ) -> Dict[str, torch.Tensor]:
                data, labels = batch
                output = self.model(data)
                loss1 = output['loss1']
                loss2 = output['loss2']
                self.context.backward(loss1)
                self.context.backward(loss2)
                self.context.step_optimizer(self.optimizer, backward_passes_per_step=2)
                return {"loss1": loss1, "loss2": loss2}

        """
        if self._managed_training:
            if self._use_apex:
                raise det.errors.InvalidExperimentException(
                    "Must call wrap_optimizer() before configure_apex_amp.",
                )
            if backward_passes_per_step < 1:
                raise det.errors.InvalidExperimentException(
                    "backward_passes_per_step for local gradient aggregation must be >= 1; "
                    f"got {backward_passes_per_step}.",
                )

            if self.distributed.size > 1 and self._distributed_backend.use_horovod():
                # We always override default fp16_compression setting if passed in directly
                if fp16_compression is None:
                    fp16_compression = self._fp16_compression_default

                hvd = horovod.hvd
                optimizer = hvd.DistributedOptimizer(
                    optimizer,
                    named_parameters=self._filter_named_parameters(optimizer),
                    backward_passes_per_step=backward_passes_per_step * self._aggregation_frequency,
                    compression=hvd.Compression.fp16 if fp16_compression else hvd.Compression.none,
                )
                logger.debug(
                    "Initialized optimizer for distributed and optimized parallel training."
                )

        if average_aggregated_gradients is None:
            average_aggregated_gradients = self._average_aggregated_gradients_default

        self._optimizer_configs[optimizer] = {
            "average_aggregated_gradients": average_aggregated_gradients
        }
        self.optimizers.append(optimizer)
        return optimizer

    def wrap_lr_scheduler(
        self,
        lr_scheduler: torch.optim.lr_scheduler._LRScheduler,
        step_mode: pytorch.LRScheduler.StepMode,
        frequency: int = 1,
    ) -> torch.optim.lr_scheduler._LRScheduler:
        """
        Returns a wrapped LR scheduler.

        The LR scheduler must use an optimizer wrapped by :meth:`wrap_optimizer`.  If ``apex.amp``
        is in use, the optimizer must also have been configured with :meth:`configure_apex_amp`.
        """
        if isinstance(lr_scheduler, torch.optim.lr_scheduler.ReduceLROnPlateau):
            if step_mode != pytorch.LRScheduler.StepMode.MANUAL_STEP:
                raise det.errors.InvalidExperimentException(
                    "detected that context.wrap_lr_scheduler() was called with an instance of "
                    "torch.optim.lr_scheduler.ReduceLROnPlateau as the lr_scheduler.  This lr "
                    "scheduler class does not have the usual step() parameters, and so it can "
                    "only be used with step_mode=MANUAL_STEP.\n"
                    "\n"
                    "For example, if you wanted to step it on every validation step, you might "
                    "wrap your lr_scheduler and pass it to a callback like this:\n"
                    "\n"
                    "class MyLRStepper(PyTorchCallback):\n"
                    "    def __init__(self, wrapped_lr_scheduler):\n"
                    "        self.wrapped_lr_scheduler = wrapped_lr_scheduler\n"
                    "\n"
                    "    def on_validation_end(self, metrics):\n"
                    '        self.wrapped_lr_scheduler.step(metrics["validation_error"])\n'
                )

        opt = getattr(lr_scheduler, "optimizer", None)
        if opt is not None:
            if opt not in self.optimizers:
                raise det.errors.InvalidExperimentException(
                    "Must use an optimizer that is returned by wrap_optimizer().",
                )
        wrapped = pytorch.LRScheduler(lr_scheduler, step_mode, frequency)
        self.lr_schedulers.append(wrapped)

        # Return the original LR scheduler to the user in case they have customizations that we
        # don't care about.
        return lr_scheduler

    def set_profiler(self, *args: List[str], **kwargs: Any) -> None:
        """
        ``set_profiler()`` is a thin wrapper around the native PyTorch profiler, torch-tb-profiler.
        It overrides the ``on_trace_ready`` parameter to the determined tensorboard path, while all
        other arguments are passed directly into ``torch.profiler.profile``. Stepping the profiler
        will be handled automatically during the training loop.

        See the `PyTorch profiler plugin
        <https://github.com/pytorch/kineto/tree/master/tb_plugin>`_ for details.

        Examples:

        Profiling GPU and CPU activities, skipping batch 1, warming up on batch 2, and profiling
        batches 3 and 4.

        .. code-block:: python

            self.context.set_profiler(
                activities=[
                    torch.profiler.ProfilerActivity.CPU,
                    torch.profiler.ProfilerActivity.CUDA,
                ],
                schedule=torch.profiler.schedule(
                    wait=1,
                    warmup=1,
                    active=2
                ),
            )
        """
        self.profiler = torch.profiler.profile(
            on_trace_ready=torch.profiler.tensorboard_trace_handler(
                str(self.get_tensorboard_path())
            ),
            *args,
            **kwargs,
        )

    def _filter_named_parameters(self, optimizer: torch.optim.Optimizer) -> List:
        """_filter_named_parameters filters the named parameters of a specified optimizer out
        of all the named parameters from a specified model. We need this function because
        a ``torch.optim.Optimizer`` doesn't store parameter names, and we need the names of
        the parameters when mapping parameters to each ``horovod.DistributedOptimizer``.
        """
        opt_params = {p for group in optimizer.param_groups for p in group.get("params", [])}
        return [(name, p) for name, p in self._main_model.named_parameters() if p in opt_params]

    def _init_device(self) -> torch.device:
        if self.distributed.size > 1:
            if self._num_gpus > 0:
                # We launch a horovod process per GPU. Each process
                # needs to bind to a unique GPU.
                device = torch.device("cuda", self.distributed.local_rank)
                torch.cuda.set_device(device)
            else:
                device = torch.device("cpu")
        elif self._num_gpus > 0:
            device = torch.device("cuda", 0)
        else:
            device = torch.device("cpu")
        return device

    def _set_default_gradient_compression(self, gradient_compression: bool) -> None:
        self._fp16_compression_default = gradient_compression

    def _set_default_average_aggregated_gradients(self, average_aggregated_gradients: bool) -> None:
        self._average_aggregated_gradients_default = average_aggregated_gradients

    def _set_is_pre_trainer(self) -> None:
        self._is_pre_trainer = True

    def to_device(self, data: pytorch._Data) -> pytorch.TorchData:
        """Map generated data to the device allocated by the Determined cluster.

        All the data in the data loader and the models are automatically moved to the
        allocated device. This method aims at providing a function for the data generated
        on the fly.
        """
        return pytorch.to_device(data, self.device, self._to_device_warned_types)

    def wrap_scaler(self, scaler: Any) -> Any:
        """
        Prepares to use automatic mixed precision through PyTorch’s native AMP API. The returned
        scaler should be passed to ``step_optimizer``, but usage does not otherwise differ from
        vanilla PyTorch APIs. Loss should be scaled before calling ``backward``, ``unscale_`` should
        be called before clipping gradients, ``update`` should be called after stepping all
        optimizers, etc.

        PyTorch 1.6 or greater is required for this feature.

        Arguments:
            scaler (``torch.cuda.amp.GradScaler``):  Scaler to wrap and track.

        Returns:
            The scaler. It may be wrapped to add additional functionality for use in Determined.
        """

        if not HAVE_AMP:
            raise det.errors.InvalidExperimentException(
                "Using context.wrap_scaler() requires PyTorch >= 1.6.",
            )

        if self._use_apex:
            raise det.errors.InvalidExperimentException("Do not mix APEX with PyTorch AMP.")

        if self._scaler is not None:
            raise det.errors.InvalidExperimentException(
                "Please only call wrap_scaler or use_amp once.",
            )

        if self.models:
            raise det.errors.InvalidExperimentException(
                "Please call wrap_scaler before wrap_model.",
            )

        # We don't need to check if CUDA is available because if it is not, a GradScaler is
        #  disabled when initialized, and we allow for disabled scalers to exist.

        self._scaler = scaler

        return scaler

    def configure_apex_amp(
        self,
        models: Union[torch.nn.Module, List[torch.nn.Module]],
        optimizers: Union[torch.optim.Optimizer, List[torch.optim.Optimizer]],
        enabled: Optional[bool] = True,
        opt_level: Optional[str] = "O1",
        cast_model_type: Optional[torch.dtype] = None,
        patch_torch_functions: Optional[bool] = None,
        keep_batchnorm_fp32: Optional[Union[bool, str]] = None,
        master_weights: Optional[bool] = None,
        loss_scale: Optional[Union[float, str]] = None,
        cast_model_outputs: Optional[torch.dtype] = None,
        num_losses: Optional[int] = 1,
        verbosity: Optional[int] = 1,
        min_loss_scale: Optional[float] = None,
        max_loss_scale: Optional[float] = 2.0**24,
    ) -> Tuple:
        """
        Configure automatic mixed precision for your models and optimizers using NVIDIA's Apex
        PyTorch extension. Note that details for ``apex.amp`` are handled automatically within
        Determined after this call.

        This function must be called **after** you have finished constructing your models and
        optimizers with :meth:`wrap_model` and :meth:`wrap_optimizer`.

        This function has the same arguments as
        `apex.amp.initialize <https://nvidia.github.io/apex/amp.html#apex.amp.initialize>`_.

        .. warning::
            When using distributed training and automatic mixed precision,
            we only support ``num_losses=1`` and calling backward on the loss once.

        Arguments:
            models (``torch.nn.Module`` or list of ``torch.nn.Module`` s):  Model(s) to modify/cast.
            optimizers (``torch.optim.Optimizer`` or list of ``torch.optim.Optimizer`` s):
                Optimizers to modify/cast. REQUIRED for training.
            enabled (bool, optional, default=True):  If False, renders all Amp calls no-ops,
                so your script should run as if Amp were not present.
            opt_level (str, optional, default="O1"):  Pure or mixed precision optimization level.
                Accepted values are "O0", "O1", "O2", and "O3", explained in detail above.
            cast_model_type (``torch.dtype``, optional, default=None):  Optional property override,
                see above.
            patch_torch_functions (bool, optional, default=None):  Optional property override.
            keep_batchnorm_fp32 (bool or str, optional, default=None):  Optional property override.
                If passed as a string, must be the string "True" or "False".
            master_weights (bool, optional, default=None):  Optional property override.
            loss_scale (float or str, optional, default=None):  Optional property override.
                If passed as a string, must be a string representing a number, e.g., "128.0",
                or the string "dynamic".
            cast_model_outputs (torch.dtype, optional, default=None):  Option to ensure that
                the outputs of your model is always cast to a particular type regardless of
                ``opt_level``.
            num_losses (int, optional, default=1):  Option to tell Amp in advance how many
                losses/backward passes you plan to use.  When used in conjunction with the
                ``loss_id`` argument to ``amp.scale_loss``, enables Amp to use a different
                loss scale per loss/backward pass, which can improve stability.
                If ``num_losses`` is left to 1, Amp will still support multiple losses/backward
                passes, but use a single global loss scale for all of them.
            verbosity (int, default=1):  Set to 0 to suppress Amp-related output.
            min_loss_scale (float, default=None):  Sets a floor for the loss scale values that
                can be chosen by dynamic loss scaling.  The default value of None means that no
                floor is imposed. If dynamic loss scaling is not used, ``min_loss_scale`` is
                ignored.
            max_loss_scale (float, default=2.**24):  Sets a ceiling for the loss scale values
                that can be chosen by dynamic loss scaling.  If dynamic loss scaling is not used,
                ``max_loss_scale`` is ignored.

        Returns:
            Model(s) and optimizer(s) modified according to the ``opt_level``.
            If  ``optimizers`` args were lists, the corresponding return value will
            also be a list.
        """
        if not enabled or not self._managed_training:
            return models, optimizers

        if self._scaler is not None and self._scaler.is_enabled():
            raise det.errors.InvalidExperimentException("Do not mix APEX with PyTorch AMP.")

        warnings.warn(
            "PyTorchTrial support for NVIDIA/apex has been deprecated and will be removed "
            "in a future version. We recommend users to migrate to Torch AMP (`torch.cuda.amp`).",
            FutureWarning,
            stacklevel=2,
        )

        if self._use_apex:
            raise det.errors.InvalidExperimentException("Please only call configure_apex_amp once.")

        if self.distributed.size > 1:
            if num_losses != 1:
                raise det.errors.InvalidExperimentException(
                    "When using distributed training, "
                    "Determined only supports configure_apex_amp with num_losses = 1.",
                )
            if self._aggregation_frequency > 1:
                raise det.errors.InvalidExperimentException(
                    "context.configure_apex_amp is not supported with "
                    "distributed training and "
                    "aggregation frequency > 1.",
                )

        if not torch.cuda.is_available():
            raise det.errors.InvalidExperimentException(
                "context.configure_apex_amp is supported only on GPU slots.",
            )

        self._use_apex = True

        if self._distributed_backend.use_torch():
            # We need to get the pre-wrapped input models to initialize APEX because
            if isinstance(models, list):
                models = [self._wrapped_models[wrapped_model] for wrapped_model in models]
            else:
                models = self._wrapped_models[models]

        logger.info(f"Enabling mixed precision training with opt_level: {opt_level}.")
        models, optimizers = apex.amp.initialize(
            models=models,
            optimizers=optimizers,
            enabled=enabled,
            opt_level=opt_level,
            cast_model_type=cast_model_type,
            patch_torch_functions=patch_torch_functions,
            keep_batchnorm_fp32=keep_batchnorm_fp32,
            master_weights=master_weights,
            loss_scale=loss_scale,
            cast_model_outputs=cast_model_outputs,
            num_losses=num_losses,
            min_loss_scale=min_loss_scale,
            max_loss_scale=max_loss_scale,
            verbosity=verbosity if self.distributed.get_rank() == 0 or self._debug_enabled else 0,
        )

        if not isinstance(models, list):
            self.models = [models]

        if self.distributed.size > 1 and self._distributed_backend.use_torch():
            # If Torch DDP is in use, re-wrap the models
            self.models = [self._PyTorchDistributedDataParallel(model) for model in self.models]

        if not isinstance(optimizers, list):
            self.optimizers = [optimizers]
        return models, optimizers

    @contextlib.contextmanager
    def _no_sync(self) -> Iterator[None]:
        assert (
            self._distributed_backend.use_torch()
        ), "_no_sync() is only applicable when using Torch DDP"
        with contextlib.ExitStack() as exit_stack:
            for ddp_model in self.models:
                exit_stack.enter_context(ddp_model.no_sync())
            yield

    def _should_communicate_and_update(self) -> bool:
        if not self._managed_training:
            return True
        if self._current_batch_idx is None:
            raise det.errors.InternalException("Training hasn't started.")
        return (self._current_batch_idx + 1) % self._aggregation_frequency == 0

    def backward(
        self,
        loss: torch.Tensor,
        gradient: Optional[torch.Tensor] = None,
        retain_graph: bool = False,
        create_graph: bool = False,
    ) -> None:
        """Compute the gradient of current tensor w.r.t. graph leaves.

        The arguments are used in the same way as ``torch.Tensor.backward``.
        See https://pytorch.org/docs/1.4.0/_modules/torch/tensor.html#Tensor.backward for details.

        .. warning::
            When using distributed training, we don't support manual gradient accumulation.
            That means the gradient on each parameter can only be calculated once on each batch.
            If a parameter is associated with multiple losses, you can either choose to call
            ``backward'' on only one of those losses, or you can set the ``require_grads`` flag of
            a parameter or module to ``False`` to avoid manual gradient accumulation on that
            parameter.
            However, you can do gradient accumulation across batches by setting
            :ref:`optimizations.aggregation_frequency <config-aggregation-frequency>` in the
            experiment configuration to be greater than 1.

        Arguments:
            gradient (Tensor or None): Gradient w.r.t. the
                tensor. If it is a tensor, it will be automatically converted
                to a Tensor that does not require grad unless ``create_graph`` is True.
                None values can be specified for scalar Tensors or ones that
                don't require grad. If a None value would be acceptable then
                this argument is optional.
            retain_graph (bool, optional): If ``False``, the graph used to compute
                the grads will be freed. Note that in nearly all cases setting
                this option to True is not needed and often can be worked around
                in a much more efficient way. Defaults to the value of
                ``create_graph``.
            create_graph (bool, optional): If ``True``, graph of the derivative will
                be constructed, allowing to compute higher order derivative
                products. Defaults to ``False``.
        """
        if self._use_apex:
            if (
                self._last_backward_batch_idx is None
                or self._current_batch_idx is None
                or self._last_backward_batch_idx < self._current_batch_idx
            ):
                self._last_backward_batch_idx = self._current_batch_idx
            else:
                raise det.errors.InvalidExperimentException(
                    "Calling context.backward(loss) multiple times is not supported "
                    "while using apex.amp and parallel/distributed training"
                )

            if loss not in self._loss_ids:
                self._loss_ids[loss] = len(self._loss_ids)
            with apex.amp.scale_loss(
                loss, self.optimizers, loss_id=self._loss_ids[loss]
            ) as scaled_loss:
                scaled_loss.backward(
                    gradient=gradient, retain_graph=retain_graph, create_graph=create_graph
                )

                if (
                    self.distributed.size > 1
                    and self._should_communicate_and_update()
                    and self._distributed_backend.use_horovod()
                ):
                    # When we exit out of this context manager, we need to finish
                    # communicating gradient updates before they are unscaled.
                    #
                    # Unfortunately, there is no clean way to support unscaling
                    # happening after synchronizing the optimizer on apex.amp.
                    # A short-term solution is to not support multiple losses nor
                    # multiple backward passes on one loss. A long-term solution is
                    # to integrate torch native AMP (https://pytorch.org/docs/stable/amp.html),
                    # which will come out soon.
                    for optimizer in self.optimizers:
                        optimizer.synchronize()  # type: ignore
        else:
            if self._scaler and self.experimental._auto_amp:
                loss = self._scaler.scale(loss)

            if (
                self.distributed.size > 1
                and self._distributed_backend.use_torch()
                and not self._should_communicate_and_update()
            ):
                # PyTorch DDP automatically syncs gradients by default on every backward pass.
                # no_sync() disables gradient all-reduce until the last iteration.
                with self._no_sync():
                    loss.backward(  # type: ignore
                        gradient=gradient, retain_graph=retain_graph, create_graph=create_graph
                    )
            else:
                loss.backward(  # type: ignore
                    gradient=gradient, retain_graph=retain_graph, create_graph=create_graph
                )

    @staticmethod
    def _average_gradients(parameters: Any, divisor: int) -> None:
        if divisor == 1:
            return

        divisor_value = float(divisor)
        for p in filter(lambda param: param.grad is not None, parameters):
            p.grad.data.div_(divisor_value)

    def step_optimizer(
        self,
        optimizer: torch.optim.Optimizer,
        clip_grads: Optional[Callable[[Iterator], None]] = None,
        auto_zero_grads: bool = True,
        scaler: Optional[Any] = None,
        # Should be ``torch.cuda.amp.GradScaler``, but:
        #   * other implementations might be possible
        #   * requiring this type forces upgrades to PyTorch 1.6+
    ) -> None:
        """
        Perform a single optimization step.

        This function must be called once for each optimizer. However, the order of
        different optimizers' steps can be specified by calling this function in different
        orders. Also, gradient accumulation across iterations is performed by the Determined
        training loop by setting the experiment configuration field
        :ref:`optimizations.aggregation_frequency <config-aggregation-frequency>`.

        Here is a code example:

        .. code-block:: python

            def clip_grads(params):
                torch.nn.utils.clip_grad_norm_(params, 0.0001),

            self.context.step_optimizer(self.opt1, clip_grads)

        Arguments:
            optimizer(``torch.optim.Optimizer``): Which optimizer should be stepped.
            clip_grads(a function, optional): This function should have one argument for
                parameters in order to clip the gradients.
            auto_zero_grads(bool, optional): Automatically zero out gradients automatically after
                stepping the optimizer. If false, you need to call ``optimizer.zero_grad()``
                manually. Note that if :ref:`optimizations.aggregation_frequency
                <config-aggregation-frequency>` is greater than 1, ``auto_zero_grads`` must be true.
            scaler(``torch.cuda.amp.GradScaler``, optional): The scaler to use for stepping the
                optimizer. This should be unset if not using AMP, and is necessary if
                ``wrap_scaler()`` was called directly.
        """

        if self._aggregation_frequency > 1 and not auto_zero_grads:
            raise det.errors.InvalidExperimentException(
                "if optimizations.aggregation_frequency is larger than 1, "
                "auto_zero_grads must be set to true. ",
            )

        if not self._should_communicate_and_update():
            return

        # Communication needs to be synchronized so that is completed
        # before we apply gradient clipping and `step()`.
        # In the case of APEX this is called in backward() instead, so that it's inside the context
        # manager and before unscaling.
        # In the case of PyTorch DDP, losses are synchronized during the backwards() pass.
        if (
            self.distributed.size > 1
            and self._distributed_backend.use_horovod()
            and not self._use_apex
        ):
            optimizer.synchronize()  # type: ignore

        parameters = (
            [p for group in optimizer.param_groups for p in group.get("params", [])]
            if not self._use_apex
            else apex.amp.master_params(optimizer)
        )

        if bool(self._optimizer_configs[optimizer]["average_aggregated_gradients"]):
            self._average_gradients(parameters=parameters, divisor=self._aggregation_frequency)

        if clip_grads is not None:
            if self._scaler and self.experimental._auto_amp:
                self._scaler.unscale_(optimizer)
            clip_grads(parameters)

        # For stepping the optimizer we will operate on the scaler passed
        # in, or fall back to the wrapped scaler (if any).
        if scaler is None and self.experimental._auto_amp:
            scaler = self._scaler
        if scaler:

            def step_fn() -> None:
                scaler.step(optimizer)  # type: ignore

        else:
            step_fn = optimizer.step  # type: ignore

        # In the case of PyTorch DDP, losses are synchronized automatically on the backwards() pass
        if self.distributed.size > 1 and self._distributed_backend.use_horovod():
            with optimizer.skip_synchronize():  # type: ignore
                step_fn()
        else:
            step_fn()

        if auto_zero_grads:
            optimizer.zero_grad()

    def is_epoch_start(self) -> bool:
        """
        Returns true if the current batch is the first batch of the epoch.

        .. warning::
            Not accurate for variable size epochs.
        """
        if self._current_batch_idx is None:
            raise det.errors.InternalException("Training hasn't started.")
        if self._epoch_len is None:
            raise det.errors.InternalException("Training DataLoader uninitialized.")
        return self._current_batch_idx % self._epoch_len == 0

    def is_epoch_end(self) -> bool:
        """
        Returns true if the current batch is the last batch of the epoch.

        .. warning::
            Not accurate for variable size epochs.
        """
        if self._current_batch_idx is None:
            raise det.errors.InternalException("Training hasn't started.")
        if self._epoch_len is None:
            raise det.errors.InternalException("Training DataLoader uninitialized.")
        return self._current_batch_idx % self._epoch_len == self._epoch_len - 1

    def current_train_epoch(self) -> int:
        if self._current_batch_idx is None:
            raise det.errors.InternalException("Training hasn't started.")
        if self._epoch_len is None:
            raise det.errors.InternalException("Training DataLoader uninitialized.")
        return self._current_batch_idx // self._epoch_len

    def current_train_batch(self) -> int:
        """
        Current global batch index
        """
        if self._current_batch_idx is None:
            raise det.errors.InternalException("Training hasn't started.")
        return self._current_batch_idx

    def get_trial_seed(self) -> int:
        if self._trial_seed is None:
            raise det.errors.InternalException("Trial seed not set.")
        return self._trial_seed

    def get_initial_batch(self) -> int:
        return self._steps_completed

    def get_data_config(self) -> Dict[str, Any]:
        """
        Return the data configuration.
        """
        return self.get_experiment_config().get("data", {})

    def get_experiment_id(self) -> int:
        """
        Return the experiment ID of the current trial.
        """
        return int(self._core.train._exp_id)

    def get_trial_id(self) -> int:
        """
        Return the trial ID of the current trial.
        """
        return int(self._core.train._trial_id)

    def get_tensorboard_path(self) -> pathlib.Path:
        """
        Get the path where files for consumption by TensorBoard should be written
        """
        return self._core.train.get_tensorboard_path()

    def get_tensorboard_writer(self) -> Any:
        """
        This function returns an instance of ``torch.utils.tensorboard.SummaryWriter``

        Trials users who wish to log to TensorBoard can use this writer object.
        We provide and manage a writer in order to save and upload TensorBoard
        files automatically on behalf of the user.

        Usage example:

        .. code-block:: python

           class MyModel(PyTorchTrial):
               def __init__(self, context):
                   ...
                   self.writer = context.get_tensorboard_writer()

               def train_batch(self, batch, epoch_idx, batch_idx):
                   self.writer.add_scalar('my_metric', np.random.random(), batch_idx)
                   self.writer.add_image('my_image', torch.ones((3,32,32)), batch_idx)
        """

        if self._tbd_writer is None:
            # As of torch v1.9.0, torch.utils.tensorboard has a bug that is exposed by setuptools
            # 59.6.0.  The bug is that it attempts to import distutils then access distutils.version
            # without actually importing distutils.version.  We can workaround this by prepopulating
            # the distutils.version submodule in the distutils module.
            #
            # Except, starting with python 3.12 distutils isn't available at all.
            try:
                import distutils.version  # noqa: F401
            except ImportError:
                pass

            from torch.utils import tensorboard

            self._tbd_writer = tensorboard.SummaryWriter(
                self.get_tensorboard_path()
            )  # type: ignore

        return self._tbd_writer

    def _maybe_reset_tbd_writer(self) -> None:
        """
        Reset (close current file and open a new one) the current writer if the current epoch
        second is at least one second greater than the epoch second of the last reset.

        The TensorFlow event writer names each event file by the epoch second it is created, so
        if events are written quickly in succession (< 1 second apart), they will overwrite each
        other.

        This effectively batches event writes so each event file may contain more than one event.
        """
        if self._tbd_writer is None:
            return

        current_ts = time.time()

        if self._last_tb_reset_ts is None:
            self._last_tb_reset_ts = current_ts

        if int(current_ts) > int(self._last_tb_reset_ts):
            self._tbd_writer.close()
            self._last_tb_reset_ts = current_ts
        else:
            # If reset didn't happen, flush, so that upstream uploads will reflect the latest
            # metric writes. reset() flushes automatically.
            self._tbd_writer.flush()

    class _PyTorchDistributedDataParallel(
        torch.nn.parallel.DistributedDataParallel  # type: ignore
    ):
        """
        Pass-through Model Wrapper to enable access to inner module attributes
        when using PyTorch DDP
        """

        def __getattr__(self, name: str) -> Any:
            try:
                return super().__getattr__(name)
            except AttributeError:
                return getattr(self.module, name)
