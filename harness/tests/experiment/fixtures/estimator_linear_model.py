from typing import Any, List

import numpy as np
import tensorflow as tf

from determined import estimator

TRAINING_LENGTH = 100
VALIDATION_LENGTH = 10


def validation_label_sum():
    """The custom metrics return a sum of labels of the validation dataset."""
    return sum(range(VALIDATION_LENGTH))


def range_data_loader(batch_size, length):
    """Return a dataloader that yields tuples like ({"x": val}, val) for LinearEstimator."""
    data = tf.data.Dataset.range(length).map(lambda x: tf.cast(x, tf.float32)).batch(batch_size)
    label = tf.data.Dataset.range(length).map(lambda x: tf.cast(x, tf.float32)).batch(batch_size)
    return tf.data.Dataset.zip(({"x": data}, label))


def sum_reducer(batch_metrics: List):
    """A function that is able to operate as a custom reducer."""
    return np.hstack(batch_metrics).sum()


class SumReducer(estimator.MetricReducer):
    """A class that is able to operate as a custom reducer."""

    def __init__(self):
        self.sum = 0

    def accumulate(self, metric: Any):
        self.sum += metric.sum()
        return self.sum

    def cross_slot_reduce(self, per_slot_metrics: List):
        return sum(per_slot_metrics)


class LinearEstimator(estimator.EstimatorTrial):
    def __init__(self, context: estimator.EstimatorTrialContext) -> None:
        self.context = context
        self.hparams = context.get_hparams()
        self.batch_size = self.context.get_per_slot_batch_size()

        self.dense = None

    def make_model_fn(self, feature_columns, optimizer):
        """Return a one variable linear model.  Used by LinearEstimator."""

        def model_fn(features, labels, mode):
            input_layer = tf.compat.v1.feature_column.input_layer(features, feature_columns)
            dense = tf.compat.v1.layers.Dense(
                units=1, use_bias=False, kernel_initializer=tf.zeros_initializer(), name="my_dense",
            )
            output_layer = dense(input_layer)
            predictions = tf.squeeze(output_layer, 1)

            if mode == tf.estimator.ModeKeys.PREDICT:
                return tf.estimator.EstimatorSpec(mode, predictions=predictions)

            loss = tf.losses.mean_squared_error(labels, predictions)

            if mode == tf.estimator.ModeKeys.EVAL:
                # Use the custom metrics API.
                fn_sum = self.context.experimental.make_metric(labels, sum_reducer, np.float32)
                cls_sum = self.context.experimental.make_metric(labels, SumReducer(), np.float32)

                return tf.estimator.EstimatorSpec(
                    mode,
                    loss=loss,
                    eval_metric_ops={"label_sum_fn": fn_sum, "label_sum_cls": cls_sum},
                )

            if mode == tf.estimator.ModeKeys.TRAIN:
                train_op = optimizer.minimize(
                    loss, global_step=tf.compat.v1.train.get_global_step()
                )
                return tf.estimator.EstimatorSpec(mode, loss=loss, train_op=train_op)

        return model_fn

    def build_estimator(self) -> tf.compat.v1.estimator.Estimator:
        feature_columns = [tf.feature_column.numeric_column("x", shape=(), dtype=tf.int64)]
        optimizer = tf.compat.v1.train.GradientDescentOptimizer(
            learning_rate=self.hparams["learning_rate"],
        )
        optimizer = self.context.wrap_optimizer(optimizer)

        estimator = tf.compat.v1.estimator.Estimator(
            model_fn=self.make_model_fn(feature_columns, optimizer)
        )

        return estimator

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        def fn():
            ds = range_data_loader(self.context.get_per_slot_batch_size(), TRAINING_LENGTH)
            return self.context.wrap_dataset(ds)

        return tf.estimator.TrainSpec(fn)

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        def fn():
            ds = range_data_loader(self.context.get_per_slot_batch_size(), VALIDATION_LENGTH)
            return self.context.wrap_dataset(ds)

        return tf.estimator.EvalSpec(fn)
