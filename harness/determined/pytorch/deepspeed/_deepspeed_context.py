import json
import logging
import time
from importlib import util as importutil
from typing import Any, Dict, List, Optional, Set, Type, Union, cast

import deepspeed
import torch
from deepspeed.runtime import config_utils

import determined as det
from determined import pytorch, util
from determined.pytorch import deepspeed as det_ds

logger = logging.getLogger("determined.pytorch")


def overwrite_deepspeed_config(
    base_ds_config: Union[str, Dict], source_ds_dict: Dict[str, Any]
) -> Dict[str, Any]:
    """Overwrite a base_ds_config with values from a source_ds_dict.

    You can use source_ds_dict to overwrite leaf nodes of the base_ds_config.
    More precisely, we will iterate depth first into source_ds_dict and if a node corresponds to
    a leaf node of base_ds_config, we copy the node value over to base_ds_config.

    Arguments:
        base_ds_config (str or Dict): either a path to a DeepSpeed config file or a dictionary.
        source_ds_dict (Dict): dictionary with fields that we want to copy to base_ds_config
    Returns:
        The resulting dictionary when base_ds_config is overwritten with source_ds_dict.
    """
    if isinstance(base_ds_config, str):
        base_ds_config = json.load(
            open(base_ds_config, "r"),
            object_pairs_hook=config_utils.dict_raise_error_on_duplicate_keys,
        )
    else:
        if not isinstance(base_ds_config, dict):
            raise TypeError("Expected string or dict for base_ds_config argument.")

    return util.merge_dicts(cast(Dict[str, Any], base_ds_config), source_ds_dict)


