"""
Tensorpack has a similar issue to TF Estimators in that it wants to control your training loop:
you give it your model and it runs a function that does everything for you and can't be called more
than once. This trial class addresses the issue by creating a long-running subordinate process that
runs the training function. The main process then does very little besides tell the subordinate
process which workloads to run and return the results back to the harness.

In the subordinate process, we give Tensorpack a callback of type `ManagerCallback`, which allows us
to control what Tensorpack is doing even while inside the single long call to the training function.
At the end of each training step, the callback blocks until it receives a new workload from the main
process and then reacts accordingly.
"""

import itertools
import logging
import pathlib
import time
from abc import abstractmethod
from typing import Any, Dict, Iterator, List, Optional, Sequence, Tuple, TypeVar, Union, cast

import tensorflow as tf
import tensorpack as tp
from tensorpack.callbacks.base import Callback
from tensorpack.tfutils.common import get_global_step_var
from tensorpack.train.tower import TowerTrainer

import determined as det
from determined import horovod, workload
from determined.horovod import hvd
from determined_common import check

IMPOSSIBLY_LARGE_EPOCHS = 9999999999999


T = TypeVar("T")


def pairs(iterable: Union[Iterator[T], Sequence[T]]) -> Iterator[Tuple[T, T]]:
    """s -> (s0,s1), (s1,s2), (s2, s3), ..."""
    a, b = itertools.tee(iterable)
    next(b, None)
    return zip(a, b)


def is_determined_ai_tensorpack() -> bool:
    return hasattr(tp, "__is_determined_ai__")


class Evaluator(Callback):  # type: ignore
    """
    A class that defines how to compute validation metrics for a model. This can run arbitrary
    Python code for each validation.
    """

    _chief_only = False

    @abstractmethod
    def set_up_graph(self, trainer: tp.Trainer) -> None:
        pass

    @abstractmethod
    def compute_validation_metrics(self) -> Dict[str, Any]:
        pass


class NamedTensorsEvaluator(Evaluator):
    """
    A class that specifies that the values of tensors with certain names should be returned as
    validation metrics.
    """

    def __init__(self, names: List[str]):
        self.names = names

    def set_up_graph(self, trainer: tp.Trainer) -> None:
        raise NotImplementedError()

    def compute_validation_metrics(self) -> Dict[str, Any]:
        raise NotImplementedError()


