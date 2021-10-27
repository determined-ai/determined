# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable, List

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import to_jsonable as jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fapi import ApiClient


class _ModelsApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_get_model(self, model_id: int) -> Awaitable[m.V1GetModelResponse]:
        path_params = {"modelId": str(model_id)}

        return self.api_client.request(
            type_=m.V1GetModelResponse,
            method="GET",
            url="/api/v1/models/{modelId}",
            path_params=path_params,
        )

    def _build_for_get_model_version(self, model_id: str, model_version: int) -> Awaitable[m.V1GetModelVersionResponse]:
        path_params = {"modelId": str(model_id), "modelVersion": str(model_version)}

        return self.api_client.request(
            type_=m.V1GetModelVersionResponse,
            method="GET",
            url="/api/v1/models/{modelId}/versions/{modelVersion}",
            path_params=path_params,
        )

    def _build_for_get_model_versions(
        self, model_id: int, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None
    ) -> Awaitable[m.V1GetModelVersionsResponse]:
        path_params = {"modelId": str(model_id)}

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
            url="/api/v1/models/{modelId}/versions",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_get_models(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        name: str = None,
        description: str = None,
        labels: List[str] = None,
        archived: bool = None,
        users: List[str] = None,
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
        if labels is not None:
            query_params["labels"] = [str(labels_item) for labels_item in labels]
        if archived is not None:
            query_params["archived"] = str(archived)
        if users is not None:
            query_params["users"] = [str(users_item) for users_item in users]

        return self.api_client.request(
            type_=m.V1GetModelsResponse,
            method="GET",
            url="/api/v1/models",
            params=query_params,
        )

    def _build_for_patch_model(self, model_id: int, body: m.V1PatchModelRequest) -> Awaitable[m.V1PatchModelResponse]:
        path_params = {"model.id": str(model_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PatchModelResponse,
            method="PATCH",
            url="/api/v1/models/{model.id}",
            path_params=path_params,
            json=body,
        )

    def _build_for_post_model(self, body: m.V1Model) -> Awaitable[m.V1PostModelResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(type_=m.V1PostModelResponse, method="POST", url="/api/v1/models", json=body)

    def _build_for_post_model_version(
        self, model_id: int, body: m.V1PostModelVersionRequest
    ) -> Awaitable[m.V1PostModelVersionResponse]:
        path_params = {"modelId": str(model_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PostModelVersionResponse,
            method="POST",
            url="/api/v1/models/{modelId}/versions",
            path_params=path_params,
            json=body,
        )


class AsyncModelsApi(_ModelsApi):
    async def get_model(self, model_id: int) -> m.V1GetModelResponse:
        return await self._build_for_get_model(model_id=model_id)

    async def get_model_version(self, model_id: str, model_version: int) -> m.V1GetModelVersionResponse:
        return await self._build_for_get_model_version(model_id=model_id, model_version=model_version)

    async def get_model_versions(
        self, model_id: int, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None
    ) -> m.V1GetModelVersionsResponse:
        return await self._build_for_get_model_versions(
            model_id=model_id, sort_by=sort_by, order_by=order_by, offset=offset, limit=limit
        )

    async def get_models(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        name: str = None,
        description: str = None,
        labels: List[str] = None,
        archived: bool = None,
        users: List[str] = None,
    ) -> m.V1GetModelsResponse:
        return await self._build_for_get_models(
            sort_by=sort_by,
            order_by=order_by,
            offset=offset,
            limit=limit,
            name=name,
            description=description,
            labels=labels,
            archived=archived,
            users=users,
        )

    async def patch_model(self, model_id: int, body: m.V1PatchModelRequest) -> m.V1PatchModelResponse:
        return await self._build_for_patch_model(model_id=model_id, body=body)

    async def post_model(self, body: m.V1Model) -> m.V1PostModelResponse:
        return await self._build_for_post_model(body=body)

    async def post_model_version(
        self, model_id: int, body: m.V1PostModelVersionRequest
    ) -> m.V1PostModelVersionResponse:
        return await self._build_for_post_model_version(model_id=model_id, body=body)


class SyncModelsApi(_ModelsApi):
    def get_model(self, model_id: int) -> m.V1GetModelResponse:
        coroutine = self._build_for_get_model(model_id=model_id)
        return get_event_loop().run_until_complete(coroutine)

    def get_model_version(self, model_id: str, model_version: int) -> m.V1GetModelVersionResponse:
        coroutine = self._build_for_get_model_version(model_id=model_id, model_version=model_version)
        return get_event_loop().run_until_complete(coroutine)

    def get_model_versions(
        self, model_id: int, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None
    ) -> m.V1GetModelVersionsResponse:
        coroutine = self._build_for_get_model_versions(
            model_id=model_id, sort_by=sort_by, order_by=order_by, offset=offset, limit=limit
        )
        return get_event_loop().run_until_complete(coroutine)

    def get_models(
        self,
        sort_by: str = None,
        order_by: str = None,
        offset: int = None,
        limit: int = None,
        name: str = None,
        description: str = None,
        labels: List[str] = None,
        archived: bool = None,
        users: List[str] = None,
    ) -> m.V1GetModelsResponse:
        coroutine = self._build_for_get_models(
            sort_by=sort_by,
            order_by=order_by,
            offset=offset,
            limit=limit,
            name=name,
            description=description,
            labels=labels,
            archived=archived,
            users=users,
        )
        return get_event_loop().run_until_complete(coroutine)

    def patch_model(self, model_id: int, body: m.V1PatchModelRequest) -> m.V1PatchModelResponse:
        coroutine = self._build_for_patch_model(model_id=model_id, body=body)
        return get_event_loop().run_until_complete(coroutine)

    def post_model(self, body: m.V1Model) -> m.V1PostModelResponse:
        coroutine = self._build_for_post_model(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def post_model_version(self, model_id: int, body: m.V1PostModelVersionRequest) -> m.V1PostModelVersionResponse:
        coroutine = self._build_for_post_model_version(model_id=model_id, body=body)
        return get_event_loop().run_until_complete(coroutine)
