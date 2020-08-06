import abc
from typing import Iterator, Optional, Tuple, Union, cast

import tensorflow as tf

from determined import keras
from determined_common import check


class _TrainingInputManager(metaclass=abc.ABCMeta):
    """
    Base class for managing validation input data and the metadata related to it for tf.keras trial.
    """

    def __init__(
        self, context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext]
    ) -> None:
        self._context = context

    @abc.abstractmethod
    def get_initial_epoch(self) -> int:
        pass

    @abc.abstractmethod
    def get_training_input_and_batches_per_epoch(
        self,
    ) -> Tuple[Union[Iterator, tf.data.Dataset], int]:
        pass


class _ValidationInputManager(metaclass=abc.ABCMeta):
    """
    Base class for managing training input data and the metadata related to it for tf.keras trial.
    """

    def __init__(
        self, context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext]
    ) -> None:
        self._context = context

    @abc.abstractmethod
    def get_validation_input_and_num_batches(
        self,
    ) -> Tuple[Union[Iterator, tf.data.Dataset], Optional[int]]:
        pass

    @abc.abstractmethod
    def stop_validation_input_and_get_num_inputs(self) -> Optional[int]:
        pass


class _TrainingDataLayerTFDatasetManager(_TrainingInputManager):
    def __init__(
        self,
        context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext],
        train_config: keras.TFKerasTrainConfig,
    ) -> None:
        super().__init__(context=context)

        self._training_cacheable = self._context.experimental.get_train_cacheable()
        self._training_dataset = train_config.training_data

        check.true(
            self._training_cacheable.is_decorator_used(),
            "Please use `@context.experimental.cache_train_dataset(dataset_name, dataset_version)`"
            " for the training dataset.",
        )
        check.false(
            self._context.dataset_initialized,
            "Please do not use: `context.wrap_dataset(dataset)` if using "
            "`@context.experimental.cache_train_dataset()` and "
            "`@context.experimental.cache_validation_dataset()`.",
        )
        check.is_instance(
            train_config.training_data,
            tf.data.Dataset,
            "Pass in a `tf.data.Dataset` object if using "
            "`@context.experimental.cache_train_dataset()`.",
        )

    def get_initial_epoch(self) -> int:
        batches_seen = self._context.env.initial_workload.total_batches_processed
        inputs_seen = batches_seen * self._context.get_per_slot_batch_size()
        initial_epoch = inputs_seen // self._training_cacheable.get_dataset_length()

        return initial_epoch

    def get_training_input_and_batches_per_epoch(self) -> Tuple[tf.data.Dataset, int]:
        # Make sure we never run out of training data.
        self._training_dataset = self._training_dataset.repeat()  # type: ignore

        # TODO: When using a tf.dataset and setting `steps_per_epoch` to None
        # tf.keras determines the length of the dataset based on the first epoch
        # which means that we can not rely on tf.dataset to signal the ends of epochs
        # when resuming mid-epoch. Additionally if using tf.dataset.Data.from_generator(),
        # when setting the `steps_per_epoch` to None, we must not set
        # `inter_op_parallelism_threads` to 1 for TF 1.* because this causes a hang after
        # the second epoch if the length of the dataset is divisible by the batch size.
        steps_per_epoch = (
            self._training_cacheable.get_dataset_length() // self._context.get_per_slot_batch_size()
        )
        return self._training_dataset, steps_per_epoch


class _ValidationDataLayerTFDatasetManager(_ValidationInputManager):
    def __init__(
        self,
        context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext],
        train_config: keras.TFKerasTrainConfig,
    ) -> None:
        super().__init__(context=context)

        self._validation_cacheable = self._context.experimental.get_validation_cacheable()
        self._validation_dataset = train_config.validation_data

        check.true(
            self._validation_cacheable.is_decorator_used(),
            "Please use `@context.experimental.cache_validation_dataset(dataset_name, "
            "dataset_version)` for the validation dataset.",
        )
        check.false(
            self._context.dataset_initialized,
            "Please do not use: `context.wrap_dataset(dataset)` if using "
            "`@context.experimental.cache_train_dataset()` and "
            "`@context.experimental.cache_validation_dataset()`.",
        )
        check.is_instance(
            train_config.validation_data,
            tf.data.Dataset,
            "Pass in a `tf.data.Dataset` object if using "
            "`@context.experimental.cache_validation_dataset()`.",
        )

    def get_validation_input_and_num_batches(self) -> Tuple[tf.data.Dataset, Optional[int]]:
        return self._validation_dataset, None

    def stop_validation_input_and_get_num_inputs(self) -> int:
        # If the validation dataset is not evenly split amongst the ranks,
        # the num_inputs is rounded up.
        return (
            self._validation_cacheable.get_dataset_length() * self._context.distributed.get_size()
        )


class _TrainingTFDatasetManager(_TrainingInputManager):
    def __init__(
        self,
        context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext],
        train_config: keras.TFKerasTrainConfig,
    ) -> None:
        super().__init__(context=context)

        check.true(
            self._context.dataset_initialized,
            "Please use: `context.wrap_dataset(dataset)` if using `tf.data.Dataset`.",
        )

        self._training_dataset = train_config.training_data

    def get_initial_epoch(self) -> int:
        return 0

    def get_training_input_and_batches_per_epoch(self) -> Tuple[tf.data.Dataset, int]:
        # Tensorflow dataset doesn't provide length api so use the configured scheduling_unit.
        steps_per_epoch = self._context.env.experiment_config.scheduling_unit()
        return self._training_dataset.repeat(), steps_per_epoch  # type: ignore