class ManagerCallback(tp.callbacks.Callback):  # type: ignore
    """
    ManagerCallback contains the logic for running workloads in the subordinate process.

    `_trigger_epoch` runs at the end of each Tensorpack epoch (corresponding to a Determined step).
    It sends the batch metrics back to the main process and blocks until the main process sends
    information about the next workload to run.

    `_before_run` and `_after_run` extract the metrics from the graph.
    """

    _chief_only = False

    def __init__(
        self,
        metric_names: List[str],
        evaluator: Optional[Evaluator],
        validation_metrics_names: Optional[List[str]],
        workloads: workload.Stream,
        is_chief: bool,
        machine_rank: int,
        context: Any,
    ) -> None:
        self.metric_names = metric_names
        self.batch_metrics = []  # type: List[Dict[str, Any]]
        self.evaluator = evaluator
        self.validation_metrics_names = validation_metrics_names
        self.workloads = workloads
        self.is_chief = is_chief
        self.machine_rank = machine_rank
        self.context = context

        # Store the response_func for train_for_step workloads while we do the training.
        self.train_response_func = None  # type: Optional[workload.ResponseFunc]

    def get_tensor(self, name: str) -> Any:
        """
        Attempt to turn a name into a sensible value from the graph.
        """
        # Look for the output of an operation with the given name.
        g = tf.get_default_graph()
        try:
            x = g.get_operation_by_name(name)
            if len(x.outputs) != 1:
                raise ValueError(f"Operation {name} does not have exactly one output")
            return x.outputs[0]
        except (KeyError, ValueError):
            pass

        # Look for a tensor with the name.
        try:
            return g.get_tensor_by_name(name)
        except (KeyError, ValueError):
            pass

        # Look for an existing variable with the name.
        with tf.variable_scope("", reuse=True):
            try:
                return tf.get_variable(name)
            except (KeyError, ValueError):
                pass

        # Look in the first tower for the tensor.
        if isinstance(self.trainer, TowerTrainer):
            try:
                return self.trainer.towers.training()[0][name]
            except KeyError:
                pass

        raise ValueError("Tensor not found: {}".format(name))

    # _setup_graph, _before_run, and _after_run gather batch metrics from each
    # run (adapted from tp.callbacks.ProcessTensors).
    def _setup_graph(self) -> None:
        if self.evaluator:
            self.evaluator.set_up_graph(self.trainer)

        # Fetch the requested metrics, along with the global step for debugging.
        fetches = (
            {n: self.get_tensor(n) for n in self.metric_names},
            tf.train.get_or_create_global_step(),
        )
        self._fetch = tf.train.SessionRunArgs(fetches=fetches)

        # Set up model saving logic (taken from tp.callbacks.ModelSaver).
        self.saver = tf.train.Saver(
            max_to_keep=None, write_version=tf.train.SaverDef.V2, save_relative_paths=True
        )
        tf.add_to_collection(tf.GraphKeys.SAVERS, self.saver)

        with tf.name_scope(None):
            self.gs_val = tf.placeholder(tf.int64, shape=())
            self.gs_set_op = tf.assign(
                get_global_step_var(), self.gs_val, name="DET_SET_GLOBAL_STEP"
            ).op

    def _before_run(self, _: Any) -> tf.train.SessionRunArgs:
        self.before_time = time.time()
        return self._fetch

    def _after_run(self, _: Any, rv: tf.train.SessionRunValues) -> None:
        metrics, _ = rv.results
        dt = time.time() - self.before_time
        logging.debug(
            f"after_run machine_rank={self.machine_rank}, "
            f"local step={self.trainer.local_step,}, gs={self.trainer.global_step}, dt={dt:.6f}"
        )
        self.trainer.loop._global_step += 1
        # The results are already in dict form, since we constructed the SessionRunArgs as one.
        self.batch_metrics.append(metrics)

    def _before_train(self) -> Any:
        self._control_loop()

    def _compute_validation_metrics(self) -> Any:
        """
        Computes validation metrics using either Evaluator() or CustomInferenceRunner().
        """
        if self.evaluator:
            check.is_none(self.validation_metrics_names)
            metrics = self.evaluator.compute_validation_metrics()
        else:
            check.is_not_none(self.validation_metrics_names)
            # Find our custom Inference callback.
            custom_inference_callback = None  # type: Optional[CustomInferenceRunner]
            for callback in self.trainer._callbacks.cbs:
                if isinstance(callback, CustomInferenceRunner):
                    custom_inference_callback = callback
                    break
            custom_inference_callback = cast(CustomInferenceRunner, custom_inference_callback)
            self.validation_metrics_names = cast(List[str], self.validation_metrics_names)
            metrics = custom_inference_callback.trigger_on_validation_step(
                self.validation_metrics_names
            )

        if not self.is_chief:
            return workload.Skipped()

        return {"validation_metrics": metrics}

    def _trigger_epoch(self) -> None:
        """
        This runs at the end of each training step, sends the metrics back to the main process, and
        decides what to do next.
        """

        check.is_not_none(self.train_response_func, "no response_func at end of train_for_step")
        self.train_response_func = cast(workload.ResponseFunc, self.train_response_func)

        if self.is_chief:
            response = {
                "metrics": det.util.make_metrics(None, self.batch_metrics),
                "stop_requested": self.context.get_stop_requested(),
            }
            self.train_response_func(response)
        else:
            self.train_response_func(workload.Skipped())

        self.train_response_func = None
        self.batch_metrics = []

        self._control_loop()

    def _control_loop(self) -> None:
        for wkld, args, response_func in self.workloads:
            if wkld.kind == workload.Workload.Kind.RUN_STEP:
                # Move on to the next step.
                self.train_response_func = response_func
                break
            elif wkld.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                response_func(
                    det.util.wrap_metrics(
                        self._compute_validation_metrics(), self.context.get_stop_requested()
                    )
                )
            elif wkld.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                check.len_eq(args, 1)
                check.is_instance(args[0], pathlib.Path)
                path = cast(pathlib.Path, args[0])
                response_func(self.save_checkpoint(path))
            elif wkld.kind == workload.Workload.Kind.TERMINATE:
                response_func({} if self.is_chief else workload.Skipped())
                raise det.errors.WorkerFinishedGracefully("Exiting normally.")
            else:
                raise AssertionError(f"Unknown wkld kind {wkld.kind}")

    def save_checkpoint(self, path: pathlib.Path) -> workload.Response:
        if not self.is_chief:
            return workload.Skipped()

        # save() interprets its argument as a string prefix rather than a directory name; this tells
        # it to save everything into files inside the given directory with filenames starting with
        # "model".
        prefix = path.joinpath("model")
        self.trainer.sess.run(self.gs_set_op, feed_dict={self.gs_val: self.trainer.global_step})
        self.saver.save(self.trainer.sess, str(prefix), global_step=self.trainer.global_step)

        return {}


