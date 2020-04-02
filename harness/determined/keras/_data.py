import math
from typing import Any, Dict, Iterator, List, Optional, Tuple, Union

import numpy as np
import tensorflow
from packaging import version

import determined as det
from determined_common import check

# Handle TensorFlow compatibility issues.
if version.parse(tensorflow.__version__) >= version.parse("1.14.0"):
    import tensorflow.compat.v1 as tf
else:
    import tensorflow as tf

ArrayLike = Union[np.ndarray, List[np.ndarray], Dict[str, np.ndarray]]


def _is_list_of_numpy_array(x: Any) -> bool:
    return isinstance(x, (list, tuple)) and all(isinstance(v, np.ndarray) for v in x)


def _is_dict_of_numpy_array(x: Any) -> bool:
    return isinstance(x, dict) and all(isinstance(x[k], np.ndarray) for k in x)


def _length_of_multi_arraylike(data: ArrayLike) -> int:
    if isinstance(data, np.ndarray):
        return len(data)
    elif isinstance(data, (list, tuple)):
        return len(data[0])
    elif isinstance(data, dict):
        return len(list(data.values())[0])
    else:
        raise det.errors.InternalException(f"Unsupported data type: {type(data)}.")


def _get_elements_in_multi_arraylike(data: ArrayLike, start: int, end: int) -> Any:
    if isinstance(data, np.ndarray):
        return data[start:end]
    elif isinstance(data, (list, tuple)):
        return [arraylike[start:end] for arraylike in data]
    elif isinstance(data, dict):
        return {name: data[name][start:end] for name in data}
    else:
        raise det.errors.InternalException(f"Unsupported data type: {type(data)}.")


class _ArrayLikeAdapter(tf.keras.utils.Sequence):  # type: ignore
    """This adapter adapts np.ndarray, a list of np.ndarray, and a dict of
    np.ndarray into a tf.keras.utils.Sequence instance.
    """

    def __init__(
        self,
        x: ArrayLike,
        y: ArrayLike,
        batch_size: int,
        sample_weight: Optional[np.ndarray] = None,
        drop_leftovers: bool = False,
    ):
        self._x_length = _length_of_multi_arraylike(x)
        self._y_length = _length_of_multi_arraylike(y)

        check.eq(self._x_length, self._y_length, "Length of x and y do not match.")
        check.check_gt_eq(self._x_length, batch_size, "Batch size is too large for the input data.")
        if sample_weight is not None:
            check.eq(
                self._x_length,
                len(sample_weight),
                "Lengths of input data and sample weights do not match.",
            )

        self.x = x
        self.y = y
        self.sample_weight = sample_weight

        self.batch_size = batch_size
        self.drop_leftovers = drop_leftovers

    def __len__(self) -> int:
        # Returns number of batches (keeps last partial batch).
        if self.drop_leftovers:
            return math.floor(self._x_length / self.batch_size)
        else:
            return math.ceil(self._x_length / self.batch_size)

    def __getitem__(
        self, index: int
    ) -> Union[Tuple[ArrayLike, ArrayLike], Tuple[ArrayLike, ArrayLike, np.ndarray]]:
        # Gets batch at position index.
        start = index * self.batch_size
        # The end is not `(index + 1) * self.batch_size` if the
        # last batch is not a full `self.batch_size`
        end = min((index + 1) * self.batch_size, self._x_length)

        if self.sample_weight is None:
            return (
                _get_elements_in_multi_arraylike(self.x, start, end),
                _get_elements_in_multi_arraylike(self.y, start, end),
            )
        else:
            return (
                _get_elements_in_multi_arraylike(self.x, start, end),
                _get_elements_in_multi_arraylike(self.y, start, end),
                self.sample_weight[start:end],
            )


class _SequenceWithOffset(tf.keras.utils.Sequence):  # type: ignore
    def __init__(self, sequence: tf.keras.utils.Sequence, batch_offset: int = 0):
        self._sequence = sequence
        self._batch_offset = batch_offset

    def __len__(self):  # type: ignore
        return len(self._sequence)

    def __getitem__(self, index):  # type: ignore
        index = (index + self._batch_offset) % len(self)
        return self._sequence[index]


