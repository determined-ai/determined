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
from cifar_model import build_model, build_optimizer, compile_model
from data import load_numpy_data
from tensorflow.keras.models import Sequential

from determined import keras


class CIFARTrial(keras.TFKerasTrial):
    def __init__(self, context: keras.TFKerasTrialContext) -> None:
        self.context = context
        self.train_np, self.test_np = load_numpy_data(self.context)

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
        return [keras.callbacks.TensorBoard(update_freq="batch", profile_batch=0, histogram_freq=1)]

    def build_training_data_loader(self) -> keras.InputData:
        hparams = self.context.get_hparams()

        train_ds = self.context.wrap_dataset(tf.data.Dataset.from_tensor_slices(self.train_np))
        augmentation = tf.keras.Sequential(
            [
                tf.keras.layers.RandomFlip(mode="horizontal"),
                tf.keras.layers.RandomTranslation(
                    height_factor=hparams.get("height_factor", 0.0),
                    width_factor=hparams.get("width_factor", 0.0),
                ),
            ]
        )
        train_ds = train_ds.batch(self.context.get_per_slot_batch_size())
        train_ds = train_ds.map(
            lambda x, y: (augmentation(x), y), num_parallel_calls=tf.data.experimental.AUTOTUNE
        )
        train_ds = train_ds.prefetch(tf.data.experimental.AUTOTUNE)
        return train_ds

    def build_validation_data_loader(self) -> keras.InputData:
        test_ds = self.context.wrap_dataset(tf.data.Dataset.from_tensor_slices(self.test_np))
        test_ds = test_ds.batch(1)
        return test_ds
