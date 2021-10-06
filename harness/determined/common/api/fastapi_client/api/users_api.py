# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable

from determined.common.api.fastapi_client import models as m
from fastapi.encoders import jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _UsersApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_user(self, username: str) -> Awaitable[m.V1GetUserResponse]:
        path_params = {"username": str(username)}

        return self.api_client.request(
            type_=m.V1GetUserResponse,
            method="GET",
            url="/api/v1/users/{username}",
            path_params=path_params,
        )

    def _build_for_determined_get_users(
        self,
    ) -> Awaitable[m.V1GetUsersResponse]:
        return self.api_client.request(
            type_=m.V1GetUsersResponse,
            method="GET",
            url="/api/v1/users",
        )

    def _build_for_determined_post_user(self, body: m.V1PostUserRequest) -> Awaitable[m.V1PostUserResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(type_=m.V1PostUserResponse, method="POST", url="/api/v1/users", json=body)

    def _build_for_determined_set_user_password(
        self, username: str, body: str
    ) -> Awaitable[m.V1SetUserPasswordResponse]:
        path_params = {"username": str(username)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1SetUserPasswordResponse,
            method="POST",
            url="/api/v1/users/{username}/password",
            path_params=path_params,
            json=body,
        )


class AsyncUsersApi(_UsersApi):
    async def determined_get_user(self, username: str) -> m.V1GetUserResponse:
        return await self._build_for_determined_get_user(username=username)

    async def determined_get_users(
        self,
    ) -> m.V1GetUsersResponse:
        return await self._build_for_determined_get_users()

    async def determined_post_user(self, body: m.V1PostUserRequest) -> m.V1PostUserResponse:
        return await self._build_for_determined_post_user(body=body)

    async def determined_set_user_password(self, username: str, body: str) -> m.V1SetUserPasswordResponse:
        return await self._build_for_determined_set_user_password(username=username, body=body)


class SyncUsersApi(_UsersApi):
    def determined_get_user(self, username: str) -> m.V1GetUserResponse:
        coroutine = self._build_for_determined_get_user(username=username)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_users(
        self,
    ) -> m.V1GetUsersResponse:
        coroutine = self._build_for_determined_get_users()
        return get_event_loop().run_until_complete(coroutine)

    def determined_post_user(self, body: m.V1PostUserRequest) -> m.V1PostUserResponse:
        coroutine = self._build_for_determined_post_user(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_set_user_password(self, username: str, body: str) -> m.V1SetUserPasswordResponse:
        coroutine = self._build_for_determined_set_user_password(username=username, body=body)
        return get_event_loop().run_until_complete(coroutine)
