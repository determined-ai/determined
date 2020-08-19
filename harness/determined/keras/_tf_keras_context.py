import inspect
import logging
from typing import Any, Callable, List, NamedTuple, Optional, Union

import tensorflow as tf

import determined as det
from determined import _data_layer, errors, horovod, keras
from determined.horovod import hvd


class TFKerasTrainConfig(NamedTuple):
    training_data: Union[keras.SequenceAdapter, tf.data.Dataset]
    validation_data: Union[keras.SequenceAdapter, tf.data.Dataset]
    callbacks: List[tf.keras.callbacks.Callback]


class TFKerasContext:
    """
    Base context class that contains runtime information for any Determined
    workflow that uses the ``tf.keras`` API.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        self.env = env
        self.hvd_config = hvd_config
        self.dataset_initialized = False

        self.experimental = TFKerasExperimentalContext(env=env, hvd_config=hvd_config)

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

                training_data = keras.SequenceAdapter(
                    fit_generator_args.arguments["generator"],
                    use_multiprocessing=fit_generator_args.arguments["use_multiprocessing"],
                    workers=fit_generator_args.arguments["workers"],
                )
                validation_data = keras.SequenceAdapter(
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
                """Communicate a model, data, and other training configuration with the harness.

                Parameters:
                    the same as tf.keras.Model.fit except for this function only handles the
                    following cases of data

                    x: Input data. It could be:
                        1) A Numpy array (or array-like), or a list of arrays (in case the model
                        has multiple inputs).
                        2) A dict mapping input names to the corresponding array, if the model
                        has named inputs.
                        3) A tf.data dataset. Should return a tuple of either (inputs, targets) or
                        (inputs, targets, sample_weights).
                        4) A keras.utils.Sequence returning (inputs, targets) or (inputs, targets,
                        sample weights).

                    y: Target data. Like the input data x, it could be either Numpy array(s).
                        If x is a dataset or keras.utils.Sequence instance, y should not be
                        specified(since targets will be obtained from x).

                    validation_data: Data on which to evaluate the loss and any model metrics
                        at the end of each epoch. The model will not be trained on this data.
                        validation_data will override validation_split. validation_data could be:
                        1) tuple (x_val, y_val) of Numpy arrays
                        2) tuple (x_val, y_val, val_sample_weights) of Numpy arrays
                        3) dataset For the first two cases, batch_size must be provided.
                        For the last case, validation_steps could be provided.
                """
                if not self.compile_args:
                    raise errors.InvalidExperimentException(
                        "Must call .compile before calling .fit()."
                    )

                fit_args = inspect.signature(model.fit).bind(*args, **kwargs)
                fit_args.apply_defaults()

                # TODO: Use batch size from context instead of fit call.
                training_data = keras._adapt_keras_data(
                    x=fit_args.arguments["x"],
                    y=fit_args.arguments["y"],
                    sample_weight=fit_args.arguments["sample_weight"],
                    batch_size=self.env.per_slot_batch_size,
                    use_multiprocessing=fit_args.arguments["use_multiprocessing"],
                    workers=fit_args.arguments["workers"],
                    max_queue_size=fit_args.arguments["max_queue_size"],
                    drop_leftovers=True,
                )

                val_x, val_y, val_sample_weight = keras._get_x_y_and_sample_weight(
                    input_data=fit_args.arguments["validation_data"]
                )
                validation_data = keras._adapt_keras_data(
                    x=val_x,
                    y=val_y,
                    sample_weight=val_sample_weight,
                    batch_size=self.env.per_slot_batch_size,
                    use_multiprocessing=fit_args.arguments["use_multiprocessing"],
                    workers=fit_args.arguments["workers"],
                    max_queue_size=fit_args.arguments["max_queue_size"],
                    drop_leftovers=False,
                )

                self.train_config = TFKerasTrainConfig(
                    training_data=training_data,
                    validation_data=validation_data,
                    callbacks=fit_args.arguments["callbacks"],
                )

                if train_fn:
                    train_fn()

        return _WrappedModel()

    def wrap_dataset(self, dataset: Any, shard_dataset: bool = True) -> Any:
        """
        This should be used to wrap ``tf.data.Dataset`` objects immediately after
        they have been created. Users should use the output of this wrapper as the
        new instance of their dataset. If users create multiple datasets (e.g.,
        one for training and one for validation), users should wrap each dataset
        independently.

        Args:
            dataset: tf.data.Dataset
            shard_dataset:
                When performing multi-slot (distributed) training, this
                controls whether the dataset is sharded so that each training process
                (one per slot) sees unique data. If set to False, users must manually
                configure each process to use unique data.
        """
        self.dataset_initialized = True
        if not self.hvd_config.use or not isinstance(dataset, tf.data.Dataset) or not shard_dataset:

            if self.hvd_config and not shard_dataset:
                logging.info("Dataset sharding skipped.")
            return dataset

        hvd.require_horovod_type("tensorflow.keras", "TFKerasContext.wrap_dataset was called.")
        dataset = dataset.shard(hvd.size(), hvd.rank())
        logging.debug(f"Sharded dataset to index {hvd.rank()} of {hvd.size()}.")
        return dataset


class TFKerasTrialContext(det.TrialContext, TFKerasContext):
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        det.TrialContext.__init__(self, env, hvd_config)
        TFKerasContext.__init__(self, env, hvd_config)

    def wrap_model(self, model: Any) -> Any:
        """
        This should be used to wrap ``tf.keras.Model`` objects immediately after
        they have been created but before they have been compiled. This function
        takes a ``tf.keras.Model`` and returns a wrapped version of the model;
        the return value should be used in place of the original model.

        Args:
            model: tf.keras.Model
        """
        return self._wrap_model_with_train_fn(model, None)


class TFKerasNativeContext(det.NativeContext, TFKerasContext):
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        det.NativeContext.__init__(self, env, hvd_config)
        TFKerasContext.__init__(self, env, hvd_config)

    def wrap_model(self, model: Any) -> Any:
        return self._wrap_model_with_train_fn(model, self._train_fn)


class TFKerasExperimentalContext(_data_layer.DataLayerContext):
    """
    Context class that contains experimental runtime information and features
    for any Determined workflow that uses the ``tf.keras`` API.

    ``TFKerasExperimentalContext`` extends ``EstimatorTrialContext`` under
    the ``context.experimental`` namespace.
    """

    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        super().__init__(env=env, hvd_config=hvd_config)
