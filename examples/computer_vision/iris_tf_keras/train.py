"""
This example shows you how to train a model with Determined's keras callback.

Useful References:
    https://docs.determined.ai/latest/reference/training/api-keras-reference.html
    https://keras.io/api/

Based off of: https://medium.com/@nickbortolotti/iris-species-categorization-using-tf-keras-tf-data-
              and-differences-between-eager-mode-on-and-off-9b4693e0b22
"""
import argparse
import logging
from typing import List

import pandas as pd
from tensorflow.keras import layers, losses, metrics, models, utils
from tensorflow.keras.optimizers import legacy

import determined as det
import determined.keras

# Where to download data from.
TRAIN_DATA = "http://download.tensorflow.org/data/iris_training.csv"
TEST_DATA = "http://download.tensorflow.org/data/iris_test.csv"

# Constants about the data set.
NUM_CLASSES = 3

# The first row of each data set is not a typical CSV header with column labels, but rather a
# dataset descriptor of the following format:
#
# <num observations>,<num features>,<species 0 label>,<species 1 label>,<species 2 label>
#
# The remaining rows then contain observations, with the four features followed by label.  The
# label values in the observation rows take on the values 0, 1, or 2 which correspond to the
# three species in the header.  Define the columns explicitly here so that we can more easily
# separate features and labels below.
LABEL_HEADER = "Species"
DS_COLUMNS = [
    "SepalLength",
    "SepalWidth",
    "PetalLength",
    "PetalWidth",
    LABEL_HEADER,
]


def get_train_data():
    # Ignore header line and read the training CSV observations into a pandas DataFrame.
    train = pd.read_csv(TRAIN_DATA, names=DS_COLUMNS, header=0)
    train_features, train_labels = train, train.pop(LABEL_HEADER)

    # Since we're building a classifier, convert the labels in the raw
    # dataset (0, 1, or 2) to one-hot vector encodings that we'll to
    # construct the Sequence data loaders that Determined expects.
    train_labels_categorical = utils.to_categorical(train_labels, num_classes=3)

    return train_features.values, train_labels_categorical


def get_test_data():
    test = pd.read_csv(TEST_DATA, names=DS_COLUMNS, header=0)
    test_features, test_labels = test, test.pop(LABEL_HEADER)
    test_labels_categorical = utils.to_categorical(test_labels, num_classes=3)
    return test_features.values, test_labels_categorical


def main(core_context, strategy, checkpoint, continue_id, hparams, epochs):
    # Download train and test data.
    train_x, train_y = get_train_data()
    validation_data = get_test_data()

    # Create and compile the model within a strategy's scope.
    with strategy.scope():
        inputs = layers.Input(shape=(4,))
        dense1 = layers.Dense(hparams["layer1_dense_size"])(inputs)
        dense2 = layers.Dense(NUM_CLASSES, activation="softmax")(dense1)
        model = models.Model(inputs=inputs, outputs=dense2)

        optimizer = legacy.RMSprop(
            lr=hparams["learning_rate"],
            decay=hparams["learning_rate_decay"],
        )

        model.compile(
            optimizer,
            losses.categorical_crossentropy,
            [metrics.categorical_accuracy],
        )

    # Create the main DeterminedCallback that connects training to the Determined cluster.
    det_cb = det.keras.DeterminedCallback(
        core_context,
        checkpoint=checkpoint,
        continue_id=continue_id,
        # Iris epochs are very short, so we don't even bother to save checkpoints until we finish.
        checkpoint_epochs=0,
    )

    # Also include a Determined-aware version of the Keras' TensorBoard callback.
    tb_cb = det.keras.TensorBoard(
        core_context, update_freq="batch", profile_batch=0, histogram_freq=1
    )

    # Call model.fit() with our callbacks.
    model.fit(
        x=train_x,
        y=train_y,
        batch_size=hparams["global_batch_size"],
        validation_data=validation_data,
        epochs=epochs,
        callbacks=[det_cb, tb_cb],
    )


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    parser = argparse.ArgumentParser()
    parser.add_argument("--epochs", type=int, default=500, help="how long to train for")
    args = parser.parse_args()

    info = det.get_cluster_info()
    if info and info.task_type == "TRIAL":
        # We are a training a trial on-cluster.
        continue_id = info.trial.trial_id
        checkpoint = info.latest_checkpoint
        # Use the hparams selected by the searcher for this trial.
        hparams = info.trial.hparams
    else:
        # We are either in a notebook on-cluster or off-cluster entirely.
        continue_id = "local-train-task"
        checkpoint = None
        # Pick some hparams for ourselves.
        hparams = {
            "learning_rate": 1.0e-4,
            "learning_rate_decay": 1.0e-6,
            "layer1_dense_size": 16,
            "global_batch_size": 16,
        }

    distributed, strategy = det.core.DistributedContext.from_tf_config()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context, strategy, checkpoint, continue_id, hparams, args.epochs)
