"""
A noop estimator which will pause for 15 seconds after the chief exits to test that the chief is
allowed to finish its post-terminate callback.
"""

import random
import time
from typing import List

import numpy as np
import tensorflow as tf
from packaging import version

from determined import estimator


def rand_rand_reducer(batch_metrics: List):
    return random.random()


def np_rand_reducer(batch_metrics: List):
    return np.random.random()


def tf_rand_reducer(batch_metrics: List):
    if version.parse(tf.__version__) >= version.parse("2.0.0"):
        return tf.random.get_global_generator().uniform
    else:
        return 0.0


class ChiefPauseOnTerminateRunHook(estimator.RunHook):
    def __init__(self, ctx):
        self.ctx = ctx

    def on_trial_close(self) -> None:
        if self.ctx.distributed.get_rank() == 0:
            time.sleep(15)
            print("rank 0 has completed on_trial_close")


class NoopEstimator(estimator.EstimatorTrial):
    def __init__(self, context: estimator.EstimatorTrialContext) -> None:
        self.context = context

    def create_model_fn(self):
        def noop_model_fn(features, labels, mode):
            if mode == tf.estimator.ModeKeys.EVAL:

                eval_metric_ops = {
                    name: self.context.experimental.make_metric(metric, reducer, np.float64)
                    for name, metric, reducer in [
                        ("rand_rand", tf.constant([[]]), rand_rand_reducer),
                        ("np_rand", tf.constant([[]]), np_rand_reducer),
                        ("tf_rand", tf.constant([[]]), tf_rand_reducer),
                    ]
                }

                return tf.estimator.EstimatorSpec(
                    mode, loss=tf.constant([0.0], dtype=tf.float32), eval_metric_ops=eval_metric_ops
                )

            if mode == tf.estimator.ModeKeys.TRAIN:
                return tf.estimator.EstimatorSpec(
                    mode,
                    loss=tf.constant([0.0], dtype=tf.float32),
                    train_op=tf.no_op(),
                )

        return noop_model_fn

    def build_estimator(self) -> tf.estimator.Estimator:
        _ = [tf.feature_column.numeric_column("x", shape=(), dtype=tf.int64)]
        optimizer = tf.compat.v1.train.GradientDescentOptimizer(learning_rate=0.0)
        _ = self.context.wrap_optimizer(optimizer)

        return tf.compat.v1.estimator.Estimator(model_fn=self.create_model_fn())

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        def fn():
            ds = tf.data.Dataset.range(100).batch(self.context.get_per_slot_batch_size())
            return self.context.wrap_dataset(ds)

        return tf.estimator.TrainSpec(fn, hooks=[ChiefPauseOnTerminateRunHook(self.context)])

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        def fn():
            ds = tf.data.Dataset.range(100).batch(self.context.get_per_slot_batch_size())
            return self.context.wrap_dataset(ds)

        return tf.estimator.EvalSpec(fn)
