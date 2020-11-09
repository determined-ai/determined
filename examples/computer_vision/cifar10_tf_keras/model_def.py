"""
This example shows how you could use Keras `Sequence`s and multiprocessing/multithreading for Keras
models in Determined. Information for how this can be configured can be found in
`make_data_loaders()`.

Tutorial based on this example:
    https://docs.determined.ai/latest/tutorials/tf-cifar-tutorial.html

Useful References:
    https://docs.determined.ai/latest/reference/api/keras.html
    https://www.tensorflow.org/guide/keras

Based on: https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py

"""
import os
import tarfile
import urllib.request
from typing import List

import numpy as np
import tensorflow as tf
from tensorflow.keras.layers import Activation, Conv2D, Dense, Dropout, Flatten, MaxPooling2D
from tensorflow.keras.losses import categorical_crossentropy
from tensorflow.keras.metrics import categorical_accuracy
from tensorflow.keras.models import Sequential
from tensorflow.keras.optimizers import RMSprop

from data import NUM_CLASSES, augment_data, get_data, preprocess_data, preprocess_labels

from determined import keras

# Constants about the data set.
IMAGE_SIZE = 32
NUM_CHANNELS = 3


def download_cifar10_tf_sequence(download_directory: str, url: str) -> str:
    os.makedirs(download_directory, exist_ok=True)
    filepath = os.path.join(download_directory, "data.tar.gz")
    urllib.request.urlretrieve(url, filename=filepath)
    tar = tarfile.open(filepath)
    tar.extractall(path=download_directory)
    return os.path.join(download_directory, "cifar-10-batches-py")


def categorical_error(y_true: np.ndarray, y_pred: np.ndarray) -> float:
    return 1.0 - categorical_accuracy(y_true, y_pred)  # type: ignore


class CIFARTrial(keras.TFKerasTrial):
    def __init__(self, context: keras.TFKerasTrialContext) -> None:
        self.context = context
        self.base_learning_rate = context.get_hparam("learning_rate")  # type: float
        self.learning_rate_decay = context.get_hparam("learning_rate_decay")  # type: float
        self.layer1_dropout = context.get_hparam("layer1_dropout")  # type: float
        self.layer2_dropout = context.get_hparam("layer2_dropout")  # type: float
        self.layer3_dropout = context.get_hparam("layer3_dropout")  # type: float

        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

    def session_config(self) -> tf.compat.v1.ConfigProto:
        if self.context.get_hparams().get("disable_CPU_parallelism", False):
            return tf.compat.v1.ConfigProto(
                intra_op_parallelism_threads=1, inter_op_parallelism_threads=1
            )
        else:
            return tf.compat.v1.ConfigProto()

    def build_model(self) -> Sequential:
        model = Sequential()
        model.add(tf.keras.Input(shape=(IMAGE_SIZE, IMAGE_SIZE, NUM_CHANNELS), name="image"))
        model.add(Conv2D(32, (3, 3), padding="same"))
        model.add(Activation("relu"))
        model.add(Conv2D(32, (3, 3)))
        model.add(Activation("relu"))
        model.add(MaxPooling2D(pool_size=(2, 2)))
        model.add(Dropout(self.layer1_dropout))

        model.add(Conv2D(64, (3, 3), padding="same"))
        model.add(Activation("relu"))
        model.add(Conv2D(64, (3, 3)))
        model.add(Activation("relu"))
        model.add(MaxPooling2D(pool_size=(2, 2)))
        model.add(Dropout(self.layer2_dropout))

        model.add(Flatten())
        model.add(Dense(512))
        model.add(Activation("relu"))
        model.add(Dropout(self.layer3_dropout))
        model.add(Dense(NUM_CLASSES, name="label"))
        model.add(Activation("softmax"))

        # Wrap the model.
        model = self.context.wrap_model(model)

        # Create and wrap the optimizer.
        optimizer = RMSprop(lr=self.base_learning_rate, decay=self.learning_rate_decay)
        optimizer = self.context.wrap_optimizer(optimizer)

        model.compile(
            optimizer,
            categorical_crossentropy,
            [categorical_accuracy, categorical_error],
        )

        return model

    def keras_callbacks(self) -> List[tf.keras.callbacks.Callback]:
        return [keras.TFKerasTensorBoard(update_freq="batch", profile_batch=0, histogram_freq=1)]

    def build_training_data_loader(self) -> keras.InputData:
        """
        In this example we added some fields of note under the `data` field in
        the YAML experiment configuration: the `acceleration` field. Under this
        field, you can configure multithreading by setting
        `use_multiprocessing` to `False`, or set it to `True` for
        multiprocessing. You can also configure the number of workers
        (processes or threads depending on `use_multiprocessing`).

        Another thing of note are the data augmentation fields in
        hyperparameters. The fields here get passed through to Keras'
        `ImageDataGenerator` for real-time data augmentation.
        """
        if not self.data_downloaded:
            self.download_directory = download_cifar10_tf_sequence(
                download_directory=self.download_directory,
                url=self.context.get_data_config()["url"],
            )
            self.data_downloaded = True

        hparams = self.context.get_hparams()
        width_shift_range = hparams.get("width_shift_range", 0.0)
        height_shift_range = hparams.get("height_shift_range", 0.0)
        horizontal_flip = hparams.get("horizontal_flip", False)
        batch_size = self.context.get_per_slot_batch_size()

        (train_data, train_labels), (_, _) = get_data(self.download_directory)

        # Setup training data loader.
        data_augmentation = {
            "width_shift_range": width_shift_range,
            "height_shift_range": height_shift_range,
            "horizontal_flip": horizontal_flip,
        }

        # Returns a tf.keras.Sequence.
        train = augment_data(train_data, train_labels, batch_size, data_augmentation)

        return train

    def build_validation_data_loader(self) -> keras.InputData:
        if not self.data_downloaded:
            self.download_directory = download_cifar10_tf_sequence(
                download_directory=self.download_directory,
                url=self.context.get_data_config()["url"],
            )
            self.data_downloaded = True

        (_, _), (test_data, test_labels) = get_data(self.download_directory)

        return preprocess_data(test_data), preprocess_labels(test_labels)
