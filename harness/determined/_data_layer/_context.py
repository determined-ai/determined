from typing import Callable

import determined as det
from determined import _data_layer, horovod


class DataLayerContext:
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        self._training_cacheable = _data_layer._CacheableDecorator(
            env=env,
            hvd_config=hvd_config,
            training=True,
            per_slot_batch_size=env.per_slot_batch_size,
        )
        self._validation_cacheable = _data_layer._CacheableDecorator(
            env=env,
            hvd_config=hvd_config,
            training=False,
            per_slot_batch_size=env.per_slot_batch_size,
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
        stored in a cache, keyed by ``dataset_id`` and ``dataset_version``. The cache is
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
        stored in a cache, keyed by ``dataset_id`` and ``dataset_version``. The cache is
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

    def get_train_cacheable(self) -> _data_layer._CacheableDecorator:
        return self._training_cacheable

    def get_validation_cacheable(self) -> _data_layer._CacheableDecorator:
        return self._validation_cacheable
