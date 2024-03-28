import datetime
import logging
import queue
import threading
from typing import Any, Dict, List, Optional

from determined.common import api
from determined.common.api import bindings

logger = logging.getLogger("determined.core")

METRICS_QUEUE_MAX_SIZE = 1000


class MetricsContext:
    """
    ``MetricsContext`` gives access to metrics reporting during trial tasks.
    """

    def __init__(
        self,
        session: api.Session,
        trial_id: int,
        run_id: int,
    ) -> None:
        self._shipper = _MetricsShipper(session=session, trial_id=trial_id, run_id=run_id)

    def start(self) -> None:
        self._shipper.start()

    def close(self) -> None:
        self._shipper.stop()

    def publish(
        self,
        group: str,
        metrics: Dict[str, Any],
        steps_completed: Optional[int] = None,
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
        timestamp: Optional[datetime.datetime] = None,
    ) -> None:
        self._shipper.publish(
            group=group,
            steps_completed=steps_completed,
            metrics=metrics,
            batch_metrics=batch_metrics,
        )


class _TrialMetrics:
    def __init__(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
        timestamp: Optional[datetime.datetime] = None,
    ):
        self.group = group
        self.steps_completed = steps_completed
        self.metrics = metrics
        self.batch_metrics = batch_metrics
        self.timestamp = timestamp


class _MetricsShipper(threading.Thread):
    def __init__(
        self,
        session: api.Session,
        trial_id: int,
        run_id: int,
    ):
        self._queue = queue.Queue(maxsize=METRICS_QUEUE_MAX_SIZE)
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id
        super().__init__()

    def publish(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
        timestamp: Optional[datetime.datetime] = None,
    ) -> None:
        self._queue.put(
            _TrialMetrics(
                group=group,
                steps_completed=steps_completed,
                metrics=metrics,
                batch_metrics=batch_metrics,
                timestamp=timestamp,
            )
        )

    def run(self) -> None:
        """Start the thread and ship metrics in queue to master."""
        while True:
            msg = self._queue.get()
            if msg is None:
                # Received shutdown message, exit.
                return
            self._report(metrics=msg)

    def _report(self, metrics: _TrialMetrics) -> None:
        v1metrics = bindings.v1Metrics(
            avgMetrics=metrics.metrics, batchMetrics=metrics.batch_metrics
        )
        v1TrialMetrics = bindings.v1TrialMetrics(
            metrics=v1metrics,
            stepsCompleted=metrics.steps_completed,
            trialId=self._trial_id,
            trialRunId=self._run_id,
            reportTime=metrics.timestamp,
        )
        body = bindings.v1ReportTrialMetricsRequest(metrics=v1TrialMetrics, group=metrics.group)
        bindings.post_ReportTrialMetrics(self._session, body=body, metrics_trialId=self._trial_id)

    def stop(self) -> None:
        self._queue.put(None)
        self.join()


class DummyMetricsContext(MetricsContext):
    def __init__(self):
        pass

    def publish(
        self,
        group: str,
        metrics: Dict[str, Any],
        steps_completed: Optional[int] = None,
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
        timestamp: Optional[datetime.datetime] = None,
    ) -> None:
        pass
