import dataclasses
import datetime
from typing import Any, Dict, List, Optional, cast

from determined.common import util
from determined.common.api import bindings


@dataclasses.dataclass
class TrialMetrics:
    """
    Specifies a metric that the trial reported.

    Attributes:
        trial_id
        trial_run_id
        total_batches
        end_time
        metrics
        batch_metrics
    """

    trial_id: int
    trial_run_id: int
    total_batches: int
    end_time: datetime.datetime
    metrics: Dict[str, Any]
    batch_metrics: Optional[List[Dict[str, Any]]] = None

    @classmethod
    def _from_bindings(cls, metric_report: bindings.v1MetricsReport, group: str) -> "TrialMetrics":
        key = "validation_metrics" if group == util._LEGACY_VALIDATION else "avg_metrics"
        return cls(
            trial_id=metric_report.trialId,
            trial_run_id=metric_report.trialRunId,
            total_batches=metric_report.totalBatches,
            end_time=util.parse_protobuf_timestamp(metric_report.endTime),
            metrics=metric_report.metrics[key],
            batch_metrics=metric_report.metrics.get("batch_metrics", None),
        )

    @property
    def steps_completed(self) -> int:
        """@deprecated: Use total_batches instead."""
        return self.total_batches

    @steps_completed.setter
    def steps_completed(self, value: int) -> None:
        self.total_batches = value


class TrainingMetrics(TrialMetrics):
    """
    Specifies a training metric report that the trial reported.
    """

    def __init__(self, steps_completed: Optional[int] = None, **kwargs: Any):
        if steps_completed is not None:
            kwargs["total_batches"] = steps_completed
        super().__init__(**kwargs)

    @classmethod
    def _from_bindings(  # type: ignore
        cls,
        metric_report: bindings.v1MetricsReport,
    ) -> "TrainingMetrics":
        return cast("TrainingMetrics", super()._from_bindings(metric_report, util._LEGACY_TRAINING))


class ValidationMetrics(TrialMetrics):
    """
    Specifies a validation metric report that the trial reported.
    """

    def __init__(self, steps_completed: Optional[int] = None, **kwargs: Any):
        if steps_completed is not None:
            kwargs["total_batches"] = steps_completed
        super().__init__(**kwargs)

    @classmethod
    def _from_bindings(  # type: ignore
        cls, metric_report: bindings.v1MetricsReport
    ) -> "ValidationMetrics":
        return cast(
            "ValidationMetrics", super()._from_bindings(metric_report, util._LEGACY_VALIDATION)
        )


class InferenceMetrics(TrialMetrics):
    """
    Specifies a validation metric report that the trial reported.
    """

    def __init__(self, steps_completed: Optional[int] = None, **kwargs: Any):
        if steps_completed is not None:
            kwargs["total_batches"] = steps_completed
        super().__init__(**kwargs)

    @classmethod
    def _from_bindings(  # type: ignore
        cls, metric_report: bindings.v1MetricsReport
    ) -> "InferenceMetrics":
        return cast("InferenceMetrics", super()._from_bindings(metric_report, util._INFERENCE))