class _ValidationTFDatasetManager(_ValidationInputManager):
    def __init__(
        self,
        context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext],
        train_config: keras.TFKerasTrainConfig,
    ) -> None:
        super().__init__(context=context)

        check.true(
            self._context.dataset_initialized,
            "Please use: `context.wrap_dataset(dataset)` if using `tf.data.Dataset`.",
        )

        self._validation_dataset = train_config.validation_data

    def get_validation_input_and_num_batches(self) -> Tuple[tf.data.Dataset, Optional[int]]:
        return self._validation_dataset, None

    def stop_validation_input_and_get_num_inputs(self) -> Optional[int]:
        return None


class _TrainingSequenceAdapterManager(_TrainingInputManager):
    def __init__(
        self,
        context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext],
        train_config: keras.TFKerasTrainConfig,
    ) -> None:
        super().__init__(context=context)

        self._training_sequence_adapter = cast(keras.SequenceAdapter, train_config.training_data)
        self._training_iterator = None  # type: Optional[Iterator]

    def get_initial_epoch(self) -> int:
        batches_seen = self._context.env.initial_workload.total_batches_processed
        initial_epoch = batches_seen // len(self._training_sequence_adapter)

        return initial_epoch

    def get_training_input_and_batches_per_epoch(self) -> Tuple[Iterator, int]:
        training_iterator_offset = self._context.env.initial_workload.total_batches_processed

        sequence_adapter = self._training_sequence_adapter
        if self._context.hvd_config.use:
            # When using horovod each worker starts at a unique offset
            # so that all workers are processing unique data on each step.
            batch_rank_offset = (
                len(sequence_adapter) // self._context.distributed.get_size()
            ) * self._context.distributed.get_rank()
            training_iterator_offset += batch_rank_offset

        sequence_adapter.start(batch_offset=training_iterator_offset)
        self._training_iterator = sequence_adapter.get_iterator()

        steps_per_epoch = len(sequence_adapter) // self._context.distributed.get_size()

        return self._training_iterator, steps_per_epoch


class _ValidationSequenceAdapterManager(_ValidationInputManager):
    def __init__(
        self,
        context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext],
        train_config: keras.TFKerasTrainConfig,
    ) -> None:
        super().__init__(context=context)

        self._validation_sequence_adapter = cast(
            keras.SequenceAdapter, train_config.validation_data
        )

    def get_validation_input_and_num_batches(self) -> Tuple[Iterator, int]:
        sequence_adapter = self._validation_sequence_adapter
        val_iterator_offset = 0
        val_num_batches = len(sequence_adapter)

        if self._context.hvd_config.use:
            num_slots = self._context.distributed.get_size()
            rank = self._context.distributed.get_rank()

            leftover_validation_batches = len(sequence_adapter) % num_slots
            val_num_batches = len(sequence_adapter) // num_slots
            val_iterator_offset = val_num_batches * rank + min(leftover_validation_batches, rank)
            if rank < leftover_validation_batches:
                val_num_batches += 1

        sequence_adapter.start(batch_offset=val_iterator_offset, is_validation=True)
        validation_iterator = sequence_adapter.get_iterator()

        return validation_iterator, val_num_batches

    def stop_validation_input_and_get_num_inputs(self) -> Optional[int]:
        sequence_adapter = self._validation_sequence_adapter
        sequence_adapter.stop()

        # TODO: This does not account for incomplete batches.
        return len(sequence_adapter) * self._context.get_per_slot_batch_size()


def _init_input_managers(
    context: Union[keras.TFKerasTrialContext, keras.TFKerasNativeContext],
    train_config: keras.TFKerasTrainConfig,
) -> Tuple[_TrainingInputManager, _ValidationInputManager]:

    training_input_manager = None  # type: Optional[_TrainingInputManager]
    training_cacheable = context.experimental.get_train_cacheable()
    if training_cacheable.is_decorator_used():
        training_input_manager = _TrainingDataLayerTFDatasetManager(
            context=context, train_config=train_config
        )

    elif isinstance(train_config.training_data, tf.data.Dataset):
        training_input_manager = _TrainingTFDatasetManager(
            context=context, train_config=train_config
        )

    else:
        training_input_manager = _TrainingSequenceAdapterManager(
            context=context, train_config=train_config
        )

    validation_input_manager = None  # type: Optional[_ValidationInputManager]
    validation_cacheable = context.experimental.get_validation_cacheable()
    if validation_cacheable.is_decorator_used():
        validation_input_manager = _ValidationDataLayerTFDatasetManager(
            context=context, train_config=train_config
        )

    elif isinstance(train_config.validation_data, tf.data.Dataset):
        validation_input_manager = _ValidationTFDatasetManager(
            context=context, train_config=train_config
        )

    else:
        validation_input_manager = _ValidationSequenceAdapterManager(
            context=context, train_config=train_config
        )

    return training_input_manager, validation_input_manager
