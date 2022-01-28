import enum
import numbers
from typing import Any, Callable, Dict, List, Mapping, Optional, Sequence, Type, TypeVar

from determined.common import schemas
from determined.common.schemas import expconf

PRIMITIVE_JSON_TYPES = (numbers.Number, str, bool, type(None))

# `typing` has some awkward APIs for type inspection.  In general, the only thing we can rely on
# across python versions is that annotations are hashable, and equality tests work.  As there is
# only a small, fixed number of types that we actually need to know how to work with, we just build
# some lookup tables.
KNOWN_OPTIONAL_TYPES = {}  # type: Dict[Any, Any]

KNOWN_DICT_TYPES = {}  # type: Dict[Any, Any]

KNOWN_LIST_TYPES = {}  # type: Dict[Any, Any]

KNOWN_CUSTOM_PARSERS = {}  # type: Dict[Any, Callable]

R = TypeVar("R")


def register_known_type(cls: R) -> R:
    KNOWN_OPTIONAL_TYPES[Optional[cls]] = cls
    KNOWN_OPTIONAL_TYPES[Optional[List[cls]]] = List[cls]  # type: ignore
    KNOWN_OPTIONAL_TYPES[Optional[Dict[str, cls]]] = Dict[str, cls]  # type: ignore

    KNOWN_LIST_TYPES[List[cls]] = cls  # type: ignore

    # Note that schemas only support Dict[str, *], since json objects only support string keys.
    KNOWN_DICT_TYPES[Dict[str, cls]] = cls  # type: ignore

    return cls


def register_custom_parser(anno: Any, parse_fn: Callable) -> None:
    KNOWN_CUSTOM_PARSERS[anno] = parse_fn


# Start with registering some basic types.
register_known_type(int)
register_known_type(float)
register_known_type(bool)
register_known_type(str)
register_known_type(Any)


def _to_dict(val: Any, explicit_nones: bool) -> Any:
    """Recurse through an object, calling .to_dict() on all subclasses of SchemaBase."""
    if isinstance(val, SchemaBase):
        return val.to_dict(explicit_nones)
    if isinstance(val, PRIMITIVE_JSON_TYPES):
        return val
    if isinstance(val, enum.Enum):
        return val.value
    if isinstance(val, Mapping):
        return {k: _to_dict(v, explicit_nones) for k, v in val.items()}
    if isinstance(val, Sequence):
        return [_to_dict(i, explicit_nones) for i in val]
    raise ValueError(f"invalid type in _to_dict: {type(val).__name__}")


def _fill_defaults(val: Any) -> None:
    """Recurse through an object, calling .fill_defaults() on all subclasses of SchemaBase."""
    if isinstance(val, SchemaBase):
        val.fill_defaults()
        return
    if isinstance(val, enum.Enum):
        return
    if isinstance(val, PRIMITIVE_JSON_TYPES):
        return
    if isinstance(val, Mapping):
        for v in val.values():
            _fill_defaults(v)
        return
    if isinstance(val, Sequence):
        for v in val:
            _fill_defaults(v)
        return
    raise ValueError(f"invalid type in _fill_defaults: {type(val).__name__}")


def _copy(val: Any) -> Any:
    """Recurse through an object, calling .copy() on all subclasses of SchemaBase."""
    if isinstance(val, SchemaBase):
        return val.copy()
    if isinstance(val, PRIMITIVE_JSON_TYPES):
        return val
    if isinstance(val, enum.Enum):
        return type(val)(val.value)
    if isinstance(val, Mapping):
        return {k: _copy(v) for k, v in val.items()}
    if isinstance(val, Sequence):
        return [_copy(i) for i in val]
    raise ValueError(f"invalid type in _copy: {type(val).__name__}")