class _SequenceAdapter:
    """
    A class to assist with restoring and saving iterators for a dataset.
    """

    def __init__(
        self,
        data: tf.keras.utils.Sequence,
        use_multiprocessing: bool = False,
        workers: int = 1,
        max_queue_size: int = 10,
    ):
        """
        Multiprocessing or multithreading for native Python generators is not supported.
        If you want these performance accelerations, please consider using a Sequence.

        Args:
            sequence: A tf.keras.utils.Sequence that holds the data.
            use_multiprocessing: If True, use process-based threading. If unspecified,
                `use_multiprocessing` will default to False. Note that because this implementation
                relies on multiprocessing, you should not pass non-picklable arguments for the
                data loaders as they can't be passed easily to children processes.
            workers: Maximum number of processes to spin up when using process-based threading.
                If unspecified, workers will default to 1. If 0, will execute the data loading on
                the main thread.
            max_queue_size: Maximum size for the generator queue. If unspecified, `max_queue_size`
                will default to 10.
        """
        self._max_queue_size = max_queue_size
        if not len(data):
            raise ValueError("tf.keras.utils.Sequence objects should have a non-zero length.")
        self._sequence = _SequenceWithOffset(data)
        self._use_multiprocessing = use_multiprocessing
        self._workers = workers

    def __len__(self) -> int:
        return len(self._sequence)

    def start(self, batch_offset: int = 0, is_validation: bool = False) -> None:
        """
        Sets a batch offset and starts the pre-processing of data.

        Pre-processing of data only happens if workers >0. If the underlying data type is an
        iterator, we are unable to set a batch_offset.

        Args:
            batch_offset: Batch number to start at.
            is_validation: Whether this iterator will be used for validation. This is necessary
                because `get_iterator` usually returns an infinite iterator. When `is_validation`
                is True, the iterator stops at the end of the epoch.
        """
        self._is_validation = is_validation
        self._sequence._batch_offset = batch_offset
        if self._workers > 0:
            self._enqueuer = tf.keras.utils.OrderedEnqueuer(
                self._sequence, use_multiprocessing=self._use_multiprocessing
            )
            self._enqueuer.start(workers=self._workers, max_queue_size=self._max_queue_size)

    def get_iterator(self) -> Iterator:
        """
        Gets an Iterator over the data.

        `start` must be called prior to calling this function"
        """

        def _make_finite(iterator: Iterator, num_steps: int) -> Iterator:
            for _ in range(num_steps):
                yield next(iterator)

        def _iter_sequence_infinite(sequence: tf.keras.utils.Sequence) -> Iterator:
            while True:
                yield from sequence

        if self._is_validation:
            if self._workers > 0:
                iterator = self._enqueuer.get()
                return _make_finite(iterator, len(self._sequence))
            return iter(self._sequence)

        if self._workers > 0:
            return self._enqueuer.get()  # type: ignore
        return _iter_sequence_infinite(self._sequence)

    def stop(self, timeout: Optional[int] = None) -> None:
        """
        Stops processing the data.

        If workers is >0, this will stop any related threads and processes.
        Otherwise this is a no-op.

        Args:
            timeout: Maximum time to wait.
        """
        if self._workers > 0:
            self._enqueuer.stop(timeout=timeout)


