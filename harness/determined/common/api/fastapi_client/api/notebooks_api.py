# flake8: noqa E501
from asyncio import get_event_loop
from typing import TYPE_CHECKING, Awaitable, List

from determined.common.api.fastapi_client import models as m
from determined.common.api.fapi import jsonable_encoder

if TYPE_CHECKING:
    from determined.common.api.fastapi_client.api_client import ApiClient


class _NotebooksApi:
    def __init__(self, api_client: "ApiClient"):
        self.api_client = api_client

    def _build_for_determined_get_notebook(self, notebook_id: str) -> Awaitable[m.V1GetNotebookResponse]:
        path_params = {"notebookId": str(notebook_id)}

        return self.api_client.request(
            type_=m.V1GetNotebookResponse,
            method="GET",
            url="/api/v1/notebooks/{notebookId}",
            path_params=path_params,
        )

    def _build_for_determined_get_notebooks(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> Awaitable[m.V1GetNotebooksResponse]:
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
            type_=m.V1GetNotebooksResponse,
            method="GET",
            url="/api/v1/notebooks",
            params=query_params,
        )

    def _build_for_determined_kill_notebook(self, notebook_id: str) -> Awaitable[m.V1KillNotebookResponse]:
        path_params = {"notebookId": str(notebook_id)}

        return self.api_client.request(
            type_=m.V1KillNotebookResponse,
            method="POST",
            url="/api/v1/notebooks/{notebookId}/kill",
            path_params=path_params,
        )

    def _build_for_determined_launch_notebook(
        self, body: m.V1LaunchNotebookRequest
    ) -> Awaitable[m.V1LaunchNotebookResponse]:
        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1LaunchNotebookResponse, method="POST", url="/api/v1/notebooks", json=body
        )

    def _build_for_determined_notebook_logs(
        self, notebook_id: str, offset: int = None, limit: int = None, follow: bool = None
    ) -> Awaitable[m.StreamResultOfV1NotebookLogsResponse]:
        path_params = {"notebookId": str(notebook_id)}

        query_params = {}
        if offset is not None:
            query_params["offset"] = str(offset)
        if limit is not None:
            query_params["limit"] = str(limit)
        if follow is not None:
            query_params["follow"] = str(follow)

        return self.api_client.request(
            type_=m.StreamResultOfV1NotebookLogsResponse,
            method="GET",
            url="/api/v1/notebooks/{notebookId}/logs",
            path_params=path_params,
            params=query_params,
        )

    def _build_for_determined_set_notebook_priority(
        self, notebook_id: str, body: m.V1SetNotebookPriorityRequest
    ) -> Awaitable[m.V1SetNotebookPriorityResponse]:
        path_params = {"notebookId": str(notebook_id)}

        body = jsonable_encoder(body)

        return self.api_client.request(
            type_=m.V1SetNotebookPriorityResponse,
            method="POST",
            url="/api/v1/notebooks/{notebookId}/set_priority",
            path_params=path_params,
            json=body,
        )


class AsyncNotebooksApi(_NotebooksApi):
    async def determined_get_notebook(self, notebook_id: str) -> m.V1GetNotebookResponse:
        return await self._build_for_determined_get_notebook(notebook_id=notebook_id)

    async def determined_get_notebooks(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> m.V1GetNotebooksResponse:
        return await self._build_for_determined_get_notebooks(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, users=users
        )

    async def determined_kill_notebook(self, notebook_id: str) -> m.V1KillNotebookResponse:
        return await self._build_for_determined_kill_notebook(notebook_id=notebook_id)

    async def determined_launch_notebook(self, body: m.V1LaunchNotebookRequest) -> m.V1LaunchNotebookResponse:
        return await self._build_for_determined_launch_notebook(body=body)

    async def determined_notebook_logs(
        self, notebook_id: str, offset: int = None, limit: int = None, follow: bool = None
    ) -> m.StreamResultOfV1NotebookLogsResponse:
        return await self._build_for_determined_notebook_logs(
            notebook_id=notebook_id, offset=offset, limit=limit, follow=follow
        )

    async def determined_set_notebook_priority(
        self, notebook_id: str, body: m.V1SetNotebookPriorityRequest
    ) -> m.V1SetNotebookPriorityResponse:
        return await self._build_for_determined_set_notebook_priority(notebook_id=notebook_id, body=body)


class SyncNotebooksApi(_NotebooksApi):
    def determined_get_notebook(self, notebook_id: str) -> m.V1GetNotebookResponse:
        coroutine = self._build_for_determined_get_notebook(notebook_id=notebook_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_get_notebooks(
        self, sort_by: str = None, order_by: str = None, offset: int = None, limit: int = None, users: List[str] = None
    ) -> m.V1GetNotebooksResponse:
        coroutine = self._build_for_determined_get_notebooks(
            sort_by=sort_by, order_by=order_by, offset=offset, limit=limit, users=users
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_kill_notebook(self, notebook_id: str) -> m.V1KillNotebookResponse:
        coroutine = self._build_for_determined_kill_notebook(notebook_id=notebook_id)
        return get_event_loop().run_until_complete(coroutine)

    def determined_launch_notebook(self, body: m.V1LaunchNotebookRequest) -> m.V1LaunchNotebookResponse:
        coroutine = self._build_for_determined_launch_notebook(body=body)
        return get_event_loop().run_until_complete(coroutine)

    def determined_notebook_logs(
        self, notebook_id: str, offset: int = None, limit: int = None, follow: bool = None
    ) -> m.StreamResultOfV1NotebookLogsResponse:
        coroutine = self._build_for_determined_notebook_logs(
            notebook_id=notebook_id, offset=offset, limit=limit, follow=follow
        )
        return get_event_loop().run_until_complete(coroutine)

    def determined_set_notebook_priority(
        self, notebook_id: str, body: m.V1SetNotebookPriorityRequest
    ) -> m.V1SetNotebookPriorityResponse:
        coroutine = self._build_for_determined_set_notebook_priority(notebook_id=notebook_id, body=body)
        return get_event_loop().run_until_complete(coroutine)
