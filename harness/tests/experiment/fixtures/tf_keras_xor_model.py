from typing import Any, List, cast

import tensorflow as tf
from packaging import version
from tensorflow.keras import layers, losses, metrics, models

if version.parse(tf.__version__) < version.parse("2.11.0"):
    from tensorflow.keras import optimizers as opt
else:
    from tensorflow.keras.optimizers import legacy as opt  # TODO MLG-443

from determined import keras
from tests.experiment import tf_utils


class StopVeryEarlyCallback(keras.callbacks.Callback):
    def on_train_workload_end(self, _: int, logs: Any = None) -> None:
        self.model.stop_training = True


def categorical_error(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return 1.0 - metrics.categorical_accuracy(y_true, y_pred)


def predictions(y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
    return y_pred


class XORTrial(keras.TFKerasTrial):
    """
    Models a lightweight neural network model with one hidden layer to
    learn a binary XOR function. See Deep Learning Book, chapter 6.1 for
    the solution with a hidden size of 2, and a MSE loss function.
    """

    _searcher_metric = "val_loss"

    def __init__(self, context: keras.TFKerasTrialContext):
        self.context = context
        # In-memory Sequences work best with workers=0.
        self.context.configure_fit(verbose=False, workers=0)

    def build_model(self) -> models.Sequential:
        model = models.Sequential()
        model.add(
            layers.Dense(
                self.context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,)
            )
        )
        model.add(layers.Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            self.context.wrap_optimizer(opt.SGD(lr=self.context.get_hparam("learning_rate"))),
            losses.binary_crossentropy,
            metrics=[categorical_error],
        )
        return cast(models.Sequential, model)

    def batch_size(self) -> int:
        return self.context.get_per_slot_batch_size()

    def session_config(self) -> tf.compat.v1.ConfigProto:
        return tf.compat.v1.ConfigProto(
            intra_op_parallelism_threads=1, inter_op_parallelism_threads=1
        )

    def build_training_data_loader(self) -> keras.InputData:
        train, _ = tf_utils.make_xor_data_sequences(batch_size=4)
        return train

    def build_validation_data_loader(self) -> keras.InputData:
        _, test = tf_utils.make_xor_data_sequences(batch_size=4)
        return test

    def keras_callbacks(self) -> List[tf.keras.callbacks.Callback]:
        return [StopVeryEarlyCallback()] if self.context.env.hparams.get("stop_early") else []


class XORTrialOldOptimizerAPI(XORTrial):
    def build_model(self) -> models.Sequential:
        model = models.Sequential()
        model.add(
            layers.Dense(
                self.context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,)
            )
        )
        model.add(layers.Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            opt.SGD(lr=self.context.get_hparam("learning_rate")),
            losses.binary_crossentropy,
            metrics=[categorical_error],
        )
        return cast(models.Sequential, model)


class XORTrialWithTrainingMetrics(XORTrial):
    def build_model(self) -> models.Sequential:
        model = models.Sequential()
        model.add(
            layers.Dense(
                self.context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,)
            )
        )
        model.add(layers.Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            self.context.wrap_optimizer(opt.SGD(lr=self.context.get_hparam("learning_rate"))),
            losses.binary_crossentropy,
            metrics=[categorical_error, metrics.categorical_accuracy, predictions],
        )
        return cast(models.Sequential, model)


class CustomOptimizer(opt.SGD):  # type: ignore
    pass


class CustomDenseLayer(layers.Dense):  # type: ignore
    pass


class XORTrialWithCustomObjects(XORTrial):
    def custom_loss_fn(self, y_true: tf.Tensor, y_pred: tf.Tensor) -> tf.Tensor:
        return losses.binary_crossentropy(y_true, y_pred)

    def custom_activation_fn(self, x: tf.Tensor) -> tf.Tensor:
        return tf.keras.activations.sigmoid(x)

    def build_model(self) -> models.Sequential:
        model = models.Sequential()
        model.add(
            CustomDenseLayer(
                self.context.get_hparam("hidden_size"),
                activation=self.custom_activation_fn,
                input_shape=(2,),
            )
        )
        model.add(layers.Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            self.context.wrap_optimizer(
                CustomOptimizer(lr=self.context.get_hparam("learning_rate"))
            ),
            loss=self.custom_loss_fn,
            metrics=[categorical_error, metrics.categorical_accuracy, predictions],
        )
        return cast(models.Sequential, model)


class XORTrialWithOptimizerState(XORTrial):
    def build_model(self) -> models.Sequential:
        model = models.Sequential()
        model.add(
            layers.Dense(
                self.context.get_hparam("hidden_size"), activation="sigmoid", input_shape=(2,)
            )
        )
        model.add(layers.Dense(1))
        model = self.context.wrap_model(model)
        model.compile(
            self.context.wrap_optimizer(opt.Adam(lr=self.context.get_hparam("learning_rate"))),
            losses.binary_crossentropy,
            metrics=[categorical_error],
        )
        return cast(models.Sequential, model)
