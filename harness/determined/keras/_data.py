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


class SequenceAdapter:
    """
    Deprecated: use context.configure_fit() instead.
    """

    def __init__(
        self,
        sequence: tf.keras.utils.Sequence,
        use_multiprocessing: bool = False,
        workers: int = 1,
        max_queue_size: int = 10,
    ):
        """
        Deprecated: use context.configure_fit() instead.
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
