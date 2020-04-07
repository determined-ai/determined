import logging
import math
import pathlib
import sys
from datetime import datetime, timezone
from typing import List, Optional, cast

import determined as det
from determined import tensorboard, workload
from determined_common import storage
from determined_common.check import check_eq, check_len, check_not_eq, check_not_isinstance


def _current_timestamp() -> datetime:
    """Returns the current time as a datetime object in the UTC timezone."""
    return datetime.now(timezone.utc)


class WorkloadManager(workload.Source):
    """
    WorkloadManager handles workload messages after they are received on the
    WebSocket. Each WorkloadManager may allow different workload messages.
    """

    def __init__(
        self,
        env: det.EnvContext,
        workloads: workload.Stream,
        rendezvous_info: det.RendezvousInfo,
        storage_mgr: storage.StorageManager,
        tensorboard_mgr: tensorboard.TensorboardManager,
        metric_writer: tensorboard.BatchMetricWriter,
    ) -> None:
        self.env = env
        self.workloads = workloads
        self.rendezvous_info = rendezvous_info
        self.storage_mgr = storage_mgr
        self.tensorboard_mgr = tensorboard_mgr
        self.callbacks = [metric_writer]  # type: List[det.callback.Callback]


def build_workload_manager(
    env: det.EnvContext,
    workloads: workload.Stream,
    rendezvous_info: det.RendezvousInfo,
    storage_mgr: storage.StorageManager,
    tensorboard_mgr: tensorboard.TensorboardManager,
    metric_writer: tensorboard.BatchMetricWriter,
) -> WorkloadManager:
    """
    Build the WorkloadManager as specified by the container environment.
    """
    if env.workload_manager_type == "TRIAL_WORKLOAD_MANAGER":
        return _TrialWorkloadManager(
            env, workloads, rendezvous_info, storage_mgr, tensorboard_mgr, metric_writer
        )
    raise ValueError("Unexpected workload manager type: {}", env.workload_manager_type)