class CustomInferenceRunner(tp.InferenceRunner):  # type: ignore
    """
    CustomInferenceRunner is the callback that is used when users
    specify validation metrics rather than using Evaluator().
    Unlike `tp.InferenceRunner` this callback will run on every
    RUN_VALIDATION. `tp.InferenceRunner` would run after every RUN_STEP.
    """

    _chief_only = False

    def __init__(self, machine_rank: int, *args: Any, **kwargs: Any) -> None:
        self._machine_rank = machine_rank
        super().__init__(*args, **kwargs)

    def _trigger(self) -> None:
        """
        Overwrites the `tp.InferenceRunner._trigger()` for `trigger_on_validation_step()`.
        """
        pass

    def trigger_on_validation_step(self, validation_metrics_names: List[str]) -> Any:
        """
        Called by `ManagerCallback` for each RUN_VALIDATION step.
        """

        if self._machine_rank != 0:
            return {}

        # When compute validation is called as first step, need to make sure
        # that `_before_train` of this callback has already been called.
        if not hasattr(self, "_hooked_sess"):
            super()._before_train()

        super()._trigger()

        validation_metrics = {}
        for validation_metrics_name in validation_metrics_names:
            validation_metrics[validation_metrics_name] = self.trainer.monitors.get_latest(
                validation_metrics_name
            )
        return validation_metrics


class CustomScalarStats(tp.ScalarStats):  # type: ignore
    """
    CustomScalarStats is the callback used for monitoring stats when
    users specify validation metrics rather than using Evaluator().
    """

    def names_with_prefix(self) -> List[str]:
        names = []
        for name in self.names:
            names.append(f"{self.prefix}_{name}")
        return names


