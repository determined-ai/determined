from typing import Tuple

import determined as det
import numpy as np
import tensorflow as tf


def load_numpy_data(
    context: det.core.Context,
) -> Tuple[Tuple[np.ndarray, np.ndarray], Tuple[np.ndarray, np.ndarray]]:
    # When running distributed, we don't want multiple ranks on the same node to download the
    # data simultaneously, since they'll overwrite each other. So we only download on
    # local rank 0.
    if context.distributed.get_local_rank() == 0:
        tf.keras.datasets.cifar10.load_data()
    # Wait until local rank 0 is done downloading.
    context.distributed.allgather_local(None)
    # Now that the data is downloaded, each rank can load it.
    (X_train, Y_train), (X_test, Y_test) = tf.keras.datasets.cifar10.load_data()
    # Convert from pixel values to [0, 1] range floats, and one-hot encode labels.
    X_train = X_train.astype("float32") / 255
    X_test = X_test.astype("float32") / 255
    Y_train = tf.keras.utils.to_categorical(Y_train, num_classes=10)
    Y_test = tf.keras.utils.to_categorical(Y_test, num_classes=10)
    return (X_train, Y_train), (X_test, Y_test)
