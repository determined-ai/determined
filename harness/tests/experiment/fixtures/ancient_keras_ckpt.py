"""
This model exists to confirm that old models and their old checkpoints can still
be loaded in new Determined.
"""

from typing import Any, Dict, cast

import tensorflow as tf
from tensorflow import raw_ops
from tensorflow.keras import layers, losses, models
from tensorflow.keras.optimizers import legacy  # TODO MLG-443

from determined import keras


def make_one_var_tf_dataset_loader(hparams: Dict[str, Any], batch_size: int) -> raw_ops.ZipDataset:
    dataset_range = hparams["dataset_range"]

    xtrain = tf.data.Dataset.range(dataset_range).batch(batch_size)
    ytrain = tf.data.Dataset.range(dataset_range).batch(batch_size)

    train_ds = tf.data.Dataset.zip((xtrain, ytrain))
    return train_ds


class AncientTrial(keras.TFKerasTrial):
    """
    An old model that should always reload from its equally old checkpoints.

    Don't change this model architecture or add any fancy features or the test won't be valid.
    """

    _searcher_metric = "val_loss"

    def __init__(self, context: keras.TFKerasTrialContext):
        self.context = context
        self.my_batch_size = self.context.get_per_slot_batch_size()
        self.my_learning_rate = self.context.get_hparam("learning_rate")

    def build_training_data_loader(self) -> keras.InputData:
        dataset = make_one_var_tf_dataset_loader(self.context.get_hparams(), self.my_batch_size)
        dataset = self.context.wrap_dataset(dataset)
        return dataset

    def build_validation_data_loader(self) -> keras.InputData:
        dataset = make_one_var_tf_dataset_loader(self.context.get_hparams(), self.my_batch_size)
        dataset = self.context.wrap_dataset(dataset)
        return dataset

    def build_model(self) -> models.Sequential:
        model = models.Sequential()
        model.add(
            layers.Dense(
                1, activation=None, use_bias=False, kernel_initializer="zeros", input_shape=(1,)
            )
        )
        model = self.context.wrap_model(model)
        model.compile(legacy.SGD(lr=self.my_learning_rate), losses.mean_squared_error)
        return cast(models.Sequential, model)