def _merge(obj: Any, src: Any) -> Any:
    """Recursively merge two objects and return the result"""
    if src is None:
        return obj
    if obj is None:
        return src
    if type(obj) is not type(src):
        raise AssertionError("merge must be called with matching types")
    if isinstance(obj, SchemaBase):
        return obj.merge(src)
    if isinstance(obj, enum.Enum):
        return obj
    if isinstance(obj, PRIMITIVE_JSON_TYPES):
        return obj
    if isinstance(obj, Mapping):
        src.update(obj)
        return src
    if isinstance(obj, Sequence):
        return obj
    raise ValueError(f"invalid type in merge: {type(obj).__name__}")


def _instance_from_annotation(anno: type, value: Any, prevalidated: bool = False) -> Any:
    """
    During calls to .from_dict(), use the type annotation to create a new object from value.
    """
    if anno == Any:
        # In the special case of typing.Any, we just return the value directly.
        return value

    # Handle Optionals (strip the Optional part).
    if anno in KNOWN_OPTIONAL_TYPES:
        anno = KNOWN_OPTIONAL_TYPES[anno]
    elif Optional[anno] == anno:
        raise TypeError(f"unrecognized Optional ({anno}), maybe use @schemas.register_known_type?")

    # Detect List[*] types, where issubclass(x, List) is unsafe.
    if anno in KNOWN_LIST_TYPES:
        subanno = KNOWN_LIST_TYPES[anno]
        if value is None:
            return None
        if not isinstance(value, Sequence):
            raise TypeError(f"unable to create instance of {anno} from {value}")
        return [_instance_from_annotation(subanno, v, prevalidated) for v in value]

    # Detect Dict[*] types, where issubclass(x, Dict) is unsafe.
    if anno in KNOWN_DICT_TYPES:
        subanno = KNOWN_DICT_TYPES[anno]
        if value is None:
            return None
        if not isinstance(value, Mapping):
            raise TypeError(f"unable to create instance of {anno} from {value}")
        return {k: _instance_from_annotation(subanno, v, prevalidated) for k, v in value.items()}

    # Detect custom types which have hand-written parsers.
    if anno in KNOWN_CUSTOM_PARSERS:
        parse_fn = KNOWN_CUSTOM_PARSERS[anno]
        return parse_fn(value, prevalidated)

    # Detect Union[*] types and convert them to their UnionBase class.
    if anno in schemas.UnionBase._union_types:
        anno = schemas.UnionBase._union_types[anno]

    # Any valid annotations must be plain types by now, which will allow us to use issubclass().
    if not isinstance(anno, type):
        raise TypeError(
            f"invalid compound annotation {anno}, maybe use @schemas.register_known_type?"
        )

    if issubclass(anno, enum.Enum):
        return anno(value)
    if issubclass(anno, SchemaBase):
        # For subclasses of SchemaBase we just call either from_dict() or from_none().
        if value is None:
            return anno.from_none()
        return anno.from_dict(value, prevalidated)
    if issubclass(anno, PRIMITIVE_JSON_TYPES):
        # For json literal types, we just include them directly.
        return value

    raise TypeError(f"invalid type annotation on SchemaBase object: {anno}")


T = TypeVar("T", bound="SchemaBase")


class SchemaBaseMeta(type):
    """
    SchemaBaseMeta simply marks registers all SchemaBase objects with KNOWN_OPTIONAL_TYPES.
    """

    def __new__(cls, *arg: List[Any]) -> Any:
        x = super().__new__(cls, *arg)
        register_known_type(x)
        return x


