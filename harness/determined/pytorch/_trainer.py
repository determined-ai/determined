import contextlib
import logging
import random
import sys
from typing import Any, Dict, Iterator, Optional

import numpy as np
import torch
import torch.distributed as dist

import determined as det
from determined import core, gpu, horovod, profiler, pytorch
from determined.horovod import hvd


class Trainer:
    def __init__(self, trial: pytorch.PyTorchTrial, context: pytorch.PyTorchTrialContext):
        self._trial = trial
        self._context = context
        self._core = self._context._core
        self._distributed_backend = det._DistributedBackend()
        self._det_profiler = None  # type: Optional[profiler.ProfilerAgent]
        self._info = det.get_cluster_info()
        self._local_training = self._info is None or self._info.task_type != "TRIAL"

    def configure_profiler(
        self, sync_timings: bool, enabled: bool, begin_on_batch: int, end_after_batch: int
    ) -> None:
        assert self._info, "Determined profiler must be run on cluster"
        self._det_profiler = profiler.ProfilerAgent(
            trial_id=str(self._info.trial.trial_id),
            agent_id=self._info.agent_id,
            master_url=self._info.master_url,
            profiling_is_enabled=enabled,
            global_rank=self._core.distributed.get_rank(),
            local_rank=self._core.distributed.get_local_rank(),
            begin_on_batch=begin_on_batch,
            end_after_batch=end_after_batch,
            sync_timings=sync_timings,
        )

    def fit(
        self,
        checkpoint_period: Optional[pytorch.TrainUnit] = None,
        validation_period: Optional[pytorch.TrainUnit] = None,
        max_length: Optional[pytorch.TrainUnit] = None,
        reporting_period: Optional[pytorch.TrainUnit] = None,
        aggregation_frequency: Optional[int] = None,
        checkpoint_policy: Optional[str] = None,
        test_mode: Optional[bool] = None,
    ) -> None:

        # Set context and training variables
        if aggregation_frequency:
            self._context._aggregation_frequency = aggregation_frequency

        # Set defaults
        checkpoint_policy = checkpoint_policy or "best"
        checkpoint_period = checkpoint_period or pytorch.Batch(sys.maxsize)
        validation_period = validation_period or pytorch.Batch(sys.maxsize)
        test_mode = test_mode or False

        if self._local_training:
            if checkpoint_policy == "best":
                logging.warning(
                    "checkpoint_policy='best' is not supported in local training mode. "
                    "Falling back to 'all'"
                )
                checkpoint_policy = "all"
            assert max_length, "max_length must be defined in local training mode"
            assert isinstance(
                max_length.value, int
            ), "max_length must be configured in TrainUnit(int)"

            if self._det_profiler:
                logging.warning("Determined profiler will be ignored in local training mode")

            latest_checkpoint = None
            smaller_is_better = True
            searcher_metric_name = None
            steps_completed = 0
            reporting_period = reporting_period or pytorch.Batch(sys.maxsize)
            step_zero_validation = False
            global_batch_size = None
        else:

            assert not test_mode, "test_mode is only supported in local training mode"
            assert self._info, "Unable to detect cluster info"
            if max_length is not None:
                logging.warning(
                    "max_length is ignored when training on-cluster. Please configure the searcher instead"
                )

            latest_checkpoint = self._info.latest_checkpoint
            smaller_is_better = bool(self._info.trial._config["searcher"]["smaller_is_better"])

            searcher_metric_name = self._info.trial._config["searcher"]["metric"]
            steps_completed = int(self._info.trial._steps_completed)
            reporting_period = reporting_period or pytorch.Batch(
                int(self._info.trial._config["scheduling_unit"])
            )
            step_zero_validation = bool(self._info.trial._config["perform_initial_validation"])
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
            det_profiler=self._det_profiler,
            global_batch_size=global_batch_size,
        )

        trial_controller.run()


def _initialize_distributed_backend() -> Optional[core.DistributedContext]:
    info = det.get_cluster_info()

    distributed_backend = det._DistributedBackend()
    if distributed_backend.use_horovod():
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
            "determined.launch.horovod"
        )
    return None


def _set_random_seeds(seed: int) -> None:
    # Set identical random seeds on all training processes.
    # When using horovod, each worker will start at a unique
    # offset in the dataset, ensuring that it is processing a unique
    # training batch.
    random.seed(seed)
    np.random.seed(seed)
    torch.random.manual_seed(seed)


def _generate_local_seed() -> int:
    return random.randint(0, 1 << 31)


@contextlib.contextmanager
def init(
    *,
    hparams: Optional[Dict] = None,
    exp_conf: Optional[Dict[str, Any]] = None,
    distributed: Optional[core.DistributedContext] = None
) -> pytorch.PyTorchTrialContext:
    cluster_info = det.get_cluster_info()
    local_training = cluster_info is None or cluster_info.task_type != "TRIAL"

    # Pre-execute steps: initialize distributed backend and random seeds
    distributed_context = distributed

    if not local_training:
        distributed_context = _initialize_distributed_backend()

    # Initialize default values
    if local_training:
        trial_seed = _generate_local_seed()

        # XXX: figure out if better way to handle this
        aggregation_frequency = exp_conf and int(exp_conf.get("optimizations", {}).get("aggregation_frequency", 1))  # type: ignore
        fp16_compression = False
        average_aggregated_gradients = True
        steps_completed = 0
        managed_training = True
        debug_enabled = False
        num_gpus = len(gpu.get_gpu_uuids())
    else:
        assert cluster_info, "Unable to detect cluster info"

        trial_seed = cluster_info.trial.trial_seed
        exp_conf = cluster_info.trial._config
        aggregation_frequency = int(exp_conf["optimizations"]["aggregation_frequency"])
        fp16_compression = bool(exp_conf["optimizations"]["gradient_compression"])
        average_aggregated_gradients = bool(
            exp_conf["optimizations"]["average_aggregated_gradients"]
        )
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
        )

        # Set here for backwards-compatibility: future code-paths should call wrap_optimizer
        context._set_gradient_compression(fp16_compression)
        context._set_average_aggregated_gradients(average_aggregated_gradients)

        yield context