class _TFDatasetAdapter:
    """
    A class to assist with restoring and saving iterators for a dataset.
    """

    def __init__(self, dataset: tf.data.Dataset, prefetch_buffer: int = 1) -> None:
        self.dataset = dataset
        self.prefetch_buffer = prefetch_buffer

    def get_iterator(self, repeat: bool = False) -> tf.data.Iterator:
        """
        Return a tf.data.Iterator

        Arguments:
            repeat:
                Indicate if dataset should be pre-transformed with a repeat().
        """
        temp = self.dataset
        if repeat:
            # Having an extra repeat should be ok, so we don't need to check if
            # the dataset already has one.
            temp = temp.repeat()

        if self.prefetch_buffer > 0:
            temp = temp.prefetch(self.prefetch_buffer)

        return temp.make_one_shot_iterator()

    def save_iterator(
        self, iterator: tf.data.Iterator, save_path: str, save_session: tf.Session
    ) -> None:
        """
        Save an iterator to a checkpoint.

        Arguments:
            iterator:
                The iterator to be saved.
            save_path:
                The path to a checkpoint used for restoring an iterator.
            save_session:
                The TensorFlow session which should be used for restoring an
                iterator from a checkpoint.
        """
        saveable = tf.data.experimental.make_saveable_from_iterator(iterator)
        saver = tf.train.Saver({"iterator": saveable})
        saver.save(save_session, save_path)

    def restore_iterator(
        self,
        iterator: tf.data.Iterator,
        restore_path: str,
        restore_session: tf.Session,
        run_options: tf.RunOptions = None,
    ) -> tf.data.Iterator:
        """
        Restore an iterator from a checkpoint.

        Arguments:
            iterator:
                The iterator to be restored.
            restore_path:
                The path to a checkpoint used for restoring an iterator.
            restore_session:
                The TensorFlow session which should be used for restoring an
                iterator from a checkpoint.
            run_options:
                The tf.RunOptions to pass to the tf.Session during
                tf.Saver.restore().
        """
        saveable = tf.data.experimental.make_saveable_from_iterator(iterator)
        restorer = tf.train.Saver({"iterator": saveable})
        restorer.restore(restore_session, restore_path, options=run_options)
        return iterator


InputData = Union[tf.keras.utils.Sequence, tf.data.Dataset, _TFDatasetAdapter, _SequenceAdapter]


def adapt_keras_data(
    x: Any,
    y: Any = None,
    sample_weight: Optional[np.ndarray] = None,
    batch_size: Optional[int] = None,
    use_multiprocessing: bool = False,
    workers: int = 1,
    max_queue_size: int = 10,
    drop_leftovers: bool = False,
) -> InputData:
    """adapt_keras_data adapts input and target data to a _SequenceAdapter or a
    _TFDatasetAdapter, both of which are designed to support random access efficiently,
    for the purpose of supporting Determined managed training loop.

    Multiprocessing or multithreading for native Python generators is not supported.
    If you want these performance accelerations, please consider using a Sequence as x.

    Args:
        x: Input data. It could be:
            1) A Numpy array (or array-like), or a list of arrays (in case the model
            has multiple inputs).
            2) A dict mapping input names to the corresponding array, if the model
            has named inputs.
            3) A tf.data dataset. Should return a tuple of either (inputs, targets) or
            (inputs, targets, sample_weights).
            4) A keras.utils.Sequence returning (inputs, targets) or (inputs, targets,
            sample weights).

        y: Target data. Like the input data x, it could be either Numpy array(s).
            If x is a dataset or keras.utils.Sequence instance, y should not be specified
            (since targets will be obtained from x).

        use_multiprocessing: If True, use process-based threading. If unspecified,
            `use_multiprocessing` will default to False. Note that because this implementation
            relies on multiprocessing, you should not pass non-picklable arguments for the
            data loaders as they can't be passed easily to children processes. This argument
            is ignored if x is a Dataset.

        workers: Maximum number of processes to spin up when using process-based threading.
            If unspecified, workers will default to 1. If 0, will execute the data loading on
            the main thread. This argument is ignored if x is a Dataset.

        max_queue_size: Maximum size for the generator queue. If unspecified, `max_queue_size`
            will default to 10. This argument is ignored if x is a Dataset.

        drop_leftovers: If True, drop the data that cannot complete the last batch. This
            argument is ignored if x is a Sequence or a Dataset.
    """

    def check_y_is_none(y: Any) -> None:
        if y is not None:
            raise det.errors.InvalidDataTypeException(
                type(y),
                "If x is a keras.utils.Sequence or a tf.data.Dataset, "
                "y should not be specified (since targets will be obtained from x)."
                f"See the instruction below for details: \n{adapt_keras_data.__doc__}",
            )

    if isinstance(x, np.ndarray) or _is_list_of_numpy_array(x) or _is_dict_of_numpy_array(x):
        if not (
            (isinstance(y, np.ndarray) or _is_list_of_numpy_array(y))
            and isinstance(batch_size, int)
        ):
            raise det.errors.InvalidDataTypeException(
                type(y),
                "If x is a numpy array or list/dict of numpy arrays, "
                "y must also be a numpy array. "
                f"See the instruction below for details: \n{adapt_keras_data.__doc__}",
            )
        return _SequenceAdapter(
            _ArrayLikeAdapter(x, y, batch_size, sample_weight, drop_leftovers),
            use_multiprocessing,
            workers,
            max_queue_size,
        )

    elif isinstance(x, tf.keras.utils.Sequence):
        check_y_is_none(y)
        return _SequenceAdapter(x, use_multiprocessing, workers, max_queue_size)

    elif isinstance(x, tf.data.Dataset):
        check_y_is_none(y)
        return _TFDatasetAdapter(x)

    elif isinstance(x, (_SequenceAdapter, _TFDatasetAdapter)):
        check_y_is_none(y)
        return x

    else:
        raise det.errors.InvalidDataTypeException(
            type(x),
            f"x is invalid type. x={x}\n"
            f"See the instruction below for details: \n"
            f"{adapt_keras_data.__doc__}",
        )


