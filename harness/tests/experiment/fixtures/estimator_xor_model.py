from typing import Callable, Dict, Tuple, Union

import tensorflow as tf

from determined.estimator import (
    EstimatorNativeContext,
    EstimatorTrial,
    EstimatorTrialContext,
    ServingInputReceiverFn,
)
from tests.experiment.utils import xor_data


def xor_input_fn(
    context: Union[EstimatorNativeContext, EstimatorTrialContext],
    batch_size: int,
    shuffle: bool = False,
) -> Callable[[], Tuple[tf.Tensor, tf.Tensor]]:
    def _input_fn() -> Tuple[tf.Tensor, tf.Tensor]:
        data, labels = xor_data()
        dataset = tf.data.Dataset.from_tensor_slices((data, labels))
        dataset = context.wrap_dataset(dataset)
        if shuffle:
            dataset = dataset.shuffle(1000)

        def map_dataset(x, y):
            return {"input": x}, y

        dataset = dataset.batch(batch_size)
        dataset = dataset.map(map_dataset)

        return dataset

    return _input_fn


def xor_input_fn_data_layer(
    context: Union[EstimatorNativeContext, EstimatorTrialContext],
    training: bool,
    batch_size: int,
    shuffle: bool = False,
) -> Callable[[], Tuple[tf.Tensor, tf.Tensor]]:
    def _input_fn() -> Tuple[tf.Tensor, tf.Tensor]:
        cacheable = (
            context.experimental.cache_train_dataset
            if training
            else context.experimental.cache_validation_dataset
        )

        @cacheable("xor_input_fn_data_layer", "xor_data", shuffle=shuffle)
        def make_dataset() -> tf.data.Dataset:
            data, labels = xor_data()
            ds = tf.data.Dataset.from_tensor_slices((data, labels))
            return ds

        dataset = make_dataset()

        def map_dataset(x, y):
            return {"input": x}, y

        dataset = dataset.batch(batch_size)
        dataset = dataset.map(map_dataset)

        return dataset

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
            )
        )

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        return tf.estimator.EvalSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=False,
            )
        )

    def build_serving_input_receiver_fns(self) -> Dict[str, ServingInputReceiverFn]:
        _input = tf.feature_column.numeric_column("input", shape=(2,), dtype=tf.int64)
        return {
            "inference": tf.estimator.export.build_parsing_serving_input_receiver_fn(
                tf.feature_column.make_parse_example_spec([_input])
            )
        }


class XORTrialDataLayer(XORTrial):
    def build_train_spec(self) -> tf.estimator.TrainSpec:
        return tf.estimator.TrainSpec(
            xor_input_fn_data_layer(
                context=self.context,
                training=True,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=self.context.get_hparam("shuffle"),
            )
        )

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        return tf.estimator.EvalSpec(
            xor_input_fn_data_layer(
                context=self.context,
                training=False,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=False,
            )
        )
