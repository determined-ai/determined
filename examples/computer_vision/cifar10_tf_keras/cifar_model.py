"""
Original CIFAR-10 CNN Keras model code from:
https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py.
"""

import numpy as np
import tensorflow as tf
from tensorflow.keras import layers
from tensorflow.keras import losses
from tensorflow.keras import metrics
from tensorflow.keras import models
from tensorflow.keras import optimizers

from data import NUM_CLASSES

# Constants about the data set.
IMAGE_SIZE = 32
NUM_CHANNELS = 3


def categorical_error(y_true: np.ndarray, y_pred: np.ndarray) -> float:
    return 1.0 - metrics.categorical_accuracy(y_true, y_pred)  # type: ignore


def build_model(layer1_dropout, layer2_dropout, layer3_dropout):
    model = models.Sequential()
    model.add(tf.keras.Input(shape=(IMAGE_SIZE, IMAGE_SIZE, NUM_CHANNELS), name="image"))
    model.add(layers.Conv2D(32, (3, 3), padding="same"))
    model.add(layers.Activation("relu"))
    model.add(layers.Conv2D(32, (3, 3)))
    model.add(layers.Activation("relu"))
    model.add(layers.MaxPooling2D(pool_size=(2, 2)))
    model.add(layers.Dropout(layer1_dropout))

    model.add(layers.Conv2D(64, (3, 3), padding="same"))
    model.add(layers.Activation("relu"))
    model.add(layers.Conv2D(64, (3, 3)))
    model.add(layers.Activation("relu"))
    model.add(layers.MaxPooling2D(pool_size=(2, 2)))
    model.add(layers.Dropout(layer2_dropout))

    model.add(layers.Flatten())
    model.add(layers.Dense(512))
    model.add(layers.Activation("relu"))
    model.add(layers.Dropout(layer3_dropout))
    model.add(layers.Dense(NUM_CLASSES, name="label"))
    model.add(layers.Activation("softmax"))

    return model


def build_optimizer(learning_rate, learning_rate_decay):
    lr_schedule = optimizers.schedules.ExponentialDecay(
        initial_learning_rate=learning_rate,
        decay_steps=10000,
        decay_rate=learning_rate_decay,
    )
    return optimizers.RMSprop(learning_rate=lr_schedule)


def compile_model(model, optimizer):
    model.compile(
        optimizer,
        losses.categorical_crossentropy,
        [metrics.categorical_accuracy, categorical_error],
    )
