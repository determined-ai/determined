"""
This example shows how to use Determined to implement a tf.keras-based CNN to
perform image classification on the Fashion-MNIST dataset.

Based off: https://www.tensorflow.org/tutorials/keras/classification
"""
import tempfile

import tensorflow as tf
from tensorflow import keras

from determined.keras import TFKerasTrial, TFKerasTrialContext, InputData

import data


class MNISTTrial(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = tempfile.mkdtemp()

    def build_model(self):
        model = keras.Sequential(
            [
                keras.layers.Flatten(input_shape=(28, 28)),
                keras.layers.Dense(128, activation="relu"),
                keras.layers.Dense(10),
            ]
        )
        model = self.context.wrap_model(model)
        model.compile(
            optimizer="adam",
            loss=tf.keras.losses.SparseCategoricalCrossentropy(from_logits=True),
            metrics=[tf.keras.metrics.SparseCategoricalAccuracy(name="accuracy")],
        )
        return model

    def build_training_data_loader(self) -> InputData:
        (train_images, train_labels), (_, _) = data.load_data(self.download_directory)
        train_images = train_images / 255.0

        return train_images, train_labels

    def build_validation_data_loader(self) -> InputData:
        (_, _), (test_images, test_labels) = data.load_data(self.download_directory)
        test_images = test_images / 255.0

        return test_images, test_labels
