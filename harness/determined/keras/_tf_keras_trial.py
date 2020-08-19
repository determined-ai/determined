r"""
"""

import inspect
import logging
import os
import pathlib
import random
import sys
from abc import abstractmethod
from typing import Any, Dict, List, Optional, TextIO, cast

import h5py
import numpy as np
import tensorflow as tf
from packaging import version
from tensorflow.keras.models import Model
from tensorflow.python.keras.callbacks import make_logs
from tensorflow.python.keras.saving.hdf5_format import load_optimizer_weights_from_hdf5_group
from tensorflow.python.keras.utils.mode_keys import ModeKeys

import determined as det
from determined import horovod, keras, profile, workload
from determined.horovod import hvd
from determined_common import check

IMPOSSIBLY_LARGE_EPOCHS = sys.maxsize


def load_optimizer_weights(model: Model, load_path: pathlib.Path) -> None:
    """
    Load the optimizer states from a tf.keras model saved with
    tf.keras.models.save_model(). Ignores and prints a warning message when
    encountering a graph network. This implementation is lifted from
    tf.keras.models.load_model().
    """
    f = h5py.File(str(load_path), mode="r")
    if "optimizer_weights" in f:
        tf2_2_or_newer = version.parse(tf.__version__) >= version.parse("2.2.0")
        if model._is_graph_network or tf2_2_or_newer:  # pylint: disable=protected-access
            if tf2_2_or_newer:
                try:
                    model.optimizer._create_all_weights(model.trainable_variables)
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

            optimizer_weight_values = load_optimizer_weights_from_hdf5_group(f)
            try:
                model.optimizer.set_weights(optimizer_weight_values)
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


class DeterminedEarlyStoppingCallback(tf.keras.callbacks.Callback):  # type: ignore
    """
    DeterminedEarlyStoppingCallback converts a stop request, so that Determined
    can handle the stop request by finishing the step and checkpointing.
    """

    def __init__(self, tf_keras_trial_controller: "TFKerasTrialController") -> None:
        self.tf_keras_trial_controller = tf_keras_trial_controller

    def _convert_stop_training(self) -> None:
        # We use stop_training to exit out of the training loop, but we set
        # expect_terminate when we do so.
        if self.model.stop_training and not self.tf_keras_trial_controller.expect_terminate:
            self.model.stop_training = False
            self.tf_keras_trial_controller.context.set_stop_requested(True)

    def on_epoch_end(self, _: int, logs: Any = None) -> None:
        self._convert_stop_training()

    def on_train_end(self, _: int, logs: Any = None) -> None:
        self._convert_stop_training()


class WaitForInstructionsCallback(tf.keras.callbacks.Callback):  # type: ignore
    """
    WaitForInstructionsCallback allows a separate process to control this trial.
    This callback, which is triggered from inside the model.fit(), checks with the
    main process if it should stay inside the fit loop (training step or checkpoint)
    or if it should exit the fit() loop (validation).
    """

    # Set the callback to run on all workers.
    _chief_worker_only = False

    def __init__(self, tf_keras_trial_controller: "TFKerasTrialController") -> None:
        self.tf_keras_trial_controller = tf_keras_trial_controller
        self.batches_processed = 0
        self.metrics = []  # type: List[Dict[str, Any]]

    def on_train_batch_end(self, _: int, logs: Any = None) -> None:
        check.is_in("loss", logs)

        # Remove default keras metrics we aren't interested in like "batch" and
        # "size".
        self.metrics.append({k: v for k, v in logs.items() if k not in {"batch", "size"}})
        self.batches_processed += 1
        if self.batches_processed != self.tf_keras_trial_controller.num_batches:
            return

        check.is_not_none(
            self.tf_keras_trial_controller.train_response_func,
            "Callback should avoid calling model.predict(), "
            "as this will affect Determined training behavior",
        )
        response_func = cast(
            workload.ResponseFunc, self.tf_keras_trial_controller.train_response_func
        )

        # TODO(DET-1278): Average training metrics across GPUs when using Horovod.
        num_inputs = (
            self.tf_keras_trial_controller.num_batches * self.tf_keras_trial_controller.batch_size
        )

        if self.tf_keras_trial_controller.is_chief:
            response = {
                "metrics": det.util.make_metrics(num_inputs, self.metrics),
                "stop_requested": self.tf_keras_trial_controller.context.get_stop_requested(),
            }
            response_func(response)
        else:
            response_func(workload.Skipped())

        self.tf_keras_trial_controller.train_response_func = None
        self.metrics = []
        self.batches_processed = 0

        self.tf_keras_trial_controller.run()

        if self.model.stop_training and version.parse(tf.__version__) >= version.parse("2.2.0"):
            # Starting with TF 2.2, `model.stop_training` is only checked at the end of epochs.
            raise det.errors.WorkerFinishedGracefully


