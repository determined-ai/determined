from typing import Any, List, cast

import tensorflow as tf
from tensorflow.keras.layers import Dense
from tensorflow.keras.losses import binary_crossentropy
from tensorflow.keras.metrics import categorical_accuracy
from tensorflow.keras.models import Sequential
from tensorflow.keras.optimizers import SGD, Adam

from determined import keras
from tests.experiment.utils import make_xor_data_sequences, xor_data  # noqa: I202, I100


class StopVeryEarlyCallback(keras.callbacks.Callback):  # type: ignore
    def on_train_workload_end(self, _: int, logs: Any = None) -> None:
        self.model.stop_training = True


def categorical_error(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return 1.0 - categorical_accuracy(y_true, y_pred)


def predictions(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return y_pred


class XORTrial(keras.TFKerasTrial):
    _searcher_metric = "val_loss"
    """
    Models a lightweight neural network model with one hidden layer to
    learn a binary XOR function. See Deep Learning Book, chapter 6.1 for
    the solution with a hidden size of 2, and a MSE loss function.
    """

    def __init__(self, context: keras.TFKerasTrialContext):
        self.context = context
        # In-memory Sequences work best with workers=0.
        self.context.configure_fit(verbose=False, workers=0)

    def build_model(self) -> Sequential:
        model = Sequential()
        model.add(
            Dense(self.context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,))
        )
        model.add(Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            self.context.wrap_optimizer(SGD(lr=self.context.get_hparam("learning_rate"))),
            binary_crossentropy,
            metrics=[categorical_error],
        )
        return cast(Sequential, model)

    def batch_size(self) -> int:
        return self.context.get_per_slot_batch_size()

    def session_config(self) -> tf.compat.v1.ConfigProto:
        return tf.compat.v1.ConfigProto(
            intra_op_parallelism_threads=1, inter_op_parallelism_threads=1
        )

    def build_training_data_loader(self) -> keras.InputData:
        train, _ = make_xor_data_sequences(batch_size=4)
        return train

    def build_validation_data_loader(self) -> keras.InputData:
        _, test = make_xor_data_sequences(batch_size=4)
        return test

    def keras_callbacks(self) -> List[tf.keras.callbacks.Callback]:
        return [StopVeryEarlyCallback()] if self.context.env.hparams.get("stop_early") else []


class XORTrialOldOptimizerAPI(XORTrial):
    def build_model(self) -> Sequential:
        model = Sequential()
        model.add(
            Dense(self.context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,))
        )
        model.add(Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            SGD(lr=self.context.get_hparam("learning_rate")),
            binary_crossentropy,
            metrics=[categorical_error],
        )
        return cast(Sequential, model)


class XORTrialWithTrainingMetrics(XORTrial):
    def build_model(self) -> Sequential:
        model = Sequential()
        model.add(
            Dense(self.context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,))
        )
        model.add(Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            self.context.wrap_optimizer(SGD(lr=self.context.get_hparam("learning_rate"))),
            binary_crossentropy,
            metrics=[categorical_error, categorical_accuracy, predictions],
        )
        return cast(Sequential, model)


class CustomOptimizer(SGD):  # type: ignore
    pass


class CustomDenseLayer(Dense):  # type: ignore
    pass


class XORTrialWithCustomObjects(XORTrial):
    def custom_loss_fn(self, y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
        return binary_crossentropy(y_true, y_pred)

    def custom_activation_fn(self, x: tf.Tensor) -> tf.Tensor:
        return tf.keras.activations.sigmoid(x)

    def build_model(self) -> Sequential:
        model = Sequential()
        model.add(
            CustomDenseLayer(
                self.context.get_hparam("hidden_size"),
                activation=self.custom_activation_fn,
                input_shape=(2,),
            )
        )
        model.add(Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            self.context.wrap_optimizer(
                CustomOptimizer(lr=self.context.get_hparam("learning_rate"))
            ),
            loss=self.custom_loss_fn,
            metrics=[categorical_error, categorical_accuracy, predictions],
        )
        return cast(Sequential, model)


class XORTrialWithOptimizerState(XORTrial):
    def build_model(self) -> Sequential:
        model = Sequential()
        model.add(
            Dense(self.context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,))
        )
        model.add(Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            self.context.wrap_optimizer(Adam(lr=self.context.get_hparam("learning_rate"))),
            binary_crossentropy,
            metrics=[categorical_error],
        )
        return cast(Sequential, model)


class XORTrialWithDataLayer(XORTrial):
    def build_training_data_loader(self) -> keras.InputData:
        @self.context.experimental.cache_train_dataset("XORTrialWithDataLayer", "xor_data")
        def make_dataset() -> tf.data.Dataset:
            data, labels = xor_data()
            ds = tf.data.Dataset.from_tensor_slices((data, labels))
            return ds

        dataset = make_dataset()
        dataset = dataset.batch(batch_size=4)
        return dataset

    def build_validation_data_loader(self) -> keras.InputData:
        @self.context.experimental.cache_validation_dataset("XORTrialWithDataLayer", "xor_data")
        def make_dataset() -> tf.data.Dataset:
            data, labels = xor_data()
            ds = tf.data.Dataset.from_tensor_slices((data, labels))
            return ds

        dataset = make_dataset()
        dataset = dataset.batch(batch_size=4)
        return dataset
