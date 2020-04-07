"""
This example a simple example that shows how to implemented a CNN based on the CIFAR10 in
Determined.

Based off: https://www.tensorflow.org/tutorials/images/cnn
"""

import tensorflow as tf
from tensorflow import keras

from determined.keras import InMemorySequence, TFKerasTrial, TFKerasTrialContext


class MNISTTrial(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context

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

    def build_training_data_loader(self):
        fashion_mnist = keras.datasets.fashion_mnist
        (train_images, train_labels), (_, _) = fashion_mnist.load_data()
        train_images = train_images / 255.0

        batch_size = self.context.get_per_slot_batch_size()
        train = InMemorySequence(data=train_images, labels=train_labels, batch_size=batch_size)

        return train

    def build_validation_data_loader(self):
        fashion_mnist = keras.datasets.fashion_mnist
        (_, _), (test_images, test_labels) = fashion_mnist.load_data()
        test_images = test_images / 255.0

        batch_size = self.context.get_per_slot_batch_size()
        test = InMemorySequence(data=test_images, labels=test_labels, batch_size=batch_size)

        return test
