import math
from typing import Any, Dict, List, Optional, Tuple, Union

import numpy as np
import tensorflow as tf

import determined as det
from determined import keras
from determined_common import check

ArrayLike = Union[np.ndarray, List[np.ndarray], Dict[str, np.ndarray]]

InputData = Union[tf.keras.utils.Sequence, tf.data.Dataset, "SequenceAdapter", tuple]


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

        if not (
            isinstance(x, np.ndarray) or _is_list_of_numpy_array(x) or _is_dict_of_numpy_array(x)
        ):
            raise det.errors.InvalidDataTypeException(
                type(x),
                "Data which is not tf.data.Datasets or tf.keras.utils.Sequence objects must be a "
                "numpy array or a list/dict of numpy arrays. See the instructions below for "
                f"details:\n{keras.TFKerasTrial.build_training_data_loader.__doc__}",
            )
        if not (
            isinstance(y, np.ndarray) or _is_list_of_numpy_array(y) or _is_dict_of_numpy_array(y)
        ):
            raise det.errors.InvalidDataTypeException(
                type(y),
                "Data which is not tf.data.Datasets or tf.keras.utils.Sequence objects must be a "
                "numpy array or a list/dict of numpy arrays. See the instructions below for "
                f"details:\n{keras.TFKerasTrial.build_training_data_loader.__doc__}",
            )

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


class _DeterminedSequenceWrapper(tf.keras.utils.Sequence):  # type: ignore
    """
    A Keras Sequence with start offset, sharding, and deterministic shuffling.

    This wrapper uses shuffle-before-shard to get the highest-quality shuffle.

    When training is enabled, this wrapper will report the full length of the dataset, rather than
    the length of this shard.  This is to work around the fact that some shards might be longer
    than others, which would cause some workers to iterate through the dataset at different rates.
    """

    def __init__(
        self,
        sequence: tf.keras.utils.Sequence,
        shard_rank: int,
        shard_size: int,
        training: bool,
        shuffle: Optional[bool] = None,
        shuffle_seed: Optional[int] = None,
        prior_batches_trained: Optional[int] = None,
    ):
        check.gt_eq(
            len(sequence),
            shard_size,
            "The length of the Keras Sequence used must be greater than or equal to the number "
            "of workers used in training.",
        )

        self._sequence = sequence
        self._shard_rank = shard_rank
        self._shard_size = shard_size
        self._shuffle = shuffle
        self._shuffle_seed = shuffle_seed

        if not training:
            self._batch_offset = 0
            self._indices = list(range(shard_rank, len(sequence), shard_size))
            self._len = len(self._indices)
            return

        assert shuffle is not None
        assert shuffle_seed is not None
        assert prior_batches_trained is not None

        # Even when we are sharded, we'll repeat in a way that his holds true.
        # Example: suppose the underlying dataset is length 10, and we have 3 shards.
        #   Each shard will yield 3 sharded epochs before completing a cycle.  At the end of each
        #   cycle, all shards will be back in their original state.
        #
        #   shard 0 yields: 0 3 6 9|2 5 8|1 4 7
        #   shard 1 yields: 1 4 7|0 3 6 9|2 5 8
        #   shard 2 yields: 2 5 8|1 4 7|0 3 6 9
        #
        # Notice that each shard emits a local epoch that consists of (shard_size) cycles through
        # the total dataset.
        self._len = len(sequence)

        self._batch_offset = prior_batches_trained % self._len

        # Pre-calculate the next two epochs in indices that we will produce.  We need two epochs of
        # indices so that we can always produce one entire epoch of indices starting from any
        # batch_offset, since the only thread-safe place to reshuffle is in on_epoch_end, and that
        # always happens after exactly len(self) batches.

        if not shuffle:
            unmodded = range(shard_rank, self._len * shard_size * 2, shard_size)
            self._indices = [u % self._len for u in unmodded]
            return

        self._rng = np.random.RandomState(shuffle_seed)

        # We calculate the shuffled indices for all workers in the set, even though we are only
        # going to yield one worker's shard of values from this sequence.  This allows us to be
        # deterministic in our shuffling and to guarantee that the model actually sees all of one
        # entire global epoch before it starts to see any repeats (with possible overlap in one
        # batch at every global epoch boundary).
        self._global_epoch_indices = list(range(self._len))
        self._rng.shuffle(self._global_epoch_indices)

        # Begin in the correct local epoch's shuffle
        local_epochs_past = prior_batches_trained // self._len
        # Because local epochs are sharded from global epochs, we cycle through global epochs once
        # per shard_size per local epoch.
        for _ in range(local_epochs_past * shard_size):
            self._rng.shuffle(self._global_epoch_indices)

        self._indices = []
        self._gen_local_epoch_of_shuffled_indices()
        self._gen_local_epoch_of_shuffled_indices()

    def _gen_local_epoch_of_shuffled_indices(self) -> None:
        new_indices = []
        offset = self._shard_rank
        for _ in range(self._shard_size):
            new_indices += self._global_epoch_indices[offset :: self._shard_size]
            offset = (offset - len(self._sequence)) % self._shard_size
            self._rng.shuffle(self._global_epoch_indices)
        assert offset == self._shard_rank, "offset should have cycled back to its original value"
        assert len(new_indices) == len(self._sequence), "should have one epochs of new indices"
        self._indices += new_indices

    def __len__(self):  # type: ignore
        return self._len

    def __getitem__(self, index):  # type: ignore
        return self._sequence[self._indices[index + self._batch_offset]]

    def on_epoch_end(self) -> None:
        """
        Keras will synchronize all workers when the sequence runs out, so it is safe to reshuffle
        indices here; there is no risk of one worker reading indices while another worker is in
        this call.
        """
        if not self._shuffle:
            return

        # Left-shift self._indices by one epoch and generate another epoch of indices
        self._indices = self._indices[len(self._sequence) :]
        self._gen_local_epoch_of_shuffled_indices()


