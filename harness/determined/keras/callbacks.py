import logging
import pathlib
import time
from typing import Any, Dict, List, Optional

import tensorflow as tf
from packaging import version
from tensorflow.python.keras.utils import tf_utils

from determined import profiler, tensorboard


class Callback(tf.keras.callbacks.Callback):  # type: ignore
    """
    A Determined subclass of the ``tf.keras.callbacks.Callback`` interface which supports
    additional new callbacks.

    .. warning::

       The following behaviors differ between normal Keras operation and Keras
       operation within Determined:

        * Keras calls on_epoch_end at the end of the training dataset, but Determined calls it
          based on the records_per_epoch setting in the experiment config.

        * Keras calls on_epoch_end with training and validation logs, but Determined does not
          schedule training or validation around epochs in general, so Determined cannot
          guarantee that those values are available for on_epoch_end calls.  As a result,
          on_epoch_end will be called with an empty dictionary for its logs.

        * Keras does not support stateful callbacks, but Determined does.  Therefore:

           * The tf.keras version of ``EarlyStopping`` will not work right in Determined.  You
             should use you should use :class:`determined.keras.callbacks.EarlyStopping` instead.
           * The tf.keras version of ``ReduceLROnPlateau`` will not work right in Determined.  You
             should use you should use :class:`determined.keras.callbacks.ReduceLRScheduler`
             instead.

          The Determined versions are based around ``on_test_end`` rather than ``on_epoch_end``,
          which can be influenced by setting ``min_validation_period`` in the experiment
          configuration.
    """

    def on_train_workload_begin(
        self, total_batches_trained: int, batches_requested: Optional[int], logs: Dict
    ) -> None:
        """
        on_train_workload_begin is called before a chunk of model training.  The number of batches
        in the workload may vary, but will not exceed the scheduling_unit setting for the
        experiment.

        Parameters:
            total_batches_trained:  The number of batches trained at the start of the workload.

            batches_requested: The number of batches expected to train during the workload.

            logs: a dictionary (presently always an empty dictionary)
        """
        pass

    def on_train_workload_end(self, total_batches_trained: int, logs: Dict) -> None:
        """
        on_train_workload_end is called after a chunk of model training.

        Parameters:
            total_batches_trained:  The number of batches trained at the end of the workload.

            logs: a dictionary of training metrics aggregated during this workload.
        """
        pass

    def on_checkpoint_end(self, checkpoint_dir: str) -> None:
        """
        on_checkpoint_end is called after a checkpoint is finished, and allows users to save
        arbitrary files alongside the checkpoint.

        Parameters:
            checkpoint_dir:  The path to the checkpoint_dir where new files may be added.
        """
        pass

    def get_state(self) -> Any:
        """
        get_state should return a pickleable object that represents the state of this callback.

        When training is continued from a checkpoint, the value returned from get_state() will be
        passed back to the Callback object via load_state().
        """
        return None

    def load_state(self, state: Any) -> None:
        """
        load_state should accept the exact pickleable object returned by get_state to restore the
        internal state of a stateful Callback as it was when load_state was called.
        """
        pass


class _PolyLogs:
    """Support the tf2.3+ feature of passing numpy or tf logs to each callback."""

    def __init__(self, tf_logs: Optional[Dict]) -> None:
        self.tf_logs = tf_logs or {}
        self.np_logs = None  # type: Optional[Dict]

    if version.parse(tf.__version__) < version.parse("2.3.0"):

        def __call__(self, supports_tf_logs: bool) -> Dict:
            return self.tf_logs

    else:

        def __call__(self, supports_tf_logs: bool) -> Dict:
            if supports_tf_logs:
                return self.tf_logs
            if self.np_logs is None:
                try:
                    self.np_logs = tf_utils.to_numpy_or_python_type(self.tf_logs)
                except AttributeError:  # New method as of TF 2.5.
                    self.np_logs = tf_utils.sync_to_numpy_or_python_type(self.tf_logs)
            # `to_numpy_or_python_type` will return a dict if `self.tf_logs` is a dict,
            # but current type annotations aren't good enough to describe that.
            return self.np_logs  # type: ignore


