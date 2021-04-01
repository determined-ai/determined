"""
This example shows how you could use Keras `Sequence`s and multiprocessing/multithreading for Keras
models in Determined. Information for how this can be configured can be found in
`make_data_loaders()`.

Tutorial based on this example:
    https://docs.determined.ai/latest/tutorials/tf-cifar-tutorial.html

Useful References:
    https://docs.determined.ai/latest/reference/api/keras.html
    https://www.tensorflow.org/guide/keras

Based on: https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py

"""
from typing import List

import tensorflow as tf
from tensorflow.keras.models import Sequential

from cifar_model import (
    build_model,
    build_optimizer,
    compile_model,
)
from data import (
    download_data,
    get_training_data,
    get_validation_data,
)

from determined import keras

class CIFARTrial(keras.TFKerasTrial):
    def __init__(self, context: keras.TFKerasTrialContext) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.download_directory = download_data(
            download_directory=self.download_directory,
            url=self.context.get_data_config()["url"],
        )

    def session_config(self) -> tf.compat.v1.ConfigProto:
        if self.context.get_hparams().get("disable_CPU_parallelism", False):
            return tf.compat.v1.ConfigProto(
                intra_op_parallelism_threads=1, inter_op_parallelism_threads=1
            )
        else:
            return tf.compat.v1.ConfigProto()

    def build_model(self) -> Sequential:
        # Create model.
        model = build_model(
            layer1_dropout=self.context.get_hparam("layer1_dropout"),
            layer2_dropout=self.context.get_hparam("layer2_dropout"),
            layer3_dropout=self.context.get_hparam("layer3_dropout"),
        )

        # Wrap the model.
        model = self.context.wrap_model(model)

        # Create and wrap optimizer.
        optimizer = build_optimizer(
            learning_rate=self.context.get_hparam("learning_rate"),
            learning_rate_decay=self.context.get_hparam("learning_rate_decay"),
        )
        optimizer = self.context.wrap_optimizer(optimizer)

        # Compile model.
        compile_model(model=model, optimizer=optimizer)

        return model

    def keras_callbacks(self) -> List[tf.keras.callbacks.Callback]:
        return [
            keras.callbacks.TensorBoard(
                update_freq="batch", profile_batch=0, histogram_freq=1
            )
        ]

    def build_training_data_loader(self) -> keras.InputData:
        hparams = self.context.get_hparams()
        # Return a tf.keras.Sequence.
        return get_training_data(
            data_directory=self.download_directory,
            batch_size=self.context.get_per_slot_batch_size(),
            width_shift_range=hparams.get("width_shift_range", 0.0),
            height_shift_range=hparams.get("height_shift_range", 0.0),
            horizontal_flip=hparams.get("horizontal_flip", False),
        )

    def build_validation_data_loader(self) -> keras.InputData:
        return get_validation_data(self.download_directory)
