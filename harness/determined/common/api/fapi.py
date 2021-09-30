from determined.common.api.authentication import Authentication, cli_auth
import functools
from typing import Any, Callable, Awaitable, List, Dict
import argparse
from determined.common.api.fastapi_client import SyncApis, ApiClient, AsyncApis
from determined.common.api.fastapi_client.api_client import Send
from httpx import Request, Response
from pydantic import BaseModel # FIXME this doesn't get resolved in my IDE's language server
import json
# from determined.common.api.fastapi_client.api_client import ApiClient, AsyncApis, SyncApis
# from determined.common.api.fastapi_client.models import Pet

# TODO fix isinstance isn't returning true
# if hasattr(model_class, 'update_forward_refs'):

client = ApiClient(host="http://localhost:8080")
# client._async_client.aclose()
sync_apis = SyncApis(client)
# async_apis = fa.AsyncApis(client)

# resp = sync_apis.authentication_api.determined_login(V1LoginRequest(username='determined', password=''))
# print(resp)


def add_token(token: str):
    def f(req: Request, send: Send) -> Awaitable[Response]:
        req.headers['Authorization'] = 'Bearer ' + token
        return send(req)
    return f

def to_dict(o: BaseModel):
    rv = o
    if isinstance(o, List):
        return [to_dict(i) for i in o]
    elif hasattr(o, 'dict'):
        rv = o.dict() # type: Dict[str, Any]
        if isinstance(o, dict):
            for k, v in o.items():
                rv[k] = to_dict(v)
    return rv

def to_json(o: BaseModel):
    if isinstance(o, List):
        return [to_json(i) for i in o]
    assert hasattr(o, 'json')
    return json.loads(o.json())

def auth_required(func: Callable[[argparse.Namespace], Any]) -> Callable[..., Any]:
    """
    A decorator for cli functions.
    """

    @functools.wraps(func)
    def f(namespace: argparse.Namespace) -> Any:
        global cli_auth, sync_apis
        client = ApiClient(host=namespace.master)
        cli_auth = Authentication(namespace.master, namespace.user, try_reauth=True)
        token = cli_auth.get_session_token()
        client.add_middleware(add_token(token))
        sync_apis = SyncApis(client)

        # TODO avoid global?
        return func(namespace)

    return f
