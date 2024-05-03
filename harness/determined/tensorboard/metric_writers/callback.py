import abc
import logging
import time
from typing import TYPE_CHECKING, Any, Dict, List, Optional, Union

from determined import util

logger = logging.getLogger("determined.tensorboard")

if TYPE_CHECKING:
    import numpy as np


class MetricWriter(abc.ABC):
    @abc.abstractmethod
    def add_scalar(self, name: str, value: Union[int, float, "np.number"], step: int) -> None:
        pass

    @abc.abstractmethod
    def reset(self) -> None:
        pass

    @abc.abstractmethod
    def flush(self) -> None:
        pass


class BatchMetricWriter:
    def __init__(self, writer: MetricWriter) -> None:
        self.writer = writer
        self._last_flush_ts: Optional[float] = None

    def _maybe_write_metric(self, metric_key: str, metric_val: Any, step: int) -> None:
        # For now, we only log scalar metrics.
        if not util.is_numerical_scalar(metric_val):
            return

        self.writer.add_scalar("Determined/" + metric_key, metric_val, step)

    def on_train_step_end(
        self,
        steps_completed: int,
        metrics: Dict[str, Any],
        batch_metrics: Optional[List[Dict[str, Any]]] = None,
    ) -> None:
        logger.debug("Write training metrics for TensorBoard")
        metrics_seen = set()

        self._maybe_reset()

        # Log all batch metrics.
        if batch_metrics:
            for batch_idx, batch in enumerate(batch_metrics):
                batches_seen = steps_completed - len(batch_metrics) + batch_idx
                for name, value in batch.items():
                    self._maybe_write_metric(name, value, batches_seen)
                    metrics_seen.add(name)

        # Log avg metrics which were calculated by a custom reducer and are not in batch metrics.
        for name, value in metrics.items():
            if name in metrics_seen:
                continue
            self._maybe_write_metric(name, value, steps_completed)

        self.writer.flush()
        self._last_flush_ts = time.time()

    def on_validation_step_end(self, steps_completed: int, metrics: Dict[str, Any]) -> None:
        logger.debug("Write validation metrics for TensorBoard")

        self._maybe_reset()

        for name, value in metrics.items():
            if not name.startswith("val"):
                name = "val_" + name
            self._maybe_write_metric(name, value, steps_completed)

        self.writer.flush()
        self._last_flush_ts = time.time()

    def _maybe_reset(self) -> None:
        """
        Reset (close current file and open a new one) the current writer if the current epoch
        second is at least one second greater than the epoch second of the last reset.

        The TensorFlow event writer names each event file by the epoch second it is created, so
        if events are written quickly in succession (< 1 second apart), they will overwrite each
        other.

        This effectively batches event writes so each event file may contain more than one event.
        """
        current_ts = time.time()
        if not self._last_flush_ts:
            return

        if int(current_ts) > int(self._last_flush_ts):
            self.writer.reset()
