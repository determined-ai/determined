# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable, List

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import to_jsonable as jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fapi import ApiClient


class _TensorboardsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_tensorboard(self, tensorboard_id: str) -> Awaitable[m.V1GetTensorboardResponse]:
        path_params = {"tensorboardId": str(tensorboard_id)}

        return self.api_client.request(
            type_=m.V1GetTensorboardResponse,
            method="GET",
            url="/api/v1/tensorboards/{tensorboardId}",
            path_params=path_params,
        )

    def _build_for_determined_get_tensorboards(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> Awaitable[m.V1GetTensorboardsResponse]:
        query_params = {}
        if sort_by is not None:
            query_params["sortBy"] = str(sort_by)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if users is not None:
            query_params["users"] = [str(users_item) for users_item in users]

        return self.api_client.request(
            type_=m.V1GetTensorboardsResponse,
            method="GET",
            url="/api/v1/tensorboards",
            params=query_params,
        )

    def _build_for_determined_kill_tensorboard(self, tensorboard_id: str) -> Awaitable[m.V1KillTensorboardResponse]:
        path_params = {"tensorboardId": str(tensorboard_id)}

        return self.api_client.request(
            type_=m.V1KillTensorboardResponse,
            method="POST",
            url="/api/v1/tensorboards/{tensorboardId}/kill",
            path_params=path_params,
        )

    def _build_for_determined_launch_tensorboard(
        self, body: m.V1LaunchTensorboardRequest
    ) -> Awaitable[m.V1LaunchTensorboardResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1LaunchTensorboardResponse, method="POST", url="/api/v1/tensorboards", json=body
        )

    def _build_for_determined_set_tensorboard_priority(
        self, tensorboard_id: str, body: m.V1SetTensorboardPriorityRequest
    ) -> Awaitable[m.V1SetTensorboardPriorityResponse]:
        path_params = {"tensorboardId": str(tensorboard_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1SetTensorboardPriorityResponse,
            method="POST",
            url="/api/v1/tensorboards/{tensorboardId}/set_priority",
            path_params=path_params,
            json=body,
        )


class AsyncTensorboardsApi(_TensorboardsApi):
    async def determined_get_tensorboard(self, tensorboard_id: str) -> m.V1GetTensorboardResponse:
        return await self._build_for_determined_get_tensorboard(tensorboard_id=tensorboard_id)

    async def determined_get_tensorboards(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> m.V1GetTensorboardsResponse:
        return await self._build_for_determined_get_tensorboards(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, users=users
        )

    async def determined_kill_tensorboard(self, tensorboard_id: str) -> m.V1KillTensorboardResponse:
        return await self._build_for_determined_kill_tensorboard(tensorboard_id=tensorboard_id)

    async def determined_launch_tensorboard(self, body: m.V1LaunchTensorboardRequest) -> m.V1LaunchTensorboardResponse:
        return await self._build_for_determined_launch_tensorboard(body=body)

    async def determined_set_tensorboard_priority(
        self, tensorboard_id: str, body: m.V1SetTensorboardPriorityRequest
    ) -> m.V1SetTensorboardPriorityResponse:
        return await self._build_for_determined_set_tensorboard_priority(tensorboard_id=tensorboard_id, body=body)


class SyncTensorboardsApi(_TensorboardsApi):
    def determined_get_tensorboard(self, tensorboard_id: str) -> m.V1GetTensorboardResponse:
        coroutine = self._build_for_determined_get_tensorboard(tensorboard_id=tensorboard_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_tensorboards(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> m.V1GetTensorboardsResponse:
        coroutine = self._build_for_determined_get_tensorboards(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, users=users
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_kill_tensorboard(self, tensorboard_id: str) -> m.V1KillTensorboardResponse:
        coroutine = self._build_for_determined_kill_tensorboard(tensorboard_id=tensorboard_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_launch_tensorboard(self, body: m.V1LaunchTensorboardRequest) -> m.V1LaunchTensorboardResponse:
        coroutine = self._build_for_determined_launch_tensorboard(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_set_tensorboard_priority(
        self, tensorboard_id: str, body: m.V1SetTensorboardPriorityRequest
    ) -> m.V1SetTensorboardPriorityResponse:
        coroutine = self._build_for_determined_set_tensorboard_priority(tensorboard_id=tensorboard_id, body=body)
        return get_event_loop().run_until_complete(coroutine)
