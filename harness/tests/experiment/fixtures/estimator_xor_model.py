import os
from typing import Any, Callable, Dict, Tuple, Union

import tensorflow as tf

from determined import estimator
from tests.experiment.utils import xor_data


def xor_input_fn(
    context: Union[estimator.EstimatorNativeContext, estimator.EstimatorTrialContext],
    batch_size: int,
    shuffle: bool = False,
) -> Callable[[], Tuple[tf.Tensor, tf.Tensor]]:
    def _input_fn() -> Tuple[tf.Tensor, tf.Tensor]:
        data, labels = xor_data()
        dataset = tf.data.Dataset.from_tensor_slices((data, labels))
        dataset = context.wrap_dataset(dataset)
        if shuffle:
            dataset = dataset.shuffle(1000)

        def map_dataset(x, y):
            return {"input": x}, y

        dataset = dataset.batch(batch_size)
        dataset = dataset.map(map_dataset)

        return dataset

    return _input_fn


def xor_input_fn_data_layer(
    context: Union[estimator.EstimatorNativeContext, estimator.EstimatorTrialContext],
    training: bool,
    batch_size: int,
    shuffle: bool = False,
) -> Callable[[], Tuple[tf.Tensor, tf.Tensor]]:
    def _input_fn() -> Tuple[tf.Tensor, tf.Tensor]:
        cacheable = (
            context.experimental.cache_train_dataset
            if training
            else context.experimental.cache_validation_dataset
        )

        @cacheable("xor_input_fn_data_layer", "xor_data", shuffle=shuffle)
        def make_dataset() -> tf.data.Dataset:
            data, labels = xor_data()
            ds = tf.data.Dataset.from_tensor_slices((data, labels))
            return ds

        dataset = make_dataset()

        def map_dataset(x, y):
            return {"input": x}, y

        dataset = dataset.batch(batch_size)
        dataset = dataset.map(map_dataset)

        return dataset

    return _input_fn


class StopVeryEarly(tf.compat.v1.train.SessionRunHook):  # type: ignore
    def after_run(
        self, run_context: tf.estimator.SessionRunContext, run_values: tf.estimator.SessionRunValues
    ) -> None:
        run_context.request_stop()


class XORTrial(estimator.EstimatorTrial):
    """
    Models a lightweight neural network model with one hidden layer to
    learn a binary XOR function. See Deep Learning Book, chapter 6.1 for
    the solution with a hidden size of 2, and a MSE loss function.
    """

    _searcher_metric = "loss"

    def __init__(self, context: estimator.EstimatorTrialContext) -> None:
        self.context = context

    def build_estimator(self) -> tf.estimator.Estimator:
        _input = tf.feature_column.numeric_column("input", shape=(2,), dtype=tf.int32)

        if self.context.get_hparam("optimizer") == "adam":
            optimizer = tf.compat.v1.train.AdamOptimizer(
                learning_rate=self.context.get_hparam("learning_rate")
            )
        elif self.context.get_hparam("optimizer") == "sgd":
            optimizer = tf.compat.v1.train.GradientDescentOptimizer(
                learning_rate=self.context.get_hparam("learning_rate")
            )
        else:
            raise NotImplementedError()
        optimizer = self.context.wrap_optimizer(optimizer)

        return tf.compat.v1.estimator.DNNClassifier(
            feature_columns=[_input],
            hidden_units=[self.context.get_hparam("hidden_size")],
            activation_fn=tf.nn.sigmoid,
            config=tf.estimator.RunConfig(
                session_config=tf.compat.v1.ConfigProto(
                    intra_op_parallelism_threads=1, inter_op_parallelism_threads=1
                )
            ),
            optimizer=optimizer,
        )

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        hooks = [StopVeryEarly()] if self.context.env.hparams.get("stop_early") == "train" else []
        return tf.estimator.TrainSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=self.context.get_hparam("shuffle"),
            ),
            hooks=hooks,
        )

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        hooks = (
            [StopVeryEarly()] if self.context.env.hparams.get("stop_early") == "validation" else []
        )
        return tf.estimator.EvalSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=False,
            ),
            hooks=hooks,
        )

    def build_serving_input_receiver_fns(self) -> Dict[str, estimator.ServingInputReceiverFn]:
        _input = tf.feature_column.numeric_column("input", shape=(2,), dtype=tf.int64)
        return {
            "inference": tf.estimator.export.build_parsing_serving_input_receiver_fn(
                tf.feature_column.make_parse_example_spec([_input])
            )
        }


class XORTrialDataLayer(XORTrial):
    def build_train_spec(self) -> tf.estimator.TrainSpec:
        return tf.estimator.TrainSpec(
            xor_input_fn_data_layer(
                context=self.context,
                training=True,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=self.context.get_hparam("shuffle"),
            )
        )

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        return tf.estimator.EvalSpec(
            xor_input_fn_data_layer(
                context=self.context,
                training=False,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=False,
            )
        )


class UserDefinedHook(tf.estimator.SessionRunHook):
    def __init__(self, file_path: str) -> None:
        self._file_path = file_path
        self._idx = 0

    def after_run(self, run_context: Any, run_values: Any) -> None:
        self._idx += 1
        with open(self._file_path, "w") as fp:
            fp.write(f"{self._idx}")


class XORTrialWithHooks(XORTrial):
    def __init__(self, context: estimator.EstimatorTrialContext) -> None:
        self.context = context

        self._train_hook = UserDefinedHook(file_path=self.context.get_hparam("training_log_path"))
        self._val_hook = UserDefinedHook(file_path=self.context.get_hparam("val_log_path"))

    def build_train_spec(self) -> tf.estimator.TrainSpec:
        return tf.estimator.TrainSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=self.context.get_hparam("shuffle"),
            ),
            hooks=[self._train_hook],
        )

    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        return tf.estimator.EvalSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=False,
            ),
            hooks=[self._val_hook],
        )


class CustomHook(estimator.RunHook):
    def __init__(self):
        self._num_checkpoints = 0

    def on_checkpoint_load(self, checkpoint_dir: str) -> None:
        with open(os.path.join(checkpoint_dir, "custom.log"), "r") as fp:
            self._num_checkpoints = int(fp.readline())

    def on_checkpoint_end(self, checkpoint_dir: str) -> None:
        self._num_checkpoints += 1
        with open(os.path.join(checkpoint_dir, "custom.log"), "w") as fp:
            fp.write(f"{self._num_checkpoints}")


class XORTrialWithCustomHook(XORTrial):
    def build_train_spec(self) -> tf.estimator.TrainSpec:
        return tf.estimator.TrainSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=self.context.get_hparam("shuffle"),
            ),
            hooks=[CustomHook()],
        )


class CustomEndOfTrainingHook(estimator.RunHook):
    def __init__(self, path: str) -> None:
        self._path = path

    def on_trial_close(self) -> None:
        with open(self._path, "w") as fp:
            fp.write("success")


class XORTrialEndOfTrainingHook(XORTrial):
    def build_train_spec(self) -> tf.estimator.TrainSpec:
        return tf.estimator.TrainSpec(
            xor_input_fn(
                context=self.context,
                batch_size=self.context.get_per_slot_batch_size(),
                shuffle=self.context.get_hparam("shuffle"),
            ),
            hooks=[CustomEndOfTrainingHook(self.context.get_hparam("training_end"))],
        )