class SequenceAdapter:
    """
    A class to assist to optimize the performance of loading data with
    ``tf.keras.utils.Sequence`` and help with restoring and saving iterators for
    a dataset.
    """

    def __init__(
        self,
        sequence: tf.keras.utils.Sequence,
        use_multiprocessing: bool = False,
        workers: int = 1,
        max_queue_size: int = 10,
    ):
        """
        Multiprocessing or multithreading for native Python generators is not supported.
        If you want these performance accelerations, please consider using a Sequence.

        Args:
            sequence: A ``tf.keras.utils.Sequence`` that holds the data.

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

        self.sequence = sequence
        self.use_multiprocessing = use_multiprocessing
        self.workers = workers
        self.max_queue_size = max_queue_size


def _adapt_data_from_data_loader(
    input_data: InputData,
    batch_size: int,
) -> Union[tf.keras.utils.Sequence, SequenceAdapter, tf.data.Dataset]:
    if isinstance(input_data, tf.data.Dataset):
        return input_data

    if isinstance(input_data, (tf.keras.utils.Sequence, SequenceAdapter)):
        return input_data

    if not isinstance(input_data, tuple) or len(input_data) not in (2, 3):
        raise det.errors.InvalidDataTypeException(
            type(input_data),
            "input_data is invalid type. See the instruction below for details: \n"
            f"{keras.TFKerasTrial.build_training_data_loader.__doc__}",
        )

    x = input_data[0]
    y = input_data[1]
    sample_weight = input_data[2] if len(input_data) == 3 else None

    return _ArrayLikeAdapter(x, y, batch_size, sample_weight)


def _adapt_data_from_fit_args(
    x: Any,
    y: Any,
    sample_weight: Any,
    batch_size: int,
) -> Any:
    """
    This is the in-between layer from the Native API to the Trial API.
    """
    if isinstance(x, (tf.data.Dataset, tf.keras.utils.Sequence, SequenceAdapter)):
        if y is not None:
            raise det.errors.InvalidDataTypeException(
                type(y),
                "If x is a keras.utils.Sequence or a tf.data.Dataset, "
                "y should not be specified (since targets will be obtained from x)."
                "See the instruction below for details: "
                f"\n{keras.TFKerasTrial.build_training_data_loader.__doc__}",
            )
        return x

    return _ArrayLikeAdapter(x, y, batch_size, sample_weight)
