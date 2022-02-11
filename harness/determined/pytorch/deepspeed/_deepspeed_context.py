import contextlib
import importlib.util
import json
import logging
import os
from typing import Any, Dict, Iterator, List, Optional, Set, Tuple, Type, Union, cast

import deepspeed
import torch
import torch.nn as nn
from deepspeed.runtime import config_utils

import determined as det
from determined import profiler, pytorch
from determined.pytorch import deepspeed as det_ds


def merge_dicts(base_dict: Dict[str, Any], source_dict: Dict[str, Any]) -> Dict[str, Any]:
    for key, value in source_dict.items():
        if key in base_dict:
            if isinstance(value, dict):
                base_dict[key] = merge_dicts(base_dict[key], value)
            else:
                base_dict[key] = value
        else:
            base_dict[key] = value
    return base_dict


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
        The resulting dictionary whe base_ds_config is overwritten with source_ds_dict.
    """
    if isinstance(base_ds_config, str):
        base_ds_config = json.load(
            open(base_ds_config, "r"),
            object_pairs_hook=config_utils.dict_raise_error_on_duplicate_keys,
        )
    else:
        if not isinstance(base_ds_config, dict):
            raise TypeError("Expected string or dict for base_ds_config argument.")

    return merge_dicts(cast(Dict[str, Any], base_ds_config), source_ds_dict)


class DeepSpeedTrialContext(det.TrialContext, pytorch._PyTorchReducerContext):
    """Contains runtime information for any Determined workflow that uses the ``DeepSpeedTrial`` API.

    With this class, users can do the following things:

    1. Wrap DeepSpeed model engines which contain the model, optimizer, lr_scheduler, etc.
       This will make sure Determined can automatically provide gradient aggregation,
       checkpointing and fault tolerance.  In contrast to :class:`determined.pytorch.PyTorchTrial`,
       the user does not need to wrap optimizer and lr_scheduler as that should all be instead
       passed to the DeepSpeed initialize function (see
       https://www.deepspeed.ai/getting-started/#writing-deepspeed-models) when building the
       model engine.
    2. Overwrite a deepspeed config file or dictionary with values from Determined's
       experiment config to ensure consistency in batch size and support hyperparameter tuning.
    3. Wrap a custom model parallel configure that should subclass
       :class:`determined.pytorch.deepspeed.ModelParallelUnit`.  We will automatically set mpu for
       data parallel and standard pipeline parallel training.  This should only needed if there
       is additional model parallelism outside of DeepSpeed's supported methods.
    4. Functionalities inherited from :class:`determined.TrialContext`, including getting
       the runtime information and properly handling training data in distributed training.
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        det.TrialContext.__init__(self, *args, **kwargs)
        pytorch._PyTorchReducerContext.__init__(self, self.distributed._zmq_allgather)

        self._init_device()

        # Track which types we have issued warnings for in to_device().
        self._to_device_warned_types = set()  # type: Set[Type]

        # DeepSpeed supports mixed precision through NVidia Apex AMP.  ZeRO optimizer requires
        # Apex AMP and cannot be used with more complex AMP modes.
        apex_available = importlib.util.find_spec("apex") is not None
        if not apex_available:
            logging.warning(
                "Missing package APEX is required for ZeRO optimizer support through DeepSpeed."
            )

        # The following attributes are initialized during the lifetime of
        # a DeepSpeedTrialContext.
        self.models = []  # type: List[nn.Module]
        self.profiler = None  # type: Any
        self._epoch_len = None  # type: Optional[int]

        self._loss_ids = {}  # type: Dict[torch.Tensor, int]
        self._last_backward_batch_idx = None  # type: Optional[int]
        self._current_batch_idx = None  # type: Optional[int]

        self._determined_profiler = None  # type: Optional[profiler.ProfilerAgent]
        self._mpu = det_ds.DeterminedModelParallelUnit(
            self.distributed
        )  # type: det_ds.ModelParallelUnit
        self._called_wrap_mpu = False
        self._train_micro_batch_size_per_gpu = None  # type: Optional[int]
        self._num_micro_batches_per_slot = None  # type: Optional[int]
        self._use_pipeline_parallel = False
        self._data_repro_checks_disabled = False
        self._manual_grad_accumulation = False

        self._check_experiment_config()

    def _check_experiment_config(self) -> None:
        """
        Check if the user specified options in optimizations that are incompatible with
        DeepSpeedTrial.
        """
        optimizations_config = self.env.experiment_config.get_optimizations_config()
        if not optimizations_config.get("average_training_metrics", False):
            logging.warning(
                "DeepSpeedTrial always averages training metrics across data parallel ranks."
            )
        self._average_training_metrics = True
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

    def wrap_mpu(self, mpu: det_ds.ModelParallelUnit) -> det_ds.ModelParallelUnit:
        if self._called_wrap_mpu:
            raise det.errors.InvalidExperimentException(
                "Only one MPU can be passed to DeepSpeedTrialContext.  "
                "Please make sure wrap_mpu is only called once in the trial definition."
            )
        self._called_wrap_mpu = True
        old_mpu = self._mpu
        self._mpu = mpu
        if old_mpu.get_data_parallel_world_size() != mpu.get_data_parallel_world_size():
            (
                self.env._per_slot_batch_size,
                self.env._global_batch_size,
            ) = self._calculate_batch_sizes()
            logging.warning(
                "Wrapped MPU uses model parallelism.  Changing per slot batch size "
                f"to {self.get_per_slot_batch_size()} and "
                f"global batch size to {self.get_global_batch_size()}"
            )
        return mpu

    @property
    def mpu(self) -> det_ds.ModelParallelUnit:
        return self._mpu

    def wrap_model_engine(self, model: torch.nn.Module) -> torch.nn.Module:
        """Returns a wrapped model engine.

        In the background, we perform checks to properly handle pipeline parallelism if
        the model engine is a PipelineEngine.  We also recompute batch sizes to match
        the deepspeed config.
        """

        if self.env.managed_training:
            model = model.to(self.device)

        # Pipeline parallel model engine has its own MPU that we will use here.
        if isinstance(model, deepspeed.PipelineEngine):
            self._use_pipeline_parallel = True
            self._mpu = det_ds.DeepSpeedMPU(model.mpu)

        recompute_batch_size = False

        # Check to make sure that Determined's global batch size matches the model engine's.
        # If not, overwrite Determined's batch size.
        if self.get_global_batch_size() != model.train_batch_size():
            logging.warning(
                f"Setting global batch size to {model.train_batch_size()} to match the "
                "deepspeed config.  To prevent this from happening, you can call "
                "self.context.overwrite_deepspeed_config(base_ds_config) to get a consistent"
                "deepspeed config dict which you can pass to the config field when calling"
                "deepspeed.initialize to build the model engine."
            )
            self.env._global_batch_size = model.train_batch_size()
            recompute_batch_size = True

        if self._train_micro_batch_size_per_gpu is None:
            self._train_micro_batch_size_per_gpu = int(model.train_micro_batch_size_per_gpu())
            recompute_batch_size = True
        # If multiple model engines are wrapped, we will make sure that they have the same
        # micro batch size.
        assert (
            self._train_micro_batch_size_per_gpu == model.train_micro_batch_size_per_gpu()
        ), "micro batch size do not match across DeepSpeed model engines."

        if recompute_batch_size:
            (
                self.env._per_slot_batch_size,
                self.env._global_batch_size,
            ) = self._calculate_batch_sizes()

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
        self._data_repro_checks_disabled = True

    @property
    def use_pipeline_parallel(self) -> bool:
        return self._use_pipeline_parallel

    def _calculate_batch_sizes(self) -> Tuple[int, int]:
        """Recompute per slot batch size to account for deepspeed support for micro-batches.

        This needs to be done after the user calls wrap_model_engine to let us know the deepspeed
        config which contains the micro-batch-size.
        """
        global_batch_size = self.env.global_batch_size

        # Configure batch sizes.
        num_replicas = self._mpu.get_data_parallel_world_size()

        per_gpu_batch_size = global_batch_size // num_replicas
        self._num_micro_batches_per_slot = per_gpu_batch_size // self.train_micro_batch_size_per_gpu
        per_gpu_batch_size = self.num_micro_batches_per_slot * self.train_micro_batch_size_per_gpu
        effective_batch_size = per_gpu_batch_size * num_replicas
        if effective_batch_size != global_batch_size:
            logging.warning(
                f"`global_batch_size` changed from {global_batch_size} to {effective_batch_size} "
                f"to account for deepspeed micro batch size."
            )
        return per_gpu_batch_size, effective_batch_size

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

    def _set_determined_profiler(self, prof: profiler.ProfilerAgent) -> None:
        self._determined_profiler = prof

    @contextlib.contextmanager
    def _record_timing(self, metric_name: str, accumulate: bool = False) -> Iterator[None]:
        if not self._determined_profiler:
            yield
            return
        with self._determined_profiler.record_timing(metric_name, accumulate):
            yield

    def _init_device(self) -> None:
        self.n_gpus = len(self.env.container_gpus)
        if self.distributed.size > 1:
            if self.n_gpus > 0:
                # We launch a separate process per GPU with LOCAL_RANK set by DeepSpeed's launcher.
                # Each process needs to bind to a unique GPU.
                self.device = torch.device("cuda", int(cast(str, os.environ.get("LOCAL_RANK"))))
                torch.cuda.set_device(self.device)
            else:
                self.device = torch.device("cpu")
        elif self.n_gpus > 0:
            self.device = torch.device("cuda", 0)
        else:
            self.device = torch.device("cpu")
        assert self.device is not None, "Error setting torch device."

    def to_device(self, data: pytorch._Data) -> pytorch.TorchData:
        """Map data to the device allocated by the Determined cluster.

        Since we pass an iterable over the dataloader to `train_batch` and `evaluate_batch`
        for DeepSpeedTrial, the user is responsible for moving data to GPU if needed.  This is
        basically a helper function to make that easier.
        """
        with self._record_timing("to_device", accumulate=True):
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
