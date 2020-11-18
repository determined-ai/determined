import math
from typing import Any, Dict, Iterator, List, Optional, Tuple, Union

import numpy as np
import tensorflow as tf

import determined as det
from determined import keras
from determined_common import check

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
        sample_weights: Optional[np.ndarray] = None,
        drop_leftovers: bool = False,
    ):
        """
        If converting numpy array data to Sequence to optimize performance, consider
        using ArrayLikeAdapter.

        Args:
            x: Input data. It could be:
                1) A Numpy array (or array-like), or a list of arrays (in case the model
                has multiple inputs).
                2) A dict mapping input names to the corresponding array, if the model
                has named inputs.

            y: Target data. Like the input data x, it could be either Numpy array(s).

            batch_size: Number of samples per batch.

            sample_weights: Numpy array of weights for the samples.

            drop_leftovers: If True, drop the data that cannot complete the last batch. This
                argument is ignored if x is a Sequence or a Dataset.
        """

        self._x_length = _length_of_multi_arraylike(x)
        self._y_length = _length_of_multi_arraylike(y)

        check.eq(self._x_length, self._y_length, "Length of x and y do not match.")
        check.check_gt_eq(self._x_length, batch_size, "Batch size is too large for the input data.")
        if sample_weights is not None:
            check.eq(
                self._x_length,
                len(sample_weights),
                "Lengths of input data and sample weights do not match.",
            )

        self.x = x
        self.y = y
        self.sample_weight = sample_weights

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


class SequenceAdapter:
    """
    Deprecated: use context.configure_fit() instead.
    """

    def __init__(
        self,
        data: tf.keras.utils.Sequence,
        use_multiprocessing: bool = False,
        workers: int = 1,
        max_queue_size: int = 10,
    ):
        # TODO: Issue a deprecation warning after #1545 or #1564 land.
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


InputData = Union[tf.keras.utils.Sequence, tf.data.Dataset, SequenceAdapter, tuple]


def _get_x_y_and_sample_weight(
    input_data: Union[tf.keras.utils.Sequence, tf.data.Dataset, SequenceAdapter, tuple]
) -> Tuple[Any, Any, Any]:
    if isinstance(input_data, (tf.keras.utils.Sequence, tf.data.Dataset, SequenceAdapter)):
        return input_data, None, None

    elif isinstance(input_data, tuple) and len(input_data) == 2:
        return input_data[0], input_data[1], None

    elif isinstance(input_data, tuple) and len(input_data) == 3:
        return input_data[0], input_data[1], input_data[2]

    else:
        raise det.errors.InvalidDataTypeException(
            type(input_data),
            "input_data is invalid type. See the instruction below for details: \n"
            f"{keras.TFKerasTrial.build_training_data_loader.__doc__}",
        )


def _adapt_keras_data(
    x: Any,
    y: Any = None,
    sample_weight: Optional[np.ndarray] = None,
    batch_size: Optional[int] = None,
    use_multiprocessing: bool = False,
    workers: int = 1,
    max_queue_size: int = 10,
    drop_leftovers: bool = False,
) -> Union[SequenceAdapter, tf.data.Dataset]:
    """_adapt_keras_data adapts input and target data to a SequenceAdapter or leaves
    it as a tf.data.Dataset.

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
            data loaders as they can't be passed easily to children processes. This argument is
            ignored if x is a tf.data.Dataset.

        sample_weight: Optional Numpy array of weights for the training samples. This argument is
        ignored if x is a tf.data.Dataset or tf.keras.Sequence.

        batch_size: Number of samples per gradient update. This argument is ignored if x is a
        tf.data.Dataset or tf.keras.Sequence.

        workers: Maximum number of processes to spin up when using process-based threading.
            If unspecified, workers will default to 1. If 0, will execute the data loading on
            the main thread. This argument is ignored if x is a tf.data.Dataset.

        max_queue_size: Maximum size for the generator queue. If unspecified, `max_queue_size`
            will default to 10. This argument is ignored if x is a tf.data.Dataset.

        drop_leftovers: If True, drop the data that cannot complete the last batch. This
            argument is ignored if x is a Sequence or a Dataset. This argument is ignored if
            x is a tf.data.Dataset.
    """

    def check_y_is_none(y_data: Any) -> None:
        if y is not None:
            raise det.errors.InvalidDataTypeException(
                type(y_data),
                "If x is a keras.utils.Sequence or a tf.data.Dataset, "
                "y should not be specified (since targets will be obtained from x)."
                "See the instruction below for details: "
                f"\n{keras.TFKerasTrial.build_training_data_loader.__doc__}",
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
                "See the instruction below for details: "
                f"\n{keras.TFKerasTrial.build_training_data_loader.__doc__}",
            )
        return SequenceAdapter(
            _ArrayLikeAdapter(x, y, batch_size, sample_weight, drop_leftovers),
            use_multiprocessing,
            workers,
            max_queue_size,
        )

    elif isinstance(x, tf.keras.utils.Sequence):
        check_y_is_none(y)
        return SequenceAdapter(x, use_multiprocessing, workers, max_queue_size)

    elif isinstance(x, tf.data.Dataset):
        return x

    elif isinstance(x, SequenceAdapter):
        check_y_is_none(y)
        return x

    else:
        raise det.errors.InvalidDataTypeException(
            type(x),
            f"x is invalid type. x={x}\n"
            f"See the instruction below for details: \n"
            f"\n{keras.TFKerasTrial.build_training_data_loader.__doc__}",
        )
