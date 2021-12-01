import abc
import logging
from typing import Any, Optional, Type

import determined as det
from determined import horovod, profiler, workload
from determined.common import check
from determined.horovod import hvd


class TrialController(metaclass=abc.ABCMeta):
    """
    TrialController is the legacy class that represented the Determined-owned logic to interact with
    a user-owned Trial class.
    """

    def __init__(
        self,
        context: Any,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        workloads: Optional[workload.Stream] = None,
    ) -> None:
        self.context = context
        self.env = env
        self.hvd_config = hvd_config
        # The only time that workloads should be non-None here is unit tests or test mode.
        self.workloads = workloads

        self.prof = profiler.ProfilerAgent.from_env(
            env,
            context.distributed.cross_rank,
            context.distributed.rank,
        )

        self._check_if_trial_supports_configurations(env)

        self.batch_size = self.context.get_per_slot_batch_size()
        self.scheduling_unit = self.env.experiment_config.scheduling_unit()

        if self.hvd_config.use:
            self.is_chief = hvd.rank() == 0
        else:
            self.is_chief = True

        if self.hvd_config.use and not self.is_chief:
            log_level = (
                logging.DEBUG if self.env.experiment_config.debug_enabled() else logging.WARNING
            )
            logging.getLogger().setLevel(log_level)

    @classmethod
    @abc.abstractmethod
    def pre_execute_hook(
        cls: Type["TrialController"], env: det.EnvContext, hvd_config: horovod.HorovodContext
    ) -> Any:
        """
        Certain things must be initialized before either running user code (in the Native API case)
        or intializing user code (in the Trial API case).
        """
        pass

    @classmethod
    @abc.abstractmethod
    def from_trial(
        cls: Type["TrialController"],
        trial_inst: "det.Trial",
        context: det.TrialContext,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
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

    @classmethod
    def supports_averaging_training_metrics(cls: Type["TrialController"]) -> bool:
        return False

    def initialize_wrapper(self) -> None:
        pass

    def _check_if_trial_supports_configurations(self, env: det.EnvContext) -> None:
        if env.experiment_config.averaging_training_metrics_enabled():
            check.true(self.supports_averaging_training_metrics())

    def close(self) -> None:
        self.context.close()