class _MultiplexerBase(tf.keras.callbacks.Callback):  # type: ignore
    """
    _MultiplexerBase injects the calls so that Determined Callbacks work inside of Keras.

    _MultiplexerBase is capable of injecting certain calls, such as on_epoch_end, on its own, based
    purely on counting batches that pass through it.  However, it cannot trigger
    on_train_workload_end on it's own, as the source of truth for that call when training on the
    cluster is the determined-master.

    Therefore, _MultiplexerBase is a private object that is not useful on its own.  It is
    subclassed for either local training mode or cluster training mode.
    """

    _chief_worker_only = False
    _supports_tf_logs = True

    class SavableState:
        """SavableState makes it easy to persist state using vars(self.state)."""

        def __init__(
            self,
            epoch: int,
            want_epoch_begin: bool,
            epoch_batches: int,
            latest_training_metrics: Dict,
            total_batches: int,
        ) -> None:
            self.epoch = epoch
            self.want_epoch_begin = want_epoch_begin
            self.epoch_batches = epoch_batches
            self.latest_training_metrics = latest_training_metrics
            self.total_batches = total_batches

    def __init__(
        self,
        callbacks: List[Callback],
        is_chief: bool,
        batch_size: int,
        batches_per_epoch: Optional[None],
        load_state: Optional[Dict],
    ) -> None:
        self.all_callbacks = callbacks
        self.callbacks = [
            cb for cb in callbacks if is_chief or not getattr(cb, "_chief_worker_only", False)
        ]
        self.is_chief = is_chief
        self.batch_size = batch_size
        self.batches_per_epoch = batches_per_epoch
        self.load_state = load_state

        # batches_requested requested (for on_train_workload_begin) must be passed in from outside.
        self.batches_requested = None  # type: Optional[int]

        self.want_train_workload_begin = True

        self.state = self.SavableState(
            epoch=0,
            want_epoch_begin=True,
            epoch_batches=0,
            latest_training_metrics={},
            total_batches=0,
        )

    def set_params(self, params: Dict) -> None:
        for cb in self.callbacks:
            cb.set_params(params)

    def set_model(self, model: tf.keras.models.Model) -> None:
        self.model = model
        for cb in self.callbacks:
            cb.set_model(model)

    def on_epoch_begin(self, epoch: int, logs: Optional[Dict] = None) -> None:
        # We'll call this explicitly when we want it, via _corrected_epoch_begin().
        pass

    def on_epoch_end(self, epoch: int, logs: Optional[Dict] = None) -> None:
        # We'll call this explicitly when we want it, via _corrected_epoch_end().
        pass

    # Train-related callbacks.

    def on_train_begin(self, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_train_begin(polylogs(supports_tf_logs))
        # Now it is safe to load callback state.
        self._delayed_load_state()

    def on_train_batch_begin(self, _: int, logs: Optional[Dict] = None) -> None:
        self._check_epoch_begin()
        self._check_train_workload_begin()

        assert isinstance(logs, dict)
        if "batch" in logs:
            logs["batch"] = self.state.epoch_batches
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_train_batch_begin(self.state.epoch_batches, polylogs(supports_tf_logs))

    def on_train_batch_end(self, batch: int, logs: Optional[Dict] = None) -> None:
        assert isinstance(logs, dict)
        if "batch" in logs:
            logs["batch"] = self.state.epoch_batches
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_train_batch_end(self.state.epoch_batches, polylogs(supports_tf_logs))

        self.state.total_batches += 1
        self.state.epoch_batches += 1

        if (
            self.batches_per_epoch is not None
            and self.state.epoch_batches >= self.batches_per_epoch
        ):
            self._corrected_epoch_end(epoch=self.state.epoch, logs={})
            self.state.epoch_batches = 0
            self.state.epoch += 1
            self.state.want_epoch_begin = True

    def on_train_end(self, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_train_end(polylogs(supports_tf_logs))

    # Test-related callbacks.

    def on_test_begin(self, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_test_begin(polylogs(supports_tf_logs))

    def on_test_batch_begin(self, batch: int, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_test_batch_begin(batch, polylogs(supports_tf_logs))

    def on_test_batch_end(self, batch: int, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_test_batch_end(batch, polylogs(supports_tf_logs))

    def on_test_end(self, logs: Optional[Dict] = None) -> None:
        # Ignore on_test_end, since in TF 1.X it is called without metrics, and we want to
        # guarantee the TF2.X behavior that logs contains useful logs.  Additionally, even in
        # TF2.X these metrics would not yet be aggregated across GPUs.
        pass

    # Predict-related callbacks.

    def on_predict_begin(self, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_predict_begin(polylogs(supports_tf_logs))

    def on_predict_batch_begin(self, batch: int, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_predict_batch_begin(batch, polylogs(supports_tf_logs))

    def on_predict_batch_end(self, batch: int, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_predict_batch_end(batch, polylogs(supports_tf_logs))

    def on_predict_end(self, logs: Optional[Dict] = None) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_predict_end(polylogs(supports_tf_logs))

    def _corrected_epoch_begin(self, epoch: int, logs: Dict) -> None:
        """The real on_epoch_begin call, which is guaranteed to happen at the right time."""
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_epoch_begin(epoch, polylogs(supports_tf_logs))

    def _corrected_epoch_end(self, epoch: int, logs: Dict) -> None:
        """The real on_epoch_end call, which is guaranteed to happen at the right time."""
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_epoch_end(epoch, polylogs(supports_tf_logs))

    def _corrected_test_end(self, logs: Dict) -> None:
        polylogs = _PolyLogs(logs)
        for cb in self.callbacks:
            supports_tf_logs = getattr(cb, "_supports_tf_logs", False)
            cb.on_test_end(polylogs(supports_tf_logs))

    def _check_train_workload_begin(self) -> None:
        if not self.want_train_workload_begin:
            return
        self.want_train_workload_begin = False
        for cb in self.callbacks:
            if isinstance(cb, Callback):
                cb.on_train_workload_begin(self.state.total_batches, self.batches_requested, {})

    def _check_epoch_begin(self) -> None:
        if not self.state.want_epoch_begin:
            return
        self.state.want_epoch_begin = False
        self._corrected_epoch_begin(self.state.epoch, {})

    def _delayed_load_state(self) -> None:
        """
        Load state after on_train_begin, so that only the first on_train_begin can affect savable
        state.  This is more consistent with how we don't allow on_train_end to alter savable
        state, and also all TF-authored Callbacks would break if we didn't do it this way.
        """
        if self.load_state is None:
            return
        self.state = self.SavableState(**self.load_state["self"])
        for cb, cb_state in zip(self.all_callbacks, self.load_state["callbacks"]):
            if isinstance(cb, Callback):
                cb.load_state(cb_state)

    # Must be triggered externally.
    def _get_state(self) -> Dict:
        cb_state = [
            cb.get_state() if isinstance(cb, Callback) else None for cb in self.all_callbacks
        ]
        return {"self": vars(self.state), "callbacks": cb_state}

    # Must be triggered externally.
    def _checkpoint_end(self, checkpoint_dir: pathlib.Path) -> None:
        for cb in self.callbacks:
            if isinstance(cb, Callback):
                cb.on_checkpoint_end(str(checkpoint_dir))

    # Must be triggered externally.
    def set_batches_requested(self, batches_requested: int) -> None:
        self.batches_requested = batches_requested

    # Must be triggered externally.
    def _test_end(self, logs: Dict) -> None:
        self._corrected_test_end(logs)

    # Must be triggered externally.
    def _train_workload_end(self, metrics: Dict) -> None:
        self.state.latest_training_metrics = metrics
        for cb in self.callbacks:
            if isinstance(cb, Callback):
                cb.on_train_workload_end(self.state.total_batches, metrics)
        self.want_train_workload_begin = True


class _DeterminedProgress(Callback):
    """
    A Determined-friendly replacement for the usual verbose=1 behavior.

    It is enabled by default, but it can be disabled with ``context.configure_fit()``:

    .. code-block:: python

       class MyTFKerasTrial(TFKerasTrial):
           ...

           def __init__(self, context):
               ...
               context.configure_fit(verbose=False)
    """

    _chief_worker_only = True
    _supports_tf_logs = True

    class SavableState:
        """SavableState makes it easy to persist state using vars(self.state)."""

        def __init__(self, trained_total: int, validation_len: Optional[int]) -> None:
            self.trained_total = trained_total
            self.validation_len = validation_len

    def __init__(self) -> None:
        self.state = self.SavableState(trained_total=0, validation_len=None)
        self.train_len = None  # type: Optional[int]
        self.batches = 0
        self.percent_reported = -1

    def _report(
        self, logs: Optional[Dict], training: bool, batches: int, total: Optional[int]
    ) -> None:
        # Only report progress if we have the target total.
        if total is None:
            return

        # Don't report more often than 10% increments.
        percent_10 = int((batches / total) * 10) * 10
        if percent_10 <= self.percent_reported:
            return

        # When you do report, report to 1% accuracy.
        percent = int((batches / total) * 100)
        self.percent_reported = percent

        if training:
            report = (
                f"total batches trained: {self.state.trained_total}, "
                f"workload {percent}% complete ({batches}/{total})"
            )
        else:
            report = (
                f"validation after batch: {self.state.trained_total}, "
                f"validation {percent}% complete ({batches}/{total})"
            )
        if logs is not None:
            metrics = {k: v for k, v in logs.items() if k not in ("batch", "size")}
            report += f": {metrics}"

        print(report)

    def on_train_workload_begin(
        self, total_batches_trained: int, batches_requested: Optional[int], logs: Dict
    ) -> None:
        self.train_len = batches_requested
        self.batches = 0
        self.state.trained_total = total_batches_trained
        self.percent_reported = -1
        self._report(logs=None, training=True, batches=self.batches, total=self.train_len)

    def on_train_batch_end(self, batch: int, logs: Dict) -> None:
        self.batches += 1
        self.state.trained_total += 1
        self._report(logs=logs, training=True, batches=self.batches, total=self.train_len)

    def on_test_begin(self, logs: Dict) -> None:
        self.batches = 0
        self.percent_reported = -1
        self._report(logs=None, training=False, batches=0, total=self.state.validation_len)

    def on_test_batch_end(self, batch: int, logs: Dict) -> None:
        self.batches += 1
        self._report(
            logs=logs, training=False, batches=self.batches, total=self.state.validation_len
        )

    def on_test_end(self, logs: Dict) -> None:
        self.state.validation_len = self.batches

    def get_state(self) -> Any:
        return vars(self.state)

    def load_state(self, state: Any) -> None:
        self.state = self.SavableState(**state)


class _DeterminedHistory(Callback):
    """Like tf.keras.callbacks.History but based on validations and stateful."""

    _chief_worker_only = False
    _supports_tf_logs = True

    def __init__(self) -> None:
        self.history = {}  # type: Dict

    def on_test_end(self, logs: Dict) -> None:
        for k, v in logs.items():
            self.history.setdefault(k, []).append(v)

        # I'm not sure how this ever gets unset but keras definitely thinks it's possible.
        self.model.history = self

    def get_state(self) -> Any:
        return self.history

    def load_state(self, state: Any) -> None:
        self.history = state


def _tf_keras_callback_get_state(cb: Any) -> Any:
    """Get state based on _savable_attributes."""
    cb_name = type(cb).__name__
    for var in vars(cb):
        if var not in cb._savable_attributes and var not in cb._extra_attributes:
            raise NotImplementedError(
                f"The Determined {cb_name} is not known to work with an implementation of "
                f"{cb_name} that contains a variable named {var}."
            )
        return {var: getattr(cb, var) for var in cb._savable_attributes if hasattr(cb, var)}


def _tf_keras_callback_load_state(cb: Any, state: Any) -> None:
    """Load state based on _savable_attributes."""
    for var in cb._savable_attributes:
        if var in state:
            setattr(cb, var, state[var])


class EarlyStopping(tf.keras.callbacks.EarlyStopping, Callback):  # type: ignore
    """
    EarlyStopping behaves exactly like the ``tf.keras.callbacks.EarlyStopping`` except that it
    checks after every ``on_test_end()`` rather than every ``on_epoch_end()`` and it can save
    and restore its state after pauses in training.

    Therefore, part of configuring the Determined implementation of EarlyStopping is to
    configure ``min_validation_period`` for the experiment appropriately (likely it should be
    configured to validate every epoch).

    In Determined, ``on_test_end`` may be called slightly more often than ``min_validation_period``
    during some types of hyperparameter searches, but it is unlikely for that to occur often enough
    have a meaningful impact on this callback's operation.
    """

    _savable_attributes = {
        # tf.keras.callbacks.EarlyStopping values.
        "baseline",
        "best_weights",
        "restore_best_weights",
        "stopped_epoch",
        "wait",
        # Our own values.
        "test_end_count",
    }

    _extra_attributes = {
        # Base Callback values.
        "model",
        "params",
        "validation_data",
        "_chief_worker_only",
        "_supports_tf_logs",
        # Constant config values.
        "best",
        "min_delta",
        "monitor",
        "monitor_op",
        "patience",
        "verbose",
        # Our own values.
        "_extra_attributes",
        "_savable_attributes",
    }

    def __init__(self, *arg: Any, **kwarg: Any) -> None:
        # We have a diamond inheritance pattern, so avoid calling super().
        tf.keras.callbacks.EarlyStopping.__init__(self, *arg, **kwarg)
        self.test_end_count = 0

    def on_epoch_end(self, epoch: int, logs: Optional[Dict]) -> None:
        # Ignore on_epoch_end calls, which never contain metrics in Determined.
        pass

    def on_test_end(self, logs: Optional[Dict] = None) -> None:
        # Trigger the original EarlyStopping's on_epoch_end call.
        tf.keras.callbacks.EarlyStopping.on_epoch_end(self, self.test_end_count, logs)
        self.test_end_count += 1

    def get_state(self) -> Any:
        return _tf_keras_callback_get_state(self)

    def load_state(self, state: Any) -> None:
        _tf_keras_callback_load_state(self, state)


class ReduceLROnPlateau(tf.keras.callbacks.ReduceLROnPlateau, Callback):  # type: ignore
    """
    ReduceLROnPlateau behaves exactly like the ``tf.keras.callbacks.ReduceLROnPlateau`` except that
    it checks after every ``on_test_end()`` rather than every ``on_epoch_end()`` and it can save and
    restore its state after pauses in training.

    Therefore, part of configuring the Determined implementation of ReduceLROnPlateau is to
    configure ``min_validation_period`` for the experiment appropriately (likely it should be
    configured to validate every epoch).

    In Determined, ``on_test_end`` may be called slightly more often than ``min_validation_period``
    during some types of hyperparameter searches, but it is unlikely for that to occur often
    enough have a meaningful impact on this callback's operation.
    """

    _savable_attributes = {
        # tf.keras.callbacks.ReduceLROnPlateau values.
        "cooldown_counter",
        "wait",
        "best",
        # Our own values.
        "test_end_count",
    }

    _extra_attributes = {
        # Base Callback values.
        "model",
        "params",
        "validation_data",
        "_chief_worker_only",
        "_supports_tf_logs",
        # Constant config values.
        "cooldown",
        "factor",
        "min_delta",
        "min_lr",
        "mode",
        "monitor",
        "monitor_op",
        "patience",
        "verbose",
        # Our own values.
        "_extra_attributes",
        "_savable_attributes",
    }

    def __init__(self, *arg: Any, **kwarg: Any) -> None:
        # We have a diamond inheritance pattern, so avoid calling super().
        tf.keras.callbacks.ReduceLROnPlateau.__init__(self, *arg, **kwarg)
        self.test_end_count = 0

    def on_epoch_end(self, epoch: int, logs: Optional[Dict]) -> None:
        # Ignore on_epoch_end calls, which never contain metrics in Determined.
        pass

    def on_test_end(self, logs: Optional[Dict] = None) -> None:
        # Trigger the original ReduceLROnPlateau's on_epoch_end call.
        tf.keras.callbacks.ReduceLROnPlateau.on_epoch_end(self, self.test_end_count, logs)
        self.test_end_count += 1

    def get_state(self) -> Any:
        return _tf_keras_callback_get_state(self)

    def load_state(self, state: Any) -> None:
        _tf_keras_callback_load_state(self, state)


class TensorBoard(tf.keras.callbacks.TensorBoard, Callback):  # type: ignore
    """
    This is a thin wrapper over the TensorBoard callback that ships with ``tf.keras``.  For more
    information, see the :ref:`TensorBoard Guide <tensorboards>` or the upstream docs for
    `tf.keras.callbacks.TensorBoard
    <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/TensorBoard>`__.

    Note that if a ``log_dir`` argument is passed to the constructor, it will be ignored.
    """

    # TensorBoard uses on_epoch_end but we manually take care of that.
    _skip_epoch_end_check = True

    def __init__(self, *args: Any, **kwargs: Any):
        self.workload_end_count = 0
        user_log_dir = kwargs.pop("log_dir", None)
        if user_log_dir is not None:
            logging.warning(
                f"arg log_dir={user_log_dir} to det.keras.callbacks.TensorBoard will be ignored"
            )
        log_dir = str(tensorboard.get_base_path({}).resolve())
        tf.keras.callbacks.TensorBoard.__init__(self, log_dir=log_dir, *args, **kwargs)

    def _write_logs(self, *args: Any) -> None:
        """
        _write_logs calls the original _write_logs() function from the Keras
        TensorBoard callback. After the logs are flushed to disk, we close and
        reopen the tf event writer so that it serializes the next set of logs
        to a new file. This allows the tensorboard manager to treat the
        written files as immutable and upload them to persistent storage
        without later having to append to them. This behavior is useful for
        tensorboard backed by S3.
        """
        tf.keras.callbacks.TensorBoard._write_logs(self, *args)
        self.writer.close()
        self.writer.reopen()

    def get_state(self) -> Any:
        return {"workload_end_count": self.workload_end_count}

    def load_state(self, state: Any) -> None:
        self.workload_end_count = state["workload_end_count"]

    # Translate train workload hooks into epoch hooks.
    def on_train_workload_begin(
        self, total_batches_trained: int, batches_requested: Optional[int], logs: Dict
    ) -> None:
        tf.keras.callbacks.TensorBoard.on_epoch_begin(self, self.workload_end_count, logs)

    def on_train_workload_end(self, total_batches_trained: int, logs: Dict) -> None:
        tf.keras.callbacks.TensorBoard.on_epoch_end(self, self.workload_end_count, logs)
        self.workload_end_count += 1


class _DeterminedProfiler(Callback):
    """Hooks Keras into the profiler agent lifecycle"""

    def __init__(self, prof: profiler.ProfilerAgent, batch_size: int) -> None:
        super().__init__()
        self.total_batch = 0
        self.prof = prof
        self.batch_start = None  # type: Optional[float]
        self.batch_size = batch_size

    def get_state(self) -> Dict:
        return {"total_batch": self.total_batch}

    def load_state(self, state: Any) -> None:
        self.total_batch = state["total_batch"]

    def on_train_batch_begin(self, batch: int, _: Optional[Dict] = None) -> None:
        self.total_batch += 1
        self.batch_start = time.time()
        self.prof.update_batch_idx(self.total_batch)

    def on_train_batch_end(self, _: int, logs: Optional[Dict] = None) -> None:
        if not logs or not self.batch_start:
            return

        samples = logs.get("size", self.batch_size)
        samples_per_second = samples / (time.time() - self.batch_start)
        self.prof.record_metric("samples_per_second", samples_per_second)
