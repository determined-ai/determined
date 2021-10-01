# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable, List

from determined.common.api.fastapi_client import models as m

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _JobsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_job_queue_stats(
        self, resource_pools: List[str] = None
    ) -> Awaitable[m.V1GetJobQueueStatsResponse]:
        query_params = {}
        if resource_pools is not None:
            query_params["resourcePools"] = [str(resource_pools_item) for resource_pools_item in resource_pools]

        return self.api_client.request(
            type_=m.V1GetJobQueueStatsResponse,
            method="GET",
            url="/api/v1/resource-pools/queues/stats",
            params=query_params,
        )

    def _build_for_determined_get_jobs(
        self,
        pagination_offset: int = None,
        pagination_limit: int = None,
        resource_pools: List[str] = None,
        order_by: str = None,
    ) -> Awaitable[m.V1GetJobsResponse]:
        query_params = {}
        if pagination_offset is not None:
            query_params["pagination.offset"] = str(pagination_offset)
        if pagination_limit is not None:
            query_params["pagination.limit"] = str(pagination_limit)
        if resource_pools is not None:
            query_params["resourcePools"] = [str(resource_pools_item) for resource_pools_item in resource_pools]
        if order_by is not None:
            query_params["orderBy"] = str(order_by)

        return self.api_client.request(
            type_=m.V1GetJobsResponse,
            method="GET",
            url="/api/v1/resource-pools/queues",
            params=query_params,
        )

    def _build_for_determined_update_job_queue(
        self,
    ) -> Awaitable[m.Any]:
        return self.api_client.request(
            type_=m.Any,
            method="PATCH",
            url="/api/v1/resource-pools/queues",
        )


class AsyncJobsApi(_JobsApi):
    async def determined_get_job_queue_stats(self, resource_pools: List[str] = None) -> m.V1GetJobQueueStatsResponse:
        return await self._build_for_determined_get_job_queue_stats(resource_pools=resource_pools)

    async def determined_get_jobs(
        self,
        pagination_offset: int = None,
        pagination_limit: int = None,
        resource_pools: List[str] = None,
        order_by: str = None,
    ) -> m.V1GetJobsResponse:
        return await self._build_for_determined_get_jobs(
            pagination_offset=pagination_offset,
            pagination_limit=pagination_limit,
            resource_pools=resource_pools,
            order_by=order_by,
        )

    async def determined_update_job_queue(
        self,
    ) -> m.Any:
        return await self._build_for_determined_update_job_queue()


class SyncJobsApi(_JobsApi):
    def determined_get_job_queue_stats(self, resource_pools: List[str] = None) -> m.V1GetJobQueueStatsResponse:
        coroutine = self._build_for_determined_get_job_queue_stats(resource_pools=resource_pools)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_jobs(
        self,
        pagination_offset: int = None,
        pagination_limit: int = None,
        resource_pools: List[str] = None,
        order_by: str = None,
    ) -> m.V1GetJobsResponse:
        coroutine = self._build_for_determined_get_jobs(
            pagination_offset=pagination_offset,
            pagination_limit=pagination_limit,
            resource_pools=resource_pools,
            order_by=order_by,
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_update_job_queue(
        self,
    ) -> m.Any:
        coroutine = self._build_for_determined_update_job_queue()
        return get_event_loop().run_until_complete(coroutine)