class TensorpackTrialController(det.LoopTrialController):
    """
    The subordinate process that actually runs Tensorpack training. After doing some setup, it
    spends the rest of its life in a call to Tensorpack's `train_with_defaults` function.
    """

    def __init__(self, trial_inst: det.Trial, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)

        check.is_instance(
            trial_inst, TensorpackTrial, "TensorpackTrialController needs a TensorpackTrial"
        )
        self.trial = cast(TensorpackTrial, trial_inst)

        training_dataflow = self.trial.build_training_dataflow()
        validation_dataflow = self.trial.build_validation_dataflow()

        # Set if model is initialized from scratch.
        self.session_init = None  # type: Optional[Any]

        self._init_model(training_dataflow, validation_dataflow)

    @staticmethod
    def pre_execute_hook(env: det.EnvContext, hvd_config: horovod.HorovodContext) -> None:
        # Initialize the correct horovod.
        if hvd_config.use:
            hvd.require_horovod_type("tensorflow", "TensorpackTrial is in use.")
            hvd.init()

        TensorpackTrialController._set_random_seeds()

    @staticmethod
    def _set_random_seeds() -> None:
        # For distributed training each machine needs to use a unique
        # random seed so that the dataset is processed in a unique order.
        # TODO(DET-1124): Re-enable random seeds.
        # rank_random_seed = seed + self.rendezvous_info.get_rank()
        # random.seed(rank_random_seed)
        # np.random.seed(rank_random_seed)
        # tf.set_random_seed(rank_random_seed)
        # tp.utils.fix_rng_seed(rank_random_seed)
        pass

    @staticmethod
    def from_trial(*args: Any, **kwargs: Any) -> det.TrialController:
        return TensorpackTrialController(*args, **kwargs)

    @staticmethod
    def from_native(*args: Any, **kwargs: Any) -> det.TrialController:
        raise NotImplementedError("TensorpackTrialController does not support the Native API.")

    @staticmethod
    def supports_multi_gpu_training() -> bool:
        return True

    def _init_model(self, training_dataflow: Any, validation_dataflow: Any) -> None:
        self._load()

        logging.info("Calling build_model")
        if self.hvd_config.use:
            trainer_type = "horovod"
        else:
            trainer_type = "replicated"
        model = self.trial.build_model(trainer_type)
        logging.info("Finished build_model")

        determined_ai_tensorpack = is_determined_ai_tensorpack()

        if not determined_ai_tensorpack and self.hvd_config.aggregation_frequency > 1:
            raise AssertionError(
                "Gradient aggregation is only supported for custom DAI version of tensorpack"
            )

        if self.hvd_config.use:
            self.trainer = tp.HorovodTrainer(
                average=False,
                compression=hvd.compression.Compression.fp16,
                aggregation_frequency=self.hvd_config.aggregation_frequency,
            )
        else:
            num_gpus = len(self.env.container_gpus)
            self.trainer = tp.SyncMultiGPUTrainerReplicated(num_gpus, average=False, mode="nccl")

        inp = tp.QueueInput(training_dataflow)

        # StagingInput causes deadlocks in some code, so allow it to be disabled.
        # TODO: Figure out why.
        if not self.env.hparams.get("disable_staging_area"):
            inp = tp.StagingInput(inp, 1)

        logging.info("Calling setup_graph")
        self.trainer.setup_graph(
            model.get_input_signature(), inp, model.build_graph, model.get_optimizer
        )
        logging.info("Finished setup_graph")

        # For validation we support users specifying an Evaluator(), or passing in
        # the validation metrics they want to track. If they pass in validation
        # metrics, we create a custom InferenceRunner() callback. FasterRCNN uses the
        # Evaluator(), while all other Tensorpack example models use InferenceRunner.
        evaluator = None  # type: Optional[Evaluator]
        validation_metrics_names = None  # type: Optional[List[str]]
        inference_runner_callback = None  # type: Optional[CustomInferenceRunner]
        evaluator_or_validation_metrics = self.trial.validation_metrics()
        if isinstance(evaluator_or_validation_metrics, list):
            check.is_not_none(validation_dataflow)
            validation_scalar_stats = CustomScalarStats(
                evaluator_or_validation_metrics, prefix="val"
            )
            validation_metrics_names = validation_scalar_stats.names_with_prefix()
            inference_runner_callback = CustomInferenceRunner(
                self.rendezvous_info.get_rank(), validation_dataflow, validation_scalar_stats
            )
        else:
            evaluator = evaluator_or_validation_metrics

        metrics = ["loss", *self.trial.training_metrics()]

        if self.env.hparams.get("include_summary_metrics"):
            metrics.extend(t.op.inputs[1].name for t in tf.get_collection(tf.GraphKeys.SUMMARIES))

        manager_cb = ManagerCallback(
            metrics,
            evaluator,
            validation_metrics_names,
            self.workloads,
            self.is_chief,
            self.rendezvous_info.get_rank(),
            self.context,
        )

        # TODO: check to make sure users don't pass in InferenceRunner
        # because that will run validation after every RUN_STEP.
        self.cbs = [manager_cb, *self.trial.tensorpack_callbacks()]
        if inference_runner_callback:
            self.cbs.append(inference_runner_callback)

    def _load(self) -> None:
        if self.load_path is None:
            # If not loading from checkpoint check if backbone weights are specified.
            backbone_weights_path = self.trial.load_backbone_weights()
            if backbone_weights_path:
                self.load_path = pathlib.Path(backbone_weights_path)
        else:
            self.load_path = self.load_path.joinpath("checkpoint")

        if self.load_path is None or not self.is_chief:
            logging.info("Not loading model")
            self.session_init = None
        else:
            logging.info(f"Loading model from {self.load_path}")
            self.session_init = tp.get_model_loader(str(self.load_path))

    def run(self) -> None:
        logging.info(f"Rank {self.rendezvous_info.get_rank()} calling train_with_defaults")
        try:
            self.trainer.train_with_defaults(
                callbacks=self.cbs,
                monitors=self.trial.tensorpack_monitors(),
                steps_per_epoch=self.scheduling_unit,
                starting_epoch=self.env.first_step(),
                max_epoch=self.env.first_step() + IMPOSSIBLY_LARGE_EPOCHS,
                session_init=self.session_init,
            )
        except det.errors.WorkerFinishedGracefully:
            pass
        else:
            raise AssertionError(
                "Training loop exited unexpectedly but without throwing any errors. This is "
                "possibly due to a user callback causing the training loop to exit, which is not "
                "supported at this time."
            )


