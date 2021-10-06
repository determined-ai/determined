from determined.common.api.fastapi_client.api.experiments_api import SyncExperimentsApi
from determined.common.api.fapi import ApiClient
from determined.common.api.authentication import Authentication
import argparse
import functools
import json
from typing import Any, Awaitable, Callable, Dict, List, Optional, Type, TypeVar

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
