import enum
import functools
import inspect
import json
import numbers
import typing
from typing import (
    Any,
    Callable,
    Dict,
    List,
    Mapping,
    Optional,
    Sequence,
    Tuple,
    Type,
    TypeVar,
    cast,
)

from determined_common.schemas import expconf

PRIMITIVE_JSON_TYPES = (numbers.Number, str, bool, type(None))


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


def _remove_optional(anno: Any) -> Any:
    """Given a type annotation, which might be TYPE or Optional[TYPE], return TYPE."""
    if type(anno) is not typing._Union:  # type: ignore
        return anno
    args = list(anno.__args__)
    if type(None) in args:
        args.remove(type(None))
    if len(args) != 1:
        return type(anno)(args)
    return args[0]


def _handle_unions(anno: type) -> type:
    if type(anno) is not typing._Union:  # type: ignore
        return anno
    args = list(anno.__args__)  # type: ignore
    args = cast(List[type], args)
    # Strip any Nones, which indicate Optionals.
    if type(None) in args:
        args.remove(type(None))
    if len(args) > 1:
        # Named unions are instantiated using their associated UnionBase's from_dict() method.
        named = UnionBase._union_types.get(frozenset(args))
        if named is None:
            raise TypeError(f"no named union for {args}")
        return named
    # Normal Optional[some_type] reduce to just some_type.
    return args[0]


def _instance_from_annotation(anno: type, dict_value: Any, prevalidated: bool = False) -> Any:
    # All Union types reduce to some other type.  In the case of our union schemas, like
    # hyperparameters, that other type may be partially determined by dict_value.
    typ = _handle_unions(anno)

    if typ == typing.Any:
        # In the special case of typing.Any, we just return the dict_value directly.
        return dict_value
    if issubclass(typ, enum.Enum):
        return typ(dict_value)
    if issubclass(typ, SchemaBase):
        # For subclasses of SchemaBase we just call either from_dict() or from_none().
        if dict_value is None:
            return typ.from_none()
        return typ.from_dict(dict_value, prevalidated)
    if issubclass(typ, PRIMITIVE_JSON_TYPES):
        # For json literal types, we just include them directly.
        return dict_value
    if issubclass(typ, typing.List):
        # List[thing] annotations; create a list of things.
        args = typ.__args__  # type: ignore
        args = cast(List[type], args)
        if len(args) != 1:
            raise TypeError("got typing.List[] without any element type")
        if dict_value is None:
            return None
        if not isinstance(dict_value, typing.Sequence):
            raise TypeError(f"unable to create instance of {typ} from {dict_value}")
        return [_instance_from_annotation(args[0], dv, prevalidated) for dv in dict_value]
    if issubclass(typ, typing.Dict):
        # Dict[str, thing] annotations; create a dict of strings to things.
        args = typ.__args__  # type: ignore
        args = cast(List[type], args)
        if len(args) != 2:
            raise TypeError("got typing.Dict[] without any element type")
        if args[0] != str:
            raise TypeError("got typing.Dict[] without a string as the first type")
        if dict_value is None:
            return None
        if not isinstance(dict_value, typing.Mapping):
            raise TypeError(f"unable to create instance of {typ} from {dict_value}")
        return {
            k: _instance_from_annotation(args[1], v, prevalidated) for k, v in dict_value.items()
        }
    raise TypeError(f"invalid type annotation on SchemaBase object: {anno}")


def auto_init(old_init: Callable) -> Callable:
    """
    auto_init is a decorator for an __init__ which uses setattr() to set values in __init__ based
    on the signature of the __init__ function.

    Check out this class:

        class Thing:
            a: int
            b: Optional[TestSub] = None

            @auto_init
            def __init__(
                self,
                a: int,
                b: Optional[TestSub] = None,
            ):
                ## This is effectively what happens magically due to @auto_init
                # if a is not None:
                #     self.a = a
                # if b is not None:
                #     self.b = b
                pass

    The simpler strategy would be to skip the annotations at the Class level and just set values
    in __init__.  However, using @auto_init has several benefits:

      - By relying on class annotations for default values, you can always call thing.a, but you
        can also use `"a" in vars(thing)` to know if the value was set explicitly or not.

      - The annotations are easily recognized by type-aware systems for linting or tab-completion.

      - Listing out the types in the signature of __init__() is not actually necessary (it could
        be inferred from the annotations) but for type-awarness and tab-completion systems it is
        necessary.  (side note: you don't need explicit __init__ definitions for @dataclass
        classes when working with mypy, but that's because mypy special-cases them.)

      - Given the previous point, the annotations and the signature can easily be kept in perfect
        sync with each other, but enforcing the synchronization between the __init__ signature and
        the body of __init__ would be extremely difficult in a large body of evolving configs.

    Also see the AutoInit metaclass for an *even easier* way to get the benefits of @auto_init.
    """

    old_sig = inspect.signature(old_init)

    @functools.wraps(old_init)
    def set_all_attrs(self: T, *args: list, **kwargs: dict) -> None:
        if args:
            raise TypeError("only use keyword arguments")

        kw = dict(old_sig.bind(self, *args, **kwargs).arguments)
        del kw["self"]
        for k, v in kw.items():
            setattr(self, k, v)

        # Always call the old __init__ in case there is anything useful in there.
        old_init(self, **kwargs)

    return set_all_attrs


