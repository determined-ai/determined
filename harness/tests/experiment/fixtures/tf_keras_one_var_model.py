from typing import Any, Dict, List, cast

import numpy as np
import tensorflow as tf
from tensorflow.keras import layers, losses, models
from tensorflow.keras.optimizers import legacy  # TODO MLG-443

from determined import keras
from tests.experiment.fixtures import keras_cb_checker


def make_one_var_tf_dataset_loader(hparams: Dict[str, Any], batch_size: int):
    dataset_range = hparams["dataset_range"]

    xtrain = tf.data.Dataset.range(dataset_range).batch(batch_size)
    ytrain = tf.data.Dataset.range(dataset_range).batch(batch_size)

    train_ds = tf.data.Dataset.zip((xtrain, ytrain))
    return train_ds


class OneVarTrial(keras.TFKerasTrial):
    """
    Models a simple one variable(y = wx) neural network, and a MSE loss function.
    """

    _searcher_metric = "loss"

    def __init__(self, context: keras.TFKerasTrialContext):
        self.context = context
        self.my_batch_size = self.context.get_per_slot_batch_size()
        self.my_learning_rate = self.context.get_hparam("learning_rate")

    def build_training_data_loader(self) -> keras.InputData:
        dataset = make_one_var_tf_dataset_loader(self.context.get_hparams(), self.my_batch_size)
        dataset = self.context.wrap_dataset(dataset)
        return dataset

    def build_validation_data_loader(self) -> keras.InputData:
        dataset = make_one_var_tf_dataset_loader(self.context.get_hparams(), self.my_batch_size)
        dataset = self.context.wrap_dataset(dataset)
        return dataset

    def build_model(self) -> models.Sequential:
        model = models.Sequential()
        model.add(
            layers.Dense(
                1, activation=None, use_bias=False, kernel_initializer="zeros", input_shape=(1,)
            )
        )
        model = self.context.wrap_model(model)
        model.compile(legacy.SGD(lr=self.my_learning_rate), losses.mean_squared_error)
        return cast(models.Sequential, model)

    @staticmethod
    def calc_gradient(w: float, values: List[float]) -> float:
        # Calculate what the gradient should be for a given weight and batch of
        # input values.

        # model:            yhat = w*x
        # loss:             L = (ytrue - w*x)**2
        # gradient:         dL/dw = -2*x*(ytrue - w*x)
        # let ytrue = x:    dL/dw = -2*x*x*(1-w)

        # We know that TensorFlow averages gradients across a batch, so we take
        # the mean of the gradient from each value in `values`.
        return np.mean([-2 * v * v * (1 - w) for v in values])

    @staticmethod
    def calc_loss(w, values):
        return np.mean([(v - w * v) ** 2 for v in values])

    def keras_callbacks(self):
        epochs = self.context.get_hparams().get("epochs")
        validations = self.context.get_hparams().get("validations")
        # Include a bunch of callbacks just to make sure they work.
        # EarlyStopping changed in TF 2.5 to stop unconditionally
        # if patience=0
        return [
            keras_cb_checker.CBChecker(epochs=epochs, validations=validations),
            keras.callbacks.TensorBoard(),
            keras.callbacks.ReduceLROnPlateau(monitor="val_loss"),
            keras.callbacks.EarlyStopping(restore_best_weights=True, patience=1),
        ]
        return
