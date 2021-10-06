# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _TemplatesApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_delete_template(self, template_name: str) -> Awaitable[m.Any]:
        path_params = {"templateName": str(template_name)}

        return self.api_client.request(
            type_=m.Any,
            method="DELETE",
            url="/api/v1/templates/{templateName}",
            path_params=path_params,
        )

    def _build_for_determined_get_template(self, template_name: str) -> Awaitable[m.V1GetTemplateResponse]:
        path_params = {"templateName": str(template_name)}

        return self.api_client.request(
            type_=m.V1GetTemplateResponse,
            method="GET",
            url="/api/v1/templates/{templateName}",
            path_params=path_params,
        )

    def _build_for_determined_get_templates(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, name: str = None
    ) -> Awaitable[m.V1GetTemplatesResponse]:
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

        return self.api_client.request(
            type_=m.V1GetTemplatesResponse,
            method="GET",
            url="/api/v1/templates",
            params=query_params,
        )

    def _build_for_determined_put_template(
        self, template_name: str, body: m.V1Template
    ) -> Awaitable[m.V1PutTemplateResponse]:
        path_params = {"template.name": str(template_name)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1PutTemplateResponse,
            method="PUT",
            url="/api/v1/templates/{template.name}",
            path_params=path_params,
            json=body,
        )


class AsyncTemplatesApi(_TemplatesApi):
    async def determined_delete_template(self, template_name: str) -> m.Any:
        return await self._build_for_determined_delete_template(template_name=template_name)

    async def determined_get_template(self, template_name: str) -> m.V1GetTemplateResponse:
        return await self._build_for_determined_get_template(template_name=template_name)

    async def determined_get_templates(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, name: str = None
    ) -> m.V1GetTemplatesResponse:
        return await self._build_for_determined_get_templates(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, name=name
        )

    async def determined_put_template(self, template_name: str, body: m.V1Template) -> m.V1PutTemplateResponse:
        return await self._build_for_determined_put_template(template_name=template_name, body=body)


class SyncTemplatesApi(_TemplatesApi):
    def determined_delete_template(self, template_name: str) -> m.Any:
        coroutine = self._build_for_determined_delete_template(template_name=template_name)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_template(self, template_name: str) -> m.V1GetTemplateResponse:
        coroutine = self._build_for_determined_get_template(template_name=template_name)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_templates(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, name: str = None
    ) -> m.V1GetTemplatesResponse:
        coroutine = self._build_for_determined_get_templates(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, name=name
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_put_template(self, template_name: str, body: m.V1Template) -> m.V1PutTemplateResponse:
        coroutine = self._build_for_determined_put_template(template_name=template_name, body=body)
        return get_event_loop().run_until_complete(coroutine)
