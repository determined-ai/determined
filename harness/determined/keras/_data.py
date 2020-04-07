import math
from typing import Any, Dict, Iterator, List, Optional, Tuple, Union

import numpy as np
import tensorflow as tf

from determined_common import check


class SequenceWithOffset(tf.keras.utils.Sequence):  # type: ignore
    def __init__(self, sequence: tf.keras.utils.Sequence, batch_offset: int = 0):
        self._sequence = sequence
        self._batch_offset = batch_offset

    def __len__(self):  # type: ignore
        return len(self._sequence)

    def __getitem__(self, index):  # type: ignore
        index = (index + self._batch_offset) % len(self)
        return self._sequence[index]


class KerasBatch(object):
    """
    A class that encapsulates the inputs needed for Keras to train/test a batch.

    This class is intentionally small and minimizes the checks it makes to what could
    break our code. Validation of the fields is left to Keras as much as possible.
    """

    def __init__(
        self,
        data: Union[np.ndarray, List, Dict],
        labels: Union[np.ndarray, List, Dict],
        sample_weight: Union[np.ndarray, List, Dict],
    ):
        self.data = data
        self.labels = labels
        self.sample_weight = sample_weight

    def __len__(self) -> int:
        if isinstance(self.data, np.ndarray):
            return len(self.data)
        elif isinstance(self.data, list):
            return len(self.data[0])
        elif isinstance(self.data, dict):
            return len(self.data[next(iter(self.data))])
        raise TypeError(
            "Input data for Keras trials must be either a NumPy array, list of NumPy "
            "arrays, or a dictionary mapping input names to NumPy arrays."
        )


def _iter_sequence_infinite(sequence: tf.keras.utils.Sequence) -> Iterator:
    while True:
        yield from sequence


def _make_finite(iterator: Iterator, num_steps: int) -> Iterator:
    for _ in range(num_steps):
        yield next(iterator)


class KerasDataAdapter:
    def __init__(
        self,
        data: tf.keras.utils.Sequence,
        use_multiprocessing: bool = False,
        workers: int = 1,
        max_queue_size: int = 10,
    ):
        """
        Data encapsulation for Keras models.

        KerasDataAdapter wraps a tf.keras.utils.Sequence which would normally be fed to
        Keras' fit_generator. This behaves the same as the data inputs to fit_generator.

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
        self._sequence = SequenceWithOffset(data)
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


def make_keras_data_adapter(
    data_loader: Any,
    batch_size: int,
    multi_input_output: bool = False,
    drop_leftovers: bool = False,
) -> KerasDataAdapter:
    if isinstance(data_loader, KerasDataAdapter):
        return data_loader
    elif isinstance(data_loader, tf.keras.utils.Sequence):
        return KerasDataAdapter(data_loader)
    else:
        raise ValueError(
            "Data loaders of type {} are not supported by KerasDataAdapter".format(
                type(data_loader).__name__
            )
        )


class InMemorySequence(tf.keras.utils.Sequence):  # type: ignore
    """
    InMemorySequence is a utility class for converting a simple in-memory
    supervised numpy dataset into a tf.keras.utils.Sequence instance. This is
    useful for cases for supporting high-level training loops that accept
    in-memory data (tf.keras.Model.fit(x=data, y=labels)). It may also be used
    directly by users in their data loading implementation(s).

    Currently, this class only supports the case where data, labels, and
    (optionally) sample_weights are of np.ndarray type. For any more complex
    data types (multiple input and/or multiple output), it is recommended to
    define a full tf.keras.utils.Sequence.
    """

    def __init__(
        self,
        data: np.ndarray,
        labels: np.ndarray,
        batch_size: int,
        sample_weight: Optional[np.ndarray] = None,
        drop_leftovers: bool = False,
    ):
        check.check_eq_len(data, labels, "Length of input data and input labels do not match.")
        check.check_gt_eq(len(data), batch_size, "Batch size is too large for the input data.")
        if sample_weight is not None:
            check.check_eq_len(
                data, sample_weight, "Lengths of input data and sample weights do not match."
            )

        self.data = data
        self.labels = labels
        self.batch_size = batch_size
        self.drop_leftovers = drop_leftovers
        self.sample_weight = sample_weight

    def __len__(self) -> int:
        # Returns number of batches (keeps last partial batch)
        if self.drop_leftovers:
            return math.floor(len(self.data) / self.batch_size)
        else:
            return math.ceil(len(self.data) / self.batch_size)

    def __getitem__(self, index: int) -> Tuple[np.ndarray, np.ndarray]:
        # Gets batch at position index
        start = index * self.batch_size

        # The end is not `(index + 1) * self.batch_size` if the
        # last batch is not a full `self.batch_size`
        end = min((index + 1) * self.batch_size, len(self.data))
        return (self.data[start:end], self.labels[start:end])


KerasInputData = Union[KerasDataAdapter, tf.data.Dataset]
