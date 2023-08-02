from typing import Any, Optional, Tuple, Type

import numpy as np
from tensorflow.keras import utils as keras_utils

from determined import keras


def xor_data(dtype: Type[Any] = np.int64) -> Tuple[np.ndarray, np.ndarray]:
    training_data = np.array([[0, 0], [0, 1], [1, 0], [1, 1]], dtype=dtype)
    training_labels = np.array([0, 1, 1, 0], dtype=dtype)
    return training_data, training_labels


def make_xor_data_sequences(
    shuffle: bool = False,
    seed: Optional[int] = None,
    dtype: Type[Any] = np.int64,
    multi_input_output: bool = False,
    batch_size: int = 1,
) -> Tuple[keras_utils.Sequence, keras_utils.Sequence]:
    """
    Generates data loaders for the toy XOR problem.  The dataset only has four
    possible inputs.  For the purposes of testing, the validation set is the
    same as the training dataset.
    """
    training_data, training_labels = xor_data(dtype)

    if shuffle:
        if seed is not None:
            np.random.seed(seed)
        idxs = np.random.permutation(4)
        training_data = training_data[idxs]
        training_labels = training_labels[idxs]

    return (
        keras._ArrayLikeAdapter(training_data, training_labels, batch_size=batch_size),
        keras._ArrayLikeAdapter(training_data, training_labels, batch_size=batch_size),
    )
