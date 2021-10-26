import argparse
import functools
from enum import Enum
from json import JSONEncoder
from typing import Any, Awaitable, Callable, Dict, List, Optional, Type, TypeVar, Union

from determined.common.api.authentication import cli_auth
from determined.common.api.request import do_request
from determined.common.schemas import SchemaBase

T = TypeVar("T")
Primitives = Union[str, bool, int, float, None]
Jsonable = Union[Primitives, List["Jsonable"], Dict[str, "Jsonable"]]


class ApiClient:
    def __init__(self, host: str = "http://localhost:8080"):
        self.host = host

    def set_host(self, host: str) -> None:
        self.host = host

    async def request(
        self,
        type_: Type[SchemaBase],
        method: str,
        url: str,
        path_params: Optional[Dict[str, Any]] = None,
        **kwargs,
    ) -> Awaitable[SchemaBase]:
        if path_params is None:
            path_params = {}
        url = (self.host or "") + url.format(**path_params)
        response = do_request(method, self.host, url, auth=cli_auth, **kwargs)
        json_val = response.json()
        if hasattr(type_, "from_dict"):
            return type_.from_dict(json_val)
        else:
            return json_val


client = ApiClient()


def set_host(func: Callable[[argparse.Namespace], Any]) -> Callable[..., Any]:
    """
    A decorator for cli functions to set the host (aka master) address.
    """

    @functools.wraps(func)
    def f(namespace: argparse.Namespace) -> Any:
        client.set_host(namespace.master)
        return func(namespace)

    return f


def Field(*args, **kwargs) -> Any:
    alias = kwargs["alias"]

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
    def __init__(self, **kwargs):
        if self.__annotations__ is None:
            return
        cls_attrs = self.__annotations__.keys()
        for attr in cls_attrs:
            attr_getter, _ = self.__getattribute__(attr)
            # pass the input value to validator to compute
            # the default and enforce validations
            val = attr_getter(attr, kwargs.get(attr))
            self.__setattr__(attr, val)

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
        return super(FApiSchemaBase, cls).from_dict(d, prevalidated=True)

    def to_jsonble(self) -> Jsonable:
        d: Dict[str, Jsonable] = {}
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


def to_jsonable(o: Union[Jsonable, FApiSchemaBase]) -> Jsonable:
    if isinstance(o, List):
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
