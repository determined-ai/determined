import inspect
import json
import logging
import pathlib
import pickle
import random
import sys
import time
from abc import abstractmethod
from typing import Any, Dict, List, Optional, Tuple, Type, cast

import h5py
import numpy as np
import tensorflow as tf
from packaging import version
from tensorflow.keras.models import Model
from tensorflow.python.framework.ops import EagerTensor

import determined as det
from determined import keras, layers, tensorboard, util, workload
from determined._tf_rng import get_rng_state, set_rng_state
from determined.common import check
from determined.horovod import hvd
from determined.tensorboard.metric_writers import tensorflow

# In TF 2.6, we have to import some keras internals directly from `keras`.
if version.parse(tf.__version__) >= version.parse("2.6.0"):
    from keras.callbacks import CallbackList, make_logs, set_callback_parameters
    from keras.saving.hdf5_format import (
        load_optimizer_weights_from_hdf5_group,
        save_optimizer_weights_to_hdf5_group,
    )
    from keras.utils.mode_keys import ModeKeys
else:
    from tensorflow.python.keras.callbacks import CallbackList, make_logs, set_callback_parameters
    from tensorflow.python.keras.saving.hdf5_format import (
        load_optimizer_weights_from_hdf5_group,
        save_optimizer_weights_to_hdf5_group,
    )
    from tensorflow.python.keras.utils.mode_keys import ModeKeys


IMPOSSIBLY_LARGE_EPOCHS = sys.maxsize


def is_tf2_enabled() -> bool:
    """Checks if `tf.compat.v1.disable_v2_behavior` has been called."""
    if version.parse(tf.__version__) < version.parse("2.0.0"):
        return False

    try:
        # Try recent tf2 variant first.
        return tf._tf2.enabled()  # type: ignore
    except AttributeError:
        # Fallback to legacy option for tensorflow circa 2.2.0.
        return tf.python.tf2.enabled()  # type: ignore


def load_optimizer_weights(
    model: Model, h5group: Any, optimizer: tf.keras.optimizers.Optimizer
) -> None:
    """
    Load the optimizer states from a tf.keras model saved with
    tf.keras.models.save_model(). Ignores and prints a warning message when
    encountering a graph network. This implementation is lifted from
    tf.keras.models.load_model().
    """
    tf2_2_or_newer = version.parse(tf.__version__) >= version.parse("2.2.0")
    if model._is_graph_network or tf2_2_or_newer:  # pylint: disable=protected-access
        if tf2_2_or_newer:
            try:
                optimizer._create_all_weights(model.trainable_variables)
            except (NotImplementedError, AttributeError):
                logging.warning(
                    "Error when creating the weights of optimizer, making it "
                    "impossible to restore the saved optimizer state. As a result, "
                    "your model is starting with a freshly initialized optimizer."
                )
        else:
            # Build train function (to get weight updates).  Models that aren't
            # graph networks must wait until they are called with data to
            # _make_train_function() and so can't load optimizer weights.
            model._make_train_function()

        optimizer_weight_values = load_optimizer_weights_from_hdf5_group(h5group)
        try:
            optimizer.set_weights(optimizer_weight_values)
        except ValueError:
            logging.warning(
                "Error in loading the saved optimizer "
                "state. As a result, your model is "
                "starting with a freshly initialized "
                "optimizer."
            )
    else:
        logging.warning(
            "Sequential models without an `input_shape` "
            "passed to the first layer cannot reload their "
            "optimizer state. As a result, your model is "
            "starting with a freshly initialized optimizer."
        )


class TrialControllerMultiplexer(keras.callbacks._MultiplexerBase):
    """
    Extend _MultiplexerBase with the logic for triggering on_train_workload_end, and on_test_end
    and based on master-requested workloads.
    """

    def __init__(self, trial_controller: "TFKerasTrialController", *arg: Any, **kwarg: Any) -> None:
        super().__init__(*arg, **kwarg)
        self.trial_controller = trial_controller
        self.test_inputs = 0
        self.test_batches = 0

    def on_train_begin(self, logs: Optional[Dict] = None) -> None:
        super().on_train_begin()
        self.trial_controller._control_loop()

    def on_train_batch_end(self, batch: int, logs: Optional[Dict] = None) -> None:
        super().on_train_batch_end(batch, logs)
        assert isinstance(logs, dict)

        # Keras helpfully records the observed batch size as logs["size"].  Keras internal code
        # handles the case where logs is not present (see BaseLogger callback).  I (rb) can't
        # figure out where that would originate from, so we will include reasonable fallback
        # behavior for that case.
        num_inputs = logs.get("size", self.trial_controller.context.get_per_slot_batch_size())

        self.trial_controller._post_train_batch_end(num_inputs, logs)

    def on_test_begin(self, logs: Optional[Dict] = None) -> None:
        super().on_test_begin(logs)
        self.test_inputs = 0
        self.test_batches = 0

    def on_test_batch_end(self, batch: int, logs: Optional[Dict] = None) -> None:
        super().on_test_batch_end(batch, logs)
        assert isinstance(logs, dict)
        self.test_inputs += logs.get(
            "size", self.trial_controller.context.get_per_slot_batch_size()
        )
        self.test_batches += 1

    def _corrected_test_end(self, logs: Dict) -> None:
        super()._corrected_test_end(logs)
        self.trial_controller._stop_training_check()

    def get_test_inputs(self) -> Tuple[int, int]:
        return self.test_inputs, self.test_batches

    def _corrected_epoch_end(self, epoch: int, logs: Dict) -> None:
        super()._corrected_epoch_end(epoch, logs)
        self.trial_controller._stop_training_check()

    def on_train_end(self, logs: Optional[Dict] = None) -> None:
        # Ignore on_train_end when we manage the training loop, since in TF 2.0 (but not 2.1!) will
        # trigger an exta on_train_end when we raise the WorkerFinishedGracefully exception.
        pass

    def _corrected_train_end(self, logs: Optional[Dict] = None) -> None:
        super().on_train_end(logs)


