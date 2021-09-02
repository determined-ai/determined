import pathlib

import numpy as np
import tensorflow as tf
from tensorflow import keras

import determined as det
from determined import experimental
from determined.keras import TFKerasTrial, TFKerasTrialContext


class RuntimeErrorTrial(TFKerasTrial):
    """
    A model guaranteed to throw a runtime error, so we can check that native framework errors are
    surfaced properly.
    """

    def __init__(self, context: TFKerasTrialContext) -> None:
        self.context = context

    def build_model(self):
        model = keras.Sequential([keras.layers.Dense(10)])
        model = self.context.wrap_model(model)
        model.compile(
            optimizer=tf.keras.optimizers.Adam(name="Adam"),
            loss=tf.keras.losses.SparseCategoricalCrossentropy(from_logits=True),
            metrics=[
                tf.keras.metrics.Accuracy()
            ],  # ERR: this is the wrong accuracy, should be SparseCategoricalAccuracy
        )
        return model

    def build_training_data_loader(self):
        return np.zeros(1), np.zeros(1)

    def build_validation_data_loader(self):
        return np.zeros(1), np.zeros(1)


if __name__ == "__main__":
    experimental.create(
        trial_def=RuntimeErrorTrial,
        config={
            "description": "keras_runtime_error",
            "hyperparameters": {"global_batch_size": det.Constant(1)},
            "searcher": {"metric": "val_accuracy"},
            "data_layer": {"type": "lfs", "container_storage_path": "/tmp"},
        },
        local=True,
        test=True,
        context_dir=str(pathlib.Path.cwd()),
    )