class DeterminedProfiler(tf.keras.callbacks.Callback):  # type: ignore
    """
    DeterminedProfiler profiles the training time per batch, outputting the results to a log file.
    """

    OUTPUT_FILENAME = "/profile/det_profiling.log"

    def __init__(self, profile_frequency: Optional[int], out_file: str) -> None:
        self._profile_frequency = profile_frequency
        self._out_file = out_file
        self._profile_file: Optional[TextIO] = None
        self._count = 0

    def on_train_begin(self, _: Any) -> None:
        if self._profile_frequency:
            if not os.path.isdir(os.path.dirname(self._out_file)):
                raise AssertionError(
                    f"{self._out_file} is not a valid output file, because the directory "
                    f"{os.path.dirname(self._out_file)} does not exist"
                )
            # Set buffering to 1 because the `on_train_end` hook does not get
            # hit, and so we have no good way of ensuring the file flushes
            # before we end the process.
            self._profile_file = open(self._out_file, "a", buffering=1)

    def _should_profile(self) -> bool:
        return (
            self._profile_frequency is not None
            and self._count == self._profile_frequency - 1
            and self._profile_file is not None
        )

    def on_train_batch_begin(self, batch: int, _: Any = None) -> None:
        if self._should_profile():
            self._profile_file = cast(TextIO, self._profile_file)
            self._train_batch_start_time = profile.log_start(
                "batch", self._profile_file, batch=batch
            )

    def on_train_batch_end(self, batch: int, _: Any = None) -> None:
        if self._profile_frequency:
            if self._should_profile():
                self._profile_file = cast(TextIO, self._profile_file)
                profile.log_end(
                    "batch", self._profile_file, self._train_batch_start_time, batch=batch
                )
            self._count = (self._count + 1) % self._profile_frequency


