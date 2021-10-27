# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import to_jsonable as jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fapi import ApiClient


class _AuthenticationApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_current_user(
        self,
    ) -> Awaitable[m.V1CurrentUserResponse]:
        return self.api_client.request(
            type_=m.V1CurrentUserResponse,
            method="GET",
            url="/api/v1/auth/user",
        )

    def _build_for_login(self, body: m.V1LoginRequest) -> Awaitable[m.V1LoginResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(type_=m.V1LoginResponse, method="POST", url="/api/v1/auth/login", json=body)

    def _build_for_logout(
        self,
    ) -> Awaitable[m.Any]:
        return self.api_client.request(
            type_=m.Any,
            method="POST",
            url="/api/v1/auth/logout",
        )


class AsyncAuthenticationApi(_AuthenticationApi):
    async def current_user(
        self,
    ) -> m.V1CurrentUserResponse:
        return await self._build_for_current_user()

    async def login(self, body: m.V1LoginRequest) -> m.V1LoginResponse:
        return await self._build_for_login(body=body)

    async def logout(
        self,
    ) -> m.Any:
        return await self._build_for_logout()


class SyncAuthenticationApi(_AuthenticationApi):
    def current_user(
        self,
    ) -> m.V1CurrentUserResponse:
        coroutine = self._build_for_current_user()
        return get_event_loop().run_until_complete(coroutine)

    def login(self, body: m.V1LoginRequest) -> m.V1LoginResponse:
        coroutine = self._build_for_login(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def logout(
        self,
    ) -> m.Any:
        coroutine = self._build_for_logout()
        return get_event_loop().run_until_complete(coroutine)
