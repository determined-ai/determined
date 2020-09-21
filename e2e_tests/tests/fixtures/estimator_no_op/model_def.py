"""
A noop estimator which will pause for 15 seconds after the chief exits to test that the chief is
allowed to finish its post-terminate callback.
"""

import time

import tensorflow as tf

from determined import estimator


def noop_model_fn(features, labels, mode):
    if mode == tf.estimator.ModeKeys.EVAL:
        return tf.estimator.EstimatorSpec(
            mode,
            loss=tf.constant([0.0], dtype=tf.float32),
        )

    if mode == tf.estimator.ModeKeys.TRAIN:
        return tf.estimator.EstimatorSpec(
            mode,
            loss=tf.constant([0.0], dtype=tf.float32),
            train_op=tf.no_op(),
        )


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

    def build_estimator(self) -> tf.estimator.Estimator:
        _ = [tf.feature_column.numeric_column("x", shape=(), dtype=tf.int64)]
        optimizer = tf.compat.v1.train.GradientDescentOptimizer(learning_rate=0.0)
        _ = self.context.wrap_optimizer(optimizer)

        return tf.compat.v1.estimator.Estimator(model_fn=noop_model_fn)

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
