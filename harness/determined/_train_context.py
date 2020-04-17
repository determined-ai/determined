import abc
import logging
from typing import Any, Callable, Dict, Optional, Union, cast

import determined as det
from determined import data_layer, horovod


class _TrainContext(metaclass=abc.ABCMeta):
    """
    _TrainContext is the API to query the system about the trial as it's running.
    These methods should be made available to both Native and Trial APIs.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        self.env = env  # type: det.EnvContext
        self.hvd_config = hvd_config  # type: horovod.HorovodContext
        self.distributed = DistributedContext(env, hvd_config)

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


class _DataLayerContext:
    def __init__(
        self,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        train_context: Union[NativeContext, TrialContext],
    ) -> None:
        self._training_cacheable = data_layer.CacheableDecorator(
            env=env,
            hvd_config=hvd_config,
            training=True,
            per_slot_batch_size=train_context.get_per_slot_batch_size(),
        )
        self._validation_cacheable = data_layer.CacheableDecorator(
            env=env,
            hvd_config=hvd_config,
            training=False,
            per_slot_batch_size=train_context.get_per_slot_batch_size(),
        )

    def cache_train_dataset(
        self,
        dataset_id: str,
        dataset_version: str,
        shuffle: bool = False,
        skip_shuffle_at_epoch_end: bool = False,
    ) -> Callable:
        """cache_train_dataset is a decorator for creating your training dataset.  It should
        decorate a function that outputs a ``tf.data.Dataset`` object. The dataset will be
        stored in a cache, keyed by ``dataset_id`` and ``dataset_version``. The cache is re-used
        re-used in subsequent calls.

        Args:
            dataset_id: A string that will be used as part of the
                unique identifier for this dataset.

            dataset_version: A string that will be used as part of the
                unique identifier for this dataset.

            shuffle: A bool indicating if the dataset should be shuffled. Shuffling will
                be performed with the trial's random seed which can be set in
                :ref:`experiment-configuration`.

            skip_shuffle_at_epoch_end: A bool indicating if shuffling should be skipped
                at the end of epochs.


        Example Usage:

        .. code-block:: python

            def make_train_dataset(self):
                @self.context.experimental.cache_train_dataset("range_dataset", "v1")
                def make_dataset():
                    ds = tf.data.Dataset.range(10)
                    return ds

                dataset = make_dataset()
                dataset = dataset.batch(self.context.get_per_slot_batch_size())
                dataset = dataset.map(...)
                return dataset

        .. note::
            ``dataset.batch()`` and runtime augmentation should be done after caching.
            Additionally, users should never need to call ``dataset.repeat()``.

        """

        return self._training_cacheable.cache_dataset(
            dataset_id=dataset_id,
            dataset_version=dataset_version,
            shuffle=shuffle,
            skip_shuffle_at_epoch_end=skip_shuffle_at_epoch_end,
        )

    def cache_validation_dataset(
        self, dataset_id: str, dataset_version: str, shuffle: bool = False,
    ) -> Callable:
        """cache_validation_dataset is a decorator for creating your validation dataset.  It should
        decorate a function that outputs a ``tf.data.Dataset`` object. The dataset will be
        stored in a cache, keyed by ``dataset_id`` and ``dataset_version``. The cache is re-used
        re-used in subsequent calls.

        Args:
            dataset_id: A string that will be used as part of the
                unique identifier for this dataset.

            dataset_version: A string that will be used as part of the
                unique identifier for this dataset.

            shuffle: A bool indicating if the dataset should be shuffled. Shuffling will
                be performed with the trial's random seed which can be set in
                :ref:`experiment-configuration`.

        """

        return self._validation_cacheable.cache_dataset(
            dataset_id=dataset_id,
            dataset_version=dataset_version,
            shuffle=shuffle,
            skip_shuffle_at_epoch_end=True,
        )

    def get_train_cacheable(self) -> data_layer.CacheableDecorator:
        return self._training_cacheable

    def get_validation_cacheable(self) -> data_layer.CacheableDecorator:
        return self._validation_cacheable