class _TrialWorkloadManager(WorkloadManager):
    def __init__(
        self,
        env: det.EnvContext,
        workloads: workload.Stream,
        rendezvous_info: det.RendezvousInfo,
        storage_mgr: storage.StorageManager,
        tensorboard_mgr: tensorboard.TensorboardManager,
        metric_writer: tensorboard.BatchMetricWriter,
    ) -> None:
        super().__init__(
            env, workloads, rendezvous_info, storage_mgr, tensorboard_mgr, metric_writer,
        )
        self.workload = None  # type: Optional[workload.Workload]

    def __iter__(self) -> workload.Stream:
        for w, _, response_func in self.workloads:
            if self.rendezvous_info.get_rank() == 0:
                logging.info("Running workload {}".format(w))
            else:
                logging.debug("Running workload {}".format(w))
            self.check_sane_workload(w)

            self.workload = w

            if w.kind == workload.Workload.Kind.RUN_STEP:
                yield from self.yield_train_for_step(w, response_func)
            elif w.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                yield from self.yield_compute_validation_metrics(w, response_func)
            elif w.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                yield from self.yield_checkpoint_model(w, response_func)
            elif w.kind == workload.Workload.Kind.TERMINATE:
                yield from self.yield_terminate(w, response_func)
            else:
                raise AssertionError("Unexpected workload: {}".format(w.kind))

    def check_sane_workload(self, new_workload: workload.Workload) -> None:
        # If this is the initial workload, we don't expect to start with
        # a checkpoint operation. All other workloads are reasonable.
        if self.workload is None:
            check_not_eq(new_workload.kind, workload.Workload.Kind.CHECKPOINT_MODEL)
            return

        # If this is not the initial workload, it should be compatible
        # with the previous workload that ran in this container.
        check_eq(self.workload.trial_id, new_workload.trial_id)

        if new_workload.kind == workload.Workload.Kind.RUN_STEP:
            check_eq(self.workload.step_id + 1, new_workload.step_id)
        else:
            check_eq(self.workload.step_id, new_workload.step_id)

    def yield_train_for_step(
        self, wkld: workload.Workload, respond: workload.ResponseFunc
    ) -> workload.Stream:
        start_time = _current_timestamp()

        for callback in self.callbacks:
            if wkld.step_id == 1:
                callback.on_trial_begin()
            callback.on_train_step_begin(wkld.step_id)

        def _respond(metrics: workload.Response) -> None:

            # Only the chief container should actually respond to TRAIN_FOR_STEP.
            if self.rendezvous_info.get_rank() != 0:
                respond(workload.Skipped())
                return

            check_not_isinstance(metrics, workload.Skipped, "Chief skipped a workload.")
            metrics = cast(workload.Metrics, metrics)

            batch_metrics = metrics["batch_metrics"]

            # Sanity-check training metrics.
            det.util.validate_batch_metrics(batch_metrics)
            check_len(batch_metrics, num_batches)

            for callback in self.callbacks:
                callback.on_train_step_end(wkld.step_id, batch_metrics)

            self.tensorboard_mgr.sync()

            # Send the response up.
            respond(
                {
                    "type": "WORKLOAD_COMPLETED",
                    "workload": wkld,
                    "start_time": start_time,
                    "end_time": _current_timestamp(),
                    "metrics": metrics,
                }
            )

        num_batches = self.env.experiment_config.get("batches_per_step", 100)
        yield wkld, [num_batches], _respond

    def yield_compute_validation_metrics(
        self, wkld: workload.Workload, respond: workload.ResponseFunc
    ) -> workload.Stream:
        start_time = _current_timestamp()

        def _respond(metrics: workload.Response) -> None:

            # Only the chief container should actually respond to COMPUTE_VALIDATION_METRICS.
            if self.rendezvous_info.get_rank() != 0:
                respond(workload.Skipped())
                return

            check_not_isinstance(metrics, workload.Skipped, "Chief skipped a workload.")
            metrics = cast(workload.Metrics, metrics)

            v_metrics = metrics["validation_metrics"]
            for callback in self.callbacks:
                callback.on_validation_step_end(wkld.step_id, v_metrics)

            self.tensorboard_mgr.sync()

            # Check that the validation metrics computed by the model code
            # includes the metric used by the search method.
            searcher_metric = self.env.experiment_config["searcher"]["metric"]
            if searcher_metric not in v_metrics:
                raise AssertionError(
                    "Search method is configured to use metric '{}' but model "
                    "definition returned validation metrics {}. The metric "
                    "used by the search method must be one of the validation "
                    "metrics returned by the model definition.".format(
                        searcher_metric, list(v_metrics.keys())
                    )
                )
                sys.exit(1)

            non_serializable_metrics = set()
            # NaN and bytes are not JSON serializable. None does not have a
            # canonical JSON representation. In the case of trial implementation bugs
            # or numerical instability issues, validation metric functions may
            # return None or NaN values. For now, immediately fail any trial that
            # encounters such a None metric. For NaN metrics, if it's the target of
            # the searcher, we set it to +/- max_float depending on if the searcher
            # is optimizing for the max or min. NaN metrics which are not the
            # target of the searcher are dropped.
            # TODO (DET-2495): Do not replace NaN metric values.
            for metric_name, metric_value in v_metrics.items():
                metric_is_none = metric_value is None
                metric_is_nan = tensorboard.metric_writers.util.is_numerical_scalar(
                    metric_value
                ) and math.isnan(metric_value)

                if metric_is_none:
                    raise AssertionError(
                        "Validation metric '{}' returned "
                        "an invalid scalar value: {}".format(metric_name, metric_value)
                    )
                    sys.exit(1)

                if metric_is_nan:
                    if metric_name == searcher_metric:
                        v_metrics[metric_name] = (
                            sys.float_info.max
                            if self.env.experiment_config["searcher"]["smaller_is_better"]
                            else sys.float_info.min
                        )
                        logging.warning(
                            f"Changed metrics {metric_name} from NaN to {v_metrics[metric_name]}."
                        )
                    else:
                        non_serializable_metrics.add(metric_name)

                if isinstance(metric_value, (bytes, bytearray)):
                    non_serializable_metrics.add(metric_name)

            if len(non_serializable_metrics):
                logging.warning(
                    "Removed non serializable metrics: %s", ", ".join(non_serializable_metrics)
                )
                for metric_name in non_serializable_metrics:
                    del v_metrics[metric_name]

            respond(
                {
                    "type": "WORKLOAD_COMPLETED",
                    "workload": wkld,
                    "start_time": start_time,
                    "end_time": _current_timestamp(),
                    "metrics": metrics,
                }
            )

        for callback in self.callbacks:
            callback.on_validation_step_begin(wkld.step_id)

        yield wkld, [], _respond

    def yield_checkpoint_model(
        self, wkld: workload.Workload, respond: workload.ResponseFunc
    ) -> workload.Stream:
        start_time = _current_timestamp()

        # Only the chief container should checkpoint.
        if self.rendezvous_info.get_rank() == 0:
            with self.storage_mgr.store_path() as (storage_id, path):
                yield wkld, [pathlib.Path(path)], lambda _: None

                metadata = storage.StorageMetadata(
                    storage_id, storage.StorageManager._list_directory(path)
                )

            logging.info("Saved trial to checkpoint {}".format(metadata.storage_id))
            self.tensorboard_mgr.sync()

            metadata.labels = {
                "experiment_id": str(wkld.experiment_id),
                "trial_id": str(wkld.trial_id),
                "step_id": str(wkld.step_id),
            }

            message = {
                "type": "WORKLOAD_COMPLETED",
                "workload": wkld,
                "start_time": start_time,
                "end_time": _current_timestamp(),
                "metrics": metadata,
            }  # type: workload.Response
        else:
            message = workload.Skipped()
        respond(message)

    def yield_terminate(
        self, wkld: workload.Workload, respond: workload.ResponseFunc
    ) -> workload.Stream:

        # The master can't actually handle WORKLOAD_COMPLETED messages for TERMINATE workloads.
        def _respond(_: workload.Response) -> None:
            respond(workload.Skipped())

        yield wkld, [], _respond
