import contextlib
import logging
import random
import sys
from typing import Any, Dict, Iterator, Optional

import numpy as np
import torch
import torch.distributed as dist

import determined as det
from determined import core, gpu, horovod, pytorch

logger = logging.getLogger("determined.pytorch")


class Trainer:
    """
    ``pytorch.Trainer`` is an abstraction on top of a vanilla PyTorch training loop that handles
    many training details under-the-hood, and exposes APIs for configuring training-related features
    such as automatic checkpointing, validation, profiling, metrics reporting, etc.

    ``Trainer`` must be initialized and called from within a ``pytorch.PyTorchTrialContext``.
    """

    def __init__(self, trial: pytorch.PyTorchTrial, context: pytorch.PyTorchTrialContext):
        self._trial = trial
        self._context = context
        self._core = self._context._core
        self._distributed_backend = det._DistributedBackend()
        self._info = det.get_cluster_info()
        self._local_training = self._info is None or self._info.task_type != "TRIAL"

    def configure_profiler(
        self,
        sync_timings: bool = True,
        enabled: bool = False,
        begin_on_batch: int = 0,
        end_after_batch: Optional[int] = None,
    ) -> None:
        """
        @deprecated: Configure `fit(..., profiling_enabled=True) instead`.

        Configures the Determined profiler. This functionality is only supported for on-cluster
        training. For local training mode, this method is a no-op.

        This method should only be called before .fit(), and only once within the scope of init().
        If called multiple times, the last call's configuration will be used.

        Arguments:
            sync_timings: (Optional) Specifies whether Determined should wait for all GPU kernel
                streams before considering a timing as ended. Defaults to true. Applies only for
                frameworks that collect timing metrics (currently just PyTorch).
            enabled: (Optional) Defines whether profiles should be collected or not. Defaults to
                false.
            begin_on_batch: (Optional) Specifies the batch on which profiling should begin.
                Defaults to 0.
            end_after_batch: (Optional) Specifies the batch after which profiling should end.

        .. note::

           Profiles are collected for a maximum of 5 minutes, regardless of the settings above.

        """
        logger.error(
            "`trainer.configure_profiler` has been replaced with "
            "`fit(..., profiling_enabled=True)` and will be removed in a future release."
        )

    def fit(
        self,
        checkpoint_period: Optional[pytorch.TrainUnit] = None,
        validation_period: Optional[pytorch.TrainUnit] = None,
        max_length: Optional[pytorch.TrainUnit] = None,
        reporting_period: pytorch.TrainUnit = pytorch.Batch(100),  # noqa: B008
        checkpoint_policy: str = "best",
        latest_checkpoint: Optional[str] = None,
        step_zero_validation: bool = False,
        test_mode: bool = False,
        profiling_enabled: bool = False,
    ) -> None:
        """
        ``fit()`` trains a ``PyTorchTrial`` configured from the ``Trainer`` and handles
        checkpointing and validation steps, and metrics reporting.

        Arguments:
            checkpoint_period: The number of steps to train for before checkpointing. This is
                a ``TrainUnit`` type (``Batch`` or ``Epoch``) which can take an ``int`` or
                instance of ``collections.abc.Container`` (list, tuple, etc.). For example,
                ``Batch(100)`` would checkpoint every 100 batches, while ``Batch([5, 30, 45])``
                would checkpoint after every 5th, 30th, and 45th batch.
            validation_period: The number of steps to train for before validating. This is a
                ``TrainUnit`` type (``Batch`` or ``Epoch``) which can take an ``int`` or instance
                of ``collections.abc.Container`` (list, tuple, etc.). For example, ``Batch(100)``
                would validate every 100 batches, while ``Batch([5, 30, 45])`` would validate
                after every 5th, 30th, and 45th batch.
            max_length: The maximum number of steps to train for. This value is required and
                only applicable in local training mode. For on-cluster training, this value will
                be ignored; the searcher’s ``max_length`` must be configured from the experiment
                configuration. This is a ``TrainUnit`` type (``Batch`` or ``Epoch``) which takes an
                ``int``. For example, ``Epoch(1)`` would train for a maximum length of one epoch.
            reporting_period: The number of steps to train for before reporting metrics and
                searcher progress. For local training mode, metrics are printed to stdout. This
                is a ``TrainUnit`` type (``Batch`` or ``Epoch``) which can take an ``int`` or
                instance of ``collections.abc.Container`` (list, tuple, etc.). For example,
                ``Batch(100)`` would report every 100 batches, while ``Batch([5, 30, 45])`` would
                report after every 5th, 30th, and 45th batch.
            checkpoint_policy: Controls how Determined performs checkpoints after validation
                operations, if at all. Should be set to one of the following values:

                    best (default): A checkpoint will be taken after every validation operation
                        that performs better than all previous validations for this experiment.
                        Validation metrics are compared according to the ``metric`` and
                        ``smaller_is_better`` fields in the searcher configuration. This option
                        is only supported for on-cluster training.
                    all: A checkpoint will be taken after every validation, no matter the
                        validation performance.
                    none: A checkpoint will never be taken due to a validation. However,
                        even with this policy selected, checkpoints are still expected to be taken
                        after the trial is finished training, due to cluster scheduling decisions,
                        before search method decisions, or due to ``min_checkpoint_period``.
            latest_checkpoint: Configures the checkpoint used to start or continue training.
                This value should be set to ``det.get_cluster_info().latest_checkpoint`` for
                standard continue training functionality.
            step_zero_validation: Configures whether to perform an initial validation before
                training. Defaults to false.
            test_mode: Runs a minimal loop of training for testing and debugging purposes. Will
                train and validate one batch. Defaults to false.
            profiling_enabled: Enables system metric profiling functionality for on-cluster
                training. Defaults to false.
        """
        # Set defaults.
        if checkpoint_period is None:
            checkpoint_period = pytorch.Batch(sys.maxsize)

        if validation_period is None:
            validation_period = pytorch.Batch(sys.maxsize)

        if self._local_training:
            if checkpoint_policy == "best":
                logger.warning(
                    "checkpoint_policy='best' is not supported in local training mode. "
                    "Falling back to 'all'."
                )
                checkpoint_policy = "all"
            if max_length is None:
                raise ValueError("max_length must be defined in local training mode.")

            if not isinstance(max_length.value, int):
                raise TypeError("max_length must be configured in TrainUnit(int) types.")

            if profiling_enabled:
                logger.warning("Profiling is not supported in local training mode.")

            smaller_is_better = True
            searcher_metric_name = None
            steps_completed = 0
            global_batch_size = None
        else:
            if test_mode:
                raise ValueError("test_mode is only supported in local training mode.")

            if max_length is not None:
                logger.warning(
                    "max_length is ignored when training on-cluster. Please configure the "
                    "searcher instead."
                )

            assert self._info, "Unable to detect cluster info."
            if latest_checkpoint is None and self._info.latest_checkpoint is not None:
                logger.warning(
                    "latest_checkpoint has not been configured. Pause/resume training will not "
                    "be able to continue from latest checkpoint. Did you mean to set "
                    "`fit(latest_checkpoint=info.latest_checkpoint)'?"
                )

            smaller_is_better = bool(self._info.trial._config["searcher"]["smaller_is_better"])
            searcher_metric_name = self._info.trial._config["searcher"]["metric"]
            steps_completed = int(self._info.trial._steps_completed)
            global_batch_size = self._info.trial.hparams.get("global_batch_size", None)
            if global_batch_size:
                global_batch_size = int(global_batch_size)

        trial_controller = pytorch._PyTorchTrialController(
            trial_inst=self._trial,
            context=self._context,
            checkpoint_period=checkpoint_period,
            validation_period=validation_period,
            smaller_is_better=smaller_is_better,
            steps_completed=steps_completed,
            latest_checkpoint=latest_checkpoint,
            local_training=self._local_training,
            test_mode=test_mode,
            reporting_period=reporting_period,
            searcher_metric_name=searcher_metric_name,
            checkpoint_policy=checkpoint_policy,
            step_zero_validation=step_zero_validation,
            max_length=max_length,
            global_batch_size=global_batch_size,
            profiling_enabled=profiling_enabled,
        )

        trial_controller.run()


