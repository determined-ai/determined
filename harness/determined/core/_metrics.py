import datetime
import logging
import queue
import threading
from typing import Any, Dict, List, Optional

from determined.common import api
from determined.common.api import bindings

logger = logging.getLogger("determined.core")


class _MetricsContext:
    """Gives access to metrics reporting during trial tasks.

    Metrics reported to ``_MetricsContext`` are published to a queue, which is consumed by a
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

        self._error_queue: queue.Queue = queue.Queue()
        self._shipper = _Shipper(
            session=self._session,
            trial_id=self._trial_id,
            run_id=self._run_id,
            error_queue=self._error_queue,
        )

    def report(
        self,
        group: str,
        metrics: Dict[str, Any],
        steps_completed: Optional[int] = None,
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
        report_time: Optional[datetime.datetime] = None,
    ) -> None:
        """Adds metrics to a queue to be reported.

        Before publishing to the queue, check for exceptions that might have occurred in the
        background reporting thread from a previous report and raise them.
        """
        self._maybe_raise_exception()
        self._shipper.publish_metrics(
            group=group,
            steps_completed=steps_completed,
            metrics=metrics,
            batch_metrics=batch_metrics,
            report_time=report_time,
        )

    def _maybe_raise_exception(self) -> None:
        """Check the error queue for exceptions and raise if there are any."""
        if not self._error_queue.empty():
            err_msg = self._error_queue.get(block=False)
            logger.error(f"Error reporting metrics: {err_msg}")
            raise err_msg

    def start(self) -> None:
        self._shipper.start()

    def close(self) -> None:
        self._maybe_raise_exception()
        self._shipper.stop()
        self._join_with_timeout()

    def _join_with_timeout(self) -> None:
        while not self._shipper._queue.empty():
            # If the queue isn't empty, wait for metric reporting and print logs intermittently.
            self._shipper.join(timeout=10)
            self._maybe_raise_exception()
            if self._shipper.is_alive():
                logger.info("Waiting for _Shipper thread to finish reporting metrics...")
            else:
                return

        # Metrics queue is empty, join with timeout to avoid hangs and add logs to help debug.
        self._shipper.join(timeout=1)
        if self._shipper.is_alive():
            logger.info("Waiting for _Shipper thread to finish...")
            self._shipper.join(timeout=5)
            if self._shipper.is_alive():
                logger.warning("Failed to complete _Shipper cleanup.")
            else:
                logger.info("_Shipper cleanup complete.")


class _TrialMetrics:
    def __init__(
        self,
        group: str,
        metrics: Dict[str, Any],
        steps_completed: Optional[int] = None,
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
        report_time: Optional[datetime.datetime] = None,
    ):
        self.group = group
        self.steps_completed = steps_completed
        self.metrics = metrics
        self.batch_metrics = batch_metrics
        self.report_time = report_time


class _Shipper(threading.Thread):
    METRICS_QUEUE_MAXSIZE = 1000

    def __init__(
        self,
        session: api.Session,
        trial_id: int,
        run_id: int,
        error_queue: queue.Queue,
    ) -> None:
        self._queue: queue.Queue = queue.Queue(maxsize=self.METRICS_QUEUE_MAXSIZE)
        self._error_queue = error_queue
        self._session = session
        self._trial_id = trial_id
        self._run_id = run_id

        super().__init__(daemon=True, name="MetricsShipperThread")

    def publish_metrics(
        self,
        group: str,
        metrics: Dict[str, Any],
        steps_completed: Optional[int] = None,
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
        report_time: Optional[datetime.datetime] = None,
    ) -> None:
        self._queue.put(
            _TrialMetrics(
                group=group,
                steps_completed=steps_completed,
                metrics=metrics,
                batch_metrics=batch_metrics,
                report_time=report_time,
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
                    report_time=msg.report_time,
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
        report_time: Optional[datetime.datetime] = None,
    ) -> None:
        v1metrics = bindings.v1Metrics(avgMetrics=metrics, batchMetrics=batch_metrics)
        v1TrialMetrics = bindings.v1TrialMetrics(
            metrics=v1metrics,
            trialId=self._trial_id,
            stepsCompleted=steps_completed,
            trialRunId=self._run_id,
            reportTime=report_time.isoformat() if report_time else None,
        )
        body = bindings.v1ReportTrialMetricsRequest(metrics=v1TrialMetrics, group=group)
        bindings.post_ReportTrialMetrics(self._session, body=body, metrics_trialId=self._trial_id)


class _DummyMetricsContext(_MetricsContext):
    def __init__(self) -> None:
        pass

    def report(
        self,
        group: str,
        metrics: Dict[str, Any],
        steps_completed: Optional[int] = None,
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
        report_time: Optional[datetime.datetime] = None,
    ) -> None:
        pass

    def start(self) -> None:
        pass

    def close(self) -> None:
        pass
