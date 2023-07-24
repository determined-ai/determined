"""
Original CIFAR-10 CNN Keras model code from:
https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py.
"""

import numpy as np
import tensorflow as tf
from packaging import version
from tensorflow.keras.layers import Activation, Conv2D, Dense, Dropout, Flatten, MaxPooling2D
from tensorflow.keras.losses import categorical_crossentropy
from tensorflow.keras.metrics import categorical_accuracy
from tensorflow.keras.models import Sequential

# This example is used in some tests with an explicit tf1 image,
#  so we keep backwards compatibility.
if version.parse(tf.__version__) >= version.parse("2.11.0"):
    from tensorflow.keras.optimizers.legacy import RMSprop
else:
    from tensorflow.keras.optimizers import RMSprop

# Constants about the data set.
IMAGE_SIZE = 32
NUM_CHANNELS = 3
NUM_CLASSES = 10


def categorical_error(y_true: np.ndarray, y_pred: np.ndarray) -> float:
    return 1.0 - categorical_accuracy(y_true, y_pred)  # type: ignore


def build_model(layer1_dropout, layer2_dropout, layer3_dropout):
    model = Sequential()
    model.add(tf.keras.Input(shape=(IMAGE_SIZE, IMAGE_SIZE, NUM_CHANNELS), name="image"))
    model.add(Conv2D(32, (3, 3), padding="same"))
    model.add(Activation("relu"))
    model.add(Conv2D(32, (3, 3)))
    model.add(Activation("relu"))
    model.add(MaxPooling2D(pool_size=(2, 2)))
    model.add(Dropout(layer1_dropout))

    model.add(Conv2D(64, (3, 3), padding="same"))
    model.add(Activation("relu"))
    model.add(Conv2D(64, (3, 3)))
    model.add(Activation("relu"))
    model.add(MaxPooling2D(pool_size=(2, 2)))
    model.add(Dropout(layer2_dropout))

    model.add(Flatten())
    model.add(Dense(512))
    model.add(Activation("relu"))
    model.add(Dropout(layer3_dropout))
    model.add(Dense(NUM_CLASSES, name="label"))
    model.add(Activation("softmax"))

    return model


def build_optimizer(learning_rate, learning_rate_decay):
    return RMSprop(learning_rate=learning_rate, decay=learning_rate_decay)


def compile_model(model, optimizer):
    model.compile(
        optimizer,
        categorical_crossentropy,
        [categorical_accuracy, categorical_error],
    )
