import enum
import logging
from typing import Any, Dict, List, Optional, Set

import determined as det
from determined import tensorboard
from determined.common.api import errors
from determined.common.experimental.session import Session

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
        session: Session,
        trial_id: int,
        run_id: int,
        exp_id: int,
        tbd_mgr: Optional[tensorboard.TensorboardManager],
        tbd_writer: Optional[tensorboard.BatchMetricWriter],
    ) -> None:
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id
        self._exp_id = exp_id
        self._tbd_mgr = tbd_mgr
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

        body = {
            "trial_run_id": self._run_id,
            "steps_completed": steps_completed,
            "metrics": metrics,
        }
        if batch_metrics is not None:
            body["batch_metrics"] = batch_metrics
        logger.info(
            f"report_training_metrics(steps_completed={steps_completed}, metrics={metrics})"
        )
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/training_metrics",
            data=det.util.json_encode(body),
        )

        # Also sync tensorboard.
        if self._tbd_writer and self._tbd_mgr:
            self._tbd_writer.on_train_step_end(steps_completed, metrics, batch_metrics)
            self._tbd_mgr.sync()

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

        serializable_metrics = self._get_serializable_metrics(metrics)
        reportable_metrics = {k: metrics[k] for k in serializable_metrics}

        body = {
            "trial_run_id": self._run_id,
            "steps_completed": steps_completed,
            "metrics": reportable_metrics,
        }
        logger.info(
            f"report_validation_metrics(steps_completed={steps_completed}, metrics={metrics})"
        )
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/validation_metrics",
            data=det.util.json_encode(body),
        )

        # Also sync tensorboard (all metrics, not just json-serializable ones).
        if self._tbd_writer and self._tbd_mgr:
            self._tbd_writer.on_validation_step_end(steps_completed, metrics)
            self._tbd_mgr.sync()

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
    def __init__(self) -> None:
        pass

    def set_status(self, status: str) -> None:
        logger.info(f"status: {status}")

    def _get_last_validation(self) -> Optional[int]:
        return None

    def report_training_metrics(
        self,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        logger.info(
            f"report_training_metrics(steps_completed={steps_completed}, metrics={metrics})"
        )
        logger.debug(
            f"report_training_metrics(steps_completed={steps_completed},"
            f" batch_metrics={batch_metrics})"
        )

    def report_validation_metrics(self, steps_completed: int, metrics: Dict[str, Any]) -> None:
        serializable_metrics = self._get_serializable_metrics(metrics)
        metrics = {k: metrics[k] for k in serializable_metrics}
        logger.info(
            f"report_validation_metrics(steps_completed={steps_completed} metrics={metrics})"
        )

    def report_early_exit(self, reason: EarlyExitReason) -> None:
        logger.info(f"report_early_exit({reason})")

    def get_experiment_best_validation(self) -> Optional[float]:
        return None