class TFKerasTrialController(det.LoopTrialController):
    def __init__(
        self,
        model: tf.keras.models.Model,
        session: tf.compat.v1.ConfigProto,
        train_config: keras.TFKerasTrainConfig,
        *args: Any,
        **kwargs: Any,
    ) -> None:
        super().__init__(*args, **kwargs)

        self.model = model
        self.session = session

        self._train_input_manager, self._validation_input_manager = keras._init_input_managers(
            context=self.context, train_config=train_config
        )

        # If callbacks are set to None, then use an empty list.
        self.tf_keras_callbacks = train_config.callbacks or []

        # If a load path is provided, load weights and restore the data location.
        self._load()

        self.fit_loop_started = False

        # Store the response_func for train_for_step workloads while we do the training.
        self.train_response_func = None  # type: Optional[workload.ResponseFunc]

        self.model.stop_training = False
        self.expect_terminate = False

    @staticmethod
    def _configure_session(
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
        session_config: tf.compat.v1.ConfigProto,
    ) -> Optional[tf.compat.v1.Session]:
        if not tf.executing_eagerly():
            session_config.gpu_options.allow_growth = True
            if hvd_config.use:
                # We launch a horovod process per GPU. Each process
                # needs to bind to a unique GPU.
                session_config.gpu_options.visible_device_list = str(env.slot_ids[hvd.local_rank()])

            session = tf.compat.v1.Session(
                graph=tf.compat.v1.get_default_graph(), config=session_config
            )

            tf.compat.v1.keras.backend.set_session(session)

            return session
        else:
            gpus = tf.config.experimental.list_physical_devices("GPU")

            if len(gpus) > 0:
                local_rank = hvd.local_rank() if hvd_config.use else 0
                gpu = gpus[local_rank]
                tf.config.set_visible_devices(gpu, "GPU")
                tf.config.experimental.set_memory_growth(gpu, True)

            return None

    @staticmethod
    def pre_execute_hook(env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        # Initialize the correct horovod.
        if hvd_config.use:
            hvd.require_horovod_type("tensorflow.keras", "TFKerasTrial is in use.")
            hvd.init()

        # Start with a clean graph.
        tf.compat.v1.reset_default_graph()

        TFKerasTrialController._set_random_seeds(env.trial_seed)

        # For the Native API we must configure the Session before running user code.
        if env.experiment_config.native_enabled():
            session_config = tf.compat.v1.ConfigProto(allow_soft_placement=True)
            TFKerasTrialController._configure_session(env, hvd_config, session_config)

    @staticmethod
    def _set_random_seeds(seed: int) -> None:
        # Set identical random seeds on all training processes. When using horovod, each worker will
        # start at a unique offset in the dataset, ensuring it's processing a unique training batch.
        random.seed(seed)
        np.random.seed(seed)
        tf.compat.v1.set_random_seed(seed)

    @staticmethod
    def from_trial(
        trial_inst: det.Trial,
        context: det.TrialContext,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: det.RendezvousInfo,
        hvd_config: horovod.HorovodContext,
    ) -> det.TrialController:
        check.is_instance(
            context, keras.TFKerasTrialContext, "TFKerasTrialController needs a TFKerasTrialContext"
        )
        context = cast(keras.TFKerasTrialContext, context)

        check.is_instance(trial_inst, TFKerasTrial, "TFKerasTrialController needs a TFKerasTrial")
        trial = cast(TFKerasTrial, trial_inst)

        session = TFKerasTrialController._configure_session(env, hvd_config, trial.session_config())

        training_x, training_y, training_sample_weight = keras._get_x_y_and_sample_weight(
            input_data=trial.build_training_data_loader()
        )
        training_data = keras._adapt_keras_data(
            x=training_x,
            y=training_y,
            sample_weight=training_sample_weight,
            batch_size=context.get_per_slot_batch_size(),
            drop_leftovers=True,
        )

        val_x, val_y, val_sample_weight = keras._get_x_y_and_sample_weight(
            input_data=trial.build_validation_data_loader()
        )
        validation_data = keras._adapt_keras_data(
            x=val_x,
            y=val_y,
            sample_weight=val_sample_weight,
            batch_size=context.get_per_slot_batch_size(),
            drop_leftovers=False,
        )

        trial.build_model()
        check.is_not_none(context.model, "Please call wrap_model(...).")

        check.is_not_none(context.compile_args, "Please call model.compile(...).")
        compile_args = cast(inspect.BoundArguments, context.compile_args)

        TFKerasTrialController.compile_model(
            context=context, compile_args=compile_args, env=env, hvd_config=hvd_config
        )

        tf_keras_callbacks = trial.keras_callbacks()

        return TFKerasTrialController(
            context.model,
            session,
            keras.TFKerasTrainConfig(training_data, validation_data, tf_keras_callbacks),
            context,
            env,
            workloads,
            load_path,
            rendezvous_info,
            hvd_config,
        )

    @staticmethod
    def from_native(
        context: det.NativeContext,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: det.RendezvousInfo,
        hvd_config: horovod.HorovodContext,
    ) -> det.TrialController:
        check.is_instance(
            context,
            keras.TFKerasNativeContext,
            "TFKerasTrialController needs a TFKerasSprinkleContext",
        )
        context = cast(keras.TFKerasNativeContext, context)

        check.is_not_none(context.model, "Please call wrap_model(...).")

        check.is_not_none(context.compile_args, "Please call model.compile(...).")
        check.is_not_none(
            context.train_config, "Please call model.fit(...) or model.fit_generator(...)."
        )

        # For the Native API, we would break the user's model if we changed the session
        # right now, so we have to trust the user did not modify what we set previously.
        #
        # TODO(ryan): Fix this, probably with a function for configuring the backend session.
        session = tf.compat.v1.keras.backend.get_session()

        compile_args = cast(inspect.BoundArguments, context.compile_args)
        train_config = cast(keras.TFKerasTrainConfig, context.train_config)

        TFKerasTrialController.compile_model(
            context=context, compile_args=compile_args, env=env, hvd_config=hvd_config
        )

        return TFKerasTrialController(
            context.model,
            session,
            train_config,
            context,
            env,
            workloads,
            load_path,
            rendezvous_info,
            hvd_config,
        )

    @staticmethod
    def compile_model(
        context: keras.TFKerasContext,
        compile_args: inspect.BoundArguments,
        env: det.EnvContext,
        hvd_config: horovod.HorovodContext,
    ) -> None:
        (
            context.model,
            compile_args.arguments["optimizer"],
        ) = keras._get_multi_gpu_model_and_optimizer(
            pre_compiled_model=context.model,
            optimizer=compile_args.arguments["optimizer"],
            env=env,
            hvd_config=hvd_config,
            profile_frequency=env.experiment_config.profile_frequency(),
            profile_filename=DeterminedProfiler.OUTPUT_FILENAME,
        )

        if hvd_config.use and version.parse("2.0.0") <= version.parse(
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

    @staticmethod
    def support_determined_native() -> bool:
        return True

    @staticmethod
    def supports_multi_gpu_training() -> bool:
        return True

    def _load(self) -> None:
        if not self.load_path:
            return

        # Load model.
        if self.load_path.joinpath("determined-keras-model.h5").exists():
            full_ckpt_path = self.load_path.joinpath("determined-keras-model.h5")
        else:
            full_ckpt_path = self.load_path.joinpath("determined-keras-model")

        logging.info(f"Restoring checkpoint from {full_ckpt_path}")
        self.model.load_weights(str(full_ckpt_path))
        load_optimizer_weights(self.model, full_ckpt_path)

    def _save_checkpoint(self, path: pathlib.Path) -> workload.Response:
        # We assume that at least one training step has completed when saving a
        # checkpoint.

        if not self.is_chief:
            return workload.Skipped()

        # Save training data iterator position.
        path.mkdir(parents=True, exist_ok=True)

        # Save model.
        self.model.save(path.joinpath("determined-keras-model.h5"), save_format="h5")

        det.util.write_user_code(path)

        return {"framework": f"tensorflow-{tf.__version__}", "format": "h5"}

    def run(self) -> None:
        for wkld, args, response_func in self.workloads:
            logging.debug(f"Received wkld {wkld.kind} with args {args}.")

            if wkld.kind == workload.Workload.Kind.RUN_STEP:
                # Store the train_response_func for later.
                self.train_response_func = response_func
                self.num_batches = wkld.num_batches

                # There are two possibilities when a RUN_STEP workload is recieved.
                # 1) This is the first training step seen by the trial
                #    container. In this case, enter the tf.keras fit() training loop.
                # 2) This is _not_ the first training step, meaning that the
                #    tf.keras fit() training loop is already active and paused.
                #    break to re-enter the training loop.
                if not self.fit_loop_started:
                    try:
                        self._launch_fit()
                    except det.errors.WorkerFinishedGracefully:
                        pass

                    if not self.expect_terminate:
                        raise AssertionError(
                            "Training loop exited unexpectedly but without throwing any errors. "
                            "This is possibly due to a user callback causing the training loop to "
                            "exit, which is not supported at this time."
                        )
                break

            elif wkld.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                response_func(
                    det.util.wrap_metrics(
                        self.compute_validation_metrics(), self.context.get_stop_requested()
                    )
                )
            elif wkld.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.len_eq(args, 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                response_func(self._save_checkpoint(path))
            elif wkld.kind == workload.Workload.Kind.TERMINATE:
                self.model.stop_training = True
                self.expect_terminate = True
                response_func({} if self.is_chief else workload.Skipped())
                break
            else:
                raise AssertionError(f"Unknown wkld kind {wkld.kind}.")

    def _launch_fit(self) -> None:
        check.false(self.fit_loop_started)
        self.fit_loop_started = True

        self.tf_keras_callbacks.append(DeterminedEarlyStoppingCallback(self))
        self.tf_keras_callbacks.append(WaitForInstructionsCallback(self))

        profile_frequency = self.env.experiment_config.profile_frequency()
        if profile_frequency:
            self.tf_keras_callbacks.append(
                DeterminedProfiler(profile_frequency, DeterminedProfiler.OUTPUT_FILENAME)
            )

        if self.hvd_config.use:
            # When using horovod broadcast initial variable states from rank 0 to
            # all other processes.
            self.tf_keras_callbacks.append(hvd.callbacks.BroadcastGlobalVariablesCallback(0))

        (
            training_input,
            batches_per_epoch,
        ) = self._train_input_manager.get_training_input_and_batches_per_epoch()

        _ = self.model.fit(
            training_input,
            callbacks=self.tf_keras_callbacks,
            shuffle=False,
            steps_per_epoch=batches_per_epoch,
            initial_epoch=self._train_input_manager.get_initial_epoch(),
            epochs=IMPOSSIBLY_LARGE_EPOCHS,
            validation_split=0,
            verbose=0,
        ).history

    def compute_validation_metrics(self) -> workload.Response:
        (
            validation_data,
            validation_steps,
        ) = self._validation_input_manager.get_validation_input_and_num_batches()

        metrics_values = self.model.evaluate(validation_data, steps=validation_steps, verbose=0)

        # If the model was compiled with metrics=None, metrics_value will be a single value.
        if not isinstance(metrics_values, (tuple, list)):
            metrics_values = (metrics_values,)

        if self.hvd_config.use:
            for index, metric_value in enumerate(metrics_values):
                metrics_values[index] = np.array(hvd.allreduce(metric_value))

        num_inputs = self._validation_input_manager.stop_validation_input_and_get_num_inputs()

        if not self.is_chief:
            return workload.Skipped()

        metrics = make_logs(self.model, {}, metrics_values, ModeKeys.TEST, prefix="val_")
        check.gt(len(metrics), 0)

        return {"num_inputs": num_inputs, "validation_metrics": metrics}


class TFKerasTrial(det.Trial):
    """
    To implement a new ``tf.keras`` trial, subclass this class and
    implement the abstract methods described below (:meth:`build_model`,
    :meth:`build_training_data_loader`, and :meth:`build_validation_data_loader`).
    In most cases you should provide a custom :meth:`__init__` method as well.

    By default, experiments use TensorFlow 1.x. To configure your trial to use
    TensorFlow 2.x, specify a TensorFlow 2.x image in the
    :ref:`environment.image <exp-environment-image>` field of the experiment
    configuration (e.g.,
    ``determinedai/environments:cuda-10.1-pytorch-1.4-tf-2.2-gpu-0.5.0``).

    Trials default to using eager execution with TensorFlow 2.x but not with
    TensorFlow 1.x. To override the default behavior, call the appropriate
    function in your ``__init__`` method. For example, if you want to disable
    eager execution while using TensorFlow 2.x, call
    ``tf.compat.v1.disable_eager_execution`` at the top of your ``__init__`` method.

    For more information on writing ``tf.keras`` trial classes, refer to the
    :ref:`tutorial <tf-mnist-tutorial>`.
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
            <https://www.tensorflow.org/versions/r1.14/api_docs/python/tf/data/Dataset>`__ returning
            a tuple of either ``(inputs, targets)`` or ``(inputs, targets, sample_weights)``.

            4) A `keras.utils.Sequence
            <https://tensorflow.org/api_docs/python/tf/keras/utils/Sequence>`__ returning a tuple
            of either ``(inputs, targets)`` or ``(inputs, targets, sample weights)``.

            5) A :class:`determined.keras.SequenceAdapter` returning a tuple of either
            ``(inputs, targets)`` or ``(inputs, targets, sample weights)``.

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
            <https://www.tensorflow.org/versions/r1.14/api_docs/python/tf/data/Dataset>`__ returning
            a tuple of either ``(inputs, targets)`` or ``(inputs, targets, sample_weights)``.

            4) A `keras.utils.Sequence
            <https://tensorflow.org/api_docs/python/tf/keras/utils/Sequence>`__ returning a tuple
            of either ``(inputs, targets)`` or ``(inputs, targets, sample weights)``.

            5) A :class:`determined.keras.SequenceAdapter` returning a tuple of either
            (inputs, targets) or (inputs, targets, sample weights).

        When using ``tf.data.Dataset``, you must wrap the dataset using
        :meth:`determined.keras.TFKerasTrialContext.wrap_dataset`. This wrapper is used
        to shard the dataset for distributed training. For optimal performance, users
        should wrap a dataset immediately after creating it.
        """
        pass

    def session_config(self) -> tf.compat.v1.ConfigProto:
        """
        Specifies the `tf.ConfigProto
        <https://www.tensorflow.org/api_docs/python/tf/ConfigProto>`__ to be
        used by the TensorFlow session. By default,
        ``tf.ConfigProto(allow_soft_placement=True)`` is used.
        """
        return tf.compat.v1.ConfigProto(allow_soft_placement=True)

    def keras_callbacks(self) -> List[tf.keras.callbacks.Callback]:
        """
        Specifies a list of `tf.keras.callback.Callback
        <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/Callback>`__
        objects to be used during the trial’s lifetime.

        Callbacks should avoid calling ``model.predict()``, as this will affect
        Determined training behavior.

        .. note::
            If you specify a Keras callback that uses the `on_epoch_begin
            <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/Callback#on_epoch_begin>`__
            or <`on_epoch_end
            <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/Callback#on_epoch_end>`__
            interfaces, epoch boundaries are determined by the length of the
            training data set, not by the value of the Determined configuration
            setting :ref:`records_per_epoch <config-records-per-epoch>`.
        """
        return []
