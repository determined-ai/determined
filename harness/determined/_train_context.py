import abc
import logging
from typing import Any, Callable, Dict, Optional, Tuple, cast

import determined as det
from determined import horovod
from determined_common import check


def _calculate_batch_sizes(env: det.EnvContext) -> Tuple[int, int]:
    if "global_batch_size" not in env.hparams.keys():
        raise AssertionError(
            "Please specify `global_batch_size` under `hyperparameters` in experiment config."
        )

    if "batch_size" in env.hparams.keys():
        logging.warning(
            "Use `global_batch_size` not `batch_size` under `hyperparameters` in experiment config."
        )

    global_batch_size = env.hparams["global_batch_size"]
    check.is_instance(global_batch_size, int, "`global_batch_size` hparam must be an int.")
    global_batch_size = cast(int, global_batch_size)

    if env.experiment_config.native_parallel_enabled():
        return global_batch_size, global_batch_size

    # Configure batch sizes.
    slots_per_trial = env.experiment_config.slots_per_trial()
    if global_batch_size < slots_per_trial:
        raise AssertionError(
            "Please set the `global_batch_size` hyperparameter to be greater or equal to the "
            f"number of slots. Current batch_size: {global_batch_size}, slots_per_trial: "
            f"{slots_per_trial}."
        )

    per_gpu_batch_size = global_batch_size // slots_per_trial
    effective_batch_size = per_gpu_batch_size * slots_per_trial
    if effective_batch_size != global_batch_size:
        logging.warning(
            f"`global_batch_size` changed from {global_batch_size} to {effective_batch_size} to "
            f"divide equally across {slots_per_trial} slots."
        )

    return per_gpu_batch_size, effective_batch_size


class _TrainContext(metaclass=abc.ABCMeta):
    """
    _TrainContext is the API to query the system about the trial as it's running.
    These methods should be made available to both Native and Trial APIs.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        self.env = env  # type: det.EnvContext
        self.hvd_config = hvd_config  # type: horovod.HorovodContext
        self.distributed = DistributedContext(env, hvd_config)

        self._per_slot_batch_size, self._global_batch_size = _calculate_batch_sizes(env)

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
        return self._global_batch_size

    def get_per_slot_batch_size(self) -> int:
        """
        Return the per-slot batch size. When a model is trained with a single GPU, this is equal to
        the global batch size. When multi-GPU training is used, this is equal to the global batch
        size divided by the number of GPUs used to train the model.
        """
        return self._per_slot_batch_size

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
        Return the hyperparameter value for the given name.
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
