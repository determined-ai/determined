"""
This example shows how you could use data layer with tf.keras in Determined.
"""
from typing import Any, List

import numpy as np
import tensorflow as tf
import tensorflow_datasets as tfds
from tensorflow.keras.metrics import categorical_accuracy
from tensorflow.keras.models import Sequential
from tensorflow.keras.optimizers import RMSprop

from determined.keras import (
    TFKerasTensorBoard,
    TFKerasTrial,
    TFKerasTrialContext,
)

# Constants about the data set.
IMAGE_SIZE = 32
NUM_CHANNELS = 3


def categorical_error(y_true: np.ndarray, y_pred: np.ndarray) -> float:
    return 1.0 - categorical_accuracy(y_true, y_pred)  # type: ignore


@tfds.decode.make_decoder(output_dtype=tf.float32)
def decode_image(example, feature) -> Any:
    return tf.cast(feature.decode_example(example), dtype=tf.float32) / 255


class MnistTrial(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context
        self.base_learning_rate = context.get_hparam("learning_rate")  # type: float
        self.learning_rate_decay = context.get_hparam("learning_rate_decay")  # type: float

    def session_config(self) -> tf.compat.v1.ConfigProto:
        return tf.compat.v1.ConfigProto()

    def build_model(self) -> Sequential:
        image = tf.keras.layers.Input(shape=(28, 28, 1))

        y = tf.keras.layers.Conv2D(filters=32, kernel_size=5, padding="same", activation="relu")(
            image
        )
        y = tf.keras.layers.MaxPooling2D(pool_size=(2, 2), strides=(2, 2), padding="same")(y)
        y = tf.keras.layers.Conv2D(filters=32, kernel_size=5, padding="same", activation="relu")(y)
        y = tf.keras.layers.MaxPooling2D(pool_size=(2, 2), strides=(2, 2), padding="same")(y)
        y = tf.keras.layers.Flatten()(y)
        y = tf.keras.layers.Dense(1024, activation="relu")(y)
        y = tf.keras.layers.Dropout(0.4)(y)

        probs = tf.keras.layers.Dense(10, activation="softmax")(y)

        model = tf.keras.models.Model(image, probs, name="mnist")

        # Wrap the model.
        model = self.context.wrap_model(model)

        # Create and wrap the optimizer.
        optimizer = RMSprop(lr=self.base_learning_rate, decay=self.learning_rate_decay)
        optimizer = self.context.wrap_optimizer(optimizer)

        model.compile(
            optimizer=optimizer,
            loss="sparse_categorical_crossentropy",
            metrics=["sparse_categorical_accuracy"],
        )

        return model

    def keras_callbacks(self) -> List[tf.keras.callbacks.Callback]:
        return [TFKerasTensorBoard(update_freq="batch", profile_batch=0, histogram_freq=1)]

    def build_training_data_loader(self) -> tf.data.Dataset:
        @self.context.experimental.cache_train_dataset("mnist-tf-keras", "v1", shuffle=True)
        def make_dataset() -> tf.data.Dataset:
            mnist_builder = tfds.builder("mnist")
            mnist_builder.download_and_prepare(download_dir="/cifar")
            mnist_train = mnist_builder.as_dataset(
                split="train", decoders={"image": decode_image()}, as_supervised=True,
            )
            return mnist_train

        ds = make_dataset()
        ds = ds.batch(self.context.get_per_slot_batch_size())
        return ds

    def build_validation_data_loader(self) -> tf.data.Dataset:
        @self.context.experimental.cache_validation_dataset("mnist-tf-keras", "v1")
        def make_dataset() -> tf.data.Dataset:
            mnist_builder = tfds.builder("mnist")
            mnist_builder.download_and_prepare(download_dir="/cifar")
            mnist_val = mnist_builder.as_dataset(
                split="test", decoders={"image": decode_image()}, as_supervised=True,
            )
            return mnist_val

        ds = make_dataset()
        ds = ds.batch(self.context.get_per_slot_batch_size())
        return ds
