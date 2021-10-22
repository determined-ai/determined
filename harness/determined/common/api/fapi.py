import argparse
import functools
import json
from typing import Any, Awaitable, Callable, Dict, List, Optional, Type, TypeVar

from determined.common.api.authentication import Authentication
from determined.common.api.fastapi_client.api.experiments_api import SyncExperimentsApi
from determined.common.api.request import do_request

# TODO fix isinstance isn't returning true
# if hasattr(model_class, 'update_forward_refs'):


T = TypeVar("T")


class ApiClient:
    def __init__(self, host: str = "http://localhost:8080"):
        self.host = host
        self.auth: Optional[Authentication] = None

    # @setter
    def set_auth(self, auth: Authentication):
        self.auth = auth

    async def request(
        self, type_: Type[T], method: str, url: str, path_params: Dict[str, Any] = None, **kwargs
    ) -> Awaitable[T]:
        if path_params is None:
            path_params = {}
        url = (self.host or "") + url.format(**path_params)
        response = do_request(method, self.host, url, auth=self.auth, **kwargs)
        return  type_.from_dict(response.json()) # type: ignore


client = ApiClient(host="http://localhost:8080")
experiments_api = SyncExperimentsApi(client)  # type: ignore


def auth_required(func: Callable[[argparse.Namespace], Any]) -> Callable[..., Any]:
    """
    A decorator for cli functions.
    """

    @functools.wraps(func)
    def f(namespace: argparse.Namespace) -> Any:
        global client
        client.set_auth(Authentication(namespace.master, namespace.user, try_reauth=True))
        return func(namespace)

    return f
