"""
This example shows how you could use Keras `Sequence`s and multiprocessing/multithreading for Keras
models in Determined.

Useful References:
    http://docs.determined.ai/latest/keras.html
    https://keras.io/utils/

Based off of: https://medium.com/@nickbortolotti/iris-species-categorization-using-tf-keras-tf-data-
              and-differences-between-eager-mode-on-and-off-9b4693e0b22
"""
from typing import List

import pandas as pd
import tensorflow as tf
from tensorflow.keras.layers import Dense, Input
from tensorflow.keras.losses import categorical_crossentropy
from tensorflow.keras.metrics import categorical_accuracy
from tensorflow.keras.models import Model
from tensorflow.keras.optimizers import RMSprop
from tensorflow.keras.utils import to_categorical

from determined import keras

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


class IrisTrial(keras.TFKerasTrial):
    def __init__(self, context: keras.TFKerasTrialContext) -> None:
        self.context = context

    def build_model(self) -> Model:
        """
        Define model for iris classification.

        This is a simple model with one hidden layer to predict iris species (setosa, versicolor, or
        virginica) based on four input features (length and width of sepals and petals).
        """
        inputs = Input(shape=(4,))
        dense1 = Dense(self.context.get_hparam("layer1_dense_size"))(inputs)
        dense2 = Dense(NUM_CLASSES, activation="softmax")(dense1)

        # Wrap the model.
        model = self.context.wrap_model(Model(inputs=inputs, outputs=dense2))

        # Create and wrap the optimizer.
        optimizer = RMSprop(
            lr=self.context.get_hparam("learning_rate"),
            decay=self.context.get_hparam("learning_rate_decay"),
        )
        optimizer = self.context.wrap_optimizer(optimizer)

        model.compile(
            optimizer,
            categorical_crossentropy,
            [categorical_accuracy],
        )

        return model

    def keras_callbacks(self) -> List[tf.keras.callbacks.Callback]:
        return [keras.callbacks.TensorBoard(update_freq="batch", profile_batch=0, histogram_freq=1)]

    def build_training_data_loader(self) -> keras.InputData:
        # Ignore header line and read the training CSV observations into a pandas DataFrame.
        train = pd.read_csv(self.context.get_data_config()["train_url"], names=DS_COLUMNS, header=0)
        train_features, train_labels = train, train.pop(LABEL_HEADER)

        # Since we're building a classifier, convert the labels in the raw
        # dataset (0, 1, or 2) to one-hot vector encodings that we'll to
        # construct the Sequence data loaders that Determined expects.
        train_labels_categorical = to_categorical(train_labels, num_classes=3)

        return train_features.values, train_labels_categorical

    def build_validation_data_loader(self) -> keras.InputData:
        # Ignore header line and read the test CSV observations into a pandas DataFrame.
        test = pd.read_csv(self.context.get_data_config()["test_url"], names=DS_COLUMNS, header=0)
        test_features, test_labels = test, test.pop(LABEL_HEADER)

        # Since we're building a classifier, convert the labels in the raw
        # dataset (0, 1, or 2) to one-hot vector encodings that we'll to
        # construct the Sequence data loaders that Determined expects.
        test_labels_categorical = to_categorical(test_labels, num_classes=3)

        return test_features.values, test_labels_categorical