class DeepSpeedTrialContext(det.TrialContext, pytorch._PyTorchReducerContext):
    """Contains runtime information for any Determined workflow that uses the ``DeepSpeedTrial``
    API.

    With this class, users can do the following things:

    1. Wrap DeepSpeed model engines that contain the model, optimizer, lr_scheduler, etc.
       This will make sure Determined can automatically provide gradient aggregation,
       checkpointing and fault tolerance.  In contrast to :class:`determined.pytorch.PyTorchTrial`,
       the user does not need to wrap optimizer and lr_scheduler as that should all be instead
       passed to the DeepSpeed initialize function (see
       https://www.deepspeed.ai/getting-started/#writing-deepspeed-models) when building the
       model engine.
    2. Overwrite a deepspeed config file or dictionary with values from Determined's
       experiment config to ensure consistency in batch size and support hyperparameter tuning.
    3. Set a custom model parallel configuration that should instantiate a
       :class:`determined.pytorch.deepspeed.ModelParallelUnit` dataclass.  We automatically set the
       mpu for data parallel and standard pipeline parallel training.  This should only be needed
       if there is additional model parallelism outside DeepSpeed's supported methods.
    4. Disable data reproducibility checks to allow custom data loaders.
    5. Disable automatic gradient aggregation for non-pipeline-parallel training.
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        det.TrialContext.__init__(self, *args, **kwargs)
        pytorch._PyTorchReducerContext.__init__(self, self.distributed.allgather)

        self._init_device()

        # Track which types we have issued warnings for in to_device().
        self._to_device_warned_types = set()  # type: Set[Type]

        # DeepSpeed supports mixed precision through Nvidia Apex AMP.  ZeRO optimizer requires
        # Apex AMP and cannot be used with more complex AMP modes.
        apex_available = importutil.find_spec("apex") is not None
        if not apex_available:
            logger.warning(
                "Missing package APEX is required for ZeRO optimizer support through DeepSpeed."
            )

        # The following attributes are initialized during the lifetime of
        # a DeepSpeedTrialContext.
        self.models = []  # type: List[deepspeed.DeepSpeedEngine]
        self._epoch_len = None  # type: Optional[int]

        self._loss_ids = {}  # type: Dict[torch.Tensor, int]
        self._last_backward_batch_idx = None  # type: Optional[int]
        self._current_batch_idx = None  # type: Optional[int]

        self.profiler = None  # type: Any

        self._mpu = det_ds.make_data_parallel_mpu(
            self.distributed
        )  # type: det_ds.ModelParallelUnit
        self._called_set_mpu = False
        self._train_micro_batch_size_per_gpu = None  # type: Optional[int]
        self._num_micro_batches_per_slot = None  # type: Optional[int]
        self._use_pipeline_parallel = False
        self._data_repro_checks_disabled = False
        self._manual_grad_accumulation = False

        self._check_experiment_config_optimizations()

        self._tbd_writer = None  # type: Optional[Any]
        self._enable_tensorboard_logging = True
        # Timestamp for batching TensorBoard uploads
        self._last_tb_reset_ts: Optional[float] = None

    def _check_experiment_config_optimizations(self) -> None:
        """
        Check if the user specified options in optimizations are incompatible with
        DeepSpeedTrial.
        """
        optimizations_config = self.env.experiment_config.get_optimizations_config()
        self._average_training_metrics = optimizations_config.get("average_training_metrics", False)

        mixed_precision_val = optimizations_config.get("mixed_precision", "O0")
        if mixed_precision_val != "O0":
            raise det.errors.InvalidExperimentException(
                "Mixed precision is specified through the deepspeed config instead of the "
                "Determined experiment config.",
            )
        aggregation_frequency = optimizations_config.get("aggregation_frequency", 1)
        if aggregation_frequency > 1:
            raise det.errors.InvalidExperimentException(
                "Gradient aggregation is specified through the deepspeed config instead of the "
                "Determined experiment config.",
            )
        other_optimizations_default_values = {
            "average_aggregated_gradients": True,
            "gradient_compression": False,
            "tensor_fusion_threshold": 64,
            "tensor_fusion_cycle_time": 5,
            "autotune_tensor_fusion": False,
        }
        for opt_field, default_value in other_optimizations_default_values.items():
            opt_value = optimizations_config.get(opt_field, default_value)
            if opt_value != default_value:
                logger.warning(
                    f"{opt_field}={opt_value} ignored since the setting does not apply "
                    "to DeepSpeedTrial."
                )

    def set_mpu(self, mpu: det_ds.ModelParallelUnit) -> None:
        """Use a custom model parallel configuration.

        The argument ``mpu`` should implement a
        :class:`determined.pytorch.deepspeed.ModelParallelUnit` dataclass to provide information
        on data parallel topology and whether a rank should compute metrics/build data loaders.

        This should only be needed if training with custom model parallelism.

        In the case of multiple model parallel engines, we assume that the MPU and data loaders
        correspond to the first wrapped model engine.
        """
        if len(self.models) == 0:
            raise det.errors.InvalidExperimentException(
                "Please call `wrap_model` before setting the mpu."
            )
        if self._called_set_mpu:
            raise det.errors.InvalidExperimentException(
                "Only one MPU can be passed to DeepSpeedTrialContext. "
                "Please make sure wrap_mpu is only called once in the trial definition."
            )
        if self.distributed.rank == 0:
            if not self._mpu.should_report_metrics and not self._average_training_metrics:
                raise det.errors.InvalidExperimentException(
                    "Please set optimizations.average_training_metrics in the experiment config "
                    "to true so that metrics will exist on the chief for report to the master."
                )
        self._called_set_mpu = True
        self._mpu = mpu

    def wrap_model_engine(self, model: deepspeed.DeepSpeedEngine) -> deepspeed.DeepSpeedEngine:
        """Register a DeepSpeed model engine.

        In the background, we track the model engine for checkpointing, set batch size information,
        using the first wrapped model engine, and perform checks to properly handle pipeline
        parallelism if the model engine is a PipelineEngine.
        """
        model = model.to(self.device)

        # Pipeline parallel model engine has its own MPU that we will use here.
        if isinstance(model, deepspeed.PipelineEngine):
            self._use_pipeline_parallel = True
            if len(self.models) == 0:
                self._mpu = det_ds.make_deepspeed_mpu(model.mpu)
            else:
                logger.warning("Using the MPU corresponding to the first wrapped model engine. ")

        if len(self.models) == 0:
            self._train_micro_batch_size_per_gpu = int(model.train_micro_batch_size_per_gpu())
            self._num_micro_batches_per_slot = int(model.gradient_accumulation_steps())
        else:
            # If multiple model engines are wrapped, we will raise a warning if the micro batch
            # size for additional model engines does not match that of the first model engine.
            if model.train_micro_batch_size_per_gpu() != self._train_micro_batch_size_per_gpu:
                logger.warning(
                    f"Train micro batch size for wrapped model engine {len(self.models) + 1} does "
                    "not match that for the first wrapped engine.  Num sample reporting will only "
                    "apply to wrapped model engine 1."
                )

        self.models.append(model)
        return model

    def disable_auto_grad_accumulation(self) -> None:
        """
        Prevent the DeepSpeedTrialController from automatically calling train_batch multiple times
        to process enough micro batches to meet the per slot batch size.  Thus, the user is
        responsible for manually training on enough micro batches in train_batch to meet the
        expected per slot batch size.
        """
        self._manual_grad_accumulation = True

    def disable_dataset_reproducibility_checks(self) -> None:
        """
        ``disable_dataset_reproducibility_checks()`` allows you to return an arbitrary
        ``DataLoader`` from
        :meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.build_training_data_loader` or
        :meth:`~determined.pytorch.deepspeed.DeepSpeedTrial.build_validation_data_loader`.

        Normally you would be required to return a ``det.pytorch.DataLoader`` instead, which would
        guarantee that an appropriate ``Sampler`` is used that ensures:

        - When ``shuffle=True``, the shuffle is reproducible.
        - The dataset will start at the right location, even after pausing/continuing.
        - Proper sharding is used during distributed training.

        However, there can be cases where either reproducibility of the dataset is not needed or
        where the nature of the dataset can cause the ``det.pytorch.DataLoader`` to be unsuitable.

        In those cases, you can call ``disable_dataset_reproducibility_checks()`` and you will be
        free to return any ``torch.utils.data.DataLoader`` you like.  Dataset reproducibility will
        still be possible, but it will be your responsibility.  The ``Sampler`` classes in
        :mod:`determined.pytorch.samplers` can help in this regard.
        """
        self._data_repro_checks_disabled = True

    @property
    def use_pipeline_parallel(self) -> bool:
        return self._use_pipeline_parallel

    @property
    def train_micro_batch_size_per_gpu(self) -> int:
        if self._train_micro_batch_size_per_gpu is None:
            raise det.errors.InvalidExperimentException(
                "Please call wrap_model_engine before accessing train_micro_batch_size."
            )
        return self._train_micro_batch_size_per_gpu

    @property
    def num_micro_batches_per_slot(self) -> int:
        if self._num_micro_batches_per_slot is None:
            raise det.errors.InvalidExperimentException(
                "Please call wrap_model_engine before accessing num_micro_batches_per_slot."
            )
        return self._num_micro_batches_per_slot

    def _init_device(self) -> None:
        self.n_gpus = len(self.env.container_gpus)
        if not self.n_gpus:
            raise det.errors.InvalidExperimentException("GPUs required for DeepSpeedTrial.")
        if self.distributed.size > 1:
            self.device = torch.device("cuda", self.distributed.get_local_rank())
            torch.cuda.set_device(self.device)
        else:
            self.device = torch.device("cuda", 0)
        assert self.device is not None, "Error setting torch device."

    def to_device(self, data: pytorch._Data) -> pytorch.TorchData:
        """Map data to the device allocated by the Determined cluster.

        Since we pass an iterable over the data loader to ``train_batch`` and ``evaluate_batch``
        for DeepSpeedTrial, the user is responsible for moving data to GPU if needed.  This is
        basically a helper function to make that easier.
        """
        return pytorch.to_device(data, self.device, self._to_device_warned_types)

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

    def set_profiler(self, *args: List[str], **kwargs: Any) -> None:
        """
        ``set_profiler()`` is a thin wrapper around PyTorch profiler, torch-tb-profiler.
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
