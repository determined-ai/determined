import abc
import logging
import os
from typing import Any, Optional, Type

import determined as det
from determined import profiler, tensorboard, workload
from determined.common import api


class _DistributedBackend:
    """
    _DistributedBackend contains the supported backends for distributed training. These constants
    are read from environment variables to determine which backends are in use.
    """

    HOROVOD = "USE_HOROVOD"
    DEEPSPEED = "USE_DEEPSPEED"
    TORCH = "USE_TORCH_DISTRIBUTED"

    def use_horovod(self) -> bool:
        return bool(os.environ.get(self.HOROVOD, None))

    def use_torch(self) -> bool:
        return bool(os.environ.get(self.TORCH, None))

    def use_deepspeed(self) -> bool:
        return bool(os.environ.get(self.DEEPSPEED, None))


def _profiler_agent_from_env(
    session: api.Session, env: det.EnvContext, global_rank: int, local_rank: int
) -> profiler.ProfilerAgent:
    """
    This used to be ProfilerAgent.from_env(), but it was demoted to being a helper function here.

    The purpose of demoting it is isolating the EnvContext object to the smallest footprint
    possible.  As EnvContext was part of the legacy Trial-centric harness architecture, and as this
    functionality was only required in this legacy file, this is a good home for it.
    """

    begin_on_batch, end_after_batch = env.experiment_config.profiling_interval()
    return profiler.ProfilerAgent(
        session=session,
        trial_id=env.det_trial_id,
        agent_id=env.det_agent_id,
        profiling_is_enabled=env.experiment_config.profiling_enabled(),
        global_rank=global_rank,
        local_rank=local_rank,
        begin_on_batch=begin_on_batch,
        end_after_batch=end_after_batch,
        sync_timings=env.experiment_config.profiling_sync_timings(),
    )


class TrialController(metaclass=abc.ABCMeta):
    """
    TrialController is the legacy class that represented the Determined-owned logic to interact with
    a user-owned Trial class.
    """

    def __init__(
        self,
        context: Any,
        env: det.EnvContext,
        workloads: Optional[workload.Stream] = None,
    ) -> None:
        self.context = context
        self.env = env
        # The only time that workloads should be non-None here is unit tests or test mode.
        self.workloads = workloads

        if hasattr(context._core.train, "_session"):
            sess = context._core.train._session
            self.prof = _profiler_agent_from_env(
                sess, env, context.distributed.rank, context.distributed.local_rank
            )
        else:
            self.prof = profiler.DummyProfilerAgent()

        distributed_backend = _DistributedBackend()
        self.use_horovod = distributed_backend.use_horovod()
        self.use_torch = distributed_backend.use_torch()

        self.scheduling_unit = self.env.experiment_config.scheduling_unit()

        self.is_chief = context.distributed.rank == 0

        if context.distributed.size > 1 and not self.is_chief:
            log_level = (
                logging.DEBUG if self.env.experiment_config.debug_enabled() else logging.WARNING
            )
            logging.getLogger().setLevel(log_level)

    @classmethod
    @abc.abstractmethod
    def pre_execute_hook(
        cls: Type["TrialController"], env: det.EnvContext, distributed_backend: _DistributedBackend
    ) -> Any:
        """
        Certain things must be initialized before either running user code (in the Native API case)
        or initializing user code (in the Trial API case).
        """
        pass

    @classmethod
    @abc.abstractmethod
    def from_trial(
        cls: Type["TrialController"],
        trial_inst: "det.LegacyTrial",
        context: det.TrialContext,
        env: det.EnvContext,
        workloads: Optional[workload.Stream] = None,
    ) -> "TrialController":
        """
        Create a TrialController from an instantiated framework-matched Trial.
        """
        pass

    @abc.abstractmethod
    def run(self) -> None:
        """
        The main control loop for executing user code.
        """
        pass

    @classmethod
    def supports_mixed_precision(cls: Type["TrialController"]) -> bool:
        return False

    def close(self) -> None:
        self.context.close()

    def upload_tb_files(self) -> None:
        self.context._core.train.upload_tensorboard_files(
            (lambda _: True) if self.is_chief else (lambda p: not p.match("*tfevents*")),
            tensorboard.util.get_rank_aware_path,
        )
