import logging
import os
import tarfile

import requests
import tensorflow as tf

WORK_DIRECTORY = "/tmp/pedl-mnist-estimator-work-dir"
MNIST_TF_RECORDS_FILE = "mnist-tfrecord.tar.gz"
MNIST_TF_RECORDS_URL = (
    "https://s3-us-west-2.amazonaws.com/determined-ai-test-data/"
    + MNIST_TF_RECORDS_FILE
)


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
            logging.info(
                "Downloaded {} ({} bytes)".format(MNIST_TF_RECORDS_FILE, f.size())
            )

        logging.info(
            "Extracting {} to {}".format(MNIST_TF_RECORDS_FILE, download_directory)
        )
        with tarfile.open(filepath, mode="r:gz") as f:
            f.extractall(path=download_directory)

    data_dir = os.path.join(download_directory, "mnist-tfrecord")
    assert tf.io.gfile.exists(data_dir)
    return data_dir
