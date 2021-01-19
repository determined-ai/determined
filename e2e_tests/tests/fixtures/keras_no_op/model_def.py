import random

import numpy as np
import tensorflow as tf
from packaging import version

from determined.keras import TFKerasTrial, TFKerasTrialContext


class RandomMetric(tf.keras.metrics.Metric):
    def update_state(self, *args, **kwargs):
        return None

    def result(self):
        def my_func(x):
            return random.random()

        return tf.compat.v1.py_func(my_func, [tf.ones([1], dtype=tf.float64)], tf.float64)


class NumPyRandomMetric(tf.keras.metrics.Metric):
    def update_state(self, *args, **kwargs):
        return None

    def result(self):
        def my_func(x):
            return np.random.random()

        return tf.compat.v1.py_func(my_func, [tf.ones([1], dtype=tf.float64)], tf.float64)


class TensorFlowRandomMetric(tf.keras.metrics.Metric):
    def update_state(*args, **kargs):
        pass

    def result(self):
        def my_func(x):
            if version.parse(tf.__version__) >= version.parse("2.0.0"):
                return tf.random.get_global_generator().uniform()
            else:
                return 0.0

        return tf.compat.v1.py_func(my_func, [tf.ones([1], dtype=tf.float64)], tf.float64)


class NoopKerasTrial(TFKerasTrial):
    def __init__(self, context: TFKerasTrialContext):
        self.context = context

    def build_model(self):
        model = tf.keras.Sequential(
            [
                tf.keras.layers.Dense(
                    8,
                    input_shape=(
                        8,
                        8,
                    ),
                )
            ]
        )
        model = self.context.wrap_model(model)
        optimizer = self.context.wrap_optimizer(tf.keras.optimizers.SGD())
        model.compile(
            loss=tf.keras.losses.MeanSquaredError(),
            optimizer=optimizer,
            metrics=[
                RandomMetric(name="rand_rand"),
                NumPyRandomMetric(name="np_rand"),
                TensorFlowRandomMetric(name="tf_rand"),
            ],
        )
        return model

    def build_training_data_loader(self):
        x_train = np.ones((64, 8, 8))
        y_train = np.ones((64, 8, 8))
        return (x_train, y_train)

    def build_validation_data_loader(self):
        x_val = np.ones((64, 8, 8))
        y_val = np.ones((64, 8, 8))
        return (x_val, y_val)
