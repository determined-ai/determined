"""
An example showing how to use a graph defined in low-level TensorFlow APIs in Determined.

We will be wrapping the TensorFlow graph in an Estimator and using Determined's EstimatorTrial.
"""
from typing import Any, Callable, Dict

import tensorflow.compat.v1 as tf
import tensorflow_datasets as tfds

from determined import estimator

NUM_CLASSES = 10


def calculate_logits(
    hparams: Dict[str, Any], images: tf.Tensor, training: bool
) -> tf.Tensor:
    """This example assumes you already have something like this written for defining your graph."""
    conv1 = tf.layers.conv2d(
        inputs=tf.cast(images, tf.float32),
        filters=hparams["n_filters_1"],
        kernel_size=[5, 5],
        padding="same",
        activation=tf.nn.relu,
    )
    pool1 = tf.layers.max_pooling2d(inputs=conv1, pool_size=[2, 2], strides=2)

    conv2 = tf.layers.conv2d(
        inputs=pool1,
        filters=hparams["n_filters_2"],
        kernel_size=[5, 5],
        padding="same",
        activation=tf.nn.relu,
    )
    pool2 = tf.layers.max_pooling2d(inputs=conv2, pool_size=[2, 2], strides=2)
    pool2_shape = pool2.get_shape().as_list()

    pool2_flat = tf.reshape(
        pool2, [-1, pool2_shape[1] * pool2_shape[2] * pool2_shape[3]]
    )
    dense = tf.layers.dense(inputs=pool2_flat, units=512, activation=tf.nn.relu)

    if training:
        dropout = tf.layers.dropout(inputs=dense, rate=0.5)
        logits = tf.layers.dense(inputs=dropout, units=NUM_CLASSES)
    else:
        logits = tf.layers.dense(inputs=dense, units=NUM_CLASSES)

    return logits


def calculate_loss(labels: tf.Tensor, logits: tf.Tensor) -> tf.Tensor:
    """This example assumes you already have something like this written for defining your graph."""
    return tf.reduce_mean(
        tf.nn.sparse_softmax_cross_entropy_with_logits(labels=labels, logits=logits)
    )


def calculate_predictions(logits: tf.Tensor) -> tf.Tensor:
    """This example assumes you already have something like this written for defining your graph."""
    return tf.argmax(logits, axis=1)


def calculate_error(predictions: tf.Tensor, labels: tf.Tensor) -> tf.Tensor:
    """This example assumes you already have something like this written for defining your graph."""
    correct = tf.cast(tf.equal(predictions, labels), tf.float32)
    return 1 - tf.reduce_mean(correct)


def make_model_fn(context: estimator.EstimatorTrialContext) -> Callable:
    # Define a model_fn which is the magic ingredient for wrapping a tensorflow graph in an
    # Estimator.  The Estimator training loop will call this function with different modes to
    # build graphs for either training or validation (or prediction, but that's not used by
    # Determined).
    #
    # Read more at https://www.tensorflow.org/guide/estimator.
    def model_fn(
        features: Any, mode: tf.estimator.ModeKeys
    ) -> tf.estimator.EstimatorSpec:
        # The "features" argument must be named "features", but in this simple example, it
        # contains the full output of our dataset, including the images and the labels.
        images = features["image"]
        labels = features["label"]

        if mode == tf.estimator.ModeKeys.TRAIN:
            # Build a graph for training.
            logits = calculate_logits(context.get_hparams(), images, training=True)
            loss = calculate_loss(labels, logits)

            learning_rate = context.get_hparam("learning_rate")
            optimizer = tf.train.AdamOptimizer(learning_rate=learning_rate)
            optimizer = context.wrap_optimizer(optimizer)

            train_op = optimizer.minimize(loss, global_step=tf.train.get_global_step())
            return tf.estimator.EstimatorSpec(mode, loss=loss, train_op=train_op)

        if mode == tf.estimator.ModeKeys.EVAL:
            # Build a graph for validation.
            logits = calculate_logits(context.get_hparams(), images, training=False)
            loss = calculate_loss(labels, logits)
            predictions = calculate_predictions(logits)
            error = calculate_error(predictions, labels)
            return tf.estimator.EstimatorSpec(
                mode,
                loss=loss,
                eval_metric_ops={"error": tf.metrics.mean(error)},
            )

    return model_fn


class MNistTrial(estimator.EstimatorTrial):
    def __init__(self, context: estimator.EstimatorTrialContext) -> None:
        self.context = context

    def build_estimator(self) -> tf.estimator.Estimator:
        return tf.estimator.Estimator(model_fn=make_model_fn(self.context))

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        # Write a function which returns your dataset for training...
        def input_fn() -> tf.data.Dataset:
            ds = tfds.image.MNIST().as_dataset()["train"]
            ds = self.context.wrap_dataset(ds)
            ds = ds.batch(self.context.get_per_slot_batch_size())
            return ds

        # ... then return a TrainSpec which includes that function.
        return tf.estimator.TrainSpec(input_fn)

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        # Write a function which returns your dataset for validation...
        def input_fn() -> tf.data.Dataset:
            ds = tfds.image.MNIST().as_dataset()["test"]
            ds = self.context.wrap_dataset(ds)
            ds = ds.batch(self.context.get_per_slot_batch_size())
            return ds

        # ... then return an EvalSpec which includes that function.
        return tf.estimator.EvalSpec(input_fn, steps=None)
