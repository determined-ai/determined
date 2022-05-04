import abc
from typing import TYPE_CHECKING, Any, Dict, List, Optional, Union

from determined.tensorboard.metric_writers import util

if TYPE_CHECKING:
    import numpy as np


class MetricWriter(abc.ABC):
    @abc.abstractmethod
    def add_scalar(self, name: str, value: Union[int, float, "np.number"], step: int) -> None:
        pass

    @abc.abstractmethod
    def reset(self) -> None:
        pass


class BatchMetricWriter:
    def __init__(self, writer: MetricWriter) -> None:
        self.writer = writer

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
        metrics_seen = set()

        # Log all batch metrics.
        if batch_metrics:
            for batch_idx, batch in enumerate(batch_metrics):
                batches_seen = steps_completed - len(batch) + batch_idx
                for name, value in batch.items():
                    self._maybe_write_metric(name, value, batches_seen)
                    metrics_seen.add(name)

        # Log avg metrics which were calculated by a custom reducer and are not in batch metrics.
        for name, value in metrics.items():
            if name in metrics_seen:
                continue
            self._maybe_write_metric(name, value, steps_completed)

        self.writer.reset()

    def on_validation_step_end(self, steps_completed: int, metrics: Dict[str, Any]) -> None:
        for name, value in metrics.items():
            if not name.startswith("val"):
                name = "val_" + name
            self._maybe_write_metric(name, value, steps_completed)

        self.writer.reset()
