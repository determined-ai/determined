from abc import ABC, abstractmethod
from collections import UserDict, UserList
from configparser import ConfigParser
from pathlib import Path
from typing import (Any, AnyStr, cast, Generic, IO, Iterable, Mapping, MutableMapping,
                    MutableSequence, NewType, Optional, Type,  Tuple, TypeVar, Union)
import json
import pickle

JSON_INDENT = 2
_TItem = TypeVar('_TItem')
_TKey = TypeVar('_TKey')
_TValue = TypeVar('_TValue')

base_t = Union[MutableMapping, MutableSequence, type(None), int, float, str, bool]


class BaseEnum(ABC):
    """
    Example implementation::

        class Something(BaseEnum):
            A_t = NewType('A', str)
            B_t = NewType('B', str)
            C_t = NewType('C', str)

            A: A_t = A_t.__name__
            B: B_t = B_t.__name__
            C: C_t = C_t.__name__

            all_ = (A, B, C)
            type_ = Union[A_t, B_t, C_t]

    In the following example, the ``run`` function accepts any element from the ``Something``
    enumeration::

        def run(x: Something.type_):
            pass

    In the following example, the ``run`` function accepts a subset of elements from the
    ``Something`` enumerations::

        def run(x: Union[Something.A_t, Something.B_t]:
            pass
    """
    @property
    @abstractmethod
    def type_(self) -> Union[Type[Any]]:
        raise NotImplementedError

    @property
    @abstractmethod
    def all_(self) -> Tuple['BaseEnum.type_']:
        return tuple()


class Format(BaseEnum):
    CONF_t = NewType('conf', str)
    JSON_t = NewType('json', str)
    PICKLE_t = NewType('pkl', str)
    PNG_t = NewType('png', str)
    PYTHON_t = NewType('python', str)
    TXT_t = NewType('txt', str)

    CONF: CONF_t = CONF_t.__name__
    JSON: JSON_t = JSON_t.__name__
    PICKLE: PICKLE_t = PICKLE_t.__name__
    PNG: PNG_t = PNG_t.__name__
    PYTHON: PYTHON_t = PYTHON_t.__name__
    TXT: TXT_t = TXT_t.__name__

    all_ = (CONF, JSON, PICKLE, PNG, PYTHON, TXT)
    type_ = Union[CONF_t, JSON_t, PICKLE_t, PNG_t, PYTHON_t, TXT_t]

    class InvalidFormat(ValueError):
        pass

    @classmethod
    def raise_requested_format_error(cls, *,
                                     msg: str = 'Requested format: "{requested_fmt}"', **kw):
        raise cls.InvalidFormat(msg.format(**kw))

    @classmethod
    def from_path(cls, path: Union[Path, str]) -> 'Format.type_':
        requested_fmt = Path(path).suffix.lstrip('.')
        if requested_fmt in cls.all_:
            return cast(Format.type_, requested_fmt)
        cls.raise_requested_format_error(msg='Unable to determine format from the given path:\n'
                                             '  "{path}". '
                                             'The parsed suffix (if any) was "{requested_fmt}".',
                                         requested_fmt=requested_fmt, path=path)


