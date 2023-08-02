from typing import Any

import numpy as np
import tensorflow as tf
from tensorflow import keras

import determined as det
import determined.keras


class RuntimeErrorTrial(det.keras.TFKerasTrial):
    """
    A model guaranteed to throw a runtime error, so we can check that native framework errors are
    surfaced properly.
    """

    _searcher_metric = "val_accuracy"

    def __init__(self, context: det.keras.TFKerasTrialContext) -> None:
        self.context = context

    def build_model(self) -> Any:
        model = keras.Sequential([keras.layers.Dense(10)])
        model = self.context.wrap_model(model)
        model.compile(
            # TODO MLG-443 Migrate from legacy Keras optimizers
            optimizer=tf.keras.optimizers.legacy.Adam(name="Adam"),
            loss=tf.keras.losses.SparseCategoricalCrossentropy(from_logits=True),
            metrics=[
                tf.keras.metrics.Accuracy()
            ],  # ERR: this is the wrong accuracy, should be SparseCategoricalAccuracy
        )
        return model

    def build_training_data_loader(self) -> Any:
        return np.zeros(1), np.zeros(1)

    def build_validation_data_loader(self) -> Any:
        return np.zeros(1), np.zeros(1)
