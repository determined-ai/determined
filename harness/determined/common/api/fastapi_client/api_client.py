from asyncio import get_event_loop
from typing import Any, Awaitable, Callable, Dict, Generic, Type, TypeVar, overload

from determined.common.api.fastapi_client.api.authentication_api import AsyncAuthenticationApi, SyncAuthenticationApi
from determined.common.api.fastapi_client.api.checkpoints_api import AsyncCheckpointsApi, SyncCheckpointsApi
from determined.common.api.fastapi_client.api.cluster_api import AsyncClusterApi, SyncClusterApi
from determined.common.api.fastapi_client.api.commands_api import AsyncCommandsApi, SyncCommandsApi
from determined.common.api.fastapi_client.api.experiments_api import AsyncExperimentsApi, SyncExperimentsApi
from determined.common.api.fastapi_client.api.internal_api import AsyncInternalApi, SyncInternalApi
from determined.common.api.fastapi_client.api.jobs_api import AsyncJobsApi, SyncJobsApi
from determined.common.api.fastapi_client.api.models_api import AsyncModelsApi, SyncModelsApi
from determined.common.api.fastapi_client.api.notebooks_api import AsyncNotebooksApi, SyncNotebooksApi
from determined.common.api.fastapi_client.api.profiler_api import AsyncProfilerApi, SyncProfilerApi
from determined.common.api.fastapi_client.api.shells_api import AsyncShellsApi, SyncShellsApi
from determined.common.api.fastapi_client.api.templates_api import AsyncTemplatesApi, SyncTemplatesApi
from determined.common.api.fastapi_client.api.tensorboards_api import AsyncTensorboardsApi, SyncTensorboardsApi
from determined.common.api.fastapi_client.api.trials_api import AsyncTrialsApi, SyncTrialsApi
from determined.common.api.fastapi_client.api.users_api import AsyncUsersApi, SyncUsersApi
from determined.common.api.fastapi_client.exceptions import ResponseHandlingException, UnexpectedResponse
from httpx import AsyncClient, Request, Response
from pydantic import ValidationError, parse_obj_as

ClientT = TypeVar("ClientT", bound="ApiClient")


class AsyncApis(Generic[ClientT]):
    def __init__(self, client: ClientT):
        self.client = client

        self.authentication_api = AsyncAuthenticationApi(self.client)
        self.checkpoints_api = AsyncCheckpointsApi(self.client)
        self.cluster_api = AsyncClusterApi(self.client)
        self.commands_api = AsyncCommandsApi(self.client)
        self.experiments_api = AsyncExperimentsApi(self.client)
        self.internal_api = AsyncInternalApi(self.client)
        self.jobs_api = AsyncJobsApi(self.client)
        self.models_api = AsyncModelsApi(self.client)
        self.notebooks_api = AsyncNotebooksApi(self.client)
        self.profiler_api = AsyncProfilerApi(self.client)
        self.shells_api = AsyncShellsApi(self.client)
        self.templates_api = AsyncTemplatesApi(self.client)
        self.tensorboards_api = AsyncTensorboardsApi(self.client)
        self.trials_api = AsyncTrialsApi(self.client)
        self.users_api = AsyncUsersApi(self.client)


class SyncApis(Generic[ClientT]):
    def __init__(self, client: ClientT):
        self.client = client

        self.authentication_api = SyncAuthenticationApi(self.client)
        self.checkpoints_api = SyncCheckpointsApi(self.client)
        self.cluster_api = SyncClusterApi(self.client)
        self.commands_api = SyncCommandsApi(self.client)
        self.experiments_api = SyncExperimentsApi(self.client)
        self.internal_api = SyncInternalApi(self.client)
        self.jobs_api = SyncJobsApi(self.client)
        self.models_api = SyncModelsApi(self.client)
        self.notebooks_api = SyncNotebooksApi(self.client)
        self.profiler_api = SyncProfilerApi(self.client)
        self.shells_api = SyncShellsApi(self.client)
        self.templates_api = SyncTemplatesApi(self.client)
        self.tensorboards_api = SyncTensorboardsApi(self.client)
        self.trials_api = SyncTrialsApi(self.client)
        self.users_api = SyncUsersApi(self.client)


T = TypeVar("T")
Send = Callable[[Request], Awaitable[Response]]
MiddlewareT = Callable[[Request, Send], Awaitable[Response]]


class ApiClient:
    def __init__(self, host: str = None, **kwargs: Any) -> None:
        self.host = host
        self.middleware: MiddlewareT = BaseMiddleware()
        self._async_client = AsyncClient(**kwargs)

    @overload
    async def request(
        self, *, type_: Type[T], method: str, url: str, path_params: Dict[str, Any] = None, **kwargs: Any
    ) -> T:
        ...

    @overload  # noqa F811
    async def request(
        self, *, type_: None, method: str, url: str, path_params: Dict[str, Any] = None, **kwargs: Any
    ) -> None:
        ...

    async def request(  # noqa F811
        self, *, type_: Any, method: str, url: str, path_params: Dict[str, Any] = None, **kwargs: Any
    ) -> Any:
        if path_params is None:
            path_params = {}
        url = (self.host or "") + url.format(**path_params)
        request = Request(method, url, **kwargs)
        return await self.send(request, type_)

    @overload
    def request_sync(self, *, type_: Type[T], **kwargs: Any) -> T:
        ...

    @overload  # noqa F811
    def request_sync(self, *, type_: None, **kwargs: Any) -> None:
        ...

    def request_sync(self, *, type_: Any, **kwargs: Any) -> Any:  # noqa F811
        """
        This method is not used by the generated apis, but is included for convenience
        """
        return get_event_loop().run_until_complete(self.request(type_=type_, **kwargs))

    async def send(self, request: Request, type_: Type[T]) -> T:
        response = await self.middleware(request, self.send_inner)
        if response.status_code in [200, 201]:
            try:
                return parse_obj_as(type_, response.json())
            except ValidationError as e:
                raise ResponseHandlingException(e)
        raise UnexpectedResponse.for_response(response)

    async def send_inner(self, request: Request) -> Response:
        try:
            response = await self._async_client.send(request)
        except Exception as e:
            raise ResponseHandlingException(e)
        return response

    def add_middleware(self, middleware: MiddlewareT) -> None:
        current_middleware = self.middleware

        async def new_middleware(request: Request, call_next: Send) -> Response:
            async def inner_send(request: Request) -> Response:
                return await current_middleware(request, call_next)

            return await middleware(request, inner_send)

        self.middleware = new_middleware


class BaseMiddleware:
    async def __call__(self, request: Request, call_next: Send) -> Response:
        return await call_next(request)
