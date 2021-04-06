"""
Trains a simple DNN on the MNIST dataset using the TensorFlow Estimator API.
"""
import logging
import os
import tarfile
import requests
from typing import Callable, Dict, List, Tuple

import tensorflow as tf

from determined.estimator import EstimatorTrial, EstimatorTrialContext, ServingInputReceiverFn


WORK_DIRECTORY = "/tmp/determined-mnist-estimator-work-dir"
MNIST_TF_RECORDS_FILE = "mnist-tfrecord.tar.gz"
MNIST_TF_RECORDS_URL = (
    "https://s3-us-west-2.amazonaws.com/determined-ai-test-data/" + MNIST_TF_RECORDS_FILE
)

IMAGE_SIZE = 28
NUM_CLASSES = 10


def download_data(download_directory) -> str:
    """
    Return the path of a directory with the MNIST dataset in TFRecord format.
    The dataset will be downloaded into download_directory, if it is not already
    present.
    """
    if not tf.io.gfile.exists(download_directory):
        tf.io.gfile.makedirs(download_directory)

    filepath = os.path.join(download_directory, MNIST_TF_RECORDS_FILE)
    if not tf.io.gfile.exists(filepath):
        logging.info("Downloading {}".format(MNIST_TF_RECORDS_URL))

        r = requests.get(MNIST_TF_RECORDS_URL)
        with tf.io.gfile.GFile(filepath, "wb") as f:
            f.write(r.content)
            logging.info("Downloaded {} ({} bytes)".format(MNIST_TF_RECORDS_FILE, f.size()))

        logging.info("Extracting {} to {}".format(MNIST_TF_RECORDS_FILE, download_directory))
        with tarfile.open(filepath, mode="r:gz") as f:
            f.extractall(path=download_directory)

    data_dir = os.path.join(download_directory, "mnist-tfrecord")
    assert tf.io.gfile.exists(data_dir)
    return data_dir


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
    def __init__(self, context: EstimatorTrialContext) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

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

    def _input_fn(self, files: List[str], shuffle: bool = False) -> Callable:
        def _fn() -> tf.data.TFRecordDataset:
            dataset = tf.data.TFRecordDataset(files)
            # Call `wrap_dataset` immediately after creating your dataset.
            dataset = self.context.wrap_dataset(dataset)
            if shuffle:
                dataset = dataset.shuffle(1000)
            dataset = dataset.batch(self.context.get_per_slot_batch_size())
            dataset = dataset.map(parse_mnist_tfrecord)
            return dataset

        return _fn

    # The serving input receiver is used when the model is serialized in the
    # tensorflow saved_model format. This function defines the input the model
    # expects when it is loaded from disk for inference purposes. Without this
    # function the model checkpoint will only contain the weights of the model
    # and no saved_model will be saved.
    def build_serving_input_receiver_fns(self) -> Dict[str, ServingInputReceiverFn]:
        input_column = tf.feature_column.numeric_column(
            "image", shape=(IMAGE_SIZE, IMAGE_SIZE, 1), dtype=tf.float32
        )
        return {
            "mnist_parsing": tf.estimator.export.build_parsing_serving_input_receiver_fn(
                tf.feature_column.make_parse_example_spec([input_column])
            )
        }

    @staticmethod
    def _get_filenames(directory: str) -> List[str]:
        return [os.path.join(directory, path) for path in tf.io.gfile.listdir(directory)]

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        if not self.data_downloaded:
            self.download_directory = download_data(download_directory=self.download_directory)
            self.data_downloaded = True

        train_files = self._get_filenames(os.path.join(self.download_directory, "train"))
        return tf.estimator.TrainSpec(self._input_fn(train_files, shuffle=True))

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        if not self.data_downloaded:
            self.download_directory = download_data(download_directory=self.download_directory)
            self.data_downloaded = True

        val_files = self._get_filenames(os.path.join(self.download_directory, "validation"))
        return tf.estimator.EvalSpec(self._input_fn(val_files, shuffle=False), steps=None)