T = TypeVar("T", bound="SchemaBase")


class SchemaBase:
    _id: str

    def __init__(self, **kwargs: dict) -> None:
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
            errors = expconf.validation_errors(d, cls._id)
            if errors:
                raise TypeError(f"incorrect {cls.__name__}:\n" + "\n".join(errors))

        additional_properties_anno = cls.__annotations__.get("_additional_properties")

        init_args = {}

        # For every key in the dictionary, get the type from the class annotations.  If it is a
        # sublcass of SchemaBase, call from_dict() or from_none() on it based on the value in the
        # input.  Otherwise, make sure a primitive type and pass the value to __init__ directly.
        for name, value in d.items():
            anno = cls.__annotations__.get(name, additional_properties_anno)
            if anno is None:
                # XXX: remove this continue
                continue
                raise TypeError(
                    f"{cls.__name__}.from_dict() found a key '{name}' input which has no "
                    "annotation.  This is a  bug; all SchemaBase subclasses must have annotations "
                    "which match the json schema definitions which they correspond to."
                )
            # Create an instance based on the type annotation.
            init_args[name] = _instance_from_annotation(anno, value, prevalidated=True)

        return cls(**init_args)

    def property_names(self) -> List[str]:
        return [name for name in self.__annotations__ if not name.startswith("_")]

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

        Only a few classes define this, like the ExperimentConfig (the description) and
        ReproducibilityConfig (the experiment seed).
        """
        pass

    def copy(self: T) -> T:
        return type(self)(**{k: _copy(v) for k, v in vars(self).items()})

    def merge(self, src: T) -> None:
        if type(src) is not type(self):
            raise AssertionError("merge must be called with matching types")
        src.assert_valid()
        for name, src_value in vars(src).items():
            obj_value = vars(self).get(name)
            merged_value = _merge(obj_value, src_value)
            if merged_value is not None:
                setattr(self, name, merged_value)

    # TODO: enable sanity vs completion validation
    def assert_valid(self) -> None:
        errors = expconf.validation_errors(self.to_dict(), self._id)
        if errors:
            raise AssertionError(f"incorrect {type(self).__name__}:\n" + "\n".join(errors))

    # TODO: enable sanity vs completion validation
    def assert_complete(self) -> None:
        errors = expconf.validation_errors(self.to_dict(), self._id)
        if errors:
            raise TypeError(f"incorrect {type(self).__name__}:\n" + "\n".join(errors))


class UnionBaseMeta(type):
    """
    UnionBaseMeta raises an error if you forget to set the _union_key on a UnionBase.
    """

    def __new__(cls: type, name: str, bases: Tuple, dct: dict) -> Any:
        # Allow the UnionBase class itself to skip the tests.
        if name != "UnionBase":
            if "_union_key" not in dct:
                raise TypeError(f"{name}._union_key must be defined")
            if not isinstance(dct.get("_union_key"), str):
                raise TypeError(f"{name}._union_key must be a string")

        return super().__new__(cls, name, bases, dct)  # type: ignore


U = TypeVar("U", bound=Type[SchemaBase])


class UnionBase(SchemaBase, metaclass=UnionBaseMeta):
    """
    UnionBase is a base class for handling Determined's union schemas (like hyperparameters).  Each
    subclass of UnionBase should decorate several members and should call .finalize() once with the
    typing.Union of all of the member classes.  All type annotations should use the typing.Union
    rather than the UnionBase, because the union members should never be subclasses of the UnionBase
    class at all.

    Example:

    .. code::python

       class MyUnion(schemas.UnionBase):
           _id = "..."
           _union_key = "type"

       @MyUnion.member("a")
       class MemberA(MyUnion):
           _id = "..."
           val_a: int

           @schemas.auto_init
           def __init__(self, val_a: int):
               pass


       @MyUnion.member("b")
       class MemberB(MyUnion):
           _id = "..."
           val_b: int

           @schemas.auto_init
           def __init__(self, val_b: int):
               pass

        # The Union type is the annotation you will use on larger structs.
        # Note that inheritance works poorly here because when you bump
        # versions of a union type, you may only want to bump one member
        # type.  Then it would not be clear which members should subclass
        # which version of which union type.  That's why a type annotation
        # which is separate from the union class is important.
        MyUnion_Type = Union[MemberA, MemberB]

        # Finalize the union by registering the final Union type.
        # (used by _instance_from_annotation)
        MyUnion.finalize(MyUnion_Type)

        if __name__ == "__main__":
            # returns an instance of MemberA:
            my_union = MyUnion.FromDict({"type": "a", "val_a": 1})
            # returns an instance of MemberB:
            my_union = MyUnion.FromDict({"type": "b", "val_b": 1})
            # prints "{"type": "b", "val_b": 1}"
            print(my_union.to_dict())
    """

    # _union_key must be defined on all subclasses.
    _union_key = None  # type: Optional[str]

    # _members is used by .from_dict(); each element of _members is a dictionary mapping member
    # union_ids to member types.  The key to each element of _members is a different
    # subclass of UnionBase.
    _members = {}  # type: Dict[Type[UnionBase], Dict[str, Type[SchemaBase]]]

    # _union_types maps Union[...] annotations to UnionBase classes on which to call .from_dict().
    # _union_types is used directly by _instance_from_annotation().
    _union_types = {}  # type: Dict[frozenset, Type[UnionBase]]

    def __init__(self) -> None:
        raise NotImplementedError(
            f"union type {type(self).__name__} cannot be instantiated; use .from_dict() or "
            "instantiate a member class directly"
        )

    @classmethod
    def from_dict(cls, d: dict, prevalidated: bool = False) -> SchemaBase:  # type: ignore
        if cls._union_key is None:
            raise TypeError(f"_union_key is not defined on {cls}.__name__")
        t = d.get(cls._union_key)
        if t not in UnionBase._members[cls]:
            raise ValueError("invalid union type")

        return UnionBase._members[cls][t].from_dict(d, prevalidated)

    @classmethod
    def member(cls, union_id: Any) -> Callable[[U], U]:
        def wrapper(member_cls: U) -> U:
            # Associate this key/member pair for this union base.
            UnionBase._members.setdefault(cls, {})[union_id] = member_cls

            # Add union metadata to the member.
            member_cls._union_key = cls._union_key  # type: ignore
            member_cls._union_id = union_id  # type: ignore

            return member_cls

        return wrapper

    @classmethod
    def finalize(cls, union_type: Any) -> None:
        args = union_type.__args__
        args = cast(List[type], args)
        UnionBase._union_types[frozenset(args)] = cls


if __name__ == "__main__":
    from determined_common.schemas.expconf import _v0

    root = _v0.TestRootV0.from_dict({"val_x": 1, "sub_union": {"type": "a", "val_a": 1}})
    root.assert_complete()

    print(json.dumps(root.to_dict(), indent=4))
    print("---------")
    filled = root.copy()
    filled.fill_defaults()
    print(json.dumps(filled.to_dict(), indent=4))
    print("---------")
    root.sub_obj = _v0.TestSubV0(val_y="pre-merged y")
    root.merge(filled)
    print(json.dumps(root.to_dict(), indent=4))

    # output:
    #
    #     {
    #         "val_x": 1,
    #         "sub_union": {
    #             "type": "a",
    #             "val_a": 1
    #         }
    #     }
    #     ---------
    #     {
    #         "val_x": 1,
    #         "sub_union": {
    #             "type": "a",
    #             "val_a": 1
    #         },
    #         "sub_obj": {
    #             "val_y": "default_y"
    #         }
    #     }
    #     ---------
    #     {
    #         "val_x": 1,
    #         "sub_union": {
    #             "type": "a",
    #             "val_a": 1
    #         },
    #         "sub_obj": {
    #             "val_y": "pre-merged y"
    #         }
    #     }
