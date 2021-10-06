# flake8: noqa E501
from asyncio import get_event_loop
from datetime import datetime
from typing import TYPE_CHECKING, Awaitable

from determined.common.api.fastapi_client import models as m
from fastapi.encoders import jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _ClusterApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_disable_agent(
        self, agent_id: str, body: m.V1DisableAgentRequest
    ) -> Awaitable[m.V1DisableAgentResponse]:
        path_params = {"agentId": str(agent_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1DisableAgentResponse,
            method="POST",
            url="/api/v1/agents/{agentId}/disable",
            path_params=path_params,
            json=body,
        )

    def _build_for_determined_disable_slot(self, agent_id: str, slot_id: str) -> Awaitable[m.V1DisableSlotResponse]:
        path_params = {"agentId": str(agent_id), "slotId": str(slot_id)}

        return self.api_client.request(
            type_=m.V1DisableSlotResponse,
            method="POST",
            url="/api/v1/agents/{agentId}/slots/{slotId}/disable",
            path_params=path_params,
        )

    def _build_for_determined_enable_agent(self, agent_id: str) -> Awaitable[m.V1EnableAgentResponse]:
        path_params = {"agentId": str(agent_id)}

        return self.api_client.request(
            type_=m.V1EnableAgentResponse,
            method="POST",
            url="/api/v1/agents/{agentId}/enable",
            path_params=path_params,
        )

    def _build_for_determined_enable_slot(self, agent_id: str, slot_id: str) -> Awaitable[m.V1EnableSlotResponse]:
        path_params = {"agentId": str(agent_id), "slotId": str(slot_id)}

        return self.api_client.request(
            type_=m.V1EnableSlotResponse,
            method="POST",
            url="/api/v1/agents/{agentId}/slots/{slotId}/enable",
            path_params=path_params,
        )

    def _build_for_determined_get_agent(self, agent_id: str) -> Awaitable[m.V1GetAgentResponse]:
        path_params = {"agentId": str(agent_id)}

        return self.api_client.request(
            type_=m.V1GetAgentResponse,
            method="GET",
            url="/api/v1/agents/{agentId}",
            path_params=path_params,
        )

    def _build_for_determined_get_agents(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, label: str = None
    ) -> Awaitable[m.V1GetAgentsResponse]:
        query_params = {}
        if sort_by is not None:
            query_params["sortBy"] = str(sort_by)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if label is not None:
            query_params["label"] = str(label)

        return self.api_client.request(
            type_=m.V1GetAgentsResponse,
            method="GET",
            url="/api/v1/agents",
            params=query_params,
        )

    def _build_for_determined_get_master(
        self,
    ) -> Awaitable[m.V1GetMasterResponse]:
        return self.api_client.request(
            type_=m.V1GetMasterResponse,
            method="GET",
            url="/api/v1/master",
        )

    def _build_for_determined_get_master_config(
        self,
    ) -> Awaitable[m.V1GetMasterConfigResponse]:
        return self.api_client.request(
            type_=m.V1GetMasterConfigResponse,
            method="GET",
            url="/api/v1/master/config",
        )

    def _build_for_determined_get_slot(self, agent_id: str, slot_id: str) -> Awaitable[m.V1GetSlotResponse]:
        path_params = {"agentId": str(agent_id), "slotId": str(slot_id)}

        return self.api_client.request(
            type_=m.V1GetSlotResponse,
            method="GET",
            url="/api/v1/agents/{agentId}/slots/{slotId}",
            path_params=path_params,
        )

    def _build_for_determined_get_slots(self, agent_id: str) -> Awaitable[m.V1GetSlotsResponse]:
        path_params = {"agentId": str(agent_id)}

        return self.api_client.request(
            type_=m.V1GetSlotsResponse,
            method="GET",
            url="/api/v1/agents/{agentId}/slots",
            path_params=path_params,
        )

    def _build_for_determined_master_logs(
        self, offset: int = None, limit: int = None, follow: bool = None
    ) -> Awaitable[m.StreamResultOfV1MasterLogsResponse]:
        query_params = {}
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if follow is not None:
            query_params["follow"] = str(follow)

        return self.api_client.request(
            type_=m.StreamResultOfV1MasterLogsResponse,
            method="GET",
            url="/api/v1/master/logs",
            params=query_params,
        )

    def _build_for_determined_resource_allocation_aggregated(
        self, start_date: str = None, end_date: str = None, period: str = None
    ) -> Awaitable[m.V1ResourceAllocationAggregatedResponse]:
        query_params = {}
        if start_date is not None:
            query_params["startDate"] = str(start_date)
        if end_date is not None:
            query_params["endDate"] = str(end_date)
        if period is not None:
            query_params["period"] = str(period)

        return self.api_client.request(
            type_=m.V1ResourceAllocationAggregatedResponse,
            method="GET",
            url="/api/v1/resources/allocation/aggregated",
            params=query_params,
        )

    def _build_for_determined_resource_allocation_raw(
        self, timestamp_after: datetime = None, timestamp_before: datetime = None
    ) -> Awaitable[m.V1ResourceAllocationRawResponse]:
        query_params = {}
        if timestamp_after is not None:
            query_params["timestampAfter"] = str(timestamp_after)
        if timestamp_before is not None:
            query_params["timestampBefore"] = str(timestamp_before)

        return self.api_client.request(
            type_=m.V1ResourceAllocationRawResponse,
            method="GET",
            url="/api/v1/resources/allocation/raw",
            params=query_params,
        )

    def _build_for_get_aggregated_resource_allocation_csv(
        self, start_date: str, end_date: str, period: str
    ) -> Awaitable[None]:
        query_params = {"start_date": str(start_date), "end_date": str(end_date), "period": str(period)}

        return self.api_client.request(
            type_=None,
            method="GET",
            url="/allocation/aggregated",
            params=query_params,
        )

    def _build_for_get_raw_resource_allocation_csv(
        self, timestamp_after: str, timestamp_before: str
    ) -> Awaitable[None]:
        query_params = {"timestamp_after": str(timestamp_after), "timestamp_before": str(timestamp_before)}

        return self.api_client.request(
            type_=None,
            method="GET",
            url="/allocation/raw",
            params=query_params,
        )


class AsyncClusterApi(_ClusterApi):
    async def determined_disable_agent(self, agent_id: str, body: m.V1DisableAgentRequest) -> m.V1DisableAgentResponse:
        return await self._build_for_determined_disable_agent(agent_id=agent_id, body=body)

    async def determined_disable_slot(self, agent_id: str, slot_id: str) -> m.V1DisableSlotResponse:
        return await self._build_for_determined_disable_slot(agent_id=agent_id, slot_id=slot_id)

    async def determined_enable_agent(self, agent_id: str) -> m.V1EnableAgentResponse:
        return await self._build_for_determined_enable_agent(agent_id=agent_id)

    async def determined_enable_slot(self, agent_id: str, slot_id: str) -> m.V1EnableSlotResponse:
        return await self._build_for_determined_enable_slot(agent_id=agent_id, slot_id=slot_id)

    async def determined_get_agent(self, agent_id: str) -> m.V1GetAgentResponse:
        return await self._build_for_determined_get_agent(agent_id=agent_id)

    async def determined_get_agents(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, label: str = None
    ) -> m.V1GetAgentsResponse:
        return await self._build_for_determined_get_agents(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, label=label
        )

    async def determined_get_master(
        self,
    ) -> m.V1GetMasterResponse:
        return await self._build_for_determined_get_master()

    async def determined_get_master_config(
        self,
    ) -> m.V1GetMasterConfigResponse:
        return await self._build_for_determined_get_master_config()

    async def determined_get_slot(self, agent_id: str, slot_id: str) -> m.V1GetSlotResponse:
        return await self._build_for_determined_get_slot(agent_id=agent_id, slot_id=slot_id)

    async def determined_get_slots(self, agent_id: str) -> m.V1GetSlotsResponse:
        return await self._build_for_determined_get_slots(agent_id=agent_id)

    async def determined_master_logs(
        self, offset: int = None, limit: int = None, follow: bool = None
    ) -> m.StreamResultOfV1MasterLogsResponse:
        return await self._build_for_determined_master_logs(offset=offset, limit=limit, follow=follow)

    async def determined_resource_allocation_aggregated(
        self, start_date: str = None, end_date: str = None, period: str = None
    ) -> m.V1ResourceAllocationAggregatedResponse:
        return await self._build_for_determined_resource_allocation_aggregated(
            start_date=start_date, end_date=end_date, period=period
        )

    async def determined_resource_allocation_raw(
        self, timestamp_after: datetime = None, timestamp_before: datetime = None
    ) -> m.V1ResourceAllocationRawResponse:
        return await self._build_for_determined_resource_allocation_raw(
            timestamp_after=timestamp_after, timestamp_before=timestamp_before
        )

    async def get_aggregated_resource_allocation_csv(self, start_date: str, end_date: str, period: str) -> None:
        return await self._build_for_get_aggregated_resource_allocation_csv(
            start_date=start_date, end_date=end_date, period=period
        )

    async def get_raw_resource_allocation_csv(self, timestamp_after: str, timestamp_before: str) -> None:
        return await self._build_for_get_raw_resource_allocation_csv(
            timestamp_after=timestamp_after, timestamp_before=timestamp_before
        )


class SyncClusterApi(_ClusterApi):
    def determined_disable_agent(self, agent_id: str, body: m.V1DisableAgentRequest) -> m.V1DisableAgentResponse:
        coroutine = self._build_for_determined_disable_agent(agent_id=agent_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_disable_slot(self, agent_id: str, slot_id: str) -> m.V1DisableSlotResponse:
        coroutine = self._build_for_determined_disable_slot(agent_id=agent_id, slot_id=slot_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_enable_agent(self, agent_id: str) -> m.V1EnableAgentResponse:
        coroutine = self._build_for_determined_enable_agent(agent_id=agent_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_enable_slot(self, agent_id: str, slot_id: str) -> m.V1EnableSlotResponse:
        coroutine = self._build_for_determined_enable_slot(agent_id=agent_id, slot_id=slot_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_agent(self, agent_id: str) -> m.V1GetAgentResponse:
        coroutine = self._build_for_determined_get_agent(agent_id=agent_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_agents(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, label: str = None
    ) -> m.V1GetAgentsResponse:
        coroutine = self._build_for_determined_get_agents(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, label=label
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_master(
        self,
    ) -> m.V1GetMasterResponse:
        coroutine = self._build_for_determined_get_master()
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_master_config(
        self,
    ) -> m.V1GetMasterConfigResponse:
        coroutine = self._build_for_determined_get_master_config()
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_slot(self, agent_id: str, slot_id: str) -> m.V1GetSlotResponse:
        coroutine = self._build_for_determined_get_slot(agent_id=agent_id, slot_id=slot_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_slots(self, agent_id: str) -> m.V1GetSlotsResponse:
        coroutine = self._build_for_determined_get_slots(agent_id=agent_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_master_logs(
        self, offset: int = None, limit: int = None, follow: bool = None
    ) -> m.StreamResultOfV1MasterLogsResponse:
        coroutine = self._build_for_determined_master_logs(offset=offset, limit=limit, follow=follow)
        return get_event_loop().run_until_complete(coroutine)

    def determined_resource_allocation_aggregated(
        self, start_date: str = None, end_date: str = None, period: str = None
    ) -> m.V1ResourceAllocationAggregatedResponse:
        coroutine = self._build_for_determined_resource_allocation_aggregated(
            start_date=start_date, end_date=end_date, period=period
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_resource_allocation_raw(
        self, timestamp_after: datetime = None, timestamp_before: datetime = None
    ) -> m.V1ResourceAllocationRawResponse:
        coroutine = self._build_for_determined_resource_allocation_raw(
            timestamp_after=timestamp_after, timestamp_before=timestamp_before
        )
        return get_event_loop().run_until_complete(coroutine)

    def get_aggregated_resource_allocation_csv(self, start_date: str, end_date: str, period: str) -> None:
        coroutine = self._build_for_get_aggregated_resource_allocation_csv(
            start_date=start_date, end_date=end_date, period=period
        )
        return get_event_loop().run_until_complete(coroutine)

    def get_raw_resource_allocation_csv(self, timestamp_after: str, timestamp_before: str) -> None:
        coroutine = self._build_for_get_raw_resource_allocation_csv(
            timestamp_after=timestamp_after, timestamp_before=timestamp_before
        )
        return get_event_loop().run_until_complete(coroutine)
