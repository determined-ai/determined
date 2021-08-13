import enum
import logging
import math
from typing import Any, Dict, List, Optional

import determined as det
from determined import tensorboard
from determined.common.api import errors
from determined.common.experimental.session import Session

logger = logging.getLogger("determined.generic")


class EarlyExitReason(enum.Enum):
    INVALID_HP = "EXITED_REASON_INVALID_HP"
    # This is generally unnecessary; just exit early.
    USER_REQUESTED_STOP = "EXITED_REASON_USER_REQUESTED_STOP"


class Training:
    """
    Some training-related REST API wrappers.
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
        body = {"state": status}
        logger.debug(f"set_status({status})")
        self._session.post(f"/api/v1/trials/{self._trial_id}/runner/metadata", json=body)

    def get_last_validation(self) -> Optional[int]:
        r = self._session.get(f"/api/v1/trials/{self._trial_id}")
        val = r.json()["trial"].get("latestValidation") or {}
        latest_batch = val.get("totalBatches")
        logger.debug(f"get_last_validation() -> {latest_batch}")
        return latest_batch

    def report_training_metrics(
        self,
        latest_batch: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        body = {
            "trial_run_id": self._run_id,
            "latest_batch": latest_batch,
            "metrics": metrics,
        }
        if batch_metrics is not None:
            body["batch_metrics"] = batch_metrics
        logger.info(f"report_training_metrics(latest_batch={latest_batch}, metrics={metrics})")
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/training_metrics",
            data=det.util.json_encode(body),
        )

        # Also sync tensorboard.
        if self._tbd_writer and self._tbd_mgr:
            self._tbd_writer.on_train_step_end(latest_batch, metrics, batch_metrics)
            self._tbd_mgr.sync()

    def report_validation_metrics(
        self,
        latest_batch: int,
        metrics: Dict[str, Any],
    ) -> None:
        serializable_metrics = set()
        non_serializable_metrics = set()
        # NaN and bytes are not JSON serializable.  In the case of trial implementation bugs or
        # numerical instability issues, validation metric functions may return None or NaN values.
        # For now, immediately fail any trial that encounters such a None metric. For NaN metrics,
        # if it's the target of the searcher, we set it to +/- max_float depending on if the
        # searcher is optimizing for the max or min. NaN metrics which are not the target of the
        # searcher are dropped.
        # TODO (DET-2495): Do not replace NaN metric values.
        for metric_name, metric_value in metrics.items():
            metric_is_none = metric_value is None
            metric_is_nan = tensorboard.metric_writers.util.is_numerical_scalar(
                metric_value
            ) and math.isnan(metric_value)

            if metric_is_none or metric_is_nan:
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
        reportable_metrics = {k: metrics[k] for k in serializable_metrics}

        body = {
            "trial_run_id": self._run_id,
            "latest_batch": latest_batch,
            "metrics": reportable_metrics,
        }
        logger.info(f"report_validation_metrics(latest_batch={latest_batch}, metrics={metrics})")
        self._session.post(
            f"/api/v1/trials/{self._trial_id}/validation_metrics",
            data=det.util.json_encode(body),
        )

        # Also sync tensorboard (all metrics, not just json-serializable ones).
        if self._tbd_writer and self._tbd_mgr:
            self._tbd_writer.on_validation_step_end(latest_batch, metrics)
            self._tbd_mgr.sync()

    def report_early_exit(self, reason: EarlyExitReason) -> None:
        body = {"reason": EarlyExitReason(reason).value}
        logger.info(f"report_early_exit({reason})")
        r = self._session.post(
            f"/api/v1/trials/{self._trial_id}/early_exit",
            data=det.util.json_encode(body),
        )
        if r.status_code == 400:
            logger.warn("early exit has already been reported for this trial, ignoring new value")

    def get_experiment_best_validation(self) -> Optional[float]:
        logger.debug("get_experiment_best_validation()")
        try:
            r = self._session.get(
                f"/api/v1/experiments/{self._exp_id}/searcher/best_searcher_validation_metric"
            )
        except errors.NotFoundException:
            # 404 means 'no validations yet'.
            return None
        return float(r.json()["metric"])
