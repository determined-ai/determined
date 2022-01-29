"""
A one-variable proportional model that initializes its weight to zero, and whose data and labels
are always just 1.

Useful for testing metrics or gradient updates.
"""

import tensorflow as tf

from determined import estimator


def tf_data_loader(batch_size, length):
    one_ds = tf.data.Dataset.from_tensor_slices(tf.constant([1], tf.float32))

    data = one_ds.repeat(length).batch(batch_size)
    label = one_ds.repeat(length).batch(batch_size)

    return tf.data.Dataset.zip(({"x": data}, label))


def make_model_fn(feature_columns, optimizer):
    def custom_model_fn(features, labels, mode):
        input_layer = tf.compat.v1.feature_column.input_layer(features, feature_columns)
        dense = tf.compat.v1.layers.Dense(
            units=1,
            use_bias=False,
            kernel_initializer=tf.zeros_initializer(),
            name="my_dense",
        )
        output_layer = dense(input_layer)
        predictions = tf.squeeze(output_layer, 1)

        if mode == tf.estimator.ModeKeys.PREDICT:
            return tf.estimator.EstimatorSpec(mode, predictions=predictions)

        loss = tf.losses.mean_squared_error(labels, predictions)

        if mode == tf.estimator.ModeKeys.EVAL:
            return tf.estimator.EstimatorSpec(
                mode,
                loss=loss,
            )

        if mode == tf.estimator.ModeKeys.TRAIN:
            train_op = optimizer.minimize(loss, global_step=tf.compat.v1.train.get_global_step())
            return tf.estimator.EstimatorSpec(
                mode,
                loss=loss,
                train_op=train_op,
            )

    return custom_model_fn


class MyLinearEstimator(estimator.EstimatorTrial):
    def __init__(self, context: estimator.EstimatorTrialContext) -> None:
        self.context = context
        self.hparams = context.get_hparams()
        self.batch_size = self.context.get_per_slot_batch_size()

        self.dense = None

    def build_estimator(self) -> tf.estimator.Estimator:
        feature_columns = [tf.feature_column.numeric_column("x", shape=(), dtype=tf.int64)]
        optimizer = tf.compat.v1.train.GradientDescentOptimizer(
            learning_rate=self.hparams["learning_rate"],
        )
        optimizer = self.context.wrap_optimizer(optimizer)

        estimator = tf.compat.v1.estimator.Estimator(
            model_fn=make_model_fn(feature_columns, optimizer)
        )

        return estimator

    # You need a build_serving_input_receiver_fns to use estimators with the checkpoint export APIs.
    def build_serving_input_receiver_fns(self):
        input_column = tf.feature_column.numeric_column("x", shape=(), dtype=tf.int64)
        return {
            "intput_column": tf.estimator.export.build_parsing_serving_input_receiver_fn(
                tf.feature_column.make_parse_example_spec([input_column])
            )
        }

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        def fn():
            ds = tf_data_loader(self.context.get_per_slot_batch_size(), 100)
            return self.context.wrap_dataset(ds)

        return tf.estimator.TrainSpec(fn)

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        def fn():
            ds = tf_data_loader(self.context.get_per_slot_batch_size(), 10)
            return self.context.wrap_dataset(ds)

        return tf.estimator.EvalSpec(fn, hooks=[])
