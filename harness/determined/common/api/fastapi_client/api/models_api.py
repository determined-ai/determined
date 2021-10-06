# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi_helper import to_json as jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _ModelsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_model(self, model_name: str) -> Awaitable[m.V1GetModelResponse]:
        path_params = {"modelName": str(model_name)}

        return self.api_client.request(
            type_=m.V1GetModelResponse,
            method="GET",
            url="/api/v1/models/{modelName}",
            path_params=path_params,
        )

    def _build_for_determined_get_model_version(
        self, model_name: str, model_version: int
    ) -> Awaitable[m.V1GetModelVersionResponse]:
        path_params = {"modelName": str(model_name), "modelVersion": str(model_version)}

        return self.api_client.request(
            type_=m.V1GetModelVersionResponse,
            method="GET",
            url="/api/v1/models/{modelName}/versions/{modelVersion}",
            path_params=path_params,
        )

    def _build_for_determined_get_model_versions(
        self, model_name: str, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None
    ) -> Awaitable[m.V1GetModelVersionsResponse]:
        path_params = {"modelName": str(model_name)}

        query_params = {}
        if sort_by is not None:
            query_params["sortBy"] = str(sort_by)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)

        return self.api_client.request(
            type_=m.V1GetModelVersionsResponse,
            method="GET",
            url="/api/v1/models/{modelName}/versions",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_determined_get_models(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        name: str = None,
        description: str = None,
    ) -> Awaitable[m.V1GetModelsResponse]:
        query_params = {}
        if sort_by is not None:
            query_params["sortBy"] = str(sort_by)
        if order_by is not None:
            query_params["orderBy"] = str(order_by)
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if name is not None:
            query_params["name"] = str(name)
        if description is not None:
            query_params["description"] = str(description)

        return self.api_client.request(
            type_=m.V1GetModelsResponse,
            method="GET",
            url="/api/v1/models",
            params=query_params,
        )

    def _build_for_determined_patch_model(
        self, model_name: str, body: m.V1PatchModelRequest
    ) -> Awaitable[m.V1PatchModelResponse]:
        path_params = {"model.name": str(model_name)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PatchModelResponse,
            method="PATCH",
            url="/api/v1/models/{model.name}",
            path_params=path_params,
            json=body,
        )

    def _build_for_determined_post_model(self, model_name: str, body: m.V1Model) -> Awaitable[m.V1PostModelResponse]:
        path_params = {"model.name": str(model_name)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PostModelResponse,
            method="POST",
            url="/api/v1/models/{model.name}",
            path_params=path_params,
            json=body,
        )

    def _build_for_determined_post_model_version(
        self, model_name: str, body: m.V1PostModelVersionRequest
    ) -> Awaitable[m.V1PostModelVersionResponse]:
        path_params = {"modelName": str(model_name)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PostModelVersionResponse,
            method="POST",
            url="/api/v1/models/{modelName}/versions",
            path_params=path_params,
            json=body,
        )


class AsyncModelsApi(_ModelsApi):
    async def determined_get_model(self, model_name: str) -> m.V1GetModelResponse:
        return await self._build_for_determined_get_model(model_name=model_name)

    async def determined_get_model_version(self, model_name: str, model_version: int) -> m.V1GetModelVersionResponse:
        return await self._build_for_determined_get_model_version(model_name=model_name, model_version=model_version)

    async def determined_get_model_versions(
        self, model_name: str, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None
    ) -> m.V1GetModelVersionsResponse:
        return await self._build_for_determined_get_model_versions(
            model_name=model_name, sort_by=sort_by, order_by=order_by, offset=offset, limit=limit
        )

    async def determined_get_models(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        name: str = None,
        description: str = None,
    ) -> m.V1GetModelsResponse:
        return await self._build_for_determined_get_models(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, name=name, description=description
        )

    async def determined_patch_model(self, model_name: str, body: m.V1PatchModelRequest) -> m.V1PatchModelResponse:
        return await self._build_for_determined_patch_model(model_name=model_name, body=body)

    async def determined_post_model(self, model_name: str, body: m.V1Model) -> m.V1PostModelResponse:
        return await self._build_for_determined_post_model(model_name=model_name, body=body)

    async def determined_post_model_version(
        self, model_name: str, body: m.V1PostModelVersionRequest
    ) -> m.V1PostModelVersionResponse:
        return await self._build_for_determined_post_model_version(model_name=model_name, body=body)


class SyncModelsApi(_ModelsApi):
    def determined_get_model(self, model_name: str) -> m.V1GetModelResponse:
        coroutine = self._build_for_determined_get_model(model_name=model_name)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_model_version(self, model_name: str, model_version: int) -> m.V1GetModelVersionResponse:
        coroutine = self._build_for_determined_get_model_version(model_name=model_name, model_version=model_version)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_model_versions(
        self, model_name: str, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None
    ) -> m.V1GetModelVersionsResponse:
        coroutine = self._build_for_determined_get_model_versions(
            model_name=model_name, sort_by=sort_by, order_by=order_by, offset=offset, limit=limit
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_models(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        name: str = None,
        description: str = None,
    ) -> m.V1GetModelsResponse:
        coroutine = self._build_for_determined_get_models(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, name=name, description=description
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_patch_model(self, model_name: str, body: m.V1PatchModelRequest) -> m.V1PatchModelResponse:
        coroutine = self._build_for_determined_patch_model(model_name=model_name, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_post_model(self, model_name: str, body: m.V1Model) -> m.V1PostModelResponse:
        coroutine = self._build_for_determined_post_model(model_name=model_name, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_post_model_version(
        self, model_name: str, body: m.V1PostModelVersionRequest
    ) -> m.V1PostModelVersionResponse:
        coroutine = self._build_for_determined_post_model_version(model_name=model_name, body=body)
        return get_event_loop().run_until_complete(coroutine)
