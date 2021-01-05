import logging
import math
from typing import Any, Dict, Optional, cast

import determined as det
from determined import tensorboard, workload
from determined_common.check import check_eq, check_len, check_not_eq, check_not_isinstance


class WorkloadManager(workload.Source):
    """
    WorkloadManager handles workload messages after they are received on the
    WebSocket. Each WorkloadManager may allow different workload messages.
    """

    def __init__(self, env: det.EnvContext, workloads: workload.Stream, is_chief: bool) -> None:
        self.env = env
        self.workloads = workloads
        self.is_chief = is_chief


def build_workload_manager(
    env: det.EnvContext,
    workloads: workload.Stream,
    is_chief: bool,
) -> WorkloadManager:
    """
    Build the WorkloadManager as specified by the container environment.
    """
    if env.workload_manager_type == "TRIAL_WORKLOAD_MANAGER":
        return _TrialWorkloadManager(env, workloads, is_chief)
    raise ValueError("Unexpected workload manager type: {}", env.workload_manager_type)


class _TrialWorkloadManager(WorkloadManager):
    def __init__(
        self,
        env: det.EnvContext,
        workloads: workload.Stream,
        is_chief: bool,
    ) -> None:
        super().__init__(
            env,
            workloads,
            is_chief,
        )
        self.workload = None  # type: Optional[workload.Workload]

    def __iter__(self) -> workload.Stream:
        for wkld, args, response_func in self.workloads:
            if self.is_chief:
                logging.info("Running workload {}".format(wkld))
            else:
                logging.debug("Running workload {}".format(wkld))
            self.check_sane_workload(wkld)

            self.workload = wkld

            if wkld.kind == workload.Workload.Kind.RUN_STEP:
                yield from self.yield_train_for_step(wkld, response_func)
            elif wkld.kind == workload.Workload.Kind.COMPUTE_VALIDATION_METRICS:
                yield from self.yield_compute_validation_metrics(wkld, response_func)
            elif wkld.kind == workload.Workload.Kind.CHECKPOINT_MODEL:
                # This layer does not affect CHECKPOINT_MODEL.
                yield wkld, args, response_func
            elif wkld.kind == workload.Workload.Kind.TERMINATE:
                yield from self.yield_terminate(wkld, response_func)
            else:
                raise AssertionError("Unexpected workload: {}".format(wkld.kind))

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
        def _respond(in_response: workload.Response) -> None:

            # Only the chief container should actually respond to TRAIN_FOR_STEP.
            if not self.is_chief:
                respond(workload.Skipped())
                return

            check_not_isinstance(in_response, workload.Skipped, "Chief skipped a workload.")

            in_response = cast(workload.Metrics, in_response)
            metrics = in_response["metrics"]
            metrics = cast(workload.Metrics, metrics)

            batch_metrics = metrics["batch_metrics"]

            # Sanity-check training metrics.
            det.util.validate_batch_metrics(batch_metrics)
            check_len(batch_metrics, wkld.num_batches)

            out_response = {
                "type": "WORKLOAD_COMPLETED",
                "workload": wkld,
                "metrics": metrics,
            }

            if in_response.get("stop_requested", False):
                out_response["exited_reason"] = "USER_CANCELED"

            # Send the response up.
            respond(out_response)

        yield wkld, [], _respond

    def yield_compute_validation_metrics(
        self, wkld: workload.Workload, respond: workload.ResponseFunc
    ) -> workload.Stream:
        def _respond(in_response: workload.Response) -> None:

            # Only the chief container should actually respond to COMPUTE_VALIDATION_METRICS.
            if not self.is_chief:
                respond(workload.Skipped())
                return

            check_not_isinstance(in_response, workload.Skipped, "Chief skipped a workload.")
            in_response = cast(Dict[str, Any], in_response)
            metrics = in_response["metrics"]
            metrics = cast(workload.Metrics, metrics)
            v_metrics = metrics["validation_metrics"]

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
                "metrics": metrics,
            }

            if in_response.get("stop_requested", False):
                out_response["exited_reason"] = "USER_CANCELED"

            respond(out_response)

        yield wkld, [], _respond

    def yield_terminate(
        self, wkld: workload.Workload, respond: workload.ResponseFunc
    ) -> workload.Stream:
        # The master can't actually handle WORKLOAD_COMPLETED messages for TERMINATE workloads.
        def _respond(_: workload.Response) -> None:
            respond(workload.Skipped())

        yield wkld, [], _respond
