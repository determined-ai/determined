import enum
import logging
import pathlib
from typing import Any, Callable, Dict, List, Optional, Set

import determined as det
from determined import core, tensorboard
from determined.common import api, util
from determined.common.api import bindings, errors

logger = logging.getLogger("determined.core")


class EarlyExitReason(enum.Enum):
    INVALID_HP = "EXITED_REASON_INVALID_HP"
    # This is generally unnecessary; just exit early.
    USER_REQUESTED_STOP = "EXITED_REASON_USER_REQUESTED_STOP"


class TrainContext:
    """
    ``TrainContext`` gives access to report training and validation metrics to the Determined master
    during trial tasks.
    """

    def __init__(
        self,
        session: api.Session,
        trial_id: int,
        exp_id: int,
        metrics: core._MetricsContext,
        distributed: core.DistributedContext,
        tensorboard_mode: core.TensorboardMode,
        tensorboard_manager: Optional[tensorboard.TensorboardManager],
        tbd_writer: Optional[tensorboard.BatchMetricWriter],
    ) -> None:
        self._session = session
        self._trial_id = trial_id
        self._exp_id = exp_id
        self._metrics = metrics
        self._distributed = distributed
        if tensorboard_mode != core.TensorboardMode.MANUAL and tensorboard_manager is None:
            raise ValueError("either set TensorboardMode.MANUAL, or pass a tensorboard manager.")
        self._tensorboard_mode = tensorboard_mode
        self._tensorboard_manager = tensorboard_manager
        self._tbd_writer = tbd_writer

    def set_status(self, status: str) -> None:
        """
        Report a short user-facing string that the WebUI can render to indicate what a trial is
        working on.
        """

        body = {"state": status}
        logger.debug(f"set_status({status})")
        self._session.post(f"/api/v1/trials/{self._trial_id}/runner/metadata", json=body)

    def _get_last_validation(self) -> Optional[int]:
        # This is needed by the workload sequencer, but it is not generally stable, because it is
        # easy to call this before reporting any metrics.  If your last checkpoint was older than
        # your last validation, then the value you get from this function might be higher before
        # you report metrics than after (since metrics get archived on first report of new metrics,
        # not on trial restart).  However, this bug does not happen to affect the workload sequencer
        # because of the workload sequencer's very specific use of this function.
        r = self._session.get(f"/api/v1/trials/{self._trial_id}")
        val = r.json()["trial"].get("latestValidation") or {}
        steps_completed = val.get("totalBatches")
        logger.debug(f"_get_last_validation() -> {steps_completed}")
        return steps_completed

    def set_metadata(self, metadata: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Set the metadata on the current run to the Determined master, overwrite
        existing metadata on the current run. Returns the metadata that was set.

        The metadata is a dictionary of key-value pairs that can be used for analysis,
        post-processing, or debugging.
        """
        logger.debug(f"set_metadata({metadata})")

        body = bindings.v1PostRunMetadataRequest(metadata=metadata, runId=self._trial_id)
        r = bindings.post_PostRunMetadata(
            session=self._session,
            body=body,
            runId=self._trial_id,
        )
        return r.metadata

    def get_metadata(self) -> Optional[Dict[str, Any]]:
        """
        Get the metadata of the current run from the Determined master.
        """
        r = bindings.get_GetRunMetadata(session=self._session, runId=self._trial_id)
        return r.metadata

    def _report_trial_metrics(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        """
        Report trial metrics to the master.

        You can include a list of ``batch_metrics``.  Batch metrics are not be shown in the WebUI
        but may be accessed from the master using the CLI for post-processing.
        """

        reportable_metrics = metrics
        if group == util._LEGACY_VALIDATION:
            # keep the old behavior of filtering out some metrics for validations.
            serializable_metrics = self._get_serializable_metrics(metrics)
            reportable_metrics = {k: metrics[k] for k in serializable_metrics}

        self._metrics.report(
            group=group,
            steps_completed=steps_completed,
            metrics=reportable_metrics,
            batch_metrics=batch_metrics,
        )

        # Also sync tensorboard (all metrics, not just json-serializable ones).
        if self._tensorboard_mode == core.TensorboardMode.AUTO:
            if self._tbd_writer:
                if group == util._LEGACY_TRAINING:
                    self._tbd_writer.on_train_step_end(steps_completed, metrics, batch_metrics)
                elif group == util._LEGACY_VALIDATION:
                    self._tbd_writer.on_validation_step_end(steps_completed, metrics)
            assert self._tensorboard_manager is not None
            self._tensorboard_manager.sync()

    def report_training_metrics(
        self,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        """
        Report training metrics to the master.

        You can include a list of ``batch_metrics``.  Batch metrics are not be shown in the WebUI
        but may be accessed from the master using the CLI for post-processing.
        """

        logger.info(
            f"report_training_metrics(steps_completed={steps_completed}, metrics={metrics})"
        )
        self._report_trial_metrics(util._LEGACY_TRAINING, steps_completed, metrics, batch_metrics)

    def report_validation_metrics(
        self,
        steps_completed: int,
        metrics: Dict[str, Any],
    ) -> None:
        """
        Report validation metrics to the master.
        Note that for hyperparameter search, this is independent of the need to report the searcher
        metric using ``SearcherOperation.report_completed()`` in the Searcher API.
        """

        logger.info(
            f"report_validation_metrics(steps_completed={steps_completed}, metrics={metrics})"
        )
        self._report_trial_metrics(util._LEGACY_VALIDATION, steps_completed, metrics)

    def report_metrics(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
    ) -> None:
        """
        Report metrics data to the master.

        Arguments:
            group (string): metrics group name. Can be used to partition metrics
                into different logical groups or time series.
                "training" and "validation" group names map to built-in training
                and validation time series. Note: Group cannot contain ``.`` character.
            steps_completed (int): global step number, e.g. the number of batches processed.
            metrics (Dict[str, Any]): metrics data dictionary. Must be JSON-serializable.
                When reporting metrics with the same ``group`` and ``steps_completed`` values,
                the dictionary keys must not overlap.
        """
        logger.info(
            f"report_metrics(group={group}, steps_completed={steps_completed}, metrics={metrics})"
        )
        self._report_trial_metrics(group, steps_completed, metrics)

    def get_tensorboard_path(self) -> pathlib.Path:
        """
        Get TensorBoard log directory path.
        """
        if self._tensorboard_manager is None:
            raise ValueError("tensorboard manager is required for this method")
        return self._tensorboard_manager.base_path

    def upload_tensorboard_files(
        self,
        selector: Callable[[pathlib.Path], bool] = lambda _: True,
        mangler: Callable[[pathlib.Path, int], pathlib.Path] = lambda p, __: p,
    ) -> None:
        """
        Upload files generated for consumption by Tensorboard to checkpoint storage.

        Args:
            selector: optional function returning True for a file that should be included.
                If not provided, all files are uploaded.
            mangler: optional function modifying the destination file names based on rank.
        """
        if self._tensorboard_mode == core.TensorboardMode.AUTO:
            raise RuntimeError("upload_tensorboard_files can only be used in MANUAL mode")

        if self._tensorboard_manager is None:
            raise ValueError("tensorboard manager is required for this method")
        assert self._tensorboard_manager is not None
        self._tensorboard_manager.sync(selector, mangler, self._distributed.rank)

    def _get_serializable_metrics(self, metrics: Dict[str, Any]) -> Set[str]:
        serializable_metrics = set()
        non_serializable_metrics = set()

        # In the case of trial implementation bugs, validation metric functions may return None.
        # Immediately fail any trial that encounters a None metric.
        for metric_name, metric_value in metrics.items():
            if metric_value is None:
                raise RuntimeError(
                    "Validation metric '{}' returned "
                    "an invalid scalar value: {}".format(metric_name, metric_value)
                )

            if isinstance(metric_value, (bytes, bytearray)):
                non_serializable_metrics.add(metric_name)
            else:
                serializable_metrics.add(metric_name)

        if len(non_serializable_metrics):
            logger.warning(
                "Removed non serializable metrics: %s", ", ".join(non_serializable_metrics)
            )

        return serializable_metrics

    def report_early_exit(self, reason: EarlyExitReason) -> None:
        """
        Report an early exit reason to the Determined master.

        Currenlty, the only meaningful value to report is ``EarlyExitReason.INVALID_HP``, which is
        reported automatically in ``core.Context.__exit__()`` detects an exception of type
        ``det.InvalidHP``.
        """

        body = {"reason": EarlyExitReason(reason).value}
        logger.info(f"report_early_exit({reason})")
        r = self._session.post(
            f"/api/v1/trials/{self._trial_id}/early_exit",
            data=det.util.json_encode(body),
        )
        if r.status_code == 400:
            logger.warn("early exit has already been reported for this trial, ignoring new value")

    def get_experiment_best_validation(self) -> Optional[float]:
        """
        Get the best reported validation metric reported so far, across the whole experiment.

        The returned value is the highest or lowest reported validation metric value, using the
        ``searcher.metric`` field of the experiment config as the key and
        ``searcher.smaller_is_better`` for the comparison.
        """

        logger.debug("get_experiment_best_validation()")
        try:
            r = self._session.get(
                f"/api/v1/experiments/{self._exp_id}/searcher/best_searcher_validation_metric"
            )
        except errors.NotFoundException:
            # 404 means 'no validations yet'.
            return None
        return float(r.json()["metric"])


class DummyTrainContext(TrainContext):
    def __init__(self, tensorboard_path: Optional[pathlib.Path] = None) -> None:
        self._tbd_directory = tensorboard_path

    def set_status(self, status: str) -> None:
        logger.info(f"status: {status}")

    def _get_last_validation(self) -> Optional[int]:
        return None

    def _report_trial_metrics(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        logger.debug(
            f"_report_trial_metrics(group={group}, steps_completed={steps_completed},"
            f"metrics={metrics}), batch_metrics={batch_metrics})"
        )

    def upload_tensorboard_files(
        self,
        selector: Callable[[pathlib.Path], bool] = lambda _: True,
        mangler: Callable[[pathlib.Path, int], pathlib.Path] = lambda p, __: p,
    ) -> None:
        logger.info("upload_tensorboard_files()")

    def report_early_exit(self, reason: EarlyExitReason) -> None:
        logger.info(f"report_early_exit({reason})")

    def get_experiment_best_validation(self) -> Optional[float]:
        return None

    def get_tensorboard_path(self) -> pathlib.Path:
        return self._tbd_directory  # type: ignore