def _initialize_distributed_backend() -> Optional[core.DistributedContext]:
    info = det.get_cluster_info()

    distributed_backend = det._DistributedBackend()
    if distributed_backend.use_horovod():
        hvd = horovod.hvd
        hvd.require_horovod_type("torch", "PyTorchTrial is in use.")
        hvd.init()
        return core.DistributedContext.from_horovod(horovod.hvd)
    elif distributed_backend.use_torch():
        if torch.cuda.is_available():
            dist.init_process_group(backend="nccl")  # type: ignore
        else:
            dist.init_process_group(backend="gloo")  # type: ignore
        return core.DistributedContext.from_torch_distributed()
    elif info and (len(info.container_addrs) > 1 or len(info.slot_ids) > 1):
        raise ValueError(
            "In multi-slot managed cluster training, you must wrap your training script with a "
            "distributed launch layer such as determined.launch.torch_distributed or "
            "determined.launch.horovod."
        )
    return None


def _set_random_seeds(seed: int) -> None:
    # Set identical random seeds on all training processes.
    # When doing distributed training, each worker will start at a unique
    # offset in the dataset, ensuring that it is processing a unique
    # training batch.
    random.seed(seed)
    np.random.seed(seed)
    torch.random.manual_seed(seed)


@contextlib.contextmanager
def init(
    *,
    hparams: Optional[Dict] = None,
    exp_conf: Optional[Dict[str, Any]] = None,
    distributed: Optional[core.DistributedContext] = None,
    aggregation_frequency: int = 1,
    enable_tensorboard_logging: bool = True,
) -> Iterator[pytorch.PyTorchTrialContext]:
    """
    Creates a PyTorchTrialContext for use with a PyTorchTrial. All trainer.* calls must be within
    the scope of this context because there are resources started in __enter__ that must be
    cleaned up in __exit__.

    Arguments:
        hparams: (Optional) instance of hyperparameters for the trial
        exp_conf: (Optional) for local-training mode. If unset, calling
            context.get_experiment_config() will fail.
        distributed: (Optional) custom distributed training configuration
        aggregation_frequency: number of batches before gradients are exchanged in distributed
            training. This value is configured here because it is used in context.wrap_optimizer.
        enable_tensorboard_logging: Configures if upload to tensorboard is enabled
    """
    cluster_info = det.get_cluster_info()
    local_training = cluster_info is None or cluster_info.task_type != "TRIAL"

    # Pre-execute steps: initialize distributed backend and random seeds.
    distributed_context = distributed

    if not local_training:
        distributed_context = _initialize_distributed_backend()

    # Initialize default values.
    if local_training:
        trial_seed = None
        steps_completed = 0
        managed_training = True
        debug_enabled = False
        num_gpus = len(gpu.get_gpu_uuids())
    else:
        assert cluster_info, "Unable to detect cluster info"

        trial_seed = cluster_info.trial.trial_seed
        exp_conf = cluster_info.trial._config
        steps_completed = cluster_info.trial._steps_completed
        managed_training = True
        num_gpus = len(cluster_info.gpu_uuids)
        debug_enabled = cluster_info.trial._debug

        _set_random_seeds(trial_seed)

    with core.init(
        distributed=distributed_context,
        preempt_mode=core.PreemptMode.WorkersAskChief,
        tensorboard_mode=core.TensorboardMode.MANUAL,
    ) as core_context:
        context = pytorch.PyTorchTrialContext(
            core_context=core_context,
            trial_seed=trial_seed,
            hparams=hparams,
            slots_per_trial=core_context.distributed.get_size(),
            num_gpus=num_gpus,
            exp_conf=exp_conf,
            aggregation_frequency=aggregation_frequency,
            steps_completed=steps_completed,
            managed_training=managed_training,
            debug_enabled=debug_enabled,
            enable_tensorboard_logging=enable_tensorboard_logging,
        )

        yield context
