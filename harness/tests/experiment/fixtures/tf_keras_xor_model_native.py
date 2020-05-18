import argparse
import pathlib

import tensorflow as tf
from tensorflow.keras.layers import Dense
from tensorflow.keras.losses import binary_crossentropy
from tensorflow.keras.metrics import categorical_accuracy
from tensorflow.keras.models import Sequential
from tensorflow.keras.optimizers import SGD

from determined.experimental.keras import init
from tests.experiment import utils


def categorical_error(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return 1.0 - categorical_accuracy(y_true, y_pred)


def predictions(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return y_pred


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--local", action="store_true")
    parser.add_argument("--test", action="store_true")
    parser.add_argument("--use-dataset", action="store_true")
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
        config=config, local=args.local, test=args.test, context_dir=str(pathlib.Path.cwd())
    )

    model = Sequential()
    model.add(Dense(context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,)))
    model.add(Dense(1))

    if args.use_dataset:
        data, labels = utils.xor_data()

        train = context.wrap_dataset(tf.data.Dataset.from_tensor_slices((data, labels)))
        train = train.batch(context.get_hparam("global_batch_size"))
        valid = context.wrap_dataset(tf.data.Dataset.from_tensor_slices((data, labels)))
        valid = valid.batch(context.get_hparam("global_batch_size"))
    else:
        train, valid = utils.make_xor_data_sequences(batch_size=4)

    model = context.wrap_model(model)
    model.compile(
        SGD(lr=context.get_hparam("learning_rate")),
        binary_crossentropy,
        metrics=[categorical_error],
    )
    model.fit(x=train, steps_per_epoch=100, validation_data=valid, workers=0)
