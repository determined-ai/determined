import abc
from typing import Any, Dict, List, Union

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
    def __init__(self, writer: MetricWriter, batches_per_step: int) -> None:
        self.writer = writer
        self.batches_per_step = batches_per_step

    def _maybe_write_metric(self, metric_key: str, metric_val: Any, step: int) -> None:
        # For now, we only log scalar metrics.
        if not util.is_numerical_scalar(metric_val):
            return

        if "/" in metric_key:
            return

        self.writer.add_scalar("Determined/" + metric_key, metric_val, step)

    def on_train_step_end(self, step_id: int, metrics: List[Dict[str, Any]]) -> None:
        if step_id <= 0:
            raise AssertionError(f"Expected step_id to be a positive int, but it is {step_id}")
        first_batch_in_step = (step_id - 1) * self.batches_per_step
        for batch_idx, batch_metrics in enumerate(metrics):
            batches_seen = first_batch_in_step + batch_idx
            for name, value in batch_metrics.items():
                self._maybe_write_metric(name, value, batches_seen)

        self.writer.reset()

    def on_validation_step_end(self, step_id: int, metrics: Dict[str, Any]) -> None:
        batches_seen = step_id * self.batches_per_step
        for name, value in metrics.items():
            if not name.startswith("val"):
                name = "val_" + name
            self._maybe_write_metric(name, value, batches_seen)

        self.writer.reset()