class TFKerasTrialController(det.TrialController):
    @classmethod
    def create_metric_writer(
        cls: Type["TFKerasTrialController"],
    ) -> tensorboard.BatchMetricWriter:
        writer = tensorflow.TFWriter()
        return tensorboard.BatchMetricWriter(writer)

    @classmethod
    def pre_execute_hook(
        cls: Type["TFKerasTrialController"],
        env: det.EnvContext,
        distributed_backend: det._DistributedBackend,
    ) -> None:
        # Initialize the correct horovod.
        if distributed_backend.use_horovod():
            hvd.require_horovod_type("tensorflow.keras", "TFKerasTrial is in use.")
            hvd.init()

        # Start with a clean graph.
        tf.compat.v1.reset_default_graph()

        cls._set_random_seeds(env.trial_seed)

    @classmethod
    def _set_random_seeds(cls: Type["TFKerasTrialController"], seed: int) -> None:
        # Set identical random seeds on all training processes. When using horovod, each worker will
        # start at a unique offset in the dataset, ensuring it's processing a unique training batch.
        random.seed(seed)
        np.random.seed(seed)
        tf.compat.v1.set_random_seed(seed)

    @classmethod
    def _configure_session(
        cls: Type["TFKerasTrialController"],
        env: det.EnvContext,
        session_config: tf.compat.v1.ConfigProto,
        use_horovod: bool = False,
    ) -> Optional[tf.compat.v1.Session]:
        if not tf.executing_eagerly():
            session_config.gpu_options.allow_growth = True
            if use_horovod:
                # We launch a horovod process per GPU. Each process
                # needs to bind to a unique GPU.
                session_config.gpu_options.visible_device_list = str(hvd.local_rank())

            session = tf.compat.v1.Session(
                graph=tf.compat.v1.get_default_graph(), config=session_config
            )

            tf.compat.v1.keras.backend.set_session(session)

            return session
        else:
            gpus = tf.config.experimental.list_physical_devices("GPU")

            if len(gpus) > 0:
                local_rank = hvd.local_rank() if use_horovod else 0
                gpu = gpus[local_rank]
                tf.config.experimental.set_visible_devices(gpu, "GPU")
                tf.config.experimental.set_memory_growth(gpu, True)

            return None

    @classmethod
    def compile_model(
        cls: Type["TFKerasTrialController"],
        context: keras.TFKerasTrialContext,
        compile_args: inspect.BoundArguments,
        env: det.EnvContext,
    ) -> None:
        if "optimizer" in compile_args.arguments:
            # For backwards compatibility we check if an optimizer is passed as part
            # of the compile call. If `wrap_optimizer()` is used, we will ignore this
            # this optimizer.
            compile_args.arguments["optimizer"] = context._process_optimizer_from_compile(
                compile_args.arguments["optimizer"]
            )

        # context.model is Optional[Model]. This assert signals to mypy it can't
        # be none because we check that in `from_trial`.
        assert context.model is not None

        if context.distributed.size > 1 and version.parse("2.0.0") <= version.parse(
            tf.__version__
        ) < version.parse("2.2.0"):
            logging.info(
                "Calling `model.compile(...)` with `experimental_run_tf_function=False` to ensure "
                "TensorFlow calls `optimizer.get_gradients()` to compute gradients."
            )

            context.model.compile(
                *compile_args.args, **compile_args.kwargs, experimental_run_tf_function=False
            )
        else:
            context.model.compile(*compile_args.args, **compile_args.kwargs)

    @classmethod
    def from_trial(
        cls: Type["TFKerasTrialController"],
        trial_inst: det.Trial,
        context: det.TrialContext,
        env: det.EnvContext,
        workloads: Optional[workload.Stream] = None,
    ) -> det.TrialController:
        check.is_instance(
            context, keras.TFKerasTrialContext, "TFKerasTrialController needs a TFKerasTrialContext"
        )
        context = cast(keras.TFKerasTrialContext, context)

        check.is_instance(trial_inst, TFKerasTrial, "TFKerasTrialController needs a TFKerasTrial")
        trial = cast(TFKerasTrial, trial_inst)

        # Keras only supports horovod backend for distributed training
        session = cls._configure_session(
            env, trial.session_config(), use_horovod=context.distributed.size > 1
        )

        training_data = keras._adapt_data_from_data_loader(
            input_data=trial.build_training_data_loader(),
            batch_size=context.get_per_slot_batch_size(),
        )

        validation_data = keras._adapt_data_from_data_loader(
            input_data=trial.build_validation_data_loader(),
            batch_size=context.get_per_slot_batch_size(),
        )

        trial.build_model()
        check.is_not_none(context.model, "Please call wrap_model(...).")

        check.is_not_none(context.compile_args, "Please call model.compile(...).")
        compile_args = cast(inspect.BoundArguments, context.compile_args)

        cls.compile_model(context=context, compile_args=compile_args, env=env)

        tf_keras_callbacks = trial.keras_callbacks()

        return cls(
            context.model,
            session,
            keras.TFKerasTrainConfig(training_data, validation_data, tf_keras_callbacks),
            trial,
            context,
            env,
            workloads,
        )

    def __init__(
        self,
        model: tf.keras.models.Model,
        session: tf.compat.v1.ConfigProto,
        train_config: keras.TFKerasTrainConfig,
        trial: "TFKerasTrial",
        *args: Any,
        **kwargs: Any,
    ) -> None:
        super().__init__(*args, **kwargs)

        self.model = model
        self.session = session
        self.trial = trial

        # Configure optimizers, done for backwards compatibility.
        self.context._select_optimizers()

        keras._check_if_aggregation_frequency_will_work(
            model=self.model,
            use_horovod=self.use_horovod,
            aggregation_frequency=self.context._aggregation_frequency,
        )

        self.training_data = train_config.training_data
        self.validation_data = train_config.validation_data

        # Support the deprecated SequenceAdapter API.
        if isinstance(self.training_data, keras.SequenceAdapter):
            self.context._configure_fit(
                workers=self.training_data.workers,
                use_multiprocessing=self.training_data.use_multiprocessing,
                max_queue_size=self.training_data.max_queue_size,
            )
            # Use the provided Sequence directly.
            self.training_data = self.training_data.sequence
        if isinstance(self.validation_data, keras.SequenceAdapter):
            # Ignore these settings and use the same settings as for the fit call.
            self.validation_data = self.validation_data.sequence

        if self.context.distributed.size > 1:
            assert self.use_horovod, (
                "TF Keras trial must be launched with a horovod backend if "
                "doing distributed training"
            )

        self._check_wrap_dataset(self.training_data)
        self._check_wrap_dataset(self.validation_data)

        self.enqueuers = []  # type: List[keras._Enqueuer]

        self.wlsq = None  # type: Optional[layers.WorkloadSequencer]
        if self.workloads is None:
            self.workloads, self.wlsq = layers.make_compatibility_workloads(
                self.context._core,
                self.env,
                self.context.get_global_batch_size(),
            )

        # If a load path is provided, load weights and restore the data location.
        self.multiplexer_load_state = None  # type: Optional[Dict]
        if self.env.latest_checkpoint is not None:
            logging.info(f"Restoring trial from checkpoint {self.env.latest_checkpoint}")
            with self.context._core.checkpoint.restore_path(
                self.env.latest_checkpoint
            ) as load_path:
                self._load(load_path)

        self._configure_callbacks(train_config.callbacks)

        self.train_response_func = None  # type: Optional[workload.ResponseFunc]
        self.train_workload_metrics = []  # type: List[Dict[str, Any]]
        self.train_workload_batches = 0
        self.train_workload_inputs = 0
        self.train_workload_len = 0
        self.test_inputs = 0

        self.steps_completed = self.env.steps_completed

    def _check_wrap_dataset(self, ds: Any) -> None:
        # Ignore non-tf.data.Datasets.
        if not isinstance(ds, tf.data.Dataset):
            return
        if not self.context.dataset_initialized:
            raise det.errors.InvalidExperimentException(
                "Please use either context.wrap_dataset(dataset) for tf.data.dataset inputs"
            )

    def _configure_callbacks(self, user_callbacks: Optional[List]) -> None:
        """
        If we pass a callbacks parameter to model.fit() or model.evaluate() which is a
        pre-constructed CallbackList, Keras will not alter it.  We can use this property to
        configure the exact callback order that we want in our system.

        The implementation is based closely on from the real
        tf.keras.callbacks.configure_callbacks(), with the following differences:

          - We always assume we have the original Callbacks list.
          - We prepend and append additional Determined and Horovod callbacks
          - We create a det.keras.CallbackList instead of the normal tf.keras one.
        """

        callbacks = user_callbacks or []
        check.is_instance(
            callbacks,
            list,
            "the callbacks parameter of model.fit() or model.eval() must be a list of Callbacks",
        )

        if self.env.experiment_config.get_records_per_epoch() is None:
            for cb in callbacks:
                if util.is_overridden(cb.on_epoch_end, tf.keras.callbacks.Callback) and not getattr(
                    cb, "_skip_epoch_end_check", False
                ):
                    if isinstance(cb, keras.callbacks.Callback):
                        # New callbacks must obey the rules.
                        raise AssertionError(
                            "it is unsupported to use a Callback that defines on_epoch_end "
                            f"({type(cb).__name__}) without setting the records_per_epoch value "
                            "in the experiment config"
                        )
                    else:
                        # Pre-existing callbacks only get a warning.
                        logging.warning(
                            "It is unsupported to use a Callback that defines on_epoch_end "
                            f"({type(cb).__name__})without setting the records_per_epoch value in "
                            "the experiment config. Training will continue but on_epoch_end will "
                            "never be called."
                        )

        # Standard post-callback from the real configure_callbacks().
        # Note that we are not including BaseLogger since it is only for averaging metrics over an
        # entire epoch, and we don't report any metrics in on_epoch_end at all.
        self.model.history = keras.callbacks._DeterminedHistory()
        callbacks = callbacks + [self.model.history]

        if self.context._fit_verbose:
            # Our implementation of verbose=True.
            callbacks = [keras.callbacks._DeterminedProgress()] + callbacks

        profiler = keras.callbacks._DeterminedProfiler(
            self.prof,
            self.context.get_global_batch_size(),
        )

        callbacks = callbacks + [profiler]

        # Calculate batches per epoch.  We can only handle batches per epoch, not records per epoch,
        # because we would have to communicate after every batch to know how many records were in
        # each batch on each worker in order to trigger on_epoch_end callbacks correctly.
        batches_per_epoch = None
        records_per_epoch = self.env.experiment_config.get_records_per_epoch()
        if records_per_epoch is not None:
            batches_per_epoch = records_per_epoch // self.context.get_global_batch_size()

        # We wrap all of the callbacks in a single Multiplexer.
        self.multiplexer = TrialControllerMultiplexer(
            self,
            callbacks,
            self.is_chief,
            self.context.get_per_slot_batch_size(),
            batches_per_epoch,
            self.multiplexer_load_state,
        )
        callbacks = [self.multiplexer]

        if self.context.distributed.size > 1:
            # Horovod synchronization of initial variables should happen even before we enter our
            # control loop, in case we have an initial validation requested.
            callbacks = [hvd.callbacks.BroadcastGlobalVariablesCallback(0)] + callbacks

        # The remainder of Determined control logic is done with a custom CallbackList
        self.callback_list = CallbackList(callbacks)

        # Disable timing of callbacks in some versions of keras. This can fail in some corner-cases
        # because CallbackList is not designed to allow some callbacks to call other callbacks, and
        # they can interact very poorly.
        if hasattr(self.callback_list, "_timing"):
            self.callback_list._timing["on_train_batch_begin"] = True
            self.callback_list._timing["on_train_batch_end"] = True
            self.callback_list._timing["on_test_batch_begin"] = True
            self.callback_list._timing["on_test_batch_end"] = True
            self.callback_list._timing["on_predict_batch_begin"] = True
            self.callback_list._timing["on_predict_batch_end"] = True

        # callback_model is the model given to callbacks, where we should be checking for
        # stop_training.  In horovod dtrain or non-dtrain, it should always be self.model.
        callback_model = self.model._get_callback_model()
        self.callback_list.set_model(callback_model)

        # Fill in bogus values for most of these... some of them are very complex to calculate.
        set_callback_parameters(
            self.callback_list,
            self.model,
            do_validation=False,
            batch_size=self.context.get_per_slot_batch_size(),
            epochs=None,
            steps_per_epoch=None,
            samples=None,
            verbose=False,
            mode=ModeKeys.TRAIN,
        )

        self.callback_list.model.stop_training = False

    def _save_checkpoint(self, path: pathlib.Path) -> None:
        path.mkdir(parents=True, exist_ok=True)

        # Save model weights. We use `tf` format because `h5` does not support
        # models that subclass `tf.keras.Model` and define custom `call()`
        # and/or `train_step()` functions.
        self.model.save_weights(
            str(path.joinpath("determined-keras-model-weights")), save_format="tf"
        )

        # Save optimizer(s) weights.
        with h5py.File(path.joinpath("determined-keras-optimizer-weights.h5"), "w") as h5file:
            for idx, optimizer in enumerate(self.context._optimizers):
                opt_group = h5file.create_group(f"optimizer-{idx}")
                save_optimizer_weights_to_hdf5_group(opt_group, optimizer)

        # Save RNG state.
        rng_state = get_rng_state()

        with open(path.joinpath("rng_state.pkl"), "wb") as f:
            pickle.dump(rng_state, f)

        # Save user code.
        det.util.write_user_code(path, self.env.on_cluster)

        # Save callback(s) state.
        callbacks_state = self.multiplexer._get_state()
        with path.joinpath("determined-callbacks.v1.pkl").open("wb") as f:
            pickle.dump(callbacks_state, f)

        self.multiplexer._checkpoint_end(path)

        if self.wlsq is not None:
            with path.joinpath("workload_sequencer.pkl").open("wb") as f:
                pickle.dump(self.wlsq.get_state(), f)

        trial_cls = type(self.trial)
        with open(path.joinpath("load_data.json"), "w") as f2:
            json.dump(
                {
                    "trial_type": "TFKerasTrial",
                    "experiment_config": self.context.env.experiment_config,
                    "hparams": self.context.env.hparams,
                    "trial_cls_spec": f"{trial_cls.__module__}:{trial_cls.__qualname__}",
                },
                f2,
            )

    def _load_model_weights(self, model_weights_checkpoint_path: pathlib.Path) -> None:
        logging.info(f"Restoring model weights from {model_weights_checkpoint_path}.")
        self.model.load_weights(str(model_weights_checkpoint_path))

    def _load_optimizers_weights(self, optimizer_weights_checkpoint_path: pathlib.Path) -> None:
        logging.info(f"Restoring optimizer weights from {optimizer_weights_checkpoint_path}.")
        with h5py.File(optimizer_weights_checkpoint_path, "r") as h5file:
            if "optimizer_weights" in h5file:
                load_optimizer_weights(self.model, h5file, self.model.optimizer)
                return

            for idx, optimizer in enumerate(self.context._optimizers):
                if f"optimizer-{idx}" in h5file:
                    load_optimizer_weights(self.model, h5file[f"optimizer-{idx}"], optimizer)

    def _load_model_and_optimizer_weights_v1(self, load_path: pathlib.Path) -> None:
        self._load_model_weights(load_path.joinpath("determined-keras-model"))
        self._load_optimizers_weights(load_path.joinpath("determined-keras-model"))

    def _load_model_and_optimizer_weights_v2(self, load_path: pathlib.Path) -> None:
        self._load_model_weights(load_path.joinpath("determined-keras-model.h5"))
        self._load_optimizers_weights(load_path.joinpath("determined-keras-model.h5"))

    def _load_model_and_optimizer_weights_v3(self, load_path: pathlib.Path) -> None:
        self._load_model_weights(load_path.joinpath("determined-keras-model-weights"))
        self._load_optimizers_weights(load_path.joinpath("determined-keras-optimizer-weights.h5"))

    def _load(self, load_path: pathlib.Path) -> None:
        # Find model code path, we check multiple naming conventions for backwards compatibility.
        if load_path.joinpath("determined-keras-model.h5").exists():
            self._load_model_and_optimizer_weights_v2(load_path)
        elif load_path.joinpath("determined-keras-optimizer-weights.h5").exists():
            self._load_model_and_optimizer_weights_v3(load_path)
        else:
            self._load_model_and_optimizer_weights_v1(load_path)

        # Load RNG state.
        try:
            with open(load_path.joinpath("rng_state.pkl"), "rb") as f:
                rng_state = pickle.load(f)

            set_rng_state(rng_state)
        except IOError:
            logging.warning("Checkpoint did not include RNG state.")

        # Load callbacks.
        cb_state_path = load_path.joinpath("determined-callbacks.v1.pkl")
        if cb_state_path.exists():
            with cb_state_path.open("rb") as f:
                self.multiplexer_load_state = pickle.load(f)

        # Load WorkloadSequencer state.
        wlsq_path = load_path.joinpath("workload_sequencer.pkl")
        if self.wlsq is not None and wlsq_path.exists():
            with wlsq_path.open("rb") as f:
                self.wlsq.load_state(pickle.load(f))

    def run(self) -> None:
        with self.prof:
            try:
                self._launch_fit()
            except det.errors.WorkerFinishedGracefully:
                pass
            finally:
                self._stop_enqueuers()

    def _launch_fit(self) -> None:
        training_data = self.training_data

        if isinstance(training_data, tf.keras.utils.Sequence):
            # Handle args from fit(): shuffle, workers, use_multiprocessing, and max_queue_size.
            enqueuer = keras._build_enqueuer(
                sequence=training_data,
                workers=self.context._fit_workers,
                use_multiprocessing=self.context._fit_use_multiprocessing,
                max_queue_size=self.context._fit_max_queue_size,
                shard_rank=self.context.distributed.rank,
                num_shards=self.context.distributed.size,
                repeat=True,
                shuffle=self.context._fit_shuffle,
                shuffle_seed=self.context.get_trial_seed(),
                prior_batches_trained=self.env.steps_completed,
            )
            enqueuer.start()
            self.enqueuers.append(enqueuer)
            training_data = enqueuer.data()

        if isinstance(training_data, tf.data.Dataset):
            training_data = training_data.repeat()
            if self.context._fit_shuffle:
                logging.warning(
                    "You set shuffle=True for a tf.data.Dataset, which will be ignored. "
                    "Please call .shuffle() on your dataset instead."
                )

        self.model.fit(
            training_data,
            class_weight=self.context._fit_class_weight,
            callbacks=self.callback_list,
            shuffle=False,
            steps_per_epoch=sys.maxsize,
            epochs=IMPOSSIBLY_LARGE_EPOCHS,
            validation_split=0,
            verbose=0,
            workers=0,
        )

    def _launch_evaluate(self) -> Any:
        validation_data = self.validation_data
        steps = None

        if isinstance(validation_data, tf.keras.utils.Sequence):
            # Calculate the length of our validation shard.
            steps = len(validation_data)
            if self.context.distributed.get_size() > 1:
                size = self.context.distributed.get_size()
                rank = self.context.distributed.get_rank()
                steps = steps // size + (1 if steps % size > rank else 0)

            # Handle args from fit(): shuffle, workers, use_multiprocessing, and max_queue_size.
            enqueuer = keras._build_enqueuer(
                sequence=validation_data,
                workers=self.context._fit_workers,
                use_multiprocessing=self.context._fit_use_multiprocessing,
                max_queue_size=self.context._fit_max_queue_size,
                shard_rank=self.context.distributed.get_rank(),
                num_shards=self.context.distributed.get_size(),
                repeat=False,
                shuffle=False,
                shuffle_seed=0,
                prior_batches_trained=0,
            )
            enqueuer.start()
            self.enqueuers.append(enqueuer)
            validation_data = enqueuer.data()

        if isinstance(validation_data, tf.data.Dataset):
            # Handle validation_steps, which in Keras only applies to tf.data.Datasets.
            steps = self.context._fit_validation_steps

        # Starting in TF 2.2 users may define custom test_step() that do
        # not use the model metrics.
        use_model_metrics = not (
            version.parse(tf.__version__) >= version.parse("2.2.0")
            and is_tf2_enabled()
            and tf.executing_eagerly()
        )
        evaluate_kwargs = {} if use_model_metrics else {"return_dict": True}

        if self.env.test_mode:
            steps = 1

        metrics_values = self.model.evaluate(
            validation_data,
            callbacks=self.callback_list,
            steps=steps,
            verbose=0,
            workers=0,
            **evaluate_kwargs,
        )
        logging.debug(f"Worker finished model.evaluate() with metrics: {metrics_values}.")

        # Clean up the enqueuer if we started one.
        if isinstance(self.validation_data, tf.keras.utils.Sequence):
            enqueuer.stop()
            self.enqueuers.remove(enqueuer)

            # A special side-effect of converting the keras sequence to a generator and passing
            # steps explicitly is that keras will exit our generator after N steps and the
            # Sequence.on_epoch_end() that normally runs after the last yield won't run at all
            # because the fit loop will call next() exactly `steps` times.  So we try to match the
            # exact keras behavior by manually calling on_epoch_end() here.
            self.validation_data.on_epoch_end()

        # If the model was compiled with metrics=None, metrics_value will be a single value.
        if not isinstance(metrics_values, (tuple, list, dict)):
            metrics_values = (metrics_values,)

        if use_model_metrics:
            metrics = make_logs(self.model, {}, metrics_values, ModeKeys.TEST, prefix="val_")
        else:
            check.is_instance(metrics_values, dict)
            metrics = {f"val_{k}": v for k, v in metrics_values.items()}

        return metrics

    def _control_loop(self) -> None:
        assert self.workloads is not None
        for wkld, response_func in self.workloads:
            logging.debug(f"Received wkld {wkld.kind}.")

            try:
                if wkld.kind == workload.Workload.Kind.RUN_STEP:
                    # Configure the state for a training step.
                    self.train_response_func = response_func
                    self.train_workload_batches = 0
                    self.train_workload_inputs = 0
                    self.train_workload_metrics = []
                    self.train_workload_len = wkld.num_batches
                    self.multiplexer.set_batches_requested(wkld.num_batches)
                    return

                elif wkld.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                    action = "validation"
                    response = {
                        "metrics": self._compute_validation_metrics(),
                        "stop_requested": self.context.get_stop_requested(),
                    }  # type: workload.Response

                elif wkld.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                    action = "checkpointing"
                    if self.is_chief:
                        metadata = {
                            "determined_version": det.__version__,
                            "steps_completed": self.steps_completed,
                            "framework": f"tensorflow-{tf.__version__}",
                            "format": "saved_weights",
                        }
                        with self.context._core.checkpoint.store_path(metadata) as (
                            path,
                            storage_id,
                        ):
                            self._save_checkpoint(path)
                        response = {"uuid": storage_id}
                    else:
                        response = {}

                else:
                    raise AssertionError(f"Unknown workload kind {wkld.kind}.")

            except det.InvalidHP as e:
                logging.info(f"Invalid hyperparameter exception during {action}: {e}")
                response = workload.InvalidHP()
            response_func(response)
            self.upload_tb_files()

        # End-of-training.
        self.multiplexer._corrected_train_end()
        raise det.errors.WorkerFinishedGracefully()

    def _allreduce_logs(self, logs: Dict) -> Dict:
        if not (self.context.distributed.size > 1):
            return logs
        # Reduce logs in key-sorted to be deterministic across workers.
        keys = sorted(logs)
        logging.debug(f"all-reducing logs on worker {hvd.rank()} for {len(keys)} keys {keys}.")
        return {
            key: np.array(self._hvd_allreduce(logs[key], average=True, name=key)) for key in keys
        }

    def _hvd_allreduce(self, value: Any, average: bool, name: str) -> Any:
        # The signature of our horovod allreduce changed after we rebased onto 0.21.
        hvd_sig = inspect.signature(hvd.allreduce)
        horovod_kwargs = {
            "value": value,
            "name": name,
        }  # type: Dict[str, Any]

        if "op" in hvd_sig.parameters:
            horovod_kwargs["op"] = hvd.Average if average else hvd.Sum

            # average has not yet been removed but it's deprecated. It defaults
            # to true and horovod does not support specifying an op while having
            # average be not None.
            if "average" in hvd_sig.parameters:
                horovod_kwargs["average"] = None
        else:
            horovod_kwargs["average"] = average

        return hvd.allreduce(**horovod_kwargs)

    def _convert_possible_tensor(self, possible_tensor: Any) -> Any:
        if isinstance(possible_tensor, EagerTensor):
            # Horovod and / or TensorFlow may promote scalars to tensors in eager mode.
            return possible_tensor.numpy()
        return possible_tensor

    def _post_train_batch_end(self, num_inputs: int, logs: Dict) -> None:
        # Remove default keras metrics we aren't interested in like "batch" and "size".
        self.train_workload_metrics.append(
            {
                k: self._convert_possible_tensor(v)
                for k, v in logs.items()
                if k not in {"batch", "size"}
            }
        )
        self.steps_completed += 1
        self.train_workload_inputs += num_inputs
        self.train_workload_batches += 1
        if self.train_workload_batches != self.train_workload_len:
            return

        if self.train_response_func is None:
            raise AssertionError(
                "train_response_func is not set.  This should not be possible; please file an "
                "issue at github.com/determined-ai/determined so we can fix this bug."
            )

        if self.context.distributed.size > 1:
            self.train_workload_inputs = self._hvd_allreduce(
                self.train_workload_inputs, average=False, name="train_num_inputs"
            )
            self.train_workload_inputs = self._convert_possible_tensor(self.train_workload_inputs)

        # Return only the latest metrics, which is the running average for all trained batches in
        # the step (Keras does not report individual logs, only running averages at any point).
        final_metrics = self.train_workload_metrics[-1]
        if self.env.experiment_config.average_training_metrics_enabled():
            final_metrics = self._allreduce_logs(final_metrics)

        self.multiplexer._train_workload_end(final_metrics)
        self._stop_training_check()

        if self.is_chief:
            if self.multiplexer.train_workload_begin_time is not None:
                step_duration = time.time() - self.multiplexer.train_workload_begin_time
                self.multiplexer.train_workload_begin_time = None
                logging.info(
                    det.util.make_timing_log(
                        "trained",
                        step_duration,
                        self.train_workload_inputs,
                        self.train_workload_len,
                    )
                )
            response = {
                "metrics": {
                    "num_inputs": self.train_workload_inputs,
                    "batch_metrics": self.train_workload_metrics,
                    "avg_metrics": final_metrics,
                },
                "stop_requested": self.context.get_stop_requested(),
            }  # type: workload.Response
            self.metric_writer.on_train_step_end(
                steps_completed=self.steps_completed,
                metrics=final_metrics,
                batch_metrics=self.train_workload_metrics,
            )
        else:
            response = {}

        self.train_response_func(response)
        self.train_response_func = None

        self.upload_tb_files()

        self._control_loop()

        # Always reset metrics before starting a new training step.
        self.model.reset_metrics()

    def _compute_validation_metrics(self) -> workload.Response:
        validation_start_time = time.time()
        metrics = self._launch_evaluate()
        num_inputs, num_batches = self.multiplexer.get_test_inputs()

        if self.context.distributed.size > 1:
            # Use a global ZMQ barrier here because we have observed cases where hvd.allreduce
            # may hang when called minutes apart by different workers which may happen if
            # workers complete evaluation at different speeds.
            _ = self.context.distributed.gather(None)

            num_inputs = hvd.allreduce(num_inputs, average=False, name="validation_num_inputs")
            if isinstance(num_inputs, EagerTensor):
                # Horovod will promote an int to a tensor in eager mode.
                num_inputs = num_inputs.numpy()
            num_batches = hvd.allreduce(num_batches, average=False, name="validation_num_batches")
            if isinstance(num_batches, EagerTensor):
                num_batches = num_batches.numpy()

        metrics = self._allreduce_logs(metrics)
        check.gt(len(metrics), 0)

        self.multiplexer._test_end(metrics)

        if not self.is_chief:
            return {}

        step_duration = time.time() - validation_start_time
        logging.info(det.util.make_timing_log("validated", step_duration, num_inputs, num_batches))

        self.metric_writer.on_validation_step_end(self.steps_completed, metrics)
        self.upload_tb_files()
        return {"num_inputs": num_inputs, "validation_metrics": metrics}

    def _stop_training_check(self) -> None:
        # Detect when users set stop_training and convert it to a set_stop_requested.
        if self.multiplexer.model.stop_training:
            if self.is_chief:
                self.multiplexer.model.stop_training = False
                self.context.set_stop_requested(True)
            else:
                logging.debug("cancelling model.stop_training on non-chief worker")
                self.multiplexer.model.stop_training = True

    def _stop_enqueuers(self) -> None:
        for enqueuer in self.enqueuers:
            enqueuer.stop()


