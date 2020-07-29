"""
This example demonstrates training a simple CNN with tf.keras using the Determined
Native API.
"""
import argparse
import json
import pathlib

import tensorflow as tf
from tensorflow import keras

import determined as det
from determined import experimental
from determined.experimental.keras import init
from determined.keras import _ArrayLikeAdapter, TFKerasNativeContext

import data


def build_model(context: TFKerasNativeContext) -> tf.keras.Model:
    model = keras.Sequential(
        [
            keras.layers.Flatten(input_shape=(28, 28)),
            keras.layers.Dense(context.get_hparam("dense1"), activation="relu"),
            keras.layers.Dense(10),
        ]
    )
    model = context.wrap_model(model)
    model.compile(
        optimizer=tf.keras.optimizers.Adam(name='Adam'),
        loss=tf.keras.losses.SparseCategoricalCrossentropy(from_logits=True),
        metrics=[tf.keras.metrics.SparseCategoricalAccuracy(name="accuracy")],
    )
    return model


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--config",
        dest="config",
        help="Specifies Determined Experiment configuration.",
        default="{}",
    )
    parser.add_argument("--local", action="store_true", help="Specifies local mode")
    parser.add_argument("--test", action="store_true", help="Specifies test mode")
    parser.add_argument(
        "--use-fit",
        action="store_true",
        help="If true, uses model.fit() instead of model.fit_generator()",
    )
    args = parser.parse_args()

    config = {
        "hyperparameters": {
            "global_batch_size": det.Constant(value=32),
            "dense1": det.Constant(value=128),
        },
        "searcher": {"name": "single", "metric": "val_accuracy", "max_steps": 40},
    }
    config.update(json.loads(args.config))

    context = init(config, local=args.local, test=args.test, context_dir=str(pathlib.Path.cwd()))

    train_images, train_labels = data.load_training_data()
    train_images = train_images / 255.0
    train_data = _ArrayLikeAdapter(
        x=train_images, y=train_labels, batch_size=context.get_per_slot_batch_size()
    )

    test_images, test_labels = data.load_validation_data()
    test_images = test_images / 255.0
    test_data = _ArrayLikeAdapter(
        x=test_images, y=test_labels, batch_size=context.get_per_slot_batch_size()
    )

    model = build_model(context)

    if args.use_fit:
        model.fit(
            x=train_images,
            y=train_labels,
            batch_size=context.get_per_slot_batch_size(),
            validation_data=test_data,
            use_multiprocessing=False,
            workers=1,
            max_queue_size=10,
            epochs=1,
        )
    else:
        model.fit_generator(
            generator=train_data,
            validation_data=test_data,
            use_multiprocessing=False,
            workers=1,
            max_queue_size=10,
            epochs=1,
        )
