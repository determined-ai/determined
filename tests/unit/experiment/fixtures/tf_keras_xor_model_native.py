import argparse
import pathlib

import tensorflow as tf
from tensorflow.keras.layers import Dense
from tensorflow.keras.losses import binary_crossentropy
from tensorflow.keras.metrics import categorical_accuracy
from tensorflow.keras.models import Sequential
from tensorflow.keras.optimizers import SGD

from determined import experimental
from determined.experimental.keras import init
from tests.unit.experiment import utils


def categorical_error(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return 1.0 - categorical_accuracy(y_true, y_pred)


def predictions(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return y_pred


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", dest="mode", default="cluster")
    args = parser.parse_args()

    config = {
        "hyperparameters": {
            "hidden_size": 2,
            "learning_rate": 0.1,
            "global_batch_size": 4,
            "trial_type": "default",
        }
    }

    context = init(
        config=config, mode=experimental.Mode(args.mode), context_dir=str(pathlib.Path.cwd())
    )

    model = Sequential()
    model.add(Dense(context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,)))
    model.add(Dense(1))

    train_data, val_data = utils.make_xor_data_sequences(batch_size=4)
    model = context.wrap_model(model)
    model.compile(
        SGD(lr=context.get_hparam("learning_rate")),
        binary_crossentropy,
        metrics=[categorical_error],
    )
    model.fit_generator(train_data, steps_per_epoch=100, validation_data=val_data, workers=0)
