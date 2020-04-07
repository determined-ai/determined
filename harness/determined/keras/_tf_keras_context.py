import inspect
import logging
from typing import Any, Callable, List, NamedTuple, Optional, Union

import numpy as np
import tensorflow as tf

import determined as det
from determined import errors, horovod, keras
from determined.horovod import hvd


class TFKerasTrainConfig(NamedTuple):
    training_data: Union[keras.KerasDataAdapter, tf.data.Dataset]
    validation_data: Union[keras.KerasDataAdapter, tf.data.Dataset]
    callbacks: List[tf.keras.callbacks.Callback]


class TFKerasContext:
    def __init__(self, hvd_config: horovod.HorovodContext):
        logging.debug(f"Initialized TFKerasContext with config: {hvd_config}.")
        self.hvd_config = hvd_config
        self.dataset_initialized = False

        # The following three attributes are initialized during the lifetime of a
        # TFKerasContext instance by the user calling compile() and
        # fit_generator() / fit(), respectively.
        self.model = None  # type: Optional[tf.keras.Model]
        self.compile_args = None  # type: Optional[inspect.BoundArguments]
        self.train_config = None  # type: Optional[TFKerasTrainConfig]

    def _wrap_model_with_train_fn(self, model: Any, train_fn: Optional[Callable]) -> Any:
        class _WrappedModel(type(model)):  # type: ignore
            def __init__(wrapper) -> None:
                self.model = model

            def __getattr__(wrapper, name):  # type: ignore
                return getattr(model, name)

            def __setattr__(wrapper, name, value):  # type: ignore
                return setattr(model, name, value)

            def __delattr__(wrapper, name):  # type: ignore
                return delattr(model, name)

            def compile(wrapper, *args: Any, **kwargs: Any) -> None:
                bound_arguments = inspect.signature(model.compile).bind(*args, **kwargs)
                bound_arguments.apply_defaults()

                if "optimizer" not in bound_arguments.arguments:
                    raise errors.InvalidExperimentException(
                        "Must have 'optimizer' in arguments of .compile()."
                    )

                self.compile_args = bound_arguments

            def fit_generator(wrapper, *args: Any, **kwargs: Any) -> None:
                if not self.compile_args:
                    raise errors.InvalidExperimentException(
                        "Must call .compile before calling .fit_generator()."
                    )

                fit_generator_args = inspect.signature(model.fit_generator).bind(*args, **kwargs)
                fit_generator_args.apply_defaults()

                training_data = keras.KerasDataAdapter(
                    fit_generator_args.arguments["generator"],
                    use_multiprocessing=fit_generator_args.arguments["use_multiprocessing"],
                    workers=fit_generator_args.arguments["workers"],
                )
                validation_data = keras.KerasDataAdapter(
                    fit_generator_args.arguments["validation_data"],
                    use_multiprocessing=fit_generator_args.arguments["use_multiprocessing"],
                    workers=fit_generator_args.arguments["workers"],
                )

                self.train_config = TFKerasTrainConfig(
                    training_data=training_data,
                    validation_data=validation_data,
                    callbacks=fit_generator_args.arguments["callbacks"],
                )

                if train_fn:
                    train_fn()

            def fit(wrapper, *args: Any, **kwargs: Any) -> None:
                """
                TFKerasExperiment.fit() currently handles three cases of input data:

                1) x and y arguments are both specified as in-memory numpy arrays.
                   batch_size argument is specified.
                2) x argument is specified as a tf.keras.utils.Sequence.
                3) x argument is specified as a Python generator.

                TODO(DET-2155): Support tf.dataset as input.

                Native tf.keras fit() supports a few more cases such as TF tensors,
                list of in-memory arrays, and dictionaries mapping strings to
                tensor/array values. For now, it is recommended that users implement
                a tf.keras.utils.Sequence to support any of these unsupported cases.
                """
                if not self.compile_args:
                    raise errors.InvalidExperimentException(
                        "Must call .compile before calling .fit()."
                    )

                fit_args = inspect.signature(model.fit).bind(*args, **kwargs)
                fit_args.apply_defaults()

                if isinstance(fit_args.arguments["x"], np.ndarray):
                    if not (
                        isinstance(fit_args.arguments["y"], np.ndarray)
                        and isinstance(fit_args.arguments["batch_size"], int)
                    ):
                        raise errors.InvalidExperimentException(
                            "Detected an in-memory np.ndarray as input data. "
                            "This type of input data requires an in-memory np.ndarray "
                            "as the y argument and a batch_size argument."
                        )

                    sequence = keras.InMemorySequence(
                        data=fit_args.arguments["x"],
                        labels=fit_args.arguments["y"],
                        batch_size=fit_args.arguments["batch_size"],
                    )
                    training_data = keras.KerasDataAdapter(
                        data=sequence,
                        use_multiprocessing=fit_args.arguments["use_multiprocessing"],
                        workers=fit_args.arguments["workers"],
                    )
                elif isinstance(fit_args.arguments["x"], tf.keras.utils.Sequence):
                    training_data = keras.KerasDataAdapter(
                        fit_args.arguments["x"],
                        use_multiprocessing=fit_args.arguments["use_multiprocessing"],
                        workers=fit_args.arguments["workers"],
                    )
                elif isinstance(fit_args.arguments["x"], tf.data.Dataset):
                    training_data = fit_args.arguments["x"]
                else:
                    raise errors.InvalidExperimentException(
                        "Input data type '{}' unsupported by TFKerasExperiment.fit(). "
                        "Please consider using tf.keras.utils.Sequence to represent input "
                        "data.".format(type(fit_args.arguments["x"]))
                    )

                validation_data = keras.KerasDataAdapter(
                    fit_args.arguments["validation_data"],
                    use_multiprocessing=fit_args.arguments["use_multiprocessing"],
                    workers=fit_args.arguments["workers"],
                )
                self.train_config = TFKerasTrainConfig(
                    training_data=training_data,
                    validation_data=validation_data,
                    callbacks=fit_args.arguments["callbacks"],
                )

                if train_fn:
                    train_fn()

        return _WrappedModel()

    def wrap_dataset(self, dataset: Any) -> Any:
        """
        This should be used to wrap ``tf.data.Dataset`` objects immediately after
        they have been created. Users should use the output of this wrapper as the
        new instance of their dataset. If users create multiple datasets (e.g.,
        one for training and one for testing) users should wrap each dataset
        independently.
        """
        hvd.require_horovod_type("tensorflow.keras", "TFKerasContext.wrap_dataset was called.")

        self.dataset_initialized = True
        if not self.hvd_config.use or not isinstance(dataset, tf.data.Dataset):
            return dataset
        dataset = dataset.shard(hvd.size(), hvd.rank())
        logging.debug(f"Sharded dataset to index {hvd.rank()} of {hvd.size()}.")
        return dataset


class TFKerasTrialContext(det.TrialContext, TFKerasContext):
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        det.TrialContext.__init__(self, env, hvd_config)
        TFKerasContext.__init__(self, hvd_config)

    def wrap_model(self, model: Any) -> Any:
        return self._wrap_model_with_train_fn(model, None)


class TFKerasNativeContext(det.NativeContext, TFKerasContext):
    def __init__(
        self, env: det.EnvContext, hvd_config: horovod.HorovodContext,
    ):
        det.NativeContext.__init__(self, env, hvd_config)
        TFKerasContext.__init__(self, hvd_config)

    def wrap_model(self, model: Any) -> Any:
        return self._wrap_model_with_train_fn(model, self._train_fn)