class BaseObj(ABC):
    data = None
    _path = None

    def __init__(self, *args, **kw):
        super().__init__(*args, **kw)

    @classmethod
    def get_qualname(cls):
        return cls.__qualname__

    @classmethod
    def schema(cls, fmt: Union[Format.PYTHON_t, Format.JSON_t] = Format.PYTHON) -> dict:
        if fmt == Format.PYTHON:
            return cls._schema()
        elif fmt == Format.JSON:
            json.dumps(cls._schema(), indent=JSON_INDENT)
        else:
            Format.raise_requested_format_error(requested_fmt=fmt)

    @classmethod
    def open(cls, path: Union[Path, str]):
        fmt = Format.from_path(path)
        obj = cls(cls._from_file(path, fmt))

        # Save the internal state after successful construction.
        obj._path = Path(path)

        return obj

    def path(self) -> Optional[Path]:
        return self._path

    def set_path(self, path: Union[str, Path]):
        self._path = Path(path)

    def save(self, path: Optional[Union[Path, str]] = None):
        if path is None:
            if self._path is None:
                raise FileNotFoundError('The path to save to has not been set.')
            else:
                path = self._path

        fmt = Format.from_path(path)
        self._to_file(path, fmt)

        # Save the path state only after successful save.
        self._path = Path(path)

    def validate(self):
        import jsonschema
        jsonschema.validate(self, self.schema())

    def get_filename(self, fmt: Format.type_ = Format.JSON,
                     tags: Optional[Iterable[str]] = None) -> Path:
        if tags:
            return Path(''.join((self.get_qualname(), '-', '-'.join(tags), f'.{fmt}')))
        else:
            return Path(''.join((self.get_qualname(), f'.{fmt}')))

    @classmethod
    def from_stream(cls, stream: IO[AnyStr],
                    fmt: Union[Format.CONF_t, Format.JSON_t, Format.PICKLE_t]) -> base_t:
        if fmt == Format.CONF:
            ret_dict = dict()
            cfg = ConfigParser()
            cfg.optionxform = lambda option: option
            cfg.read_file(stream)
            for key, value in cfg.items():
                ret_dict[key] = cls._serialize_to_dict(value)
            return ret_dict
        elif fmt == Format.JSON:
            return json.load(stream)
        elif fmt == Format.PICKLE:
            return pickle.load(stream)
        Format.raise_requested_format_error(requested_fmt=fmt)

    @classmethod
    def from_str(cls, a_str: AnyStr,
                 fmt: Union[Format.CONF_t, Format.JSON_t, Format.PICKLE_t]) -> base_t:
        if fmt == Format.CONF:
            cfg = ConfigParser()
            cfg.read_string(a_str)
            return cfg
        elif fmt == Format.JSON:
            return json.loads(a_str)
        elif fmt == Format.PICKLE:
            return pickle.loads(a_str)
        else:
            Format.raise_requested_format_error(requested_fmt=fmt)

    def to_stream(self, stream: IO[AnyStr], fmt: Union[Format.JSON_t, Format.PICKLE_t]):
        if fmt == Format.JSON:
            json.dump(self.data, stream, indent=JSON_INDENT)
        elif fmt == Format.PICKLE:
            pickle.dump(self.data, stream)
        elif fmt == Format.TXT:
            stream.write(str(self))
        else:
            Format.raise_requested_format_error(requested_fmt=fmt)

    def to_str(self, fmt: Format.type_) -> AnyStr:
        if fmt == Format.JSON:
            return json.dumps(self.data)
        elif fmt == Format.PICKLE:
            return pickle.dumps(self.data)
        else:
            return str(self)

    @classmethod
    def _from_file(cls, path: Union[Path, str],
                   fmt: Union[Format.CONF_t, Format.JSON_t, Format.PICKLE_t]) -> base_t:
        if fmt == Format.CONF:
            with open(path, 'r') as inf:
                return cls.from_stream(inf, fmt)
        elif fmt == Format.JSON:
            with open(path, 'r') as inf:
                return cls.from_stream(inf, fmt)
        elif fmt == Format.PICKLE:
            with open(path, 'rb') as inf:
                return cls.from_stream(inf, fmt)
        Format.raise_requested_format_error(requested_fmt=fmt)

    @staticmethod
    def _serialize_to_dict(value: Mapping) -> dict:
        if isinstance(value, dict):
            return value
        elif isinstance(value, UserDict):
            return value.data
        else:
            return dict(value)

    @staticmethod
    def _serialize_to_list(value: Iterable) -> list:
        if isinstance(value, list):
            return value
        elif isinstance(value, UserList):
            return value.data
        else:
            return list(value)

    @staticmethod
    def _serialize_to_str(value: Union[str, Any]) -> str:
        if type(value) is str:
            return value
        return str(value)

    @staticmethod
    def _serialize_to_int(value: Union[str, int]) -> int:
        if type(value) is int:
            return value
        else:
            # Base 0 means to interpret the base from the string.
            return int(value, 0)

    @staticmethod
    def _serialize_to_float(value: Union[float, int, str]):
        if type(value) is float:
            return value
        else:
            return float(value)

    @staticmethod
    def _serialize_to_bool(value: Union[bool, Any]):
        if type(value) is bool:
            return value
        else:
            return bool(value)

    @classmethod
    def _schema(cls) -> dict:
        return {}

    def _to_file(self, path: Union[Path, str],
                 fmt: Union[Format.JSON_t, Format.TXT_t, Format.PICKLE_t]):
        if fmt == Format.JSON:
            with open(path, 'w') as outf:
                self.to_stream(outf, fmt)
        elif fmt == Format.PICKLE:
            with open(path, 'wb') as outf:
                self.to_stream(outf, fmt)
        elif fmt == Format.TXT:
            with open(path, 'w') as outf:
                self.to_stream(outf, fmt)
        else:
            Format.raise_requested_format_error(requested_fmt=fmt)


