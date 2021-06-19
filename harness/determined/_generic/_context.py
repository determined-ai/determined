import contextlib
import datetime
import logging
import math
import pathlib
from typing import Any, Dict, Iterator, cast

import determined as det
from determined import tensorboard, workload
from determined.common import check, constants, storage


class Context:
    """
    generic.Context will someday evolve into a core part of the Generic API.
    """

    def __init__(
        self,
        env: det.EnvContext,
        dist: det.DistributedContext,
    ) -> None:
        self._env = env
        self._dist = dist

        self._storage_mgr = storage.build(
            env.experiment_config["checkpoint_storage"],
            container_path=None if not env.on_cluster else constants.SHARED_FS_CONTAINER_PATH,
        )

        self._tensorboard_mgr = tensorboard.build(
            env.det_cluster_id,
            env.det_experiment_id,
            env.det_trial_id,
            env.experiment_config["checkpoint_storage"],
            container_path=None if not env.on_cluster else constants.SHARED_FS_CONTAINER_PATH,
        )

        self._tensorboard_writer = tensorboard.get_metric_writer()

    @contextlib.contextmanager
    def _download_initial_checkpoint(self, checkpoint: Dict) -> Iterator[pathlib.Path]:
        """
        Wrap a storage_mgr.restore_path() context manager, but only download/cleanup on the
        local chief.
        """

        metadata = storage.StorageMetadata.from_json(checkpoint)
        logging.info("Restoring trial from checkpoint {}".format(metadata.storage_id))

        restore_path = self._dist._local_chief_contextmanager(self._storage_mgr.restore_path)
        with restore_path(metadata) as path:
            yield path

    @staticmethod
    def _current_timestamp() -> datetime.datetime:
        return datetime.datetime.now(datetime.timezone.utc)

    def _after_training(
        self,
        wkld: workload.Workload,
        start_time: datetime.datetime,
        response: workload.Response,
    ) -> workload.Response:
        if self._dist.get_rank() != 0:
            return response

        check.is_not_instance(response, workload.Skipped, "Chief skipped a workload.")

        response = cast(workload.Metrics, response)
        metrics = response["metrics"]
        metrics = cast(workload.Metrics, metrics)

        if response.get("invalid_hp", False):
            out_response = {
                "type": "WORKLOAD_COMPLETED",
                "workload": wkld,
                "start_time": start_time,
                "end_time": self._current_timestamp(),
                "metrics": metrics,
                "exited_reason": "INVALID_HP",
            }
            return out_response

        if response.get("init_invalid_hp", False):
            out_response = {
                "type": "WORKLOAD_COMPLETED",
                "workload": wkld,
                "start_time": start_time,
                "end_time": self._current_timestamp(),
                "metrics": metrics,
                "exited_reason": "INIT_INVALID_HP",
            }
            return out_response

        batch_metrics = metrics["batch_metrics"]
        # Sanity-check training metrics.
        det.util.validate_batch_metrics(batch_metrics)
        check.len_eq(batch_metrics, wkld.num_batches)

        self._tensorboard_writer.on_train_step_end(
            wkld.total_batches_processed + wkld.num_batches, metrics
        )

        self._tensorboard_mgr.sync()

        out_response = {
            "type": "WORKLOAD_COMPLETED",
            "workload": wkld,
            "start_time": start_time,
            "end_time": self._current_timestamp(),
            "metrics": metrics,
        }

        if response.get("stop_requested", False):
            out_response["exited_reason"] = "USER_CANCELED"

        return out_response

    def _after_validation(
        self,
        wkld: workload.Workload,
        start_time: datetime.datetime,
        searcher_metric: str,
        response: workload.Response,
    ) -> workload.Response:
        if self._dist.get_rank() != 0:
            return response

        check.is_not_instance(response, workload.Skipped, "Chief skipped a workload.")
        response = cast(Dict[str, Any], response)
        metrics = response["metrics"]
        metrics = cast(workload.Metrics, metrics)

        if response.get("invalid_hp", False):
            out_response = {
                "type": "WORKLOAD_COMPLETED",
                "workload": wkld,
                "start_time": start_time,
                "end_time": self._current_timestamp(),
                "metrics": metrics,
                "exited_reason": "INVALID_HP",
            }
            return out_response

        if response.get("init_invalid_hp", False):
            out_response = {
                "type": "WORKLOAD_COMPLETED",
                "workload": wkld,
                "start_time": start_time,
                "end_time": self._current_timestamp(),
                "metrics": metrics,
                "exited_reason": "INIT_INVALID_HP",
            }
            return out_response

        v_metrics = metrics["validation_metrics"]
        self._tensorboard_writer.on_validation_step_end(wkld.total_batches_processed, v_metrics)

        self._tensorboard_mgr.sync()

        # Check that the validation metrics computed by the model code
        # includes the metric used by the search method.
        if searcher_metric not in v_metrics:
            raise AssertionError(
                "Search method is configured to use metric '{}' but model "
                "definition returned validation metrics {}. The metric "
                "used by the search method must be one of the validation "
                "metrics returned by the model definition.".format(
                    searcher_metric, list(v_metrics.keys())
                )
            )

        # Check that the searcher metric has a scalar value so that it can be compared for
        # search purposes. Other metrics don't have to be scalars.
        metric_value = v_metrics[searcher_metric]
        if not tensorboard.metric_writers.util.is_numerical_scalar(metric_value):
            raise AssertionError(
                "Searcher validation metric '{}' returned "
                "a non-scalar value: {}".format(searcher_metric, metric_value)
            )

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

            if metric_is_none or metric_is_nan:
                raise AssertionError(
                    "Validation metric '{}' returned "
                    "an invalid scalar value: {}".format(metric_name, metric_value)
                )

            if isinstance(metric_value, (bytes, bytearray)):
                non_serializable_metrics.add(metric_name)

        if len(non_serializable_metrics):
            logging.warning(
                "Removed non serializable metrics: %s", ", ".join(non_serializable_metrics)
            )
            for metric_name in non_serializable_metrics:
                del v_metrics[metric_name]

        out_response = {
            "type": "WORKLOAD_COMPLETED",
            "workload": wkld,
            "start_time": start_time,
            "end_time": self._current_timestamp(),
            "metrics": metrics,
        }

        if response.get("stop_requested", False):
            out_response["exited_reason"] = "USER_CANCELED"

        return out_response

    def _after_checkpoint(
        self,
        wkld: workload.Workload,
        start_time: datetime.datetime,
        storage_id: str,
        path: str,
        response: workload.Response,
    ) -> workload.Response:
        if self._dist.get_rank() != 0:
            return response

        response = cast(Dict[str, Any], response)
        metadata = storage.StorageMetadata(
            storage_id,
            storage.StorageManager._list_directory(path),
            response.get("framework", ""),
            response.get("format", ""),
        )

        logging.info("Saved trial to checkpoint {}".format(metadata.storage_id))
        self._tensorboard_mgr.sync()

        return {
            "type": "WORKLOAD_COMPLETED",
            "workload": wkld,
            "start_time": start_time,
            "end_time": self._current_timestamp(),
            "metrics": metadata,
        }
