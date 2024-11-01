import contextlib
import logging
import os
import pathlib
import pickle
import shutil
import tempfile
from typing import Any, Dict, Optional, Tuple, Union

from tensorflow.keras import callbacks, models

import determined as det
from determined import core

logger = logging.getLogger("determined.keras")


class DeterminedCallback(callbacks.ProgbarLogger):  # type: ignore
    """
    DeterminedCallback adds Determined tracking, checkpointing, pausing, and restoring to a Keras
    ``model.fit()`` call.  Just include it as one of your callbacks.

    DeterminedCallback must not be used with a BackupAndRestore callback or a ModelCheckpoint
    callback, which have conflicting behaviors.

    When using DeterminedCallback:
      - The ``initial_epoch`` parameter to ``model.fit()`` will be overridden.  Rely on the
        ``checkpoint`` and ``continue_id`` parameters to DeterminedCallback instead.
      - Checkpoints are saved and uploaded to Determined's checkpoint storage every epoch by
        default, but can be saved less frequently based on the ``checkpoint_epochs`` parameter.
        Checkpoints are always saved when training finishes or is preempted.
      - Training will check for preemption every epoch.  This means, for instance, if you click the
        "pause" button in the UI, training will continue until the next epoch boundary.
      - The normal verbose=1 TQDM progress bars are replaced with a more log-friendly output.
      - By default, checkpoints are saved with ``model.save_weights()`` and restored with
        ``model.load_weights()``.  This is configurable by subclassing DeterminedCallback and
        implementing custom ``save_model`` and ``load_model`` methods.
      - By default, weights are saved to the path ``model_checkpoint`` inside the checkpoint
        directory, which you can pass to ``model.load_weights()`` to load a trained model from a
        downloaded checkpoint after training is complete.

    Arguments:
        core_context: the result of a ``det.core.init()`` call
        checkpoint: Either None, or a checkpoint uuid to start from.  When you are training
            on-cluster, this is likely the output of ``det.get_cluster_info().latest_checkpoint``.
        continue_id: A unique identifier that is saved with the checkpoint.  When you are training
            on-cluster, this is likely the output of ``det.get_cluster_info().trial.trial_id``.
            When loading an existing checkpoint, if the provided continue_id matches what was in the
            checkpoint, training will continue from the epoch where it left off (a pause-and-unpause
            scenario).  If the provided continue_id does not match the checkpoint, the model weights
            will be loaded but training will begin from epoch=0 (a warm-start scenario).
        train_metrics_report_period: Either the string ``"epoch"`` or a number of batches to wait
            between reporting training metrics to Determined master.  Default: ``"epoch"``.
        checkpoint_epochs: Save every N epochs.  Checkpoints are always saved when training is
            preempted, or at the end of training.  A value of `0` means to only save at those times.
            Default: 1.

    See also:
       -  :meth:`DistributedContext.from_tf_config
          <determined.core.DistributedContext.from_tf_config>`
    """

    _chief_worker_only = False
    _supports_tf_logs = False

    def __init__(
        self,
        core_context: core.Context,
        checkpoint: Optional[str],
        continue_id: Union[int, str],
        *,
        train_metrics_report_period: Union[int, str] = "epoch",
        checkpoint_epochs: int = 1,
    ) -> None:
        # We subclass ProgbarLogger to disable standard verbose=1 behavior, but really we don't
        # want any of its actual behavior.  So __init__ the supersuper class directly.
        callbacks.Callback.__init__(self)
        self._core = core_context
        self._checkpoint = checkpoint
        self._continue_id = continue_id
        self._report_period = train_metrics_report_period
        self._checkpoint_epochs = checkpoint_epochs

        self._is_chief = core_context.distributed.rank == 0
        self._is_verbose = False  # Configured by .set_params().

        # This is an undocumented workaround in case off-cluster user-code saving goes awry.
        self._save_user_code = True

        self._steps_completed = 0

        # We only track the last value from on_epoch_begin() in order to handle off-epoch reporting
        # of training metrics.
        self._epoch = -1

        # Track on_epoch_end() calls, and the last on_epoch_end() where we saved a checkpoint, in
        # order to be able to decide if we have any uncheckpointed work when we hit on_train_end().
        self._last_train_epoch = -1
        self._last_ckpt_epoch = -1

        # progress
        self._training_length: Optional[int] = None
        self._validation_length: Optional[int] = None
        self._training_batches = 0
        self._validation_batches = 0
        self._percent_reported = -1

        # We download the checkpoint, then have to keep it for a while until we can delete it.
        self._ckpt_context: Optional[contextlib.ExitStack] = None

    # Mask the inherited ProgbarLogger behavior that we don't actually want.
    def set_params(self, params: Dict[str, Any]) -> None:
        callbacks.Callback.set_params(self, params)
        self._is_verbose = bool(params.get("verbose", 0) != 0)

    # Mask the inherited ProgbarLogger behavior that we don't actually want.
    def on_predict_begin(self, logs: Optional[Dict[str, Any]]) -> None:
        pass

    # Mask the inherited ProgbarLogger behavior that we don't actually want.
    def on_predict_batch_end(self, batch: int, logs: Optional[Dict[str, Any]]) -> None:
        pass

    # Mask the inherited ProgbarLogger behavior that we don't actually want.
    def on_predict_end(self, logs: Optional[Dict[str, Any]]) -> None:
        pass

    def _implements_train_batch_hooks(self) -> bool:
        return True

    def _implements_test_batch_hooks(self) -> bool:
        # Tell keras that we don't need on_test_batch_end unless we are verbose.
        return self._is_verbose

    def _implements_predict_batch_hooks(self) -> bool:
        # Tell keras that we don't actually want any on_predict_batch_end calls.
        return False

    def _print_progress(
        self, logs: Optional[Dict[str, Any]], training: bool, batches: int, total: Optional[int]
    ) -> None:
        # Only report progress if we have the target total.
        if total is None:
            return

        # Don't report more often than 10% increments.
        percent_10 = int((batches / total) * 10) * 10
        if percent_10 <= self._percent_reported:
            return

        # When you do report, report to 1% accuracy.
        percent = int((batches / total) * 100)
        self._percent_reported = percent

        if training:
            report = (
                f"total batches trained: {self._steps_completed}, "
                f"epoch {percent}% complete ({batches}/{total})"
            )
        else:
            report = (
                f"validation after batch: {self._steps_completed}, "
                f"validation {percent}% complete ({batches}/{total})"
            )
        if logs is not None:
            metrics = {k: v for k, v in logs.items() if k not in ("batch", "size")}
            report += f": {metrics}"

        print(report)

    def on_train_begin(self, logs: Optional[Dict[str, Any]]) -> None:
        # Load initial state.  Note that we might set model._training_state, which is how we
        # override the initial_epoch provided to model.fit(), but this callback occurs just before
        # model.fit() trys to read model._training_state.
        self._ckpt_context = self._load(self._checkpoint)

    def on_epoch_begin(self, epoch: int, logs: Optional[Dict[str, Any]]) -> None:
        self._epoch = epoch
        if not self._is_chief:
            return

        # Set status.
        self._core.train.set_status("training")

        # Print progress.
        if self._is_verbose:
            self._training_batches = 0
            self._percent_reported = -1
            self._print_progress(logs=None, training=True, batches=0, total=self._training_length)

    def on_train_batch_end(self, batch: int, logs: Optional[Dict[str, Any]]) -> None:
        self._steps_completed += 1

        # Delete the initial checkpoint files, if we haven't already.
        if self._ckpt_context:
            self._ckpt_context.close()
            self._ckpt_context = None

        if not self._is_chief:
            return

        assert logs

        # Report metrics.
        if (
            isinstance(self._report_period, int)
            and self._steps_completed % self._report_period == 0
        ):
            # Skip non-metrics data from logs.
            metrics = {k: v for k, v in logs.items() if k not in ("batch", "size")}
            # Add epochs and batches.
            metrics["epochs"] = metrics.get("epochs", self._epoch + 1)
            metrics["batches"] = metrics.get("batches", self._steps_completed)
            self._core.train.report_metrics("training", self._steps_completed, metrics)

        # Print progress.
        if self._is_verbose:
            self._training_batches += 1
            self._print_progress(
                logs=logs,
                training=True,
                batches=self._training_batches,
                total=self._training_length,
            )

    def on_epoch_end(self, epoch: int, logs: Optional[Dict[str, Any]]) -> None:
        # Report metrics.
        if self._is_chief and self._report_period == "epoch":
            assert logs
            # Filter out the validation logs.
            metrics = {k: v for k, v in logs.items() if not k.startswith("val_")}
            metrics["epochs"] = metrics.get("epochs", epoch + 1)
            metrics["batches"] = metrics.get("batches", self._steps_completed)
            self._core.train.report_metrics("training", self._steps_completed, metrics)

        # Report progress.
        if self._is_chief and self.params["epochs"]:
            self._core.train.report_progress((epoch + 1) / self.params["epochs"])

        # Save a checkpoint.
        self._last_train_epoch = epoch
        if self._checkpoint_epochs > 0 and (epoch + 1) % self._checkpoint_epochs == 0:
            self._save(epoch)
            self._last_ckpt_epoch = epoch

        # Check for preemption.  Checkpointing time can be non-negligible, so we check for
        # preemption here after possibly saving a checkpoint.  If we didn't save a checkpoint but we
        # did get preempted, we'll catch that in the checkpoint fallback in on_train_end().
        if self._core.preempt.should_preempt():
            self.model.stop_training = True

        # Remember how many batches we trained, for next time.
        if self._is_chief:
            self._training_length = self._training_batches

    def on_test_begin(self, logs: Optional[Dict[str, Any]]) -> None:
        if not self._is_chief:
            return

        # Set status.
        self._core.train.set_status("validating")

        # Print progress.
        if self._is_verbose:
            self._validation_batches = 0
            self._percent_reported = -1
            self._print_progress(
                logs=None, training=False, batches=0, total=self._validation_length
            )

    def on_test_batch_end(self, batch: int, logs: Optional[Dict[str, Any]]) -> None:
        # Print progress.
        if self._is_chief and self._is_verbose:
            self._validation_batches += 1
            self._print_progress(
                logs=logs,
                training=False,
                batches=self._validation_batches,
                total=self._validation_length,
            )

    def on_test_end(self, logs: Optional[Dict[str, Any]]) -> None:
        if not self._is_chief:
            return

        assert logs
        metrics = {**logs}
        metrics["epochs"] = metrics.get("epochs", self._epoch + 1)
        metrics["batches"] = metrics.get("batches", self._steps_completed)
        self._core.train.report_metrics("validation", self._steps_completed, metrics)

        # Remember how many batches we trained, for next time.
        self._validation_length = self._validation_batches

    def on_train_end(self, logs: Optional[Dict[str, Any]]) -> None:
        # Are we exiting with some amount of uncheckpointed training?
        if self._last_train_epoch > self._last_ckpt_epoch:
            self._save(self._last_train_epoch)

        if self._is_chief:
            self._core.train.set_status("finishing")

    def _save(self, epoch: int) -> None:
        if self._is_chief:
            self._core.train.set_status("checkpointing")

        metadata = {"steps_completed": self._steps_completed}
        # Use shard=True because keras wants every worker to write a checkpoint, even though every
        # worker except the chief will end up deleting it.
        with self._core.checkpoint.store_path(metadata, shard=True) as (path, storage_id):
            # Save the model.
            self.save_model(self.model, str(path / "model_checkpoint"), self._core.distributed)
            # Only the chief saves the callback state and user code
            if self._is_chief:
                with (path / "callback_state").open("wb") as f:
                    state = {
                        "epoch": epoch,
                        "steps_completed": self._steps_completed,
                        "continue_id": self._continue_id,
                        "training_length": self._training_length,
                        "validation_length": self._validation_length,
                    }
                    pickle.dump(state, f)
                # Save user code.
                if self._save_user_code:
                    det.util.write_user_code(path, on_cluster=det.get_cluster_info() is not None)

    def _load(self, checkpoint: Optional[str]) -> Optional[contextlib.ExitStack]:
        if checkpoint is None:
            return None

        if self._is_chief:
            self._core.train.set_status("restoring")

        with contextlib.ExitStack() as exit_stack:
            path = exit_stack.enter_context(self._core.checkpoint.restore_path(checkpoint))

            # Load model.
            self.load_model(self.model, str(path / "model_checkpoint"), self._core.distributed)

            # Load our own state.
            self._load_training_state(path)

            # Success! Don't delete the checkpoint until after the first batch runs though, because
            # the checkpoint isn't actually read until then.
            return exit_stack.pop_all()

        # mypy thinks it's possible to arrive here, but it isn't.
        raise RuntimeError("impossible codepath")

    def _load_training_state(self, path: pathlib.Path) -> None:
        state_path = path / "callback_state"
        if not state_path.exists():
            return
        with state_path.open("rb") as f:
            state = pickle.load(f)
        if state["continue_id"] != self._continue_id:
            return
        # Continue training where we left off.
        self._steps_completed = state["steps_completed"]
        self._training_length = state["training_length"]
        self._validation_length = state["validation_length"]
        initial_epoch: int = state["epoch"] + 1

        # HACK: Trick the training loop into starting on a different epoch.  Internally, this is
        # how keras.callbacks.BackupAndRestore() sets the initial_epoch.
        class WorkerTrainingState:
            # For tf.keras.
            def maybe_load_initial_epoch_from_ckpt(*_: Any, **__: Any) -> int:
                return initial_epoch

            # For plain keras.
            def maybe_load_initial_counters_from_ckpt(*_: Any, **__: Any) -> Tuple[int, int]:
                # We only save on epoch boundaries.
                initial_batch = 0
                return initial_epoch, initial_batch

        self.model._training_state = WorkerTrainingState()

    def save_model(
        self, model: models.Model, path: str, distributed: core.DistributedContext
    ) -> None:
        """
        Users can subclass this if they need to customize how they save their model.

        This method is responsible for meeting the requirements of checkpointing according to the
        needs of the active Strategy.

        See the `TensorFlow docs`_ for more details.

        Arguments:
            model: the model to save
            path: the destination to save to
            distributed: the value of core_context.distributed, which can be used for detecting
                the current process's rank, or inter-worker coordination, as needed.

        .. _TensorFlow docs:
            https://www.tensorflow.org/tutorials/distribute/multi_worker_with_keras
            #model_saving_and_loading
        """

        # MultiWorkerMirroredStrategy requires everyone to save the model (to access shared
        # variables or something) but you have to delete the non-chief copies.  Brilliant.
        if distributed.rank == 0:
            model.save_weights(path)
        else:
            tempdir = tempfile.mkdtemp("save-worker-model")
            try:
                model.save_weights(os.path.join(tempdir, "model_checkpoint"))
            finally:
                shutil.rmtree(tempdir)

    def load_model(
        self, model: models.Model, path: str, distributed: core.DistributedContext
    ) -> None:
        """
        Users can subclass this if they need to customize how they load their model.

        Arguments:
            model: the model to load
            path: the destination to load from
            distributed: the value of core_context.distributed, which can be used for detecting
                the current process's rank, or inter-worker coordination, as needed.
        """

        # Users can subclass this if they just need to change how they save their model.
        model.load_weights(path)


class TensorBoard(callbacks.TensorBoard):  # type: ignore
    """
    This is a thin wrapper over the TensorBoard callback that ships with ``tf.keras``.  For more
    information, see the :ref:`TensorBoard Guide <tensorboards>` or the upstream docs for
    `tf.keras.callbacks.TensorBoard
    <https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/TensorBoard>`__.

    Note that if a ``log_dir`` argument is passed to the constructor, it will be ignored if the
    ``core_context`` is configured for tensorboard (which is the default when on-cluster).
    """

    def __init__(self, core_context: core.Context, *args: Any, **kwargs: Any):
        det_tb_path = core_context.train.get_tensorboard_path()
        if det_tb_path:
            if "log_dir" in kwargs:
                user_log_dir = kwargs.pop("log_dir")
                logger.warning(
                    f"arg log_dir={user_log_dir} to det.keras.TensorBoard will be ignored"
                )
            elif args:
                user_log_dir, args = args[0], args[1:]
                logger.warning(
                    f"arg log_dir={user_log_dir} to det.keras.TensorBoard will be ignored"
                )
            args = [det_tb_path, *args]  # type: ignore
        super().__init__(*args, **kwargs)

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
        super()._write_logs(*args)
        self.writer.close()
        self.writer.reopen()
