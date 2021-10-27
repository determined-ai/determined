# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import to_jsonable as jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fapi import ApiClient


class _InternalApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_ack_allocation_preemption_signal(
        self, allocation_id: str, body: m.V1AckAllocationPreemptionSignalRequest
    ) -> Awaitable[m.Any]:
        path_params = {"allocationId": str(allocation_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/allocations/{allocationId}/signals/ack_preemption",
            path_params=path_params,
            json=body,
        )

    def _build_for_allocation_preemption_signal(
        self, allocation_id: str, timeout_seconds: int = None
    ) -> Awaitable[m.V1AllocationPreemptionSignalResponse]:
        path_params = {"allocationId": str(allocation_id)}

        query_params = {}
        if timeout_seconds is not None:
            query_params["timeoutSeconds"] = str(timeout_seconds)

        return self.api_client.request(
            type_=m.V1AllocationPreemptionSignalResponse,
            method="GET",
            url="/api/v1/allocations/{allocationId}/signals/preemption",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_allocation_rendezvous_info(
        self, allocation_id: str, container_id: str
    ) -> Awaitable[m.V1AllocationRendezvousInfoResponse]:
        path_params = {"allocationId": str(allocation_id), "containerId": str(container_id)}

        return self.api_client.request(
            type_=m.V1AllocationRendezvousInfoResponse,
            method="GET",
            url="/api/v1/allocations/{allocationId}/rendezvous_info/{containerId}",
            path_params=path_params,
        )

    def _build_for_complete_trial_searcher_validation(
        self, trial_id: int, body: m.V1CompleteValidateAfterOperation
    ) -> Awaitable[m.Any]:
        path_params = {"trialId": str(trial_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/trials/{trialId}/searcher/completed_operation",
            path_params=path_params,
            json=body,
        )

    def _build_for_compute_hp_importance(self, experiment_id: int) -> Awaitable[m.Any]:
        path_params = {"experimentId": str(experiment_id)}

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/experiments/{experimentId}/hyperparameter-importance",
            path_params=path_params,
        )

    def _build_for_create_experiment(
        self, body: m.V1CreateExperimentRequest
    ) -> Awaitable[m.V1CreateExperimentResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1CreateExperimentResponse, method="POST", url="/api/v1/experiments", json=body
        )

    def _build_for_get_best_searcher_validation_metric(
        self, experiment_id: int
    ) -> Awaitable[m.V1GetBestSearcherValidationMetricResponse]:
        path_params = {"experimentId": str(experiment_id)}

        return self.api_client.request(
            type_=m.V1GetBestSearcherValidationMetricResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/searcher/best_searcher_validation_metric",
            path_params=path_params,
        )

    def _build_for_get_current_trial_searcher_operation(
        self, trial_id: int
    ) -> Awaitable[m.V1GetCurrentTrialSearcherOperationResponse]:
        path_params = {"trialId": str(trial_id)}

        return self.api_client.request(
            type_=m.V1GetCurrentTrialSearcherOperationResponse,
            method="GET",
            url="/api/v1/trials/{trialId}/searcher/operation",
            path_params=path_params,
        )

    def _build_for_get_hp_importance(
        self, experiment_id: int, period_seconds: int = None
    ) -> Awaitable[m.StreamResultOfV1GetHPImportanceResponse]:
        path_params = {"experimentId": str(experiment_id)}

        query_params = {}
        if period_seconds is not None:
            query_params["periodSeconds"] = str(period_seconds)

        return self.api_client.request(
            type_=m.StreamResultOfV1GetHPImportanceResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/hyperparameter-importance",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_get_resource_pools(
        self, offset: int = None, limit: int = None
    ) -> Awaitable[m.V1GetResourcePoolsResponse]:
        query_params = {}
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)

        return self.api_client.request(
            type_=m.V1GetResourcePoolsResponse,
            method="GET",
            url="/api/v1/resource-pools",
            params=query_params,
        )

    def _build_for_get_telemetry(
        self,
    ) -> Awaitable[m.V1GetTelemetryResponse]:
        return self.api_client.request(
            type_=m.V1GetTelemetryResponse,
            method="GET",
            url="/api/v1/master/telemetry",
        )

    def _build_for_idle_notebook(self, notebook_id: str, body: m.V1IdleNotebookRequest) -> Awaitable[m.Any]:
        path_params = {"notebookId": str(notebook_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any,
            method="PUT",
            url="/api/v1/notebooks/{notebookId}/report_idle",
            path_params=path_params,
            json=body,
        )

    def _build_for_mark_allocation_reservation_daemon(
        self, allocation_id: str, container_id: str, body: m.V1MarkAllocationReservationDaemonRequest
    ) -> Awaitable[m.Any]:
        path_params = {"allocationId": str(allocation_id), "containerId": str(container_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/allocations/{allocationId}/containers/{containerId}/daemon",
            path_params=path_params,
            json=body,
        )

    def _build_for_metric_batches(
        self, experiment_id: int, metric_name: str, metric_type: str, period_seconds: int = None
    ) -> Awaitable[m.StreamResultOfV1MetricBatchesResponse]:
        path_params = {"experimentId": str(experiment_id)}

        query_params = {
            "metricName": str(metric_name),
            "metricType": str(metric_type),
        }
        if period_seconds is not None:
            query_params["periodSeconds"] = str(period_seconds)

        return self.api_client.request(
            type_=m.StreamResultOfV1MetricBatchesResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/metrics-stream/batches",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_metric_names(
        self, experiment_id: int, period_seconds: int = None
    ) -> Awaitable[m.StreamResultOfV1MetricNamesResponse]:
        path_params = {"experimentId": str(experiment_id)}

        query_params = {}
        if period_seconds is not None:
            query_params["periodSeconds"] = str(period_seconds)

        return self.api_client.request(
            type_=m.StreamResultOfV1MetricNamesResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/metrics-stream/metric-names",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_post_trial_profiler_metrics_batch(
        self, body: m.V1PostTrialProfilerMetricsBatchRequest
    ) -> Awaitable[m.Any]:
        body = jsonable_encoder(body)

        return self.api_client.request(type_=m.Any, method="POST", url="/api/v1/trials/profiler/metrics", json=body)

    def _build_for_post_trial_runner_metadata(self, trial_id: int, body: m.V1TrialRunnerMetadata) -> Awaitable[m.Any]:
        path_params = {"trialId": str(trial_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/trials/{trialId}/runner/metadata",
            path_params=path_params,
            json=body,
        )

    def _build_for_report_trial_checkpoint_metadata(
        self, checkpoint_metadata_trial_id: int, body: m.V1CheckpointMetadata
    ) -> Awaitable[m.Any]:
        path_params = {"checkpointMetadata.trialId": str(checkpoint_metadata_trial_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/trials/{checkpointMetadata.trialId}/checkpoint_metadata",
            path_params=path_params,
            json=body,
        )

    def _build_for_report_trial_progress(self, trial_id: int, body: float) -> Awaitable[m.Any]:
        path_params = {"trialId": str(trial_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any, method="POST", url="/api/v1/trials/{trialId}/progress", path_params=path_params, json=body
        )

    def _build_for_report_trial_searcher_early_exit(self, trial_id: int, body: m.V1TrialEarlyExit) -> Awaitable[m.Any]:
        path_params = {"trialId": str(trial_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any, method="POST", url="/api/v1/trials/{trialId}/early_exit", path_params=path_params, json=body
        )

    def _build_for_report_trial_training_metrics(
        self, training_metrics_trial_id: int, body: m.V1TrialMetrics
    ) -> Awaitable[m.Any]:
        path_params = {"trainingMetrics.trialId": str(training_metrics_trial_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/trials/{trainingMetrics.trialId}/training_metrics",
            path_params=path_params,
            json=body,
        )

    def _build_for_report_trial_validation_metrics(
        self, validation_metrics_trial_id: int, body: m.V1TrialMetrics
    ) -> Awaitable[m.Any]:
        path_params = {"validationMetrics.trialId": str(validation_metrics_trial_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/trials/{validationMetrics.trialId}/validation_metrics",
            path_params=path_params,
            json=body,
        )

    def _build_for_trials_sample(
        self,
        experiment_id: int,
        metric_name: str,
        metric_type: str,
        max_trials: int = None,
        max_datapoints: int = None,
        start_batches: int = None,
        end_batches: int = None,
        period_seconds: int = None,
    ) -> Awaitable[m.StreamResultOfV1TrialsSampleResponse]:
        path_params = {"experimentId": str(experiment_id)}

        query_params = {
            "metricName": str(metric_name),
            "metricType": str(metric_type),
        }
        if max_trials is not None:
            query_params["maxTrials"] = str(max_trials)
        if max_datapoints is not None:
            query_params["maxDatapoints"] = str(max_datapoints)
        if start_batches is not None:
            query_params["startBatches"] = str(start_batches)
        if end_batches is not None:
            query_params["endBatches"] = str(end_batches)
        if period_seconds is not None:
            query_params["periodSeconds"] = str(period_seconds)

        return self.api_client.request(
            type_=m.StreamResultOfV1TrialsSampleResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/metrics-stream/trials-sample",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_trials_snapshot(
        self,
        experiment_id: int,
        metric_name: str,
        metric_type: str,
        batches_processed: int,
        batches_margin: int = None,
        period_seconds: int = None,
    ) -> Awaitable[m.StreamResultOfV1TrialsSnapshotResponse]:
        path_params = {"experimentId": str(experiment_id)}

        query_params = {
            "metricName": str(metric_name),
            "metricType": str(metric_type),
            "batchesProcessed": str(batches_processed),
        }
        if batches_margin is not None:
            query_params["batchesMargin"] = str(batches_margin)
        if period_seconds is not None:
            query_params["periodSeconds"] = str(period_seconds)

        return self.api_client.request(
            type_=m.StreamResultOfV1TrialsSnapshotResponse,
            method="GET",
            url="/api/v1/experiments/{experimentId}/metrics-stream/trials-snapshot",
            path_params=path_params,
            params=query_params,
        )


class AsyncInternalApi(_InternalApi):
    async def ack_allocation_preemption_signal(
        self, allocation_id: str, body: m.V1AckAllocationPreemptionSignalRequest
    ) -> m.Any:
        return await self._build_for_ack_allocation_preemption_signal(allocation_id=allocation_id, body=body)

    async def allocation_preemption_signal(
        self, allocation_id: str, timeout_seconds: int = None
    ) -> m.V1AllocationPreemptionSignalResponse:
        return await self._build_for_allocation_preemption_signal(
            allocation_id=allocation_id, timeout_seconds=timeout_seconds
        )

    async def allocation_rendezvous_info(
        self, allocation_id: str, container_id: str
    ) -> m.V1AllocationRendezvousInfoResponse:
        return await self._build_for_allocation_rendezvous_info(allocation_id=allocation_id, container_id=container_id)

    async def complete_trial_searcher_validation(
        self, trial_id: int, body: m.V1CompleteValidateAfterOperation
    ) -> m.Any:
        return await self._build_for_complete_trial_searcher_validation(trial_id=trial_id, body=body)

    async def compute_hp_importance(self, experiment_id: int) -> m.Any:
        return await self._build_for_compute_hp_importance(experiment_id=experiment_id)

    async def create_experiment(self, body: m.V1CreateExperimentRequest) -> m.V1CreateExperimentResponse:
        return await self._build_for_create_experiment(body=body)

    async def get_best_searcher_validation_metric(
        self, experiment_id: int
    ) -> m.V1GetBestSearcherValidationMetricResponse:
        return await self._build_for_get_best_searcher_validation_metric(experiment_id=experiment_id)

    async def get_current_trial_searcher_operation(self, trial_id: int) -> m.V1GetCurrentTrialSearcherOperationResponse:
        return await self._build_for_get_current_trial_searcher_operation(trial_id=trial_id)

    async def get_hp_importance(
        self, experiment_id: int, period_seconds: int = None
    ) -> m.StreamResultOfV1GetHPImportanceResponse:
        return await self._build_for_get_hp_importance(experiment_id=experiment_id, period_seconds=period_seconds)

    async def get_resource_pools(self, offset: int = None, limit: int = None) -> m.V1GetResourcePoolsResponse:
        return await self._build_for_get_resource_pools(offset=offset, limit=limit)

    async def get_telemetry(
        self,
    ) -> m.V1GetTelemetryResponse:
        return await self._build_for_get_telemetry()

    async def idle_notebook(self, notebook_id: str, body: m.V1IdleNotebookRequest) -> m.Any:
        return await self._build_for_idle_notebook(notebook_id=notebook_id, body=body)

    async def mark_allocation_reservation_daemon(
        self, allocation_id: str, container_id: str, body: m.V1MarkAllocationReservationDaemonRequest
    ) -> m.Any:
        return await self._build_for_mark_allocation_reservation_daemon(
            allocation_id=allocation_id, container_id=container_id, body=body
        )

    async def metric_batches(
        self, experiment_id: int, metric_name: str, metric_type: str, period_seconds: int = None
    ) -> m.StreamResultOfV1MetricBatchesResponse:
        return await self._build_for_metric_batches(
            experiment_id=experiment_id, metric_name=metric_name, metric_type=metric_type, period_seconds=period_seconds
        )

    async def metric_names(
        self, experiment_id: int, period_seconds: int = None
    ) -> m.StreamResultOfV1MetricNamesResponse:
        return await self._build_for_metric_names(experiment_id=experiment_id, period_seconds=period_seconds)

    async def post_trial_profiler_metrics_batch(self, body: m.V1PostTrialProfilerMetricsBatchRequest) -> m.Any:
        return await self._build_for_post_trial_profiler_metrics_batch(body=body)

    async def post_trial_runner_metadata(self, trial_id: int, body: m.V1TrialRunnerMetadata) -> m.Any:
        return await self._build_for_post_trial_runner_metadata(trial_id=trial_id, body=body)

    async def report_trial_checkpoint_metadata(
        self, checkpoint_metadata_trial_id: int, body: m.V1CheckpointMetadata
    ) -> m.Any:
        return await self._build_for_report_trial_checkpoint_metadata(
            checkpoint_metadata_trial_id=checkpoint_metadata_trial_id, body=body
        )

    async def report_trial_progress(self, trial_id: int, body: float) -> m.Any:
        return await self._build_for_report_trial_progress(trial_id=trial_id, body=body)

    async def report_trial_searcher_early_exit(self, trial_id: int, body: m.V1TrialEarlyExit) -> m.Any:
        return await self._build_for_report_trial_searcher_early_exit(trial_id=trial_id, body=body)

    async def report_trial_training_metrics(self, training_metrics_trial_id: int, body: m.V1TrialMetrics) -> m.Any:
        return await self._build_for_report_trial_training_metrics(
            training_metrics_trial_id=training_metrics_trial_id, body=body
        )

    async def report_trial_validation_metrics(self, validation_metrics_trial_id: int, body: m.V1TrialMetrics) -> m.Any:
        return await self._build_for_report_trial_validation_metrics(
            validation_metrics_trial_id=validation_metrics_trial_id, body=body
        )

    async def trials_sample(
        self,
        experiment_id: int,
        metric_name: str,
        metric_type: str,
        max_trials: int = None,
        max_datapoints: int = None,
        start_batches: int = None,
        end_batches: int = None,
        period_seconds: int = None,
    ) -> m.StreamResultOfV1TrialsSampleResponse:
        return await self._build_for_trials_sample(
            experiment_id=experiment_id,
            metric_name=metric_name,
            metric_type=metric_type,
            max_trials=max_trials,
            max_datapoints=max_datapoints,
            start_batches=start_batches,
            end_batches=end_batches,
            period_seconds=period_seconds,
        )

    async def trials_snapshot(
        self,
        experiment_id: int,
        metric_name: str,
        metric_type: str,
        batches_processed: int,
        batches_margin: int = None,
        period_seconds: int = None,
    ) -> m.StreamResultOfV1TrialsSnapshotResponse:
        return await self._build_for_trials_snapshot(
            experiment_id=experiment_id,
            metric_name=metric_name,
            metric_type=metric_type,
            batches_processed=batches_processed,
            batches_margin=batches_margin,
            period_seconds=period_seconds,
        )


class SyncInternalApi(_InternalApi):
    def ack_allocation_preemption_signal(
        self, allocation_id: str, body: m.V1AckAllocationPreemptionSignalRequest
    ) -> m.Any:
        coroutine = self._build_for_ack_allocation_preemption_signal(allocation_id=allocation_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def allocation_preemption_signal(
        self, allocation_id: str, timeout_seconds: int = None
    ) -> m.V1AllocationPreemptionSignalResponse:
        coroutine = self._build_for_allocation_preemption_signal(
            allocation_id=allocation_id, timeout_seconds=timeout_seconds
        )
        return get_event_loop().run_until_complete(coroutine)

    def allocation_rendezvous_info(self, allocation_id: str, container_id: str) -> m.V1AllocationRendezvousInfoResponse:
        coroutine = self._build_for_allocation_rendezvous_info(allocation_id=allocation_id, container_id=container_id)
        return get_event_loop().run_until_complete(coroutine)

    def complete_trial_searcher_validation(self, trial_id: int, body: m.V1CompleteValidateAfterOperation) -> m.Any:
        coroutine = self._build_for_complete_trial_searcher_validation(trial_id=trial_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def compute_hp_importance(self, experiment_id: int) -> m.Any:
        coroutine = self._build_for_compute_hp_importance(experiment_id=experiment_id)
        return get_event_loop().run_until_complete(coroutine)

    def create_experiment(self, body: m.V1CreateExperimentRequest) -> m.V1CreateExperimentResponse:
        coroutine = self._build_for_create_experiment(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def get_best_searcher_validation_metric(self, experiment_id: int) -> m.V1GetBestSearcherValidationMetricResponse:
        coroutine = self._build_for_get_best_searcher_validation_metric(experiment_id=experiment_id)
        return get_event_loop().run_until_complete(coroutine)

    def get_current_trial_searcher_operation(self, trial_id: int) -> m.V1GetCurrentTrialSearcherOperationResponse:
        coroutine = self._build_for_get_current_trial_searcher_operation(trial_id=trial_id)
        return get_event_loop().run_until_complete(coroutine)

    def get_hp_importance(
        self, experiment_id: int, period_seconds: int = None
    ) -> m.StreamResultOfV1GetHPImportanceResponse:
        coroutine = self._build_for_get_hp_importance(experiment_id=experiment_id, period_seconds=period_seconds)
        return get_event_loop().run_until_complete(coroutine)

    def get_resource_pools(self, offset: int = None, limit: int = None) -> m.V1GetResourcePoolsResponse:
        coroutine = self._build_for_get_resource_pools(offset=offset, limit=limit)
        return get_event_loop().run_until_complete(coroutine)

    def get_telemetry(
        self,
    ) -> m.V1GetTelemetryResponse:
        coroutine = self._build_for_get_telemetry()
        return get_event_loop().run_until_complete(coroutine)

    def idle_notebook(self, notebook_id: str, body: m.V1IdleNotebookRequest) -> m.Any:
        coroutine = self._build_for_idle_notebook(notebook_id=notebook_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def mark_allocation_reservation_daemon(
        self, allocation_id: str, container_id: str, body: m.V1MarkAllocationReservationDaemonRequest
    ) -> m.Any:
        coroutine = self._build_for_mark_allocation_reservation_daemon(
            allocation_id=allocation_id, container_id=container_id, body=body
        )
        return get_event_loop().run_until_complete(coroutine)

    def metric_batches(
        self, experiment_id: int, metric_name: str, metric_type: str, period_seconds: int = None
    ) -> m.StreamResultOfV1MetricBatchesResponse:
        coroutine = self._build_for_metric_batches(
            experiment_id=experiment_id, metric_name=metric_name, metric_type=metric_type, period_seconds=period_seconds
        )
        return get_event_loop().run_until_complete(coroutine)

    def metric_names(self, experiment_id: int, period_seconds: int = None) -> m.StreamResultOfV1MetricNamesResponse:
        coroutine = self._build_for_metric_names(experiment_id=experiment_id, period_seconds=period_seconds)
        return get_event_loop().run_until_complete(coroutine)

    def post_trial_profiler_metrics_batch(self, body: m.V1PostTrialProfilerMetricsBatchRequest) -> m.Any:
        coroutine = self._build_for_post_trial_profiler_metrics_batch(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def post_trial_runner_metadata(self, trial_id: int, body: m.V1TrialRunnerMetadata) -> m.Any:
        coroutine = self._build_for_post_trial_runner_metadata(trial_id=trial_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def report_trial_checkpoint_metadata(
        self, checkpoint_metadata_trial_id: int, body: m.V1CheckpointMetadata
    ) -> m.Any:
        coroutine = self._build_for_report_trial_checkpoint_metadata(
            checkpoint_metadata_trial_id=checkpoint_metadata_trial_id, body=body
        )
        return get_event_loop().run_until_complete(coroutine)

    def report_trial_progress(self, trial_id: int, body: float) -> m.Any:
        coroutine = self._build_for_report_trial_progress(trial_id=trial_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def report_trial_searcher_early_exit(self, trial_id: int, body: m.V1TrialEarlyExit) -> m.Any:
        coroutine = self._build_for_report_trial_searcher_early_exit(trial_id=trial_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def report_trial_training_metrics(self, training_metrics_trial_id: int, body: m.V1TrialMetrics) -> m.Any:
        coroutine = self._build_for_report_trial_training_metrics(
            training_metrics_trial_id=training_metrics_trial_id, body=body
        )
        return get_event_loop().run_until_complete(coroutine)

    def report_trial_validation_metrics(self, validation_metrics_trial_id: int, body: m.V1TrialMetrics) -> m.Any:
        coroutine = self._build_for_report_trial_validation_metrics(
            validation_metrics_trial_id=validation_metrics_trial_id, body=body
        )
        return get_event_loop().run_until_complete(coroutine)

    def trials_sample(
        self,
        experiment_id: int,
        metric_name: str,
        metric_type: str,
        max_trials: int = None,
        max_datapoints: int = None,
        start_batches: int = None,
        end_batches: int = None,
        period_seconds: int = None,
    ) -> m.StreamResultOfV1TrialsSampleResponse:
        coroutine = self._build_for_trials_sample(
            experiment_id=experiment_id,
            metric_name=metric_name,
            metric_type=metric_type,
            max_trials=max_trials,
            max_datapoints=max_datapoints,
            start_batches=start_batches,
            end_batches=end_batches,
            period_seconds=period_seconds,
        )
        return get_event_loop().run_until_complete(coroutine)

    def trials_snapshot(
        self,
        experiment_id: int,
        metric_name: str,
        metric_type: str,
        batches_processed: int,
        batches_margin: int = None,
        period_seconds: int = None,
    ) -> m.StreamResultOfV1TrialsSnapshotResponse:
        coroutine = self._build_for_trials_snapshot(
            experiment_id=experiment_id,
            metric_name=metric_name,
            metric_type=metric_type,
            batches_processed=batches_processed,
            batches_margin=batches_margin,
            period_seconds=period_seconds,
        )
        return get_event_loop().run_until_complete(coroutine)