class TFKerasTrial(det.Trial):
    """
    To implement a new ``tf.keras`` trial, subclass this class and
    implement the abstract methods described below (:meth:`build_model`,
    :meth:`build_training_data_loader`, and :meth:`build_validation_data_loader`).
    In most cases you should provide a custom :meth:`__init__` method as well.

    By default, experiments use TensorFlow 2.x. To configure your trial to use
    legacy TensorFlow 1.x, specify a TensorFlow 1.x image in the
    :ref:`environment.image <exp-environment-image>` field of the experiment
    configuration (e.g.,
    ``determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-0.19.10``).

    Trials default to using eager execution with TensorFlow 2.x but not with
    TensorFlow 1.x. To override the default behavior, call the appropriate
    function at the top of your code. For example, if you want to disable
    eager execution while using TensorFlow 2.x, call
    ``tf.compat.v1.disable_eager_execution`` after your import statements.
    If you are using TensorFlow 1.x in eager mode, please add
    ``experimental_run_tf_function=False`` to your model compile function.
    """

    trial_controller_class = TFKerasTrialController
    trial_context_class = keras.TFKerasTrialContext

    def __init__(self, context: keras.TFKerasTrialContext) -> None:
        """
        Initializes a trial using the provided ``context``.

        This method should typically be overridden by trial definitions: at minimum,
        it is important to store ``context`` as an instance variable so that
        it can be accessed by other methods of the trial class. This can also be a
        convenient place to initialize other state that is shared between methods.
        """
        self.context = context

    @abstractmethod
    def build_model(self) -> tf.keras.models.Model:
        """
        Returns the deep learning architecture associated with a trial.  The
        architecture might depend on the current values of the model's
        hyperparameters, which can be accessed via :func:`context.get_hparam()
        <determined.TrialContext.get_hparam>`.  This function returns a
        ``tf.keras.Model`` object.

        After constructing the ``tf.keras.Model`` object, users **must** do two
        things before returning it:

          1. Wrap the model using :meth:`context.wrap_model()
             <determined.keras.TFKerasTrialContext.wrap_model>`.

          2. Compile the model using ``model.compile()``.
        """
        pass

    @abstractmethod
    def build_training_data_loader(self) -> keras.InputData:
        """
        Defines the data loader to use during training.

        Should return one of the following:
            1) A tuple ``(x_train, y_train)``, where ``x_train`` is a NumPy array
            (or array-like), a list of arrays (in case the model has multiple inputs), or
            a dict mapping input names to the corresponding array, if the model has named inputs.
            ``y_train`` should be a NumPy array.

            2) A tuple ``(x_train, y_train, sample_weights)``
            of NumPy arrays.

            3) A `tf.data.Dataset
            <https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/data/Dataset>`__ returning
            a tuple of either ``(inputs, targets)`` or ``(inputs, targets, sample_weights)``.

            4) A `keras.utils.Sequence
            <https://tensorflow.org/api_docs/python/tf/keras/utils/Sequence>`__ returning a tuple
            of either ``(inputs, targets)`` or ``(inputs, targets, sample weights)``.

        When using ``tf.data.Dataset``, you must wrap the dataset using
        :meth:`determined.keras.TFKerasTrialContext.wrap_dataset`. This wrapper is used
        to shard the dataset for distributed training. For optimal performance, users
        should wrap a dataset immediately after creating it.

        .. warning::
            If you are using ``tf.data.Dataset``, Determined’s support for
            automatically checkpointing the dataset does not currently work correctly.
            This means that resuming workloads will start from the beginning of the dataset
            if using ``tf.data.Dataset``.
        """
        pass

    @abstractmethod
    def build_validation_data_loader(self) -> keras.InputData:
        """
        Defines the data loader to use during validation.

        Should return one of the following:
            1) A tuple ``(x_val, y_val)``, where ``x_val`` is a NumPy array
            (or array-like), a list of arrays (in case the model has multiple inputs), or
            a dict mapping input names to the corresponding array, if the model has named inputs.
            ``y_val`` should be a NumPy array.

            2) A tuple ``(x_val, y_val, sample_weights)``
            of NumPy arrays.

            3) A `tf.data.Dataset
            <https://www.tensorflow.org/versions/r1.15/api_docs/python/tf/data/Dataset>`__ returning
            a tuple of either ``(inputs, targets)`` or ``(inputs, targets, sample_weights)``.

            4) A `keras.utils.Sequence
            <https://tensorflow.org/api_docs/python/tf/keras/utils/Sequence>`__ returning a tuple
            of either ``(inputs, targets)`` or ``(inputs, targets, sample weights)``.

        When using ``tf.data.Dataset``, you must wrap the dataset using
        :meth:`determined.keras.TFKerasTrialContext.wrap_dataset`. This wrapper is used
        to shard the dataset for distributed training. For optimal performance, users
        should wrap a dataset immediately after creating it.
        """
        pass

    def session_config(self) -> tf.compat.v1.ConfigProto:
        """
        Specifies the `tf.ConfigProto
        <https://www.tensorflow.org/api_docs/python/tf/compat/v1/ConfigProto>`__ to be
        used by the TensorFlow session. By default,
        ``tf.ConfigProto(allow_soft_placement=True)`` is used.
        """
        return tf.compat.v1.ConfigProto(allow_soft_placement=True)

    def keras_callbacks(self) -> List[tf.keras.callbacks.Callback]:
        """
        Specifies a list of :class:`determined.keras.callbacks.Callback` objects to be used during
        training.

        .. note:
           Note that :class:`determined.keras.callbacks.Callback` is a subclass of
           `tf.keras.callback.Callback
           <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/Callback>`__ objects
           which supports stateful callbacks that can be checkpointed an restored mid-training.

           Please see :class:`determined.keras.callbacks.Callback` for a summary of differences
           between normal Keras callbacks and Determined Keras callbacks.

        .. warning:
           For legacy callbacks which do not subclass :class:`determined.keras.callbacks.Callback`,
           if ``records_per_epoch`` is not set in the experiement config for an experiment,
           ``on_epoch_end`` will never be called.
        """
        return []