ValidationData = Union[
    tuple, tf.keras.utils.Sequence, tf.data.Dataset, _TFDatasetAdapter, _SequenceAdapter
]


def adapt_validation_data(
    validation_data: ValidationData,
    batch_size: Optional[int] = None,
    use_multiprocessing: bool = False,
    workers: int = 1,
) -> InputData:
    """adapt_validation_data adapts inputs and targets of validation data to
    a _SequenceAdapter or _TFDatasetAdapter, both of which are designed to
    support random access efficiently, for the purpose of supporting Determined
    managed training loop.

    Multiprocessing or multithreading for native Python generators is not supported.
    If you want these performance accelerations, please consider using a Sequence as x.

    Args:
        validation_data: Data on which to evaluate the loss and any model metrics
            at the end of each epoch. The model will not be trained on this data.
            validation_data will override validation_split. validation_data could be:
            1) tuple (x_val, y_val) of Numpy arrays
            2) tuple (x_val, y_val, val_sample_weights) of Numpy arrays
            3) dataset For the first two cases, batch_size must be provided.
            For the last case, validation_steps could be provided.

        use_multiprocessing: If True, use process-based threading. If unspecified,
            `use_multiprocessing` will default to False. Note that because this implementation
            relies on multiprocessing, you should not pass non-picklable arguments for the
            data loaders as they can't be passed easily to children processes. This argument
            is ignored if x is a Dataset.

        workers: Maximum number of processes to spin up when using process-based threading.
            If unspecified, workers will default to 1. If 0, will execute the data loading on
            the main thread. This argument is ignored if x is a Dataset.
    """

    if isinstance(validation_data, (tf.keras.utils.Sequence, tf.data.Dataset)):
        return adapt_keras_data(
            x=validation_data,
            batch_size=batch_size,
            use_multiprocessing=use_multiprocessing,
            workers=workers,
        )

    elif isinstance(validation_data, tuple) and len(validation_data) == 2:
        x, y = validation_data
        return adapt_keras_data(
            x=x,
            y=y,
            batch_size=batch_size,
            use_multiprocessing=use_multiprocessing,
            workers=workers,
        )

    elif isinstance(validation_data, tuple) and len(validation_data) == 3:
        x, y, sample_weight = validation_data
        return adapt_keras_data(
            x=x,
            y=y,
            sample_weight=sample_weight,
            batch_size=batch_size,
            use_multiprocessing=use_multiprocessing,
            workers=workers,
        )

    else:
        raise det.errors.InvalidDataTypeException(
            type(validation_data),
            "validation_data is invalid type. See the instruction below for details: \n"
            f"{adapt_validation_data.__doc__}",
        )
