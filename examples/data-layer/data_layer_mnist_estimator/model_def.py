"""
Trains a simple DNN on the MNIST dataset using the TensorFlow Estimator API.
"""
import os
from typing import Callable, Dict, List, Tuple

import tensorflow as tf

from determined.estimator import EstimatorTrial, EstimatorTrialContext

import util

IMAGE_SIZE = 28
NUM_CLASSES = 10


def get_filenames(directory: str) -> List[str]:
    return [os.path.join(directory, path) for path in tf.io.gfile.listdir(directory)]


def parse_mnist_tfrecord(serialized_example: tf.Tensor) -> Tuple[Dict[str, tf.Tensor], tf.Tensor]:
    """
    Parse a TFRecord representing a single MNIST data point into an input
    feature tensor and a label tensor.

    Returns: (features: Dict[str, Tensor], label: Tensor)
    """
    raw = tf.io.parse_example(
        serialized=serialized_example, features={"image_raw": tf.io.FixedLenFeature([], tf.string)}
    )
    image = tf.io.decode_raw(raw["image_raw"], tf.float32)

    label_dict = tf.io.parse_example(
        serialized=serialized_example, features={"label": tf.io.FixedLenFeature(1, tf.int64)}
    )
    return {"image": image}, label_dict["label"]


class MNistTrial(EstimatorTrial):
    def __init__(self, context: EstimatorTrialContext):
        self.context = context

    def build_estimator(self) -> tf.estimator.Estimator:
        optimizer = tf.compat.v1.train.AdamOptimizer(
            learning_rate=self.context.get_hparam("learning_rate"),
        )
        # Call `wrap_optimizer` immediately after creating your optimizer.
        optimizer = self.context.wrap_optimizer(optimizer)
        return tf.compat.v1.estimator.DNNClassifier(
            feature_columns=[
                tf.feature_column.numeric_column(
                    "image", shape=(IMAGE_SIZE, IMAGE_SIZE, 1), dtype=tf.float32
                )
            ],
            n_classes=NUM_CLASSES,
            hidden_units=[
                self.context.get_hparam("hidden_layer_1"),
                self.context.get_hparam("hidden_layer_2"),
                self.context.get_hparam("hidden_layer_3"),
            ],
            config=tf.estimator.RunConfig(tf_random_seed=self.context.get_trial_seed()),
            optimizer=optimizer,
            dropout=self.context.get_hparam("dropout"),
        )

    def _make_train_input_fn(self) -> Callable:
        def _fn() -> tf.data.TFRecordDataset:
            @self.context.experimental.cache_train_dataset(
                "mnist-estimator-const", "v1", shuffle=True
            )
            def make_dataset() -> tf.data.TFRecordDataset:
                download_directory = util.download_data("/tmp/data")

                files = get_filenames(os.path.join(download_directory, "train"))
                ds = tf.data.TFRecordDataset(files)
                return ds

            dataset = make_dataset()
            dataset = dataset.batch(self.context.get_per_slot_batch_size())
            dataset = dataset.map(parse_mnist_tfrecord)

            return dataset

        return _fn

    def _make_validation_input_fn(self) -> Callable:
        def _fn() -> tf.data.TFRecordDataset:
            @self.context.experimental.cache_validation_dataset("mnist-estimator-const", "v1")
            def make_dataset() -> tf.data.TFRecordDataset:
                download_directory = util.download_data("/tmp/data")
                files = get_filenames(os.path.join(download_directory, "validation"))
                ds = tf.data.TFRecordDataset(files)
                return ds

            dataset = make_dataset()
            dataset = dataset.batch(self.context.get_per_slot_batch_size())
            dataset = dataset.map(parse_mnist_tfrecord)

            return dataset

        return _fn

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        return tf.estimator.TrainSpec(self._make_train_input_fn())

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        return tf.estimator.EvalSpec(self._make_validation_input_fn())
