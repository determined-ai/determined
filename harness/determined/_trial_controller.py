import abc
import logging
import pathlib
from typing import Any, Dict, Optional, cast

import determined as det
from determined import horovod, profiler, workload
from determined._rendezvous_info import RendezvousInfo
from determined.common import check
from determined.common.types import StepID
from determined.horovod import hvd


class TrialController(metaclass=abc.ABCMeta):
    """
    Abstract base class for TrialControllers.

    A TrialController is the lowest Determined-owned layer of the harness. It consumes Workloads
    from higher layers of the harness and applies framework-specific logic to execute the
    workloads.  Framework-specific details like tf.Session objects or keras.Model objects are
    handled at this level.

    Because framework APIs vary significantly, there is a wide variation in how TrialControllers
    are implemented. There are presently two major subclasses of TrialControllers:
    CallbackTrialController and LoopTrialController.

    CallbackTrialController is the legacy form of TrialController. It requires
    framework logic to be reentrant and controlled via function calls. It is
    currently only used in the integration test framework.

    LoopTrialController is the newer form of TrialController. It are distinguished by being
    designed to require owning the main control loop in the code, which is a prerequisite for
    using horovod for distributed training.
    """

    def __init__(
        self,
        context: Any,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: RendezvousInfo,
        hvd_config: horovod.HorovodContext,
        prof: profiler.ProfilerAgent,
    ) -> None:
        self.context = context
        self.env = env
        self.workloads = workloads
        self.load_path = load_path
        self.rendezvous_info = rendezvous_info
        self.hvd_config = hvd_config
        self.prof = prof

        self._check_if_trial_supports_configurations(env)

    @staticmethod
    @abc.abstractmethod
    def pre_execute_hook(env: det.EnvContext, hvd_config: horovod.HorovodContext) -> Any:
        """
        Certain things must be initialized before either running user code (in the Native API case)
        or intializing user code (in the Trial API case).
        """
        pass

    @staticmethod
    @abc.abstractmethod
    def from_trial(
        trial_inst: "det.Trial",
        prof: profiler.ProfilerAgent,
        context: det.TrialContext,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: RendezvousInfo,
        hvd_config: horovod.HorovodContext,
    ) -> "TrialController":
        """
        Create a TrialController from an instantiated framework-matched Trial.
        """
        pass

    @staticmethod
    @abc.abstractmethod
    def from_native(
        context: det.NativeContext,
        prof: profiler.ProfilerAgent,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: RendezvousInfo,
        hvd_config: horovod.HorovodContext,
    ) -> "TrialController":
        """
        Create a TrialController from either a generic Experiment object or a framework-matched
        Experiment object.
        """
        pass

    @abc.abstractmethod
    def run(self) -> None:
        """
        The main control loop for executing user code.
        """
        pass

    @staticmethod
    def supports_mixed_precision() -> bool:
        return False

    @staticmethod
    def supports_averaging_training_metrics() -> bool:
        return False

    def initialize_wrapper(self) -> None:
        pass

    def _check_if_trial_supports_configurations(self, env: det.EnvContext) -> None:
        if env.experiment_config.averaging_training_metrics_enabled():
            check.true(self.supports_averaging_training_metrics())


class CallbackTrialController(TrialController):
    """
    Abstract base class for the legacy, callback-based TrialControllers.

    Frameworks should create framework-specific subclasses and implement :func:`train_for_step`,
    :func:`compute_validation_metrics`, :func:`save`, and :func:`load`.
    """

    @staticmethod
    def from_native(*args: Any, **kwargs: Any) -> "TrialController":
        raise NotImplementedError("CallbackTrialControllers do not support the Native API")

    def run(self) -> None:
        """
        A basic control loop of the old-style (callback-based) TrialController
        classes.
        """

        for w, args, response_func in self.workloads:
            try:
                if w.kind == workload.Workload.Kind.RUN_STEP:
                    response = self.train_for_step(
                        w.step_id, w.num_batches
                    )  # type: workload.Response
                elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                    response = self.compute_validation_metrics(w.step_id)
                elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                    check.len_eq(args, 1)
                    check.is_instance(args[0], pathlib.Path)
                    path = cast(pathlib.Path, args[0])
                    self.save(path)
                    response = {}
                elif w.kind == workload.Workload.Kind.TERMINATE:
                    self.terminate()
                    response = workload.Skipped()
                else:
                    raise AssertionError("Unexpected workload: {}".format(w.kind))

            except det.errors.SkipWorkloadException:
                response = workload.Skipped()

            response_func(response)

    # Methods implemented by AF-specific subclasses.
    @abc.abstractmethod
    def train_for_step(self, step_id: StepID, num_batches: int) -> Dict[str, Any]:
        """
        Runs a trial for one step, which should consist of the training
        the model on the given number of batches.  Implemented by frameworks.

        Args:
            step_id: The index of the step to run.  This controls which batches
                to run.
            num_batches: How many batches to run this step.
            batch_loader: The training batch loader instance. Depending on the
                framework implementation, a batch loader may or may not be
                needed.

        Returns:
            The training metrics computed for each batch in the step.
        """
        pass

    @abc.abstractmethod
    def compute_validation_metrics(self, step_id: StepID) -> Dict[str, Any]:
        """
        Computes validation metrics for a trial given the current
        trial state.  Implemented by frameworks.

        Args:
            step_id: The index of the step to run.

            batch_loader: The validation batch loader instance. Depending on
                the framework implementation, a batch loader may or may not be
                needed.

        Returns:
            The validation metrics.
        """
        pass

    @abc.abstractmethod
    def save(self, path: pathlib.Path) -> None:
        """
        Saves the current model state to persistent storage. Implemented by
        frameworks.

        Args:
            path: A directory on the container file system; the trial
                should create the directory and checkpoint its current
                state into one or more files inside that directory. The
                implementation of this function creates `path`; hence,
                it should not exist before this function is called.
        """
        pass

    @abc.abstractmethod
    def load(self, path: pathlib.Path) -> None:
        """
        Loads the current model state from persistent storage. Implemented
        by frameworks.

        Args:
            path: A directory on the container file system.
        """
        pass

    def terminate(self) -> None:
        pass


class LoopTrialController(TrialController):
    def __init__(
        self,
        context: Any,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: RendezvousInfo,
        hvd_config: horovod.HorovodContext,
        prof: profiler.ProfilerAgent,
    ) -> None:
        super().__init__(
            context=context,
            env=env,
            workloads=workloads,
            load_path=load_path,
            rendezvous_info=rendezvous_info,
            hvd_config=hvd_config,
            prof=prof,
        )

        self.batch_size = self.context.get_per_slot_batch_size()
        self.scheduling_unit = self.env.experiment_config.scheduling_unit()

        logging.debug("Starting LoopTrialController initialization.")

        if self.hvd_config.use:
            self.is_chief = hvd.rank() == 0
            rank = hvd.rank()
        else:
            self.is_chief = True
            rank = 0

        if self.hvd_config.use and not self.is_chief:
            log_level = (
                logging.DEBUG if self.env.experiment_config.debug_enabled() else logging.WARNING
            )
            logging.getLogger().setLevel(log_level)

        logging.debug(
            f"TrialController initialized on rank {rank}, using hvd: {self.hvd_config.use}."
        )
