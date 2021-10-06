# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable, List

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _CommandsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_command(self, command_id: str) -> Awaitable[m.V1GetCommandResponse]:
        path_params = {"commandId": str(command_id)}

        return self.api_client.request(
            type_=m.V1GetCommandResponse,
            method="GET",
            url="/api/v1/commands/{commandId}",
            path_params=path_params,
        )

    def _build_for_determined_get_commands(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> Awaitable[m.V1GetCommandsResponse]:
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
            type_=m.V1GetCommandsResponse,
            method="GET",
            url="/api/v1/commands",
            params=query_params,
        )

    def _build_for_determined_kill_command(self, command_id: str) -> Awaitable[m.V1KillCommandResponse]:
        path_params = {"commandId": str(command_id)}

        return self.api_client.request(
            type_=m.V1KillCommandResponse,
            method="POST",
            url="/api/v1/commands/{commandId}/kill",
            path_params=path_params,
        )

    def _build_for_determined_launch_command(
        self, body: m.V1LaunchCommandRequest
    ) -> Awaitable[m.V1LaunchCommandResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1LaunchCommandResponse, method="POST", url="/api/v1/commands", json=body
        )

    def _build_for_determined_set_command_priority(
        self, command_id: str, body: m.V1SetCommandPriorityRequest
    ) -> Awaitable[m.V1SetCommandPriorityResponse]:
        path_params = {"commandId": str(command_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1SetCommandPriorityResponse,
            method="POST",
            url="/api/v1/commands/{commandId}/set_priority",
            path_params=path_params,
            json=body,
        )


class AsyncCommandsApi(_CommandsApi):
    async def determined_get_command(self, command_id: str) -> m.V1GetCommandResponse:
        return await self._build_for_determined_get_command(command_id=command_id)

    async def determined_get_commands(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> m.V1GetCommandsResponse:
        return await self._build_for_determined_get_commands(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, users=users
        )

    async def determined_kill_command(self, command_id: str) -> m.V1KillCommandResponse:
        return await self._build_for_determined_kill_command(command_id=command_id)

    async def determined_launch_command(self, body: m.V1LaunchCommandRequest) -> m.V1LaunchCommandResponse:
        return await self._build_for_determined_launch_command(body=body)

    async def determined_set_command_priority(
        self, command_id: str, body: m.V1SetCommandPriorityRequest
    ) -> m.V1SetCommandPriorityResponse:
        return await self._build_for_determined_set_command_priority(command_id=command_id, body=body)


class SyncCommandsApi(_CommandsApi):
    def determined_get_command(self, command_id: str) -> m.V1GetCommandResponse:
        coroutine = self._build_for_determined_get_command(command_id=command_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_commands(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> m.V1GetCommandsResponse:
        coroutine = self._build_for_determined_get_commands(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, users=users
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_kill_command(self, command_id: str) -> m.V1KillCommandResponse:
        coroutine = self._build_for_determined_kill_command(command_id=command_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_launch_command(self, body: m.V1LaunchCommandRequest) -> m.V1LaunchCommandResponse:
        coroutine = self._build_for_determined_launch_command(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_set_command_priority(
        self, command_id: str, body: m.V1SetCommandPriorityRequest
    ) -> m.V1SetCommandPriorityResponse:
        coroutine = self._build_for_determined_set_command_priority(command_id=command_id, body=body)
        return get_event_loop().run_until_complete(coroutine)
