"""
This files mimics keras.dataset download's function.

For parallel and distributed training, we need to account
for multiple processes (one per GPU) per agent.

For more information on data in Determined, read our data-access tutorial.
"""

import gzip
import tempfile

import numpy as np

from tensorflow.python.keras.utils.data_utils import get_file


def load_training_data():
    """Loads the Fashion-MNIST dataset.

    Returns:
        Tuple of Numpy arrays: `(x_train, y_train)`.

    License:
        The copyright for Fashion-MNIST is held by Zalando SE.
        Fashion-MNIST is licensed under the [MIT license](
        https://github.com/zalandoresearch/fashion-mnist/blob/master/LICENSE).

    """
    download_directory = tempfile.mkdtemp()
    base = "https://storage.googleapis.com/tensorflow/tf-keras-datasets/"
    files = [
        "train-labels-idx1-ubyte.gz",
        "train-images-idx3-ubyte.gz",
    ]

    paths = []
    for fname in files:
        paths.append(
            get_file(fname, origin=base + fname, cache_subdir=download_directory)
        )

    with gzip.open(paths[0], "rb") as lbpath:
        y_train = np.frombuffer(lbpath.read(), np.uint8, offset=8)

    with gzip.open(paths[1], "rb") as imgpath:
        x_train = np.frombuffer(imgpath.read(), np.uint8, offset=16).reshape(
            len(y_train), 28, 28
        )

    return x_train, y_train


def load_validation_data():
    """Loads the Fashion-MNIST dataset.

    Returns:
        Tuple of Numpy arrays: `(x_test, y_test)`.

    License:
        The copyright for Fashion-MNIST is held by Zalando SE.
        Fashion-MNIST is licensed under the [MIT license](
        https://github.com/zalandoresearch/fashion-mnist/blob/master/LICENSE).

    """
    download_directory = tempfile.mkdtemp()
    base = "https://storage.googleapis.com/tensorflow/tf-keras-datasets/"
    files = [
        "t10k-labels-idx1-ubyte.gz",
        "t10k-images-idx3-ubyte.gz",
    ]

    paths = []
    for fname in files:
        paths.append(
            get_file(fname, origin=base + fname, cache_subdir=download_directory)
        )

    with gzip.open(paths[0], "rb") as lbpath:
        y_test = np.frombuffer(lbpath.read(), np.uint8, offset=8)

    with gzip.open(paths[1], "rb") as imgpath:
        x_test = np.frombuffer(imgpath.read(), np.uint8, offset=16).reshape(
            len(y_test), 28, 28
        )

    return x_test, y_test
