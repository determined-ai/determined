import contextlib
import logging
import os
import random
import sys
import warnings
from typing import Any, Dict, Iterator, Optional

import deepspeed
import numpy as np
import torch

import determined as det
from determined import core, gpu, pytorch
from determined.pytorch import deepspeed as det_ds

logger = logging.getLogger("determined.pytorch.deepspeed")


class Trainer:
    """
    ``pytorch.deepspeed.Trainer`` is an abstraction on top of a  DeepSpeed training loop
    that handles many training details under-the-hood, and exposes APIs for configuring
    training-related features such as automatic checkpointing, validation, profiling,
    metrics reporting, etc.

    ``Trainer`` must be initialized and called from within a
    ``pytorch.deepspeed.DeepSpeedTrialContext``.
    """

    def __init__(self, trial: det_ds.DeepSpeedTrial, context: det_ds.DeepSpeedTrialContext):
        self._trial = trial
        self._context = context
        self._core = self._context._core
        self._info = det.get_cluster_info()
        self._local_training = self._info is None or self._info.task_type != "TRIAL"

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
        ``fit()`` trains a ``DeepSpeedTrial`` configured from the ``Trainer`` and handles
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
            max_length: The maximum number of steps to train for. This is a ``TrainUnit`` type
                (``Batch`` or ``Epoch``) which takes an ``int``. For example, ``Epoch(1)`` would
                train for a maximum length of one epoch.
                .. note::
                   If using an ASHA searcher, this value should match the searcher config values in
                   the experiment config (i.e. ``Epoch(1)`` = `max_time: 1` and `time_metric:
                   "epochs"`).

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

            if not isinstance(max_length, (pytorch.Batch, pytorch.Epoch)) or not isinstance(
                max_length.value, int
            ):
                raise TypeError(
                    "max_length must either be a det.pytorch.Batch(int) or det.pytorch.Epoch(int) "
                    "type"
                )

            if profiling_enabled:
                logger.warning("Profiling is not supported in local training mode.")

            smaller_is_better = True
            searcher_metric_name = None
            steps_completed = 0
            global_batch_size = None
        else:
            if test_mode:
                raise ValueError("test_mode is only supported in local training mode.")

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

            # Backwards compatibility: try to parse legacy `searcher.max_length` if `max_length`
            # isn't passed in.
            if max_length is None:
                max_length_val = core._parse_searcher_max_length(self._info.trial._config)
                if max_length_val:
                    warnings.warn(
                        "Configuring `max_length` from the `searcher.max_length` experiment "
                        "config, which was deprecated in 0.38.0 and will be removed in a future "
                        "release. Please set `fit(max_length=X)` with your desired training length "
                        "directly.",
                        FutureWarning,
                        stacklevel=2,
                    )
                    max_length_unit = core._parse_searcher_units(self._info.trial._config)
                    max_length = pytorch.TrainUnit._from_searcher_unit(
                        max_length_val, max_length_unit, global_batch_size
                    )

            # If we couldn't parse the legacy `searcher.max_length`, raise an error.
            if not max_length:
                raise ValueError(
                    "`fit(max_length=X)` must be set with your desired training length."
                )
            if not isinstance(max_length, (pytorch.Batch, pytorch.Epoch)) or not isinstance(
                max_length.value, int
            ):
                raise TypeError(
                    "max_length must either be a det.pytorch.Batch(int) or det.pytorch.Epoch(int) "
                    "type."
                )

            _check_searcher_length(exp_conf=self._info.trial._config, max_length=max_length)

        trial_controller = det_ds.DeepSpeedTrialController(
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


def _check_searcher_length(
    exp_conf: Dict[str, Any],
    max_length: pytorch.TrainUnit,
) -> None:
    """
    Certain searchers (ASHA and Adaptive ASHA) require configuring the maximum training length in
    the experiment config. We check that the `max_length` passed to `fit()` matches the experiment
    config and log warnings if it doesn't.
    """
    time_metric = exp_conf["searcher"].get("time_metric")
    if time_metric is not None:
        max_time = exp_conf["searcher"].get("max_time")
        assert max_time, "`searcher.max_time` not configured"
        if time_metric == "batches":
            if not isinstance(max_length, pytorch.Batch) or max_length.value != max_time:
                logger.warning(
                    f"`max_length` passed into `fit()` method ({max_length}) does not match "
                    f"`searcher.max_time` and `searcher.time_metric` from the experiment config "
                    f"(Batch(value={max_time})). This may result in unexpected hyperparameter "
                    f"search behavior."
                )
        elif time_metric == "epochs":
            if not isinstance(max_length, pytorch.Epoch) or max_length.value != max_time:
                logger.warning(
                    f"`max_length` passed into `fit()` method ({max_length}) does not match "
                    f"`searcher.max_time` and `searcher.time_metric` from the experiment config "
                    f"(Epoch(value={max_time})). This may result in unexpected hyperparameter "
                    f"search behavior."
                )
        else:
            logger.warning(
                "`searcher.time_metric` must be either 'batches' or 'epochs' "
                f"for training with PyTorchTrials, but got {time_metric}. "
                f"Training will proceed with {max_length} but may result in unexpected behavior."
            )


def _initialize_distributed_backend() -> Optional[core.DistributedContext]:
    info = det.get_cluster_info()
    distributed_backend = det._DistributedBackend()

    if distributed_backend.use_deepspeed():
        # We use an environment variable to allow users to enable custom initialization routine for
        # distributed training since the pre_execute_hook runs before trial initialization.
        manual_dist_init = os.environ.get("DET_MANUAL_INIT_DISTRIBUTED")
        if not manual_dist_init:
            deepspeed.init_distributed(auto_mpi_discovery=False)
        return core.DistributedContext.from_deepspeed()
    elif info and (len(info.container_addrs) > 1 or len(info.slot_ids) > 1):
        raise ValueError(
            "In multi-slot managed cluster training, you must wrap your training script with a "
            "distributed launch layer such as determined.launch.deepspeed."
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
    enable_tensorboard_logging: bool = True,
) -> Iterator[det_ds.DeepSpeedTrialContext]:
    """
    Creates a DeepSpeedTrialContext for use with a DeepSpeedTrial. All trainer.* calls
    must be within the scope of this context because there are resources started in
    __enter__ that must be cleaned up in __exit__.

    Arguments:
        hparams: (Optional) instance of hyperparameters for the trial
        exp_conf: (Optional) for local-training mode. If unset, calling
            context.get_experiment_config() will fail.
        distributed: (Optional) custom distributed training configuration
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
        num_gpus = len(gpu.get_gpu_uuids())
    else:
        assert cluster_info, "Unable to detect cluster info"

        trial_seed = cluster_info.trial.trial_seed
        exp_conf = cluster_info.trial._config
        steps_completed = cluster_info.trial._steps_completed
        num_gpus = len(cluster_info.gpu_uuids)

        _set_random_seeds(trial_seed)

    with core.init(
        distributed=distributed_context,
        preempt_mode=core.PreemptMode.WorkersAskChief,
        tensorboard_mode=core.TensorboardMode.MANUAL,
    ) as core_context:
        context = det_ds.DeepSpeedTrialContext(
            core_context=core_context,
            trial_seed=trial_seed,
            hparams=hparams,
            slots_per_trial=core_context.distributed.get_size(),
            num_gpus=num_gpus,
            exp_conf=exp_conf,
            steps_completed=steps_completed,
            enable_tensorboard_logging=enable_tensorboard_logging,
        )

        yield context
