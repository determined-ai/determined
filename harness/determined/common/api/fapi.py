from determined.common.api.authentication import Authentication, cli_auth
import functools
from typing import Any, Callable, Awaitable
import argparse
from determined.common.api.fastapi_client import SyncApis, ApiClient, AsyncApis
from determined.common.api.fastapi_client.api_client import Send
from httpx import Request, Response
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


# resp = sync_apis.cluster_api.determined_get_master()
# print(resp)


# fa.models.V1GetMasterResponse.update_forward_refs()

# pet_1 = sync_apis.pet_api.get_pet_by_id(pet_id=1)
# assert isinstance(pet_1, Pet)

# resp = sync_apis.cluster_api.determined_get_master_config()
# print(resp)

# async def get_pet_2() -> Pet:
#     pet_2 = await async_apis.pet_api.get_pet_by_id(pet_id=2)
#     assert isinstance(pet_2, Pet)
#     return pet_2

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
