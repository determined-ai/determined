import functools
import logging
import numbers
import os
import pathlib
import random
import shutil
import tempfile
from abc import abstractmethod
from typing import Any, Callable, Dict, List, Optional, cast

import numpy as np
import tensorflow as tf
from packaging import version

import determined as det
from determined import estimator, horovod, ipc, tensorboard, workload
from determined.horovod import hvd
from determined_common import check

"""
The TF Estimator API has issues around multiple invocations:

- In order to take advantage of the TF Estimator API's built-in distributed
  training, we need to use the `train_and_evaluate` method of an Estimator;
  `train` doesn't support it.
- When serializing functions, `tf.data.Dataset.map` appends a unique identifier
  to function names. The identifier counter is maintained per process. The
  `input_fn` of an estimator is called per `train` call. This creates an
  inconsistency between function names and the checkpointed function names when
  `train` is called multiple times in the same process.

While it is possible to add even more monkey patching to make it work, it
ultimately seems more elegant to give the TensorFlow API what it wants: we
spawn a subprocess for executing TensorFlow operations, passing results back to
the parent via IPC. EstimatorTrialController contains the logic of the main
(outer) process, while EstimatorTrialController contains most of the logic for the
child process from which train_and_evaluate() is executed.

To avoid the very expensive initialization overhead that occurs every
train_and_evaluate() call, EstimatorTrial controls the train_and_evaluate()
call via a tf.train.SessionRunHook. Every time training iterates for a
sufficient amount of batches, the child subprocess blocks and waits for
instructions to either continue training, compute validation metrics via the
estimator.evaluate() method, or take a checkpoint.
"""


VERY_LARGE_NUMBER = 9999999999999999


