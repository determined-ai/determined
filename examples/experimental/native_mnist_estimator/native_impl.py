"""
This example demonstrates training a simple DNN with tf.estimator using the Determined
Native API.
"""
import argparse
import json
import logging
import os
import pathlib
import tarfile
from typing import Callable, Dict, List, Tuple

import requests
import tensorflow as tf

import determined as det
from determined.estimator import EstimatorNativeContext, init

WORK_DIRECTORY = "/tmp/determined-mnist-estimator-work-dir"
MNIST_TF_RECORDS_FILE = "mnist-tfrecord.tar.gz"
MNIST_TF_RECORDS_URL = (
    "https://s3-us-west-2.amazonaws.com/determined-ai-test-data/" + MNIST_TF_RECORDS_FILE
)


IMAGE_SIZE = 28
NUM_CLASSES = 10


def download_mnist_tfrecords(download_directory) -> str:
    """
    Return the path of a directory with the MNIST dataset in TFRecord format.
    The dataset will be downloaded into WORK_DIRECTORY, if it is not already
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


def build_estimator(context: EstimatorNativeContext) -> tf.estimator.Estimator:
    optimizer = tf.compat.v1.train.AdamOptimizer(learning_rate=context.get_hparam("learning_rate"))
    # Call `wrap_optimizer` immediately after creating your optimizer.
    optimizer = context.wrap_optimizer(optimizer)
    return tf.compat.v1.estimator.DNNClassifier(
        feature_columns=[
            tf.feature_column.numeric_column(
                "image", shape=(IMAGE_SIZE, IMAGE_SIZE, 1), dtype=tf.float32
            )
        ],
        n_classes=NUM_CLASSES,
        hidden_units=[
            context.get_hparam("hidden_layer_1"),
            context.get_hparam("hidden_layer_2"),
            context.get_hparam("hidden_layer_3"),
        ],
        optimizer=optimizer,
        dropout=context.get_hparam("dropout"),
    )


def input_fn(
    context: EstimatorNativeContext, files: List[str], shuffle_and_repeat: bool = False
) -> Callable:
    def _fn() -> tf.data.TFRecordDataset:
        dataset = tf.data.TFRecordDataset(files)
        # Call `wrap_dataset` immediately after creating your dataset.
        dataset = context.wrap_dataset(dataset)
        if shuffle_and_repeat:
            dataset = dataset.apply(tf.data.experimental.shuffle_and_repeat(1000))
        dataset = dataset.batch(context.get_per_slot_batch_size())
        dataset = dataset.map(parse_mnist_tfrecord)
        return dataset

    return _fn


def _get_filenames(directory: str) -> List[str]:
    return [os.path.join(directory, path) for path in tf.io.gfile.listdir(directory)]


def build_train_spec(
    context: EstimatorNativeContext, download_data_dir: str
) -> tf.estimator.TrainSpec:
    train_files = _get_filenames(os.path.join(download_data_dir, "train"))
    return tf.estimator.TrainSpec(input_fn(context, train_files, shuffle_and_repeat=True))


def build_validation_spec(
    context: EstimatorNativeContext, download_data_dir: str
) -> tf.estimator.EvalSpec:
    val_files = _get_filenames(os.path.join(download_data_dir, "validation"))
    return tf.estimator.EvalSpec(input_fn(context, val_files, shuffle_and_repeat=False))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--config",
        dest="config",
        help="Specifies Determined Experiment configuration.",
        default="{}",
    )
    parser.add_argument(
        "--mode", dest="mode", help="Specifies test mode or submit mode.", default="submit"
    )
    args = parser.parse_args()

    config = {
        "hyperparameters": {
            "learning_rate": det.Log(-4.0, -2.0, 10),
            "global_batch_size": det.Constant(64),
            "hidden_layer_1": det.Constant(250),
            "hidden_layer_2": det.Constant(250),
            "hidden_layer_3": det.Constant(250),
            "dropout": det.Double(0.0, 0.5),
        },
        "searcher": {
            "name": "single",
            "metric": "accuracy",
            "max_steps": 10,
            "smaller_is_better": False,
        },
    }
    config.update(json.loads(args.config))

    context = init(config, mode=det.Mode(args.mode), context_dir=str(pathlib.Path.cwd()))

    # Create a unique download directory for each rank so they don't overwrite each other.
    download_directory = f"/tmp/data-rank{context.distributed.get_rank()}"
    data_dir = download_mnist_tfrecords(download_directory)

    context.train_and_evaluate(
        build_estimator(context),
        build_train_spec(context, data_dir),
        build_validation_spec(context, data_dir),
    )
