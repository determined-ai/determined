import inspect
import logging
from typing import Any, Callable, Dict, List, NamedTuple, Optional, Union

import tensorflow as tf

import determined as det
from determined import _data_layer, errors, horovod, keras
from determined.horovod import hvd
from determined_common import check


class TFKerasTrainConfig(NamedTuple):
    training_data: Union[keras.SequenceAdapter, tf.data.Dataset]
    validation_data: Union[keras.SequenceAdapter, tf.data.Dataset]
    callbacks: List[tf.keras.callbacks.Callback]


class _ArgNotProvided:
    """A singleton to distinguish between None and unprovided arguments."""

    pass


_arg_not_provided = _ArgNotProvided()


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

        self._optimizers = []  # type: List[tf.keras.optimizers.Optimizer]
        self._wrapped_optimizers = []  # type: List[tf.keras.optimizers.Optimizer]
        self._compiled_optimizer = None  # type: Optional[tf.keras.optimizers.Optimizer]

        # The following attributes may be configured via configure_fit().  Defaults match the
        # normal keras.fit() defaults.
        self._fit_verbose = True
        self._fit_class_weight = None
        self._fit_workers = 1
        self._fit_use_multiprocessing = False
        self._fit_max_queue_size = 10
        self._fit_shuffle = True
        self._fit_validation_steps = None

    def configure_fit(
        self,
        verbose: Optional[bool] = None,
        class_weight: Any = _arg_not_provided,
        workers: Optional[int] = None,
        use_multiprocessing: Optional[bool] = None,
        max_queue_size: Optional[bool] = None,
        shuffle: Optional[bool] = None,
        validation_steps: Any = _arg_not_provided,
    ) -> None:
        """
        Configure parameters of ``model.fit()``.  See the `Keras documentation
        <https://keras.io/api/>`__ for the meaning of each parameter.

        Note that the output of ``verbose=True`` will be visually different in Determined than with
        Keras, for better rendering in trial logs.

        Note that if ``configure_fit()`` is called multiple times, any keyword arguments which are
        not provided in the second call will not overwrite any settings configured by the first
        call.

        **Usage Example**

        .. code:: python

           class MyTFKerasTrial(det.keras.TFKerasTrial):
               def __init__(self, context):
                   ...
                   self.context.configure_fit(verbose=False, workers=5)

                   # It is safe to call configure_fit() multiple times.
                   self.context.configure_fit(use_multiprocessing=True)
        """
        if verbose is not None:
            self._fit_verbose = verbose
        if not isinstance(class_weight, _ArgNotProvided):
            self._fit_class_weight = class_weight
        if workers is not None:
            self._fit_workers = workers
        if use_multiprocessing is not None:
            self._fit_use_multiprocessing = use_multiprocessing
        if max_queue_size is not None:
            self._fit_max_queue_size = max_queue_size
        if shuffle is not None:
            self._fit_shuffle = shuffle
        if not isinstance(validation_steps, _ArgNotProvided):
            self._fit_validation_steps = validation_steps

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
                self.compile_args = bound_arguments

            def fit_generator(wrapper, *args: Any, **kwargs: Any) -> None:
                if not self.compile_args:
                    raise errors.InvalidExperimentException(
                        "Must call .compile before calling .fit_generator()."
                    )

                fit_generator_args = inspect.signature(model.fit_generator).bind(*args, **kwargs)
                fit_generator_args.apply_defaults()

                training_data = fit_generator_args.arguments["generator"]

                if fit_generator_args.arguments["validation_data"] is None:
                    raise errors.InvalidExperimentException(
                        "Determined requires validation_data in the call to fit_generator()."
                    )

                validation_data = keras._adapt_data_from_data_loader(
                    input_data=fit_generator_args.arguments["validation_data"],
                    batch_size=self.env.per_slot_batch_size,
                )

                self.train_config = TFKerasTrainConfig(
                    training_data=training_data,
                    validation_data=validation_data,
                    callbacks=fit_generator_args.arguments["callbacks"],
                )

                self.configure_fit(
                    verbose=fit_generator_args.arguments["verbose"],
                    class_weight=fit_generator_args.arguments["class_weight"],
                    shuffle=fit_generator_args.arguments["shuffle"],
                    workers=fit_generator_args.arguments["workers"],
                    use_multiprocessing=fit_generator_args.arguments["use_multiprocessing"],
                    max_queue_size=fit_generator_args.arguments["max_queue_size"],
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

                training_data = keras._adapt_data_from_fit_args(
                    x=fit_args.arguments["x"],
                    y=fit_args.arguments["y"],
                    sample_weight=fit_args.arguments["sample_weight"],
                    batch_size=self.env.per_slot_batch_size,
                )

                if fit_args.arguments["validation_data"] is None:
                    raise errors.InvalidExperimentException(
                        "Determined requires validation_data in the call to fit()."
                    )

                validation_data = keras._adapt_data_from_data_loader(
                    input_data=fit_args.arguments["validation_data"],
                    batch_size=self.env.per_slot_batch_size,
                )

                self.train_config = TFKerasTrainConfig(
                    training_data=training_data,
                    validation_data=validation_data,
                    callbacks=fit_args.arguments["callbacks"],
                )

                self.configure_fit(
                    verbose=fit_args.arguments["verbose"],
                    shuffle=fit_args.arguments["shuffle"],
                    class_weight=fit_args.arguments["class_weight"],
                    workers=fit_args.arguments["workers"],
                    use_multiprocessing=fit_args.arguments["use_multiprocessing"],
                    max_queue_size=fit_args.arguments["max_queue_size"],
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
        if not self.env.managed_training:
            return dataset

        self.dataset_initialized = True
        if not self.hvd_config.use or not isinstance(dataset, tf.data.Dataset) or not shard_dataset:

            if self.hvd_config and not shard_dataset:
                logging.info("Dataset sharding skipped.")
            return dataset

        hvd.require_horovod_type("tensorflow.keras", "TFKerasContext.wrap_dataset was called.")
        dataset = dataset.shard(hvd.size(), hvd.rank())
        logging.debug(f"Sharded dataset to index {hvd.rank()} of {hvd.size()}.")
        return dataset

    def _get_horovod_optimizer_if_using_horovod(
        self, optimizer: tf.keras.optimizers.Optimizer
    ) -> tf.keras.optimizers.Optimizer:
        if not self.hvd_config.use:
            return optimizer

        # Horovod doesn't know how to handle string-based optimizers.
        if isinstance(optimizer, str):
            raise det.errors.InvalidExperimentException("string optimizers are not supported")

        # The signature of our horovod optimizer changed after we rebased onto 0.21.
        hvd_sig = inspect.signature(hvd.DistributedOptimizer)
        horovod_kwargs = {
            "average_aggregated_gradients": self.hvd_config.average_aggregated_gradients,
        }  # type: Dict[str, Any]
        if "aggregation_frequency" in hvd_sig.parameters:
            horovod_kwargs["aggregation_frequency"] = self.hvd_config.aggregation_frequency
        else:
            horovod_kwargs["backward_passes_per_step"] = self.hvd_config.aggregation_frequency

        return hvd.DistributedOptimizer(optimizer, **horovod_kwargs)

    def wrap_optimizer(
        self, optimizer: tf.keras.optimizers.Optimizer
    ) -> tf.keras.optimizers.Optimizer:
        """
        This should be user to wrap ``tf.keras.optimizers.Optimizer`` objects. Users
        should use the output use the output of this wrapper as the new instance of
        their optimizer. If users create multiple optimizers, users should wrap each
        optimizer independently.

        Args:
            optimizer: tf.keras.optimizers.Optimizer
        """
        if not self.env.managed_training:
            return optimizer

        logging.debug(f"Processing wrapped optimizer {optimizer}.")
        if not self.hvd_config.use:
            self._wrapped_optimizers.append(optimizer)
            return optimizer

        hvd.require_horovod_type("tensorflow.keras", "TFKerasContext.wrap_optimizer was called.")
        if optimizer == self._compiled_optimizer:
            logging.debug(
                "Skipping wrapping optimizer as it was already wrapped during the compile call."
            )
            wrapped_optimizer = optimizer
        else:
            wrapped_optimizer = self._get_horovod_optimizer_if_using_horovod(
                optimizer=optimizer,
            )
        self._wrapped_optimizers.append(wrapped_optimizer)

        return wrapped_optimizer

    def _process_optimizer_from_compile(
        self, optimizer: tf.keras.optimizers.Optimizer
    ) -> tf.keras.optimizers.Optimizer:
        logging.debug(f"Processing compiled optimizer {optimizer}.")
        if not self.hvd_config.use:
            self._compiled_optimizer = optimizer
            return optimizer

        if optimizer in self._wrapped_optimizers:
            logging.debug(
                "Skipping wrapping optimizer that is part of the compile "
                "call as it was already wrapped explicitly via wrap_optimizer()."
            )
            wrapped_optimizer = optimizer
        else:
            wrapped_optimizer = self._get_horovod_optimizer_if_using_horovod(
                optimizer=optimizer,
            )
        self._compiled_optimizer = wrapped_optimizer

        return wrapped_optimizer

    def _select_optimizers(self) -> None:
        """
        Selects the optimizers that are going to be used. This is done for backwards
        compatibility as previously optimizers were passed in as part of the compile()
        call and are now passed in as part of `self.context.wrap_optimizers()`.
        """
        check.check_len(
            self._optimizers,
            0,
            "context._select_optimizers() called multiple times. Should be only called "
            "once by TFKerasTrialController.",
        )

        if len(self._wrapped_optimizers) > 0:
            logging.debug(f"Using wrapped optimizers: {self._wrapped_optimizers}.")
            self._optimizers = self._wrapped_optimizers
            return

        check.is_not_none(
            self._compiled_optimizer,
            "Please use `optimizer = self.context.wrap_optimizer(optimizer)` to wrap your "
            "optimizer. If using multiple optimizer, you should wrap your optimizer "
            "separately (calling wrap_optimizer() once for each optimizer).",
        )

        if self._compiled_optimizer:
            logging.info("Please switch over to using `optimizer = self.context.wrap_optimizer()`.")
            logging.debug(f"Using compiled optimizer: {self._compiled_optimizer}.")
            self._optimizers = [self._compiled_optimizer]


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

        if not self.env.managed_training:
            return model
        return self._wrap_model_with_train_fn(model, None)


class TFKerasNativeContext(det.NativeContext, TFKerasContext):
    def __init__(self, env: det.EnvContext, hvd_config: horovod.HorovodContext):
        det.NativeContext.__init__(self, env, hvd_config)
        TFKerasContext.__init__(self, env, hvd_config)

    def wrap_model(self, model: Any) -> Any:
        if not self.env.managed_training:
            return model
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