class SchedulePoint:
    def __init__(self, point: int, value: float, interp: Optional[str] = None) -> None:
        self.point = point
        self.value = value
        self.interp = interp

    def __repr__(self) -> str:
        return f"SchedulePoint(point={self.point}, value={self.value}, interp={self.interp})"


class ScheduleSetter(tp.callbacks.HyperParamSetter):  # type: ignore
    """
    Hyperparameter setter callback for step-based points that (1) does the right
    thing when resuming training from the middle by computing the value on
    every step and (2) can handle different interpolations.
    """

    def __init__(self, param: Any, schedule: List[SchedulePoint]) -> None:
        super().__init__(param)
        self.schedule = schedule
        logging.info(f"ScheduleSetter created with schedule {schedule}")

    def _get_value_to_set(self) -> float:
        v = self._real_get_value_to_set()
        logging.debug(f"Param setter: step={self.global_step} value={v}")
        return v

    def _real_get_value_to_set(self) -> float:
        step = self.global_step  # type: int
        for p0, p1 in pairs(self.schedule):
            if p0.point <= step < p1.point:
                if p0.interp is None:
                    return p0.value
                elif p0.interp == "linear":
                    t = (step - p0.point) / (p1.point - p0.point)
                    return p0.value + (p1.value - p0.value) * t

                raise ValueError(f"Unknown interpolation type: {p0.interp}")
        return self.schedule[-1].value

    def _trigger_step(self) -> None:
        self._trigger()

    def _trigger_epoch(self) -> None:
        pass


class TensorpackTrial(det.Trial):
    trial_controller_class = TensorpackTrialController

    @abstractmethod
    def build_model(self, trainer_type: str) -> tp.ModelDesc:
        """Returns the Tensorpack ModelDesc describing the model."""
        pass

    def training_metrics(self) -> List[str]:
        """Returns a list of names of tensors to use as training metrics in addition te the loss."""
        return []

    @abstractmethod
    def validation_metrics(self) -> Union[List[str], Evaluator]:
        """
        Returns either an Evaluator object that computes the validation metrics or a list of tensor
        names to use.
        """
        pass

    @abstractmethod
    def build_training_dataflow(self) -> tp.DataFlow:
        """
        Returns the tp.DataFlow to use for training.
        """
        pass

    def build_validation_dataflow(self) -> Optional[tp.DataFlow]:
        """
        Optionally returns the tp.DataFlow to use for validation.
        """
        return None

    def tensorpack_callbacks(self) -> List[tp.Callback]:
        """Returns a list of Tensorpack callbacks to use during training."""
        return []

    def tensorpack_monitors(self) -> List[tp.MonitorBase]:
        """Returns a list of Tensorpack monitors to use during training."""
        return []

    def load_backbone_weights(self) -> Optional[str]:
        """Returns the path to backbone weights"""
        return None
