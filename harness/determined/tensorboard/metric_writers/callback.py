import abc
from typing import Any, Dict, Union

import numpy as np

from determined import callback
from determined.tensorboard.metric_writers import util


class MetricWriter(abc.ABC):
    @abc.abstractmethod
    def add_scalar(self, name: str, value: Union[int, float, np.number], step: int) -> None:
        pass

    @abc.abstractmethod
    def reset(self) -> None:
        pass


class BatchMetricWriter(callback.Callback):
    def __init__(self, writer: MetricWriter) -> None:
        self.writer = writer

    def _maybe_write_metric(self, metric_key: str, metric_val: Any, step: int) -> None:
        # For now, we only log scalar metrics.
        if not util.is_numerical_scalar(metric_val):
            return

        self.writer.add_scalar("Determined/" + metric_key, metric_val, step)

    def on_train_step_end(
        self,
        step_id: int,
        num_batches: int,
        total_batches_processed: int,
        metrics: Dict[str, Any],
    ) -> None:
        if step_id <= 0:
            raise AssertionError(f"Expected step_id to be a positive int, but it is {step_id}")
        metrics_seen = set()

        # Log all batch metrics.
        for batch_idx, batch_metrics in enumerate(metrics["batch_metrics"]):
            batches_seen = total_batches_processed + batch_idx
            for name, value in batch_metrics.items():
                self._maybe_write_metric(name, value, batches_seen)
                metrics_seen.add(name)

        # Log avg metrics which were calculated by a custom reducer and are not in batch metrics.
        batches_seen = total_batches_processed + num_batches
        for name, value in metrics["avg_metrics"].items():
            if name in metrics_seen:
                continue
            self._maybe_write_metric(name, value, batches_seen)

        self.writer.reset()

    def on_validation_step_end(
        self, step_id: int, total_batches_processed: int, metrics: Dict[str, Any]
    ) -> None:
        for name, value in metrics.items():
            if not name.startswith("val"):
                name = "val_" + name
            self._maybe_write_metric(name, value, total_batches_processed)

        self.writer.reset()
