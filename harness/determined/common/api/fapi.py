import argparse
import functools
import json
from typing import Any, Awaitable, Callable, Dict, List, Optional, Type, TypeVar
import jsonpickle


from determined.common.api.authentication import Authentication
from determined.common.api.request import do_request

# TODO fix isinstance isn't returning true
# if hasattr(model_class, 'update_forward_refs'):


T = TypeVar("T")

def jsonable_encoder(d: Any) -> Any:
    return jsonpickle.encode(d, unpicklable=False)

class BaseModel():
    pass

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
        # return jsonpickle.decode(response) # type: ignore
        return response.json() # type: ignore
        # return parse_obj_as(type_, response.json())


# def to_dict(o: BaseModel):
#     rv = o
#     if isinstance(o, List):
#         return [to_dict(i) for i in o]
#     elif hasattr(o, "dict"):
#         rv = o.dict()  # type: Dict[str, Any]
#         if isinstance(o, dict):
#             for k, v in o.items():
#                 rv[k] = to_dict(v)
#     return rv


# def to_json(o: BaseModel):
#     if isinstance(o, List):
#         return [to_json(i) for i in o]
#     assert hasattr(o, "json")
#     return json.loads(o.json())
