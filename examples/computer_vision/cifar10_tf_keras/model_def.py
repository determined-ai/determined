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
from typing import Generator, List, Tuple

import numpy as np
import tensorflow as tf
from cifar_model import build_model, build_optimizer, compile_model
from tensorflow.keras.models import Sequential

import determined as det
from determined import keras


def load_numpy_data(
    context: det.core.Context,
) -> Tuple[Tuple[np.ndarray, np.ndarray], Tuple[np.ndarray, np.ndarray]]:
    # When running distributed, we don't want multiple ranks on the same node to download the
    # data simultaneously, since they'll overwrite each other. So we only download on
    # local rank 0.
    if context.distributed.get_local_rank() == 0:
        tf.keras.datasets.cifar10.load_data()
    # Wait until local rank 0 is done downloading.
    context.distributed.allgather_local(None)
    # Now that the data is downloaded, each rank can load it.
    (X_train, Y_train), (X_test, Y_test) = tf.keras.datasets.cifar10.load_data()
    # Convert from pixel values to [0, 1] range floats, and one-hot encode labels.
    X_train = X_train.astype("float32") / 255
    X_test = X_test.astype("float32") / 255
    Y_train = tf.keras.utils.to_categorical(Y_train, num_classes=10)
    Y_test = tf.keras.utils.to_categorical(Y_test, num_classes=10)
    return (X_train, Y_train), (X_test, Y_test)


def to_generator(
    xs: np.ndarray, ys: np.ndarray
) -> Generator[Tuple[np.ndarray, np.ndarray], None, None]:
    n = xs.shape[0]
    for i in range(n):
        yield xs[i], ys[i]


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

        train_ds = self.context.wrap_dataset(
            tf.data.Dataset.from_generator(
                lambda: to_generator(*self.train_np),
                output_signature=(
                    tf.TensorSpec(shape=(32, 32, 3), dtype=tf.float32),
                    tf.TensorSpec(shape=(10,), dtype=tf.float32),
                ),
            )
        )
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
        test_ds = self.context.wrap_dataset(
            tf.data.Dataset.from_generator(
                lambda: to_generator(*self.test_np),
                output_signature=(
                    tf.TensorSpec(shape=(32, 32, 3), dtype=tf.float32),
                    tf.TensorSpec(shape=(10,), dtype=tf.float32),
                ),
            )
        )
        test_ds = test_ds.batch(1)
        return test_ds