class BaseDict(BaseObj, UserDict, ABC, Generic[_TKey, _TValue]):
    class Key(ABC):
        type_: Type = NewType('BaseDict.Key', str)

        # Keys go here, e.g.
        # A_KEY = 'a_key'

        @classmethod
        def all(cls) -> Iterable[type_]:
            yield from ()

    def __init__(self, dict_=None, /, **kw):
        """This is a zero copy constructor.

        :param dict_: If None, a new dictionary will be created as assigned to the``data`` member.
            Otherwise, the ``data`` member will be assigned to ``dict_``, thus sharing the data.

        :param kw: The ``data`` member will be updated with this dictionary.
        """
        if dict_ is None:
            super().__init__()
        else:
            self.data = self._serialize_to_dict(dict_)

        if kw:
            self.update(kw)

    # Add properties for the keys of the dictionary
    # e.g.
    #
    # @property
    # def a_key(self) -> 'SomeDict':
    #     return self[self.Key.A_KEY]

    # @a_key.setter
    # def a_key(self, value: Union[dict, UserDict, 'SomeDict']):
    #     self[self.Key.A_KEY] = value

    @staticmethod
    def _deserialize(key, value) -> Any:
        return value

    @staticmethod
    def _serialize(key, value):
        return value

    def get(self, key, default=None):
        return self._deserialize(key, super().get(key, default))

    def update(self, raw: Optional[Union[dict, Tuple[Tuple[Any, Any]]]] = None, **raw2):
        """
        From dict documentation::

            D.update([E, ]**F) -> None.  Update D from dict/iterable E and F.
            If E is present and has a .keys() method, then does:  for k in E: D[k] = E[k]
            If E is present and lacks a .keys() method, then does:  for k, v in E: D[k] = v
            In either case, this is followed by: for k in F:  D[k] = F[k]
        """
        if raw is not None:
            if hasattr(raw, 'keys'):
                for key in raw.keys():
                    self[key] = self._serialize(key, raw[key])
            else:
                for key, value in raw:
                    self[key] = self._serialize(key, value)
        for key in raw2:
            self[key] = self._serialize(key, raw2[key])

    def __getitem__(self, key):
        """Deserialize from the base types here.

        Example::

         if key == self.Key.A_KEY:
             return SomeDict(super().__getitem__(item))
        """
        return self._deserialize(key, super().__getitem__(key))

    def __setitem__(self, key, value):
        """Serialize to the base types here.

        Example::

            if key == self.Key.A_KEY:
                value =  self._deserialize_to_dict(value)
            super().__setitem__(key, value)
        """
        return super().__setitem__(key, self._serialize(key, value))


class BaseList(BaseObj, UserList, ABC, Generic[_TItem]):
    def __init__(self, initlist: Optional[Iterable] = None):
        """This is a zero copy constructor.

        :param initlist: If None, a new list will be created as assigned to the``data`` member.
            Otherwise, the ``data`` member will be assigned to ``initlist``, thus sharing the data.
        """
        if initlist is None:
            super().__init__()
        else:
            self.data = self._serialize_to_list(initlist)

    def append(self, value: Any) -> None:
        super().append(self._serialize(value))

    def extend(self, iterable: Iterable[Any]):
        for value in iterable:
            self.append(value)

    def insert(self, index: int, value: Any):
        super().insert(index, self._serialize(value))

    @staticmethod
    def _deserialize(value: base_t) -> Any:
        """To be overridden by concrete implementations."""
        return value

    @staticmethod
    def _serialize(value: Any) -> base_t:
        """To be overridden by concrete implementations."""
        return value

    def __getitem__(self, idx: int):
        return self._deserialize(super().__getitem__(idx))

    def __setitem__(self, idx: int, item):
        return super().__setitem__(idx, self._serialize(item))