class DeterminedControlHook(tf.estimator.SessionRunHook):  # type: ignore
    """
    DeterminedControlHook takes control of the train_and_evaluate() loop between
    training steps to communicate with the main harness process and, in certain
    cases, execute non-training workloads.

    At the beginning of the train_and_evaluate() call and after each training
    step ends, control_loop() is triggered and blocks on recieving instructions
    for the next workload. Once instructions are recieved from the main
    process, control_loop() will compute validation, take a checkpoint, or
    break out of the loop to re-enter train_and_evaluate().
    """

    def __init__(self, estimator_trial_controller: "EstimatorTrialController") -> None:
        self.batches_processed_in_step = 0
        self.estimator_trial_controller = estimator_trial_controller

        # step_metrics keeps track of the metrics associated with a step (see
        # DeterminedControlCallback). It is cleared in between training steps.
        self.step_metrics = []  # type: List[Dict[str, Any]]
        self.batches_per_step = None  # type: Optional[int]

        self._global_step_of_last_checkpoint = None  # type: Optional[int]
        self._session = None  # type: Optional[tf.Session]
        self._current_global_step = None  # type: Optional[int]
        self._saver = None  # type: Optional[tf.train.Saver]
        self._writer = tf.compat.v1.summary.FileWriter(tensorboard.get_base_path({}))

        # Store the response_func for train_for_step workloads while we do the training.
        self.train_response_func = None  # type: Optional[workload.ResponseFunc]

    def begin(self) -> None:
        # For performance reasons, we collect per batch metrics
        # only for certain types of summaries. Other summary types,
        # are collected with a frequency of `save_summary_steps` set
        # in the RunConfig.
        summary_types_collected_every_batch = {"ScalarSummary", "TensorSummary"}
        per_batch_summaries = []
        for summary in tf.compat.v1.get_collection(tf.compat.v1.GraphKeys.SUMMARIES):
            if summary.op.type in summary_types_collected_every_batch:
                per_batch_summaries.append(summary)
                logging.debug(f"Collecting {summary} of type {summary.op.type} every batch.")
            else:
                logging.debug(f"Not collecting {summary} of type {summary.op.type} every batch.")
        self._summary_op = tf.compat.v1.summary.merge(per_batch_summaries)
        self._global_step_tensor = tf.compat.v1.train.get_global_step()

        # train_and_evaluate() is invoked before the trial controller receives
        # any workload instructions. Therefore, we need to immediately enter
        # the control loop and wait for the next workload.
        self.control_loop()

    def before_run(
        self, run_context: tf.estimator.SessionRunContext
    ) -> tf.estimator.SessionRunArgs:
        return tf.estimator.SessionRunArgs(
            {"summary": self._summary_op, "global_step": self._global_step_tensor}
        )

    def _collect_batch_metrics(self, run_values: tf.estimator.SessionRunValues) -> None:
        if "summary" not in run_values.results:
            raise AssertionError("Expected 'summary' to be run_values, but it was not.")
        summary = tf.compat.v1.summary.Summary()
        summary.ParseFromString(run_values.results["summary"])
        batch_metrics = {}  # type: Dict[str, Any]
        for val in summary.value:
            if val.HasField("simple_value"):
                batch_metrics[val.tag] = val.simple_value
            elif val.HasField("tensor"):
                batch_metrics[val.tag] = tf.make_ndarray(val.tensor)

        self.step_metrics.append(batch_metrics)

    def after_run(
        self, run_context: tf.estimator.SessionRunContext, run_values: tf.estimator.SessionRunValues
    ) -> None:
        # Check for optimizer creation here because when model_fn is passed in as a closure,
        # the optimizer is not initialized until the first training step.
        check.true(
            self.estimator_trial_controller.context.optimizer_initialized,
            "Please pass your optimizer into "
            "`det.estimator.wrap_optimizer(optimizer)` "
            "right after creating it.",
        )
        self._session = run_context.session
        self._current_global_step = run_values.results["global_step"]

        self.batches_per_step = cast(int, self.batches_per_step)
        self._collect_batch_metrics(run_values)
        self.batches_processed_in_step += 1
        if self.batches_processed_in_step < self.batches_per_step:
            return

        # TODO: Average training results across GPUs. This might
        # degrade performance due to an increase in communication.

        # Loss training metric is sometimes called `loss_1` instead of `loss`.
        for step_metrics in self.step_metrics:
            if "loss" not in step_metrics and "loss_1" in step_metrics:
                step_metrics["loss"] = step_metrics["loss_1"]

        # Send the result of the training step back to the main process.
        check.is_not_none(self.train_response_func, "no response_func at end of train_for_step")
        self.train_response_func = cast(workload.ResponseFunc, self.train_response_func)
        if self.estimator_trial_controller.is_chief:
            self.train_response_func(
                det.util.make_metrics(self.batches_processed_in_step, self.step_metrics)
            )
        else:
            self.train_response_func(workload.Skipped())

        # Reset step counter and clear the step metrics from memory.
        self.train_response_func = None
        self.batches_processed_in_step = 0
        self.step_metrics = []

        estimator._cleanup_after_train_step(self.estimator_trial_controller.estimator_dir)

        # Re-enter the control loop (block on receiving the next instruction)
        self.control_loop()

    # The following three functions are adapted from the implementation of
    # tf.train.CheckpointSaverHook.
    def after_create_session(
        self, session: tf.compat.v1.Session, coord: tf.train.Coordinator
    ) -> None:
        graph = tf.compat.v1.get_default_graph().as_graph_def(add_shapes=True)
        tf.io.write_graph(graph, str(self.estimator_trial_controller.estimator_dir), "graph.pbtxt")

        # Apart from writing the graph as a pbtxt, write the graph to a
        # tfevents file to visualize in tensorboard.
        self._writer.add_graph(graph)
        self._writer.close()
        self._writer.reopen()

    def _get_saver(self) -> tf.compat.v1.train.Saver:
        if self._saver is not None:
            return self._saver

        collection_key = tf.compat.v1.GraphKeys.SAVERS
        savers = tf.compat.v1.get_collection(collection_key)
        if not savers:
            raise RuntimeError(
                f"No items in saver collection {collection_key}. Please add a saver to "
                "the collection."
            )
        elif len(savers) > 1:
            raise RuntimeError(
                f"More than one item in savers collection {collection_key}: {savers}."
            )

        self._saver = savers[0]
        return savers[0]

    def _checkpoint_model(self, checkpoint_path: pathlib.Path) -> workload.Response:
        self._save_model()

        if not self.estimator_trial_controller.is_chief:
            return workload.Skipped()

        self._copy_latest_checkpoint(checkpoint_path=checkpoint_path)
        self._save_serving_input_receiver_fns(checkpoint_path=str(checkpoint_path))

        det.util.write_checkpoint_metadata(
            checkpoint_path,
            self.estimator_trial_controller.env,
            {"tensorflow_version": tf.__version__, "format": "saved_model"},
        )

        return {}

    def _save_model(self) -> None:
        # Only save when we have performed training since the last
        # time we saved.
        started_training = self._current_global_step is not None
        checkpoint_exists = self._global_step_of_last_checkpoint is not None
        if not started_training or (
            checkpoint_exists and self._global_step_of_last_checkpoint == self._current_global_step
        ):
            return

        logging.info(
            f"Saving checkpoints for step: {self._current_global_step} "
            f"into {self.estimator_trial_controller.estimator_dir}."
        )

        check.is_not_none(self._session)
        check.is_not_none(self._current_global_step)
        self._current_global_step = cast(int, self._current_global_step)

        self._get_saver().save(
            self._session,
            str(self.estimator_trial_controller.estimator_dir.joinpath("model.ckpt")),
            global_step=self._current_global_step,
        )
        self._global_step_of_last_checkpoint = self._current_global_step

    def _copy_latest_checkpoint(self, checkpoint_path: pathlib.Path) -> None:
        checkpoint_dir = os.path.dirname(
            self.estimator_trial_controller.estimator.latest_checkpoint()
        )
        shutil.copytree(checkpoint_dir, str(checkpoint_path))

        # Calibrate the CheckpointState metadata file to the new location.
        estimator._update_checkpoint_path_in_state_file(checkpoint_path)

    def _save_serving_input_receiver_fns(self, checkpoint_path: str) -> None:
        for name, fn in self.estimator_trial_controller.serving_input_receiver_fns.items():
            logging.info(
                f"Found a serving input receiver function '{name}', exporting as a SavedModel."
            )
            self.estimator_trial_controller.estimator.export_saved_model(
                os.path.join(checkpoint_path, name), fn
            )

    def _compute_validation_metrics(self) -> workload.Response:
        # Estimator uses the latest checkpoint to perform evaluation so
        # we need to checkpoint to model directory on every worker
        # before performing computing validation metrics.
        self._save_model()
        return self.estimator_trial_controller.compute_validation_metrics()

    def control_loop(self) -> None:
        for wkld, args, response_func in self.estimator_trial_controller.workloads:
            logging.debug(f"Received wkld {wkld.kind} with args {args}.")

            if wkld.kind == workload.Workload.Kind.RUN_STEP:
                check.len_eq(args, 1)
                check.is_instance(args[0], int)
                # Store values for the training loop.
                self.batches_per_step = cast(int, args[0])
                self.train_response_func = response_func
                # Break out of the control loop so that the train process
                # re-enters the train_and_evaluate() loop.
                break
            elif wkld.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                response_func(self._compute_validation_metrics())
            elif wkld.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.len_eq(args, 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                response_func(self._checkpoint_model(path))
            elif wkld.kind == workload.Workload.Kind.TERMINATE:
                response_func(
                    {} if self.estimator_trial_controller.is_chief else workload.Skipped()
                )
                raise det.errors.WorkerFinishedGracefully("Exiting normally.")
            else:
                raise AssertionError(f"Unknown wkld kind {wkld.kind}.")
                exit(1)


class EstimatorTrialController(det.LoopTrialController):
    def __init__(
        self,
        estimator: tf.estimator.Estimator,
        user_train_spec: tf.estimator.TrainSpec,
        val_spec: tf.estimator.EvalSpec,
        serving_input_receiver_fns: Dict[str, estimator.ServingInputReceiverFn],
        *args: Any,
        **kwargs: Any,
    ) -> None:
        super().__init__(*args, **kwargs)

        self.estimator = estimator
        self.user_train_spec = user_train_spec
        self.val_spec = val_spec
        self.serving_input_receiver_fns = serving_input_receiver_fns

        self._init_model()

    @staticmethod
    def pre_execute_hook(env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        # Initialize the correct horovod.
        if hvd_config.use:
            hvd.require_horovod_type("tensorflow", "EstimatorTrial is in use.")
            hvd.init()

        # Initialize random seeds.
        if env.experiment_config.input_from_dataflow():
            logging.debug("Using tensorpack dataflows as input.")
            process_rank = 0 if not hvd_config.use else hvd.rank()
            EstimatorTrialController.set_random_seed(env.trial_seed + process_rank)
        else:
            # Set identical random seeds on all training processes.
            # When using horovod, each worker will receive a unique
            # shard of the dataset.
            EstimatorTrialController.set_random_seed(env.trial_seed)

        if version.parse(tf.__version__) >= version.parse("2.0.0"):
            tf.compat.v1.disable_v2_behavior()

    @staticmethod
    def set_random_seed(seed: int) -> None:
        random.seed(seed)
        np.random.seed(seed)
        # This seed value will be overwritten by
        # tf.estimator.RunConfig.tf_random_seed.
        tf.compat.v1.set_random_seed(seed)

    @staticmethod
    def from_trial(
        trial_inst: det.Trial,
        context: det.TrialContext,
        env: det.EnvContext,
        *args: Any,
        **kwargs: Any,
    ) -> det.TrialController:
        check.is_instance(
            context,
            estimator.EstimatorTrialContext,
            "EstimatorTrialController needs an EstimatorTrialContext",
        )

        check.is_instance(
            trial_inst, EstimatorTrial, "EstimatorTrialController needs an EstimatorTrial"
        )
        trial_inst = cast(EstimatorTrial, trial_inst)

        return EstimatorTrialController(
            trial_inst.build_estimator(),
            trial_inst.build_train_spec(),
            trial_inst.build_validation_spec(),
            trial_inst.build_serving_input_receiver_fns(),
            context,
            env,
            *args,
            **kwargs,
        )

    @staticmethod
    def from_native(context: det.NativeContext, *args: Any, **kwargs: Any) -> det.TrialController:
        check.is_instance(
            context,
            estimator.EstimatorNativeContext,
            "EstimatorTrialController needs an EstimatorSprinkleContext",
        )
        context = cast(estimator.EstimatorNativeContext, context)

        check.true(
            hasattr(context, "estimator")
            and hasattr(context, "train_spec")
            and hasattr(context, "eval_spec"),
            "Please call TFEstimatorExperiment.train_and_evaluate().",
        )

        return EstimatorTrialController(
            context.estimator,
            context.train_spec,
            context.eval_spec,
            context.serving_input_receiver_fns,
            context,
            *args,
            **kwargs,
        )

    @staticmethod
    def supports_multi_gpu_training() -> bool:
        return True

    @staticmethod
    def support_determined_native() -> bool:
        return True

    def _check_and_repeat_train_input_fn(self, f: Callable) -> Callable:
        """
        Modifies functions that returns a `tf.data.Dataset` to repeat. This is done
        so that we never run out of training data.
        """

        @functools.wraps(f)
        def wrapper(*args: Any, **kwargs: Any) -> tf.data.Dataset:
            ds = f(*args, **kwargs)

            if self.context.experimental.get_train_cacheable().is_decorator_used():
                check.false(
                    self.context.dataset_initialized,
                    "Please do not use: `context.wrap_dataset(dataset)` if using "
                    "`@context.experimental.cache_train_dataset(dataset_name, dataset_version)` "
                    "and `@context.experimental.cache_validation_dataset(dataset_name, "
                    "dataset_version)`.",
                )
            else:
                check.true(
                    self.context.dataset_initialized,
                    "Please pass your datasets (train and test) into "
                    "`context.wrap_dataset(dataset)` right after creating them.",
                )

            if isinstance(ds, tf.data.Dataset):
                ds = ds.repeat()

            return ds

        return wrapper

    def _init_model(self) -> None:
        self._init_paths()

        self.estimator = tf.estimator.Estimator(
            model_fn=self.estimator._model_fn,
            config=self._init_run_config(self.estimator.config),
            params=self.estimator.params if self.estimator.params != {} else None,
            warm_start_from=self.estimator._warm_start_settings,
        )

        check.is_instance(
            self.estimator,
            tf.estimator.Estimator,
            "Please modify your model definition's build_estimator() implementation to return "
            "an instance of `tf.estimator.Estimator`.",
        )
        check.is_instance(
            self.user_train_spec,
            tf.estimator.TrainSpec,
            "Please modify your model definition's build_train_spec() implementation to return "
            "an instance of `tf.estimator.TrainSpec`.",
        )
        check.is_instance(
            self.val_spec,
            tf.estimator.EvalSpec,
            "Please modify your model definition's build_validation_spec() implementation "
            "to return an instance of `tf.estimator.EvalSpec`.",
        )

        all_hooks = [*self.user_train_spec.hooks]

        if self.hvd_config.use:
            all_hooks.append(hvd.BroadcastGlobalVariablesHook(0))

        # It is important that this hook is the final in the list so that if
        # any other hooks need to run _before_ the training step ends they have
        # their chance.
        all_hooks.append(DeterminedControlHook(self))

        # TODO(DET-834): Separate step ID from data loader state.
        #
        # During warm start, we initialize model weights, optimizer state
        # and input state from the checkpoint, and we set the step ID to
        # 1. Trials typically use the step ID as an index into the data
        # sequence, which means there is an inconsistency between the
        # step ID (as data index) and the optimizer state and input state.
        #
        # In the short term, behave like other trials and reset input
        # state if we are warm started. This will create an inconsistency
        # wrt saved optimizer state.

        # Repeat training dataset so we never run out of data.
        repeating_train_fn = self._check_and_repeat_train_input_fn(self.user_train_spec.input_fn)

        self.train_spec = tf.estimator.TrainSpec(input_fn=repeating_train_fn, hooks=all_hooks)
        self.eval_spec = tf.estimator.EvalSpec(input_fn=self.val_spec.input_fn, steps=None)

    def _init_run_config(self, config: tf.estimator.RunConfig) -> tf.estimator.RunConfig:
        logging.debug(f"Initializing RunConfig. Got RunConfig: {config} .")

        session_config = config.session_config
        train_distribute = None
        eval_distribute = None
        if self.hvd_config.use:
            if session_config is None:
                session_config = tf.compat.v1.ConfigProto()
            session_config.gpu_options.allow_growth = True
            session_config.gpu_options.visible_device_list = str(
                self.env.slot_ids[horovod.hvd.local_rank()]
            )
        elif len(self.env.container_gpus) > 1:
            check.true(len(self.rendezvous_info.get_addrs()) == 1)
            train_distribute = tf.distribute.MirroredStrategy()
            eval_distribute = tf.distribute.MirroredStrategy()

        config = config.replace(
            model_dir=str(self.estimator_dir),
            tf_random_seed=self.env.trial_seed,
            save_checkpoints_steps=None,
            # `train_and_evaluate()` requires that either
            # `save_checkpoints_steps` or `save_checkpoints_secs` is
            # set to greater than 0.
            save_checkpoints_secs=VERY_LARGE_NUMBER,
            session_config=session_config,
            train_distribute=train_distribute,
            eval_distribute=eval_distribute,
            experimental_distribute=None,
        )
        logging.debug(f"Initialized RunConfig with args: {config}.")
        return config

    def run(self) -> None:
        try:
            tf.estimator.train_and_evaluate(self.estimator, self.train_spec, self.eval_spec)
        except det.errors.WorkerFinishedGracefully:
            pass
        else:
            raise AssertionError(
                "Training loop exited unexpectedly but without throwing any errors. This is "
                "possibly due to either setting train_spec.max_steps to a non-None value or due to "
                "a user callback causing the training loop to exit, which is not supported at this "
                "time."
            )

    def _init_paths(self) -> None:
        """
        Create a unique model directory for each training process. If
        a load path is provided, copy the checkpoint into the model
        directory of each training process. This model directory will
        be used to initialize an Estimator. We also update the paths in
        the CheckpointState metadata file to the new directory location.
        """
        # Add suffix so that horovod processes don't overwrite each other.
        suffix = str(0) if not self.hvd_config.use else str(hvd.local_rank())
        if self.load_path is None:
            self.estimator_dir = pathlib.Path(tempfile.mkdtemp(suffix=suffix))
            logging.debug(f"Estimator directory set to {self.estimator_dir}.")
            return

        self.estimator_dir = pathlib.Path(tempfile.mkdtemp(suffix=suffix))
        if self.estimator_dir.exists():
            shutil.rmtree(str(self.estimator_dir))
        logging.debug(f"Copying from {self.load_path} to {self.estimator_dir}.")
        shutil.copytree(str(self.load_path), str(self.estimator_dir))

        # Calibrate the CheckpointState metadata file to the new location.
        estimator._update_checkpoint_path_in_state_file(self.estimator_dir)
        logging.debug(f"Load path set to {self.estimator_dir}.")

    def compute_validation_metrics(self) -> workload.Response:
        metrics = self.estimator.evaluate(
            input_fn=self.val_spec.input_fn, steps=self.val_spec.steps, hooks=self.val_spec.hooks
        )

        if self.hvd_config.use:
            metrics = self.average_metrics(metrics)
            if self.is_chief:
                logging.debug(f"Averaged validation metrics: {metrics}.")

        estimator._cleanup_after_validation_step(
            self.estimator._model_dir, self.hvd_config.use, self.is_chief
        )

        if not self.is_chief:
            return workload.Skipped()

        return {"validation_metrics": metrics}

    def average_metrics(self, metrics: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        # The chief receives the metric from every worker and computes
        # the average.
        check.true(self.hvd_config.use)
        if self.is_chief:
            self.train_process_comm_chief = cast(ipc.ZMQServer, self.train_process_comm_chief)
            logging.debug(f"Chief {hvd.rank()} beginning receiving validation metrics.")
            worker_metrics = self.train_process_comm_chief.barrier(num_connections=hvd.size() - 1)
            logging.debug(f"Chief {hvd.rank()} done receiving validation metrics.")
            for metric_name in metrics:
                if isinstance(metrics[metric_name], numbers.Number):
                    metrics[metric_name] /= hvd.size()
                else:
                    logging.warning(f"Skipping averaging metric: {metric_name}.")
            for metric_name in metrics.keys():
                for worker_metric in worker_metrics:
                    if isinstance(worker_metric[metric_name], numbers.Number):
                        metrics[metric_name] += worker_metric[metric_name] / hvd.size()
            return metrics
        else:
            self.train_process_comm_worker = cast(ipc.ZMQClient, self.train_process_comm_worker)
            logging.debug(f"Worker {hvd.rank()} sending metrics.")
            self.train_process_comm_worker.barrier(message=metrics)
            return None


class EstimatorTrial(det.Trial):
    """
    By default, experiments run with TensorFlow 1.x. To configure your trial to
    use TensorFlow 2.x, set a TF 2.x image in the experiment configuration
    (e.g. ``determinedai/environments:cuda-10.1-pytorch-1.4-tf-2.1-gpu-0.3.0``).

    ``EstimatorTrial`` supports TF 2.x; however it uses TensorFlow V1
    behavior. We have disabled TensorFlow V2 behavior for ``EstimatorTrial``,
    so there is no need for you to disable it.
    """

    trial_controller_class = EstimatorTrialController
    trial_context_class = estimator.EstimatorTrialContext

    def __init__(self, context: estimator.EstimatorTrialContext):
        """
        Initializes a trial using the provided trial_context.

        Override this function to initialize any shared state between the
        estimator, train spec, and/or validation spec.
        """
        self.context = context  # type: estimator.EstimatorTrialContext

    @abstractmethod
    def build_estimator(self) -> tf.estimator.Estimator:
        """
        Specifies the `tf.estimator.Estimator
        <https://www.tensorflow.org/api_docs/python/tf/estimator/Estimator>`__
        instance to be used during training and validation. This may be an
        instance of a `Premade Estimator
        <https://www.tensorflow.org/guide/premade_estimators>`__ provided by
        the TensorFlow team, or a `Custom Estimator
        <https://www.tensorflow.org/guide/custom_estimators>`__ created by the
        user.
        """
        pass

    @abstractmethod
    def build_train_spec(self) -> tf.estimator.TrainSpec:
        """
        Specifies the `tf.estimator.TrainSpec
        <https://www.tensorflow.org/api_docs/python/tf/estimator/TrainSpec>`__
        to be used for training steps. This training specification will contain
        a TensorFlow ``input_fn`` which constructs the input data for a
        training step. Unlike the standard Tensorflow ``input_fn`` interface,
        ``EstimatorTrial`` only supports an ``input_fn`` that returns a
        ``tf.data.Dataset`` object. A function that returns a tuple of features
        and labels is currently not supported by ``EstimatorTrial``.
        Additionally, the ``max_steps`` attribute of the training specification
        will be ignored; instead, the ``batches_per_step`` option in the
        experiment configuration is used to determine how many batches each
        training step uses.
        """
        pass

    @abstractmethod
    def build_validation_spec(self) -> tf.estimator.EvalSpec:
        """
        Specifies the `tf.estimator.EvalSpec
        <https://www.tensorflow.org/api_docs/python/tf/estimator/EvalSpec>`__
        to be used for validation steps. This evaluation spec will contain a
        TensorFlow ``input_fn`` which constructs the input data for a
        validation step. The validation step will evaluate ``steps`` batches,
        or evaluate until the ``input_fn`` raises an end-of-input exception if
        ``steps`` is ``None``.
        """
        pass

    def build_serving_input_receiver_fns(self) -> Dict[str, estimator.ServingInputReceiverFn]:
        """
        Optionally returns a Python dictionary mapping string names to
        `serving_input_receiver_fn
        <https://www.tensorflow.org/guide/saved_model#prepare_serving_inputs>`__\
                s.
        If specified, each serving input receiver function will be used to
        export a distinct `SavedModel
        <https://www.tensorflow.org/guide/saved_model>`__ inference graph when
        a Determined checkpoint is saved, using `Estimator.export_saved_model
        <https://www.tensorflow.org/api_docs/python/tf/estimator/Estimator#export_saved_model>`__.
        The exported models are saved under subdirectories named by the keys of
        the respective serving input receiver functions. For example, returning

        .. code-block:: python

           {
               "raw": tf.estimator.export.build_raw_serving_input_receiver_fn(...),
               "parsing": tf.estimator.export.build_parsing_serving_input_receiver_fn(...)
           }

        from this function would configure Determined to export two ``SavedModel``
        inference graphs in every checkpoint under ``raw`` and ``parsing``
        subdirectories, respectively. By default, this function returns an empty
        dictionary and the Determined checkpoint directory only contains metadata
        associated with the training graph.
        """
        return {}
