# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable

from determined.common.api.fastapi_client import models as m

if TYPE_CHECKING:
    from determined.common.api.fapi import ApiClient


class _ProfilerApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_get_trial_profiler_available_series(
        self, trial_id: int, follow: bool = None
    ) -> Awaitable[m.StreamResultOfV1GetTrialProfilerAvailableSeriesResponse]:
        path_params = {"trialId": str(trial_id)}

        query_params = {}
        if follow is not None:
            query_params["follow"] = str(follow)

        return self.api_client.request(
            type_=m.StreamResultOfV1GetTrialProfilerAvailableSeriesResponse,
            method="GET",
            url="/api/v1/trials/{trialId}/profiler/available_series",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_get_trial_profiler_metrics(
        self,
        labels_trial_id: int,
        labels_name: str = None,
        labels_agent_id: str = None,
        labels_gpu_uuid: str = None,
        labels_metric_type: str = None,
        follow: bool = None,
    ) -> Awaitable[m.StreamResultOfV1GetTrialProfilerMetricsResponse]:
        path_params = {"labels.trialId": str(labels_trial_id)}

        query_params = {}
        if labels_name is not None:
            query_params["labels.name"] = str(labels_name)
        if labels_agent_id is not None:
            query_params["labels.agentId"] = str(labels_agent_id)
        if labels_gpu_uuid is not None:
            query_params["labels.gpuUuid"] = str(labels_gpu_uuid)
        if labels_metric_type is not None:
            query_params["labels.metricType"] = str(labels_metric_type)
        if follow is not None:
            query_params["follow"] = str(follow)

        return self.api_client.request(
            type_=m.StreamResultOfV1GetTrialProfilerMetricsResponse,
            method="GET",
            url="/api/v1/trials/{labels.trialId}/profiler/metrics",
            path_params=path_params,
            params=query_params,
        )


class AsyncProfilerApi(_ProfilerApi):
    async def get_trial_profiler_available_series(
        self, trial_id: int, follow: bool = None
    ) -> m.StreamResultOfV1GetTrialProfilerAvailableSeriesResponse:
        return await self._build_for_get_trial_profiler_available_series(trial_id=trial_id, follow=follow)

    async def get_trial_profiler_metrics(
        self,
        labels_trial_id: int,
        labels_name: str = None,
        labels_agent_id: str = None,
        labels_gpu_uuid: str = None,
        labels_metric_type: str = None,
        follow: bool = None,
    ) -> m.StreamResultOfV1GetTrialProfilerMetricsResponse:
        return await self._build_for_get_trial_profiler_metrics(
            labels_trial_id=labels_trial_id,
            labels_name=labels_name,
            labels_agent_id=labels_agent_id,
            labels_gpu_uuid=labels_gpu_uuid,
            labels_metric_type=labels_metric_type,
            follow=follow,
        )


class SyncProfilerApi(_ProfilerApi):
    def get_trial_profiler_available_series(
        self, trial_id: int, follow: bool = None
    ) -> m.StreamResultOfV1GetTrialProfilerAvailableSeriesResponse:
        coroutine = self._build_for_get_trial_profiler_available_series(trial_id=trial_id, follow=follow)
        return get_event_loop().run_until_complete(coroutine)

    def get_trial_profiler_metrics(
        self,
        labels_trial_id: int,
        labels_name: str = None,
        labels_agent_id: str = None,
        labels_gpu_uuid: str = None,
        labels_metric_type: str = None,
        follow: bool = None,
    ) -> m.StreamResultOfV1GetTrialProfilerMetricsResponse:
        coroutine = self._build_for_get_trial_profiler_metrics(
            labels_trial_id=labels_trial_id,
            labels_name=labels_name,
            labels_agent_id=labels_agent_id,
            labels_gpu_uuid=labels_gpu_uuid,
            labels_metric_type=labels_metric_type,
            follow=follow,
        )
        return get_event_loop().run_until_complete(coroutine)
