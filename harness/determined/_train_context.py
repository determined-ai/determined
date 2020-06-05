import abc
import logging
from typing import Any, Callable, Dict, Optional, cast

import determined as det
from determined import horovod


class _TrainContext(metaclass=abc.ABCMeta):
    """
    _TrainContext is the API to query the system about the trial as it's running.
    These methods should be made available to both Native and Trial APIs.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        self.env = env  # type: det.EnvContext
        self.hvd_config = hvd_config  # type: horovod.HorovodContext
        self.distributed = DistributedContext(env, hvd_config)
        self._stop_requested = False

    def get_experiment_config(self) -> Dict[str, Any]:
        """
        Return the experiment configuration.
        """
        return self.env.experiment_config

    def get_data_config(self) -> Dict[str, Any]:
        """
        Return the data configuration.
        """
        return self.get_experiment_config().get("data", {})

    def get_experiment_id(self) -> int:
        """
        Return the experiment ID of the current trial.
        """
        return int(self.env.det_experiment_id)

    def get_global_batch_size(self) -> int:
        """
        Return the global batch size.
        """
        return self.env.global_batch_size

    def get_per_slot_batch_size(self) -> int:
        """
        Return the per-slot batch size. When a model is trained with a single GPU, this is equal to
        the global batch size. When multi-GPU training is used, this is equal to the global batch
        size divided by the number of GPUs used to train the model.
        """
        return self.env.per_slot_batch_size

    def get_trial_id(self) -> int:
        """
        Return the trial ID of the current trial.
        """
        return int(self.env.det_trial_id)

    def get_trial_seed(self) -> int:
        return self.env.trial_seed

    def get_hparams(self) -> Dict[str, Any]:
        """
        Return a dictionary of hyperparameter names to values.
        """
        return self.env.hparams

    def get_hparam(self, name: str) -> Any:
        """
        Return the current value of the hyperparameter with the given name.
        """
        if name not in self.env.hparams:
            raise ValueError(
                "Could not find name '{}' in experiment "
                "hyperparameters. Please check your experiment "
                "configuration 'hyperparameters' section.".format(name)
            )
        if name == "global_batch_size":
            logging.warning(
                "Please use `context.get_per_slot_batch_size()` and "
                "`context.get_global_batch_size()` instead of accessing "
                "`global_batch_size` directly."
            )
        return self.env.hparams[name]

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

        logging.info(
            "A trial stoppage has requested. The trial will be stopped "
            "at the end of the current step."
        )
        self._stop_requested = stop_requested


class TrialContext(_TrainContext):
    """
    A base class that all TrialContexts will inherit from.
    The context passed to the UserTrial.__init__() when we instantiate the user's Trial must
    inherit from this class.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        super().__init__(env, hvd_config)


class NativeContext(_TrainContext):
    """
    A base class that all NativeContexts will inherit when using the Native API.

    The context returned by the init() function must inherit from this class.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        super().__init__(env, hvd_config)
        self._train_fn = None  # type: Optional[Callable[[], None]]

    def _set_train_fn(self, train_fn: Callable[[], None]) -> None:
        self._train_fn = train_fn


class DistributedContext:
    """
    DistributedContext extends all TrialContexts and NativeContexts under
    the ``context.distributed`` namespace. It provides useful methods for
    effective multi-slot (parallel and distributed) training.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        self._env = env
        self._hvd_config = hvd_config

    def get_rank(self) -> int:
        """
        Return the rank of the process in the trial.
        """
        if not self._hvd_config.use:
            return 0

        return cast(int, horovod.hvd.rank())

    def get_local_rank(self) -> int:
        """
        Return the rank of the process on the agent.
        """
        if not self._hvd_config.use:
            return 0

        return cast(int, horovod.hvd.local_rank())

    def get_size(self) -> int:
        """
        Return the number of slots this trial is running on.
        """
        return self._env.experiment_config.slots_per_trial()

    def get_num_agents(self) -> int:
        """
        Return the number of agents this trial is running on.
        """
        if not self._hvd_config.use:
            return 1

        return cast(int, self.get_size() // horovod.hvd.local_size())
