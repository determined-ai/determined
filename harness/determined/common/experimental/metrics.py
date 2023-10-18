import dataclasses
import datetime
from typing import Any, Dict, List, Optional, cast

from determined.common import util
from determined.common.api import bindings


@dataclasses.dataclass
class TrialMetrics:
    """Specifies collection of metrics that the trial reported.

    Attributes:
        trial_id: The ID of the trial that reported the metric.
        trial_run_id: The ID of the trial run that reported the metric.
        steps_completed: The number of steps that the trial had completed when the metric was
            reported. Most generally, the value passed to a call to report_metrics as
            "steps_completed."
        end_time: The time when the metric was reported.
        metrics: A dict of metrics that the trial reported.
        group: The group that the metric was reported under. Usually either "validation" or
            "training", but this can be any value passed to master when reporting metrics during
            training (usually via a context's `report_metrics`).
        batch_metrics: <do not use>
    """

    trial_id: int
    trial_run_id: int
    steps_completed: int
    end_time: datetime.datetime
    metrics: Dict[str, Any]
    group: str
    batch_metrics: Optional[List[Dict[str, Any]]] = None

    @classmethod
    def _from_bindings(
        cls, metric_report: bindings.v1MetricsReport, group: Optional[str]
    ) -> "TrialMetrics":
        key = "validation_metrics" if group == util._LEGACY_VALIDATION else "avg_metrics"
        return cls(
            trial_id=metric_report.trialId,
            trial_run_id=metric_report.trialRunId,
            steps_completed=metric_report.totalBatches,
            end_time=util.parse_protobuf_timestamp(metric_report.endTime),
            metrics=metric_report.metrics[key],
            batch_metrics=metric_report.metrics.get("batch_metrics", None),
            group=metric_report.group,
        )

    @property
    def total_batches(self) -> int:
        """@deprecated: Use steps_completed instead."""
        return self.steps_completed

    @total_batches.setter
    def total_batches(self, value: int) -> None:
        """@deprecated: Use steps_completed instead."""
        self.steps_completed = value


class TrainingMetrics(TrialMetrics):
    """
    @deprecated: Use TrialMetrics instead.

    Specifies a training metric report that the trial reported.
    """

    def __init__(self, total_batches: Optional[int] = None, **kwargs: Any):
        if total_batches is not None:
            kwargs["steps_completed"] = total_batches
        kwargs["group"] = util._LEGACY_TRAINING
        super().__init__(**kwargs)

    @classmethod
    def _from_bindings(  # type: ignore
        cls,
        metric_report: bindings.v1MetricsReport,
    ) -> "TrainingMetrics":
        return cast("TrainingMetrics", super()._from_bindings(metric_report, util._LEGACY_TRAINING))


class ValidationMetrics(TrialMetrics):
    """
    @deprecated: Use TrialMetrics instead.

    Specifies a validation metric report that the trial reported.
    """

    def __init__(self, total_batches: Optional[int] = None, **kwargs: Any):
        if total_batches is not None:
            kwargs["steps_completed"] = total_batches
        kwargs["group"] = util._LEGACY_VALIDATION
        super().__init__(**kwargs)

    @classmethod
    def _from_bindings(  # type: ignore
        cls, metric_report: bindings.v1MetricsReport
    ) -> "ValidationMetrics":
        return cast(
            "ValidationMetrics", super()._from_bindings(metric_report, util._LEGACY_VALIDATION)
        )
