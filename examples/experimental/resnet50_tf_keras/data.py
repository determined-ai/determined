import tensorflow as tf
import numpy as np


class SyntheticData(tf.keras.utils.Sequence):
    def __init__(self, num_batches: int, data_shape, labels_shape) -> None:
        self.num_batches = num_batches
        self.data_shape = data_shape
        self.labels_shape = labels_shape

    def __len__(self):
        return self.num_batches

    def __getitem__(self, _):
        return (np.random.rand(*self.data_shape), np.random.rand(*self.labels_shape))
