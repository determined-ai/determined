# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable, List

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _ShellsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_shell(self, shell_id: str) -> Awaitable[m.V1GetShellResponse]:
        path_params = {"shellId": str(shell_id)}

        return self.api_client.request(
            type_=m.V1GetShellResponse,
            method="GET",
            url="/api/v1/shells/{shellId}",
            path_params=path_params,
        )

    def _build_for_determined_get_shells(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> Awaitable[m.V1GetShellsResponse]:
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
            type_=m.V1GetShellsResponse,
            method="GET",
            url="/api/v1/shells",
            params=query_params,
        )

    def _build_for_determined_kill_shell(self, shell_id: str) -> Awaitable[m.V1KillShellResponse]:
        path_params = {"shellId": str(shell_id)}

        return self.api_client.request(
            type_=m.V1KillShellResponse,
            method="POST",
            url="/api/v1/shells/{shellId}/kill",
            path_params=path_params,
        )

    def _build_for_determined_launch_shell(self, body: m.V1LaunchShellRequest) -> Awaitable[m.V1LaunchShellResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(type_=m.V1LaunchShellResponse, method="POST", url="/api/v1/shells", json=body)

    def _build_for_determined_set_shell_priority(
        self, shell_id: str, body: m.V1SetShellPriorityRequest
    ) -> Awaitable[m.V1SetShellPriorityResponse]:
        path_params = {"shellId": str(shell_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1SetShellPriorityResponse,
            method="POST",
            url="/api/v1/shells/{shellId}/set_priority",
            path_params=path_params,
            json=body,
        )


class AsyncShellsApi(_ShellsApi):
    async def determined_get_shell(self, shell_id: str) -> m.V1GetShellResponse:
        return await self._build_for_determined_get_shell(shell_id=shell_id)

    async def determined_get_shells(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> m.V1GetShellsResponse:
        return await self._build_for_determined_get_shells(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, users=users
        )

    async def determined_kill_shell(self, shell_id: str) -> m.V1KillShellResponse:
        return await self._build_for_determined_kill_shell(shell_id=shell_id)

    async def determined_launch_shell(self, body: m.V1LaunchShellRequest) -> m.V1LaunchShellResponse:
        return await self._build_for_determined_launch_shell(body=body)

    async def determined_set_shell_priority(
        self, shell_id: str, body: m.V1SetShellPriorityRequest
    ) -> m.V1SetShellPriorityResponse:
        return await self._build_for_determined_set_shell_priority(shell_id=shell_id, body=body)


class SyncShellsApi(_ShellsApi):
    def determined_get_shell(self, shell_id: str) -> m.V1GetShellResponse:
        coroutine = self._build_for_determined_get_shell(shell_id=shell_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_shells(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> m.V1GetShellsResponse:
        coroutine = self._build_for_determined_get_shells(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, users=users
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_kill_shell(self, shell_id: str) -> m.V1KillShellResponse:
        coroutine = self._build_for_determined_kill_shell(shell_id=shell_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_launch_shell(self, body: m.V1LaunchShellRequest) -> m.V1LaunchShellResponse:
        coroutine = self._build_for_determined_launch_shell(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_set_shell_priority(
        self, shell_id: str, body: m.V1SetShellPriorityRequest
    ) -> m.V1SetShellPriorityResponse:
        coroutine = self._build_for_determined_set_shell_priority(shell_id=shell_id, body=body)
        return get_event_loop().run_until_complete(coroutine)
