"""
This example demonstrates training a simple CNN with tf.keras using the Determined
Native API.
"""
import argparse
import json
import pathlib
from typing import Any, Dict, Tuple

import numpy as np
import tensorflow as tf
from tensorflow.keras.datasets import mnist
from tensorflow.keras.layers import Conv2D, Dense, Dropout, Flatten, MaxPooling2D
from tensorflow.keras.models import Sequential
from tensorflow.keras.preprocessing.image import ImageDataGenerator

import determined as det
from determined.keras import init

NUM_CLASSES = 10
INPUT_SHAPE = (28, 28, 1)
IMG_ROWS, IMG_COLS = INPUT_SHAPE[0], INPUT_SHAPE[0]


def cnn_model(hparams: Dict[str, Any]) -> tf.keras.Model:
    model = Sequential()
    model.add(
        Conv2D(
            32,
            kernel_size=(hparams["kernel_size"], hparams["kernel_size"]),
            activation="relu",
            input_shape=INPUT_SHAPE,
        )
    )
    model.add(Conv2D(64, (3, 3), activation=hparams["activation"]))
    model.add(MaxPooling2D(pool_size=(2, 2)))
    model.add(Dropout(hparams["dropout"]))
    model.add(Flatten())
    model.add(Dense(128, activation=hparams["activation"]))
    model.add(Dropout(0.5))
    model.add(Dense(NUM_CLASSES, activation="softmax"))

    return model


def load_mnist_data() -> Tuple[np.array, np.array]:
    # Download and prepare the MNIST dataset.
    (x_train, y_train), (x_test, y_test) = mnist.load_data()
    x_train = x_train.reshape(x_train.shape[0], IMG_ROWS, IMG_COLS, 1)
    x_test = x_test.reshape(x_test.shape[0], IMG_ROWS, IMG_COLS, 1)
    y_train = tf.keras.utils.to_categorical(y_train, NUM_CLASSES)
    y_test = tf.keras.utils.to_categorical(y_test, NUM_CLASSES)
    return (x_train, y_train), (x_test, y_test)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--config",
        dest="config",
        help="Specifies Determined Experiment configuration.",
        default="{}",
    )
    parser.add_argument(
        "--mode", dest="mode", help="Specifies test mode or submit mode.", default="submit"
    )
    parser.add_argument(
        "--use-fit",
        action="store_true",
        help="If true, uses model.fit() instead of model.fit_generator()",
    )
    args = parser.parse_args()

    config = {
        "hyperparameters": {
            "global_batch_size": det.Constant(value=32),
            "kernel_size": det.Constant(value=3),
            "dropout": det.Double(minval=0.0, maxval=0.5),
            "activation": det.Constant(value="relu"),
        },
        "searcher": {"name": "single", "metric": "val_accuracy", "max_steps": 40},
    }
    config.update(json.loads(args.config))

    context = init(config, mode=det.Mode(args.mode), context_dir=str(pathlib.Path.cwd()))

    (x_train, y_train), (x_test, y_test) = load_mnist_data()
    # Create training and test data generators using Keras' ImageDataGenerator.
    train_datagen = ImageDataGenerator(featurewise_center=True, featurewise_std_normalization=True)
    val_datagen = ImageDataGenerator()
    # Compute quantities required for featurewise normalization.
    train_datagen.fit(x_train)

    model = cnn_model(context.get_hparams())
    model = context.wrap_model(model)

    model.compile(
        loss=tf.keras.losses.categorical_crossentropy,
        optimizer=tf.keras.optimizers.Adadelta(),
        metrics=[tf.keras.metrics.CategoricalAccuracy(name="accuracy")],
    )

    if args.use_fit:
        model.fit(
            x_train,
            y_train,
            batch_size=context.get_per_slot_batch_size(),
            validation_data=val_datagen.flow(
                x_test, y_test, batch_size=context.get_per_slot_batch_size()
            ),
            validation_steps=y_test.shape[0] // context.get_global_batch_size(),
            use_multiprocessing=False,
            workers=1,
            max_queue_size=10,
            epochs=1,
        )
    else:
        model.fit_generator(
            train_datagen.flow(x_train, y_train, batch_size=context.get_per_slot_batch_size()),
            validation_data=val_datagen.flow(
                x_test, y_test, batch_size=context.get_per_slot_batch_size()
            ),
            validation_steps=y_test.shape[0] // context.get_global_batch_size(),
            use_multiprocessing=False,
            workers=1,
            max_queue_size=10,
            epochs=1,
        )
