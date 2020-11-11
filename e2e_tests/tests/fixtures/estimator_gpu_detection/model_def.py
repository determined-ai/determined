"""
An example of training a Tensorpack model using estimators inside Determined.
The unique code for this is `model_fn()` which takes a (largely unchanged)
Tensorpack model and creates an EstimatorSpec. Data reading is independent
of the model definition using Tensorpack code.
"""

from typing import Any, Dict, List

import tensorflow as tf
import tensorpack
from tensorflow.python.keras.backend import _preprocess_conv2d_input

from determined.estimator import EstimatorTrial, EstimatorTrialContext


class Model(tensorpack.ModelDesc):  # type: ignore
    """
    Model code is taken directly from Tensorpack examples with the slight
    modifications of using `tf.reshape` instead of `tf.extend_dims`.
    """

    def __init__(self, hparams: Dict[str, Any]) -> None:
        self.hparams = hparams
        self.image_size = 28

    def inputs(self) -> List[tf.TensorSpec]:
        """
        Define all the inputs (shape, data_type, name) that the graph will need.
        """
        return [
            tf.TensorSpec((None, self.image_size, self.image_size), tf.float32, "input"),
            tf.TensorSpec((None,), tf.int32, "label"),
        ]

    def build_graph(self, image: Any, label: Any) -> Any:
        """
        This function builds the model which takes the input
        variables and returns cost.
        """

        # In tensorflow, inputs to convolution function are assumed to be NHWC.
        # Add a single channel here.
        image = tf.reshape(image, [-1, self.image_size, self.image_size, 1])

        # Center the pixels values at zero.
        # tf.summary.image("input", (tf.expand_dims(og_image * 2 - 1, 3) + 1.0) * 128.0)
        image = image * 2 - 1

        # The context manager `argscope` sets the default option for all the layers under
        # this context. Here we use 32 channel convolution with shape 3x3.
        with tensorpack.argscope(
            tensorpack.Conv2D,
            kernel_size=3,
            activation=tf.nn.relu,
            filters=self.hparams["n_filters"],
        ):
            c0 = tensorpack.Conv2D("conv0", image)
            p0 = tensorpack.MaxPooling("pool0", c0, 2)
            c1 = tensorpack.Conv2D("conv1", p0)
            c2 = tensorpack.Conv2D("conv2", c1)
            p1 = tensorpack.MaxPooling("pool1", c2, 2)
            c3 = tensorpack.Conv2D("conv3", p1)
            fc1 = tensorpack.FullyConnected("fc0", c3, 512, nl=tf.nn.relu)
            fc1 = tensorpack.Dropout("dropout", fc1, 0.5)
            logits = tensorpack.FullyConnected("fc1", fc1, out_dim=10, nl=tf.identity)

        # This line will cause Tensorflow to detect GPU usage. If session is not properly
        # configured it causes multi-GPU runs to crash.
        _preprocess_conv2d_input(image, "channels_first")

        label = tf.reshape(label, [-1])
        cost = tf.nn.sparse_softmax_cross_entropy_with_logits(logits=logits, labels=label)
        cost = tf.reduce_mean(cost, name="cross_entropy_loss")  # the average cross-entropy loss

        correct = tf.cast(
            tf.nn.in_top_k(predictions=logits, targets=label, k=1), tf.float32, name="correct"
        )
        accuracy = tf.reduce_mean(correct, name="accuracy")
        train_error = tf.reduce_mean(1 - correct, name="train_error")
        tensorpack.summary.add_moving_summary(train_error, accuracy)

        # Use a regex to find parameters to apply weight decay.
        # Here we apply a weight decay on all W (weight matrix) of all fc layers.
        wd_cost = tf.multiply(
            self.hparams["weight_cost"],
            tensorpack.regularize_cost("fc.*/W", tf.nn.l2_loss),
            name="regularize_loss",
        )
        total_cost = tf.add_n([wd_cost, cost], name="total_cost")

        return total_cost

    def optimizer(self) -> Any:
        lr = tf.train.exponential_decay(
            learning_rate=self.hparams["base_learning_rate"],
            global_step=tensorpack.get_global_step_var(),
            decay_steps=self.hparams["decay_steps"],
            decay_rate=self.hparams["decay_rate"],
            staircase=True,
            name="learning_rate",
        )
        tf.summary.scalar("lr", lr)
        return tf.train.AdamOptimizer(lr)


def make_model_fn(context):
    def model_fn(features, labels, mode, config, params):
        """
        Configure Tensorpack model to be trained inside tf.estimators.
        """
        model = Model(context.get_hparams())

        with tensorpack.tfutils.tower.TowerContext(
            "", is_training=mode == tf.estimator.ModeKeys.TRAIN
        ):
            loss = model.build_graph(features, labels)

        train_op = None
        if mode == tf.estimator.ModeKeys.TRAIN:
            optimizer = context.wrap_optimizer(model.optimizer())
            train_op = tf.contrib.layers.optimize_loss(
                loss=loss,
                global_step=tf.train.get_or_create_global_step(),
                learning_rate=None,
                optimizer=optimizer,
            )
        return tf.estimator.EstimatorSpec(mode=mode, loss=loss, train_op=train_op)

    return model_fn


class MnistTensorpackInEstimator(EstimatorTrial):
    def __init__(self, context: EstimatorTrialContext) -> None:
        self.context = context

    def build_estimator(self) -> tf.estimator.Estimator:
        estimator = tf.estimator.Estimator(
            model_fn=make_model_fn(self.context), config=None, params=None
        )
        return estimator

    def _dataflow_to_dataset(self, train: bool) -> tf.data.Dataset:
        input_dataflow = tensorpack.dataflow.FakeData([[28, 28], [1]], size=1000)
        input_dataset = tensorpack.input_source.TFDatasetInput.dataflow_to_dataset(
            input_dataflow, types=[tf.float32, tf.int64]
        )
        input_dataset = self.context.wrap_dataset(input_dataset)
        if train:
            input_dataset = input_dataset.apply(tf.data.experimental.shuffle_and_repeat(1000))
        input_dataset = input_dataset.batch(self.context.get_per_slot_batch_size())
        return input_dataset

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        return tf.estimator.TrainSpec(lambda: self._dataflow_to_dataset(True))

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        return tf.estimator.EvalSpec(lambda: self._dataflow_to_dataset(False))
