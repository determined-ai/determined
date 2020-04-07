from typing import Callable, Dict, Tuple

import tensorflow as tf

from determined.estimator import (
    EstimatorNativeContext,
    EstimatorTrial,
    EstimatorTrialContext,
    ServingInputReceiverFn,
)
from tests.unit.frameworks.utils import xor_data


def xor_input_fn(
    context: EstimatorNativeContext, batch_size: int, repeat: bool = True, shuffle: bool = False
) -> Callable[[], Tuple[tf.Tensor, tf.Tensor]]:
    def _input_fn() -> Tuple[tf.Tensor, tf.Tensor]:
        data, labels = xor_data()
        dataset = tf.data.Dataset.from_tensor_slices((data, labels))
        dataset = context.wrap_dataset(dataset)
        if shuffle:
            dataset = dataset.shuffle(1000)
        dataset = dataset.batch(batch_size)
        if repeat:
            dataset = dataset.repeat()
        iterator = tf.compat.v1.data.make_one_shot_iterator(dataset)
        features, labels = iterator.get_next()

        tf.compat.v1.summary.tensor_summary("features", features)
        tf.compat.v1.summary.tensor_summary("labels", labels)

        return ({"input": features}, labels)

    return _input_fn


class XORTrial(EstimatorTrial):
    """
    Models a lightweight neural network model with one hidden layer to
    learn a binary XOR function. See Deep Learning Book, chapter 6.1 for
    the solution with a hidden size of 2, and a MSE loss function.
    """

    def __init__(self, context: EstimatorTrialContext) -> None:
        self.context = context

    def build_estimator(self) -> tf.estimator.Estimator:
        _input = tf.feature_column.numeric_column("input", shape=(2,), dtype=tf.int32)

        if self.context.get_hparam("optimizer") == "adam":
            optimizer = tf.compat.v1.train.AdamOptimizer(
                learning_rate=self.context.get_hparam("learning_rate")
            )
        elif self.context.get_hparam("optimizer") == "sgd":
            optimizer = tf.compat.v1.train.GradientDescentOptimizer(
                learning_rate=self.context.get_hparam("learning_rate")
            )
        else:
            raise NotImplementedError()
        optimizer = self.context.wrap_optimizer(optimizer)

        return tf.compat.v1.estimator.DNNClassifier(
            feature_columns=[_input],
            hidden_units=[self.context.get_hparam("hidden_size")],
            activation_fn=tf.nn.sigmoid,
            config=tf.estimator.RunConfig(
                session_config=tf.compat.v1.ConfigProto(
                    intra_op_parallelism_threads=1, inter_op_parallelism_threads=1
                )
            ),
            optimizer=optimizer,
        )

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        return tf.estimator.TrainSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=self.context.get_hparam("shuffle"),
                repeat=True,
            )
        )

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        return tf.estimator.EvalSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=False,
                repeat=False,
            )
        )

    def build_serving_input_receiver_fns(self) -> Dict[str, ServingInputReceiverFn]:
        _input = tf.feature_column.numeric_column("input", shape=(2,), dtype=tf.int64)
        return {
            "inference": tf.estimator.export.build_parsing_serving_input_receiver_fn(
                tf.feature_column.make_parse_example_spec([_input])
            )
        }
