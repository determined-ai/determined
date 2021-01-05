import logging
from typing import Any, Optional

from determined import tensorboard, workload
from determined.tensorboard.metric_writers import util


class TensorboardLayer(workload.Source):
    """TensorboardLayer coordinates synchronizing data to tensorboard after workloads complete."""

    def __init__(
        self,
        workloads: workload.Stream,
        tensorboard_mgr: tensorboard.TensorboardManager,
        is_chief: bool,
        writer: Optional[tensorboard.MetricWriter] = None,
    ) -> None:
        self.workloads = workloads
        self.tensorboard_mgr = tensorboard_mgr
        self.is_chief = is_chief

        if writer is None:
            try:
                from determined.tensorboard.metric_writers import tensorflow

                writer = tensorflow.TFWriter()

            except ModuleNotFoundError:
                logging.warning("Tensorflow writer not found")
                from determined.tensorboard.metric_writers import pytorch

                writer = pytorch.TorchWriter()

        self.writer = writer

    def __iter__(self) -> workload.Stream:
        for wkld, args, response_func in self.workloads:
            # Only the chief container synchronizes tensorboard.
            if not self.is_chief:
                yield wkld, args, response_func
                continue

            def _respond(in_response: workload.Response) -> None:
                if isinstance(in_response, dict):
                    if wkld.kind == workload.Workload.Kind.RUN_STEP:
                        self.upload_training_metrics(wkld, in_response)
                    elif wkld.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                        self.upload_validation_metrics(wkld, in_response)
                    # All non-terminate messages from the chief machine should sync tensorboard.
                    # (TERMINATE will result in a workload.Skipped response)
                    self.tensorboard_mgr.sync()
                response_func(in_response)

            yield wkld, args, response_func

    def _maybe_write_metric(self, metric_key: str, metric_val: Any, batches_seen: int) -> None:
        # For now, we only log scalar metrics.
        if not util.is_numerical_scalar(metric_val):
            return

        self.writer.add_scalar("Determined/" + metric_key, metric_val, batches_seen)

    def upload_training_metrics(self, wkld: workload.Workload, metrics: workload.Metrics) -> None:
        if wkld.step_id <= 0:
            raise AssertionError(f"Expected step_id to be a positive int, but it is {wkld.step_id}")

        metrics_seen = set()

        # Log all batch metrics.
        for batch_idx, batch_metrics in enumerate(metrics["batch_metrics"]):
            batches_seen = wkld.total_batches_processed + batch_idx
            for name, value in batch_metrics.items():
                self._maybe_write_metric(name, value, batches_seen)
                metrics_seen.add(name)

        # Log average metrics which were calculated via a custom reducer and not in batch metrics.
        batches_seen = wkld.total_batches_processed + wkld.num_batches
        for name, value in metrics["avg_metrics"].items():
            if name in metrics_seen:
                continue
            self._maybe_write_metric(name, value, batches_seen)

        self.writer.reset()

    def upload_validation_metrics(self, wkld: workload.Workload, metrics: workload.Metrics) -> None:
        for name, value in metrics["validation_metrics"].items():
            if not name.startswith("val"):
                name = "val_" + name
            self._maybe_write_metric(name, value, wkld.total_batches_processed)

        self.writer.reset()
