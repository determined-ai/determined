import argparse
import functools
from enum import Enum
from typing import Any, Awaitable, Callable, Dict, Optional, Type, TypeVar
from typing import Any, Dict, List, Type, TypeVar, Optional, Union  # , get_origin new in 3.8
from determined.common.schemas import SchemaBase
from json import JSONEncoder

from determined.common.api.authentication import Authentication
from determined.common.api.request import do_request


T = TypeVar("T")

class ApiClient:
    def __init__(self, host: str = "http://localhost:8080"):
        self.host = host
        self.auth: Optional[Authentication] = None

    def set_auth(self, auth: Authentication):
        self.auth = auth

    async def request(
        self, type_: Type[T], method: str, url: str, path_params: Dict[str, Any] = None, **kwargs
    ) -> Awaitable[T]:
        if path_params is None:
            path_params = {}
        url = (self.host or "") + url.format(**path_params)
        response = do_request(method, self.host, url, auth=self.auth, **kwargs)
        json_val = response.json()
        if hasattr(type_, 'from_dict'):
            return  type_.from_dict(json_val) # type: ignore
        else:
            return json_val


client = ApiClient(host="http://localhost:8080")

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


def Field(*args, **kwargs) -> Any:
    alias = kwargs['alias']

    def validator(name, val) -> Any:
        default = args[0]
        if val is None:
            if default is not Ellipsis:
                return default
            else:
                raise AttributeError(f"missing required param {name}")
        return val

    return (validator, alias)


T = TypeVar("T", bound="FApiSchemaBase")
class FApiSchemaBase(SchemaBase):
    def __init__(self, *args, **kwargs):
        if self.__annotations__ is None:
            return
        cls_attrs = self.__annotations__.keys()
        # print('args', kwargs)
        # print(self.__class__.__name__)
        for attr in cls_attrs:
            attr_getter, _ = self.__getattribute__(attr)
            # pass the input value to validator to compute
            # the default and enforce validations
            val = attr_getter(attr, kwargs.get(attr))
            self.__setattr__(attr, val)
        pass

    @classmethod
    def attr_aliases(cls) -> Dict[str, str]:
        """
        return a dict mapping from api to python repr of key names.
        """
        cls_attrs = cls.__annotations__.keys()
        aliases: Dict[str, str] = {}
        for attr in cls_attrs:
            _, alias = cls.__getattribute__(cls, attr)
            aliases[alias] = attr
        return aliases

    @classmethod
    def translate_dict(cls, d: Dict[str, Any]) -> Dict[str, Any]:
        aliases = cls.attr_aliases()
        new_d = {}
        for key in d.keys():
            new_d[aliases[key]] = d[key]
        return new_d

    @classmethod
    def from_dict(cls: Type[T], d: dict, camelCase: bool = True) -> T:
        if camelCase:
            d = cls.translate_dict(d)
        return super().from_dict(d, prevalidated=True)

    def to_jsonble(self):
        d: Dict[str, Union[str, Dict, List]] = {}
        aliases = self.attr_aliases()
        for json_key, py_key in aliases.items():
            val = self.__getattribute__(py_key)
            d[json_key] = to_jsonable(val)
        return d


class MyEncoder(JSONEncoder):
        def default(self, o):
            if isinstance(o, FApiSchemaBase):
                return super().default(o.to_jsonble())
            return super().default(o)


def to_jsonable(o: Union[Any, List[Any], Dict[str, Any], FApiSchemaBase]):
    if isinstance(o, List):  # FIXME is this enough?
        return [to_jsonable(i) for i in o]
    if isinstance(o, Dict):
        for k, v in o.items():
            o[k] = to_jsonable(v)
    if isinstance(o, FApiSchemaBase):
        return o.to_jsonble()
    if isinstance(o, Enum):
        return o.value
    return o

BaseModel = FApiSchemaBase
