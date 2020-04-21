import pathlib
from typing import Tuple

import tensorflow as tf
from tensorflow.keras.layers import Dense
from tensorflow.keras.losses import binary_crossentropy
from tensorflow.keras.metrics import categorical_accuracy
from tensorflow.keras.models import Sequential
from tensorflow.keras.optimizers import SGD

from determined import experimental
from determined.experimental import keras
from tests.unit.experiment.utils import make_xor_data_sequences  # noqa: I202, I100


def categorical_error(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return 1.0 - categorical_accuracy(y_true, y_pred)


def predictions(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return y_pred


def make_xor_single_thread_loaders() -> Tuple[tf.keras.utils.Sequence, tf.keras.utils.Sequence]:
    return make_xor_data_sequences(batch_size=4)


def train():
    context = keras.init(mode=experimental.Mode.CLUSTER, context_dir=str(pathlib.Path.cwd()))

    model = Sequential()
    model.add(Dense(context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,)))
    model.add(Dense(1))

    train_data, val_data = make_xor_single_thread_loaders()
    model = context.wrap_model(model)
    model.compile(
        SGD(lr=context.get_hparam("learning_rate")),
        binary_crossentropy,
        metrics=[categorical_error],
    )
    model.fit_generator(train_data, steps_per_epoch=100, validation_data=val_data, workers=0)


if __name__ == "__main__":
    train()
