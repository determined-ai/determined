"""
A model for debugging the execution of TensorFlow distributed training.

It tests sends and receives of sparse tensors and that batches are distributed
appropriately. The actual model is not so important to us. What we are mainly
looking for is something that produces deterministic results and that we can
separately verify the metrics of independent of any particular deep learning
framework.

The model has a single trainable parameter, `w`, which is used as the
coefficient of a linear mapping; the dataset is a map from `x` to `2x` for `x`
starting at 0 and incrementing.

For an input value x, the error is therefore `(2 - w) x`, so the loss is `(2 -
w)^2 x^2` and the negative gradient (and hence the update to `w`, because the
learning rate is 1) is `2 (2 - w) x^2`.

The mean loss and update to `w` for a batch are therefore the mean of the
squares of the values of the batch (`msq` below) multiplied by, respectively,
`(2 - w)^2` and `2 (2 - w)`.

For a batch size of 4, the first few batches look like this:

batch      msq   w     loss      update
-----------------------------------------
0,1,2,3    3.5   0     14        14
4,5,6,7    31.5  14    4536      -756
8,9,10,11  91.5  -742  50648544  136152

TODO(DET-616): The TensorFlow-computed losses for batches thereafter diverge
from the analytical calculations. Replace this model with a more robust one.

"""
from typing import Any, List

import numpy as np
import tensorflow as tf

from determined import estimator


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


class EstimatorDatasetTrial(estimator.EstimatorTrial):
    def __init__(self, context: estimator.EstimatorTrialContext):
        self.context = context
        self.hparams = context.get_hparams()

    def debug_print(self, *args):
        if self.hparams["print"]:
            return tf.print(*args)
        return tf.no_op()

    def model_fn(self, features, labels, mode):
        w = tf.Variable(np.float64(0.0), name="w", dtype=tf.float64, trainable=True)

        output = tf.math.multiply(w, features["x"])
        sparse = features["sparse"]
        shape = tf.shape(input=sparse)
        prod = tf.sparse.sparse_dense_matmul(
            sparse, tf.transpose(a=tf.ones(shape=shape, dtype=tf.float64))
        )
        loss = tf.compat.v1.losses.mean_squared_error(labels, output)

        print_input = self.debug_print("Input", w, features, labels)
        print_output = self.debug_print("Output", output, prod)
        print_loss = self.debug_print("Loss", loss)

        with tf.control_dependencies([print_input, print_output, print_loss]):
            loss = tf.identity(loss)

        opt = self.context.wrap_optimizer(
            tf.compat.v1.train.GradientDescentOptimizer(learning_rate=self.hparams["lr"])
        )
        train_op = opt.minimize(loss=loss, global_step=tf.compat.v1.train.get_global_step())

        eval_metrics_ops = None
        if mode == tf.estimator.ModeKeys.EVAL:
            # Use the custom metrics API.
            fn_sum = self.context.make_metric(labels, sum_reducer, np.float32)
            cls_sum = self.context.make_metric(labels, SumReducer(), np.float32)

            eval_metrics_ops = {"label_sum_fn": fn_sum, "label_sum_cls": cls_sum}

        return tf.estimator.EstimatorSpec(
            mode=mode,
            loss=loss,
            train_op=train_op,
            predictions={"output": output, "prod": prod},
            eval_metric_ops=eval_metrics_ops,
        )

    def build_estimator(self):
        return tf.estimator.Estimator(
            model_fn=self.model_fn,
            config=tf.estimator.RunConfig(
                session_config=tf.compat.v1.ConfigProto(
                    log_device_placement=False, allow_soft_placement=True
                )
            ),
        )

    def make_input(self, dataset_size, batch_size):
        """
        Make a dataset that exposes interesting edge cases:
        - Sparse tensors
        - User-defined map functions
        """

        def fn():
            size = int(dataset_size)
            x = np.arange(dataset_size, dtype=np.float64)
            # Make the identity matrix but sparse.
            sparse = tf.sparse.SparseTensor(
                indices=[[idx, int(i)] for idx, i in enumerate(x)],
                values=np.ones(size, dtype=np.float64),
                dense_shape=[size, size],
            )
            features = tf.data.Dataset.from_tensor_slices({"x": x, "sparse": sparse})
            labels = tf.data.Dataset.from_tensor_slices(2 * x)
            features, labels = (
                self.context.wrap_dataset(features),
                self.context.wrap_dataset(labels),
            )

            ds = tf.data.Dataset.zip((features, labels))
            ds = ds.map(lambda x, y: (x, y))
            ds = ds.batch(batch_size)
            ds = ds.repeat()
            return ds

        return fn

    def build_train_spec(self):
        return tf.estimator.TrainSpec(
            self.make_input(
                self.context.get_hparam("dataset_size"), self.context.get_per_slot_batch_size()
            )
        )

    def build_validation_spec(self):
        return tf.estimator.EvalSpec(
            self.make_input(
                self.context.get_hparam("dataset_size"), self.context.get_per_slot_batch_size()
            ),
            steps=self.context.get_hparam("validation_size")
            // self.context.get_global_batch_size(),
        )