class SchemaBase(metaclass=SchemaBaseMeta):
    _id: str

    def __init__(self, *args: list, **kwargs: dict) -> None:
        raise NotImplementedError(f"{type(self).__name__} must not be instantiated")

    @classmethod
    def from_none(cls: Type[T]) -> Optional[T]:
        """
        from_none is called inside from_dict, when a key is present as a literal None.

        For most objects (ResourcesConfig, for example), a None value means it is not present.

        However, some values (Hyperparameter, for example), a None value represents a real object.
        This classmethod makes it possible to customize behavior in those situations.
        """
        return None

    @classmethod
    def from_dict(cls: Type[T], d: dict, prevalidated: bool = False) -> T:
        if not isinstance(d, dict) or any(not isinstance(k, str) for k in d):
            raise ValueError("from_dict() requires an input dictionary with only string keys")

        # Validate before parsing.
        if not prevalidated:
            errors = expconf.sanity_validation_errors(d, cls._id)
            if errors:
                raise TypeError(f"incorrect {cls.__name__}:\n" + "\n".join(errors))

        init_args = {}

        # For every key in the dictionary, get the type from the class annotations.  If it is a
        # sublcass of SchemaBase, call from_dict() or from_none() on it based on the value in the
        # input.  Otherwise, make sure a primitive type and pass the value to __init__ directly.
        for name, value in d.items():
            # Special case: drop keys which match the _union_key value of the class.
            if name == getattr(cls, "_union_key", None):
                continue
            anno = cls.__annotations__.get(name)
            if anno is None:
                raise TypeError(
                    f"{cls.__name__}.from_dict() found a key '{name}' input which has no "
                    "annotation.  This is a  bug; all SchemaBase subclasses must have annotations "
                    "which match the json schema definitions which they correspond to."
                )
            # Create an instance based on the type annotation.
            init_args[name] = _instance_from_annotation(anno, value, prevalidated=True)

        return cls(**init_args)

    @classmethod
    def property_names(cls) -> List[str]:
        return [name for name in cls.__annotations__ if not name.startswith("_")]

    def to_dict(self, explicit_nones: bool = False) -> dict:
        if explicit_nones:
            # Iterate through all annotations.
            d = {k: _to_dict(getattr(self, k), explicit_nones) for k in self.property_names()}
        else:
            # Iterate through all defined values.
            d = {k: _to_dict(v, explicit_nones) for k, v in vars(self).items()}
        if hasattr(self, "_union_key"):
            d[self._union_key] = self._union_id  # type: ignore
        return d

    def fill_defaults(self) -> None:
        # Create any non-present child objects.
        for name, anno in self.__annotations__.items():
            # Ignore special annotations.
            if name.startswith("_"):
                continue
            # Ignore already-set values.
            if vars(self).get(name) is not None:
                continue

            # Get the default value.
            default_json = expconf.get_default(self._id, name)

            # Create an instance based on the type annotation.
            default = _instance_from_annotation(anno, default_json, prevalidated=False)

            if default is None:
                continue

            setattr(self, name, default)

        # Recurse into all child objects.
        for value in vars(self).values():
            _fill_defaults(value)

        # Finally, set any runtime defaults.
        self.runtime_defaults()

    def runtime_defaults(self) -> None:
        """
        runtime_defaults is called at the end of SchemaBase.fill_defaults(), where values which are
        filled out at runtime can be populated dynamically.

        Only a few classes define this, like the ExperimentConfig (the name) and
        ReproducibilityConfig (the experiment seed).
        """
        pass

    def copy(self: T) -> T:
        return type(self)(**{k: _copy(v) for k, v in vars(self).items()})

    def merge(self, src: T) -> None:
        if type(src) is not type(self):
            raise AssertionError("merge must be called with matching types")
        src.assert_sane()
        for name, src_value in vars(src).items():
            obj_value = vars(self).get(name)
            merged_value = _merge(obj_value, src_value)
            if merged_value is not None:
                setattr(self, name, merged_value)

    def assert_sane(self) -> None:
        errors = expconf.sanity_validation_errors(self.to_dict(), self._id)
        if errors:
            raise AssertionError(f"incorrect {type(self).__name__}:\n" + "\n".join(errors))

    def assert_complete(self) -> None:
        errors = expconf.completeness_validation_errors(self.to_dict(), self._id)
        if errors:
            raise TypeError(f"incorrect {type(self).__name__}:\n" + "\n".join(errors))

    def __eq__(self, other: object) -> bool:
        if type(self) != type(other):
            return False
        for name in self.property_names():
            if getattr(self, name) != getattr(other, name):
                return False
        return True
