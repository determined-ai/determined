import logging
import queue
import threading
from typing import Any, Dict, List, Optional

from determined.common import api
from determined.common.api import bindings

logger = logging.getLogger("determined.core")


class MetricsContext:
    """Gives access to metrics reporting during trial tasks.

    Metrics reported to ``MetricsContext`` are published to a queue, which is consumed by a
    background thread that reports them to the master.
    """

    def __init__(
        self,
        session: api.Session,
        trial_id: int,
        run_id: int,
    ) -> None:
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id

        self._error_queue = queue.Queue()
        self._shipper = _Shipper(
            session=self._session,
            trial_id=self._trial_id,
            run_id=self._run_id,
            error_queue=self._error_queue,
        )

    def report(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        # Check for thread exceptions here since we're not polling.
        if not self._error_queue.empty():
            err_msg = self._error_queue.get(block=False)
            logger.error(f"Error reporting metrics: {err_msg}")
            raise err_msg

        self._shipper.publish_metrics(
            group=group,
            steps_completed=steps_completed,
            metrics=metrics,
            batch_metrics=batch_metrics,
        )

    def start(self) -> None:
        self._shipper.start()

    def close(self) -> None:
        self._shipper.stop()
        self._shipper.join()


class _TrialMetrics:
    def __init__(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ):
        self.group = group
        self.steps_completed = steps_completed
        self.metrics = metrics
        self.batch_metrics = batch_metrics


class _Shipper(threading.Thread):
    METRICS_QUEUE_MAXSIZE = 1000

    def __init__(
        self,
        session: api.Session,
        trial_id: int,
        run_id: int,
        error_queue: queue.Queue,
    ):
        self._queue = queue.Queue(maxsize=self.METRICS_QUEUE_MAXSIZE)
        self._error_queue = error_queue
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id

        super().__init__()

    def publish_metrics(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ):
        self._queue.put(
            _TrialMetrics(
                group=group,
                steps_completed=steps_completed,
                metrics=metrics,
                batch_metrics=batch_metrics,
            )
        )

    def run(self) -> None:
        """Start the thread and ship metrics in queue to master."""
        try:
            while True:
                msg = self._queue.get()
                if msg is None:
                    # Received shutdown message, exit.
                    return
                self._post_metrics(
                    group=msg.group,
                    metrics=msg.metrics,
                    batch_metrics=msg.batch_metrics,
                    steps_completed=msg.steps_completed,
                )
        except Exception as e:
            self._error_queue.put(e)

    def stop(self) -> None:
        self._queue.put(None)

    def _post_metrics(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ):
        v1metrics = bindings.v1Metrics(avgMetrics=metrics, batchMetrics=batch_metrics)
        v1TrialMetrics = bindings.v1TrialMetrics(
            metrics=v1metrics,
            trialId=self._trial_id,
            stepsCompleted=steps_completed,
            trialRunId=self._run_id,
        )
        body = bindings.v1ReportTrialMetricsRequest(metrics=v1TrialMetrics, group=group)
        bindings.post_ReportTrialMetrics(self._session, body=body, metrics_trialId=self._trial_id)


class DummyMetricsContext(MetricsContext):
    def __init__(self):
        pass

    def report(
        self,
        group: str,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        pass

    def start(self) -> None:
        pass

    def close(self) -> None:
        pass
