import functools
import inspect
import json
import numbers
import typing
from typing import Any, Callable, Mapping, Optional, Sequence, Tuple, Type, TypeVar

from determined_common.schemas import expconf

PRIMITIVE_JSON_TYPES = (numbers.Number, str, bool, type(None))


def get_default(url: str, prop: str) -> Any:
    from determined_common.schemas.expconf import _gen

    return _gen.schemas[url].get("properties", {}).get(prop, {}).get("default")


def to_dict(val: Any) -> Any:
    """Recurse through an object, calling .to_dict() on all subclasses of SchemaBase."""
    if isinstance(val, SchemaBase):
        return val.to_dict()
    if isinstance(val, PRIMITIVE_JSON_TYPES):
        return val
    if isinstance(val, Mapping):
        return {k: to_dict(v) for k, v in val.items()}
    if isinstance(val, Sequence):
        return [to_dict(i) for i in val]
    raise ValueError(f"invalid type in to_dict: {type(val).__name__}")


def fill_defaults(val: Any) -> None:
    """Recurse through an object, calling .fill_defaults() on all subclasses of SchemaBase."""
    if isinstance(val, SchemaBase):
        val.fill_defaults()
        return
    if isinstance(val, PRIMITIVE_JSON_TYPES):
        return
    if isinstance(val, Mapping):
        for v in val.values():
            fill_defaults(v)
        return
    if isinstance(val, Sequence):
        for v in val:
            fill_defaults(v)
        return
    raise ValueError(f"invalid type in fill_defaults: {type(val).__name__}")


def copy(val: Any) -> Any:
    """Recurse through an object, calling .copy() on all subclasses of SchemaBase."""
    if isinstance(val, SchemaBase):
        return val.copy()
    if isinstance(val, PRIMITIVE_JSON_TYPES):
        return val
    if isinstance(val, Mapping):
        return {k: copy(v) for k, v in val.items()}
    if isinstance(val, Sequence):
        return [copy(i) for i in val]
    raise ValueError(f"invalid type in copy: {type(val).__name__}")


def merge(obj: Any, src: Any) -> Any:
    """Recursively merge two objects and return the result"""
    if src is None:
        return obj
    if obj is None:
        return src
    if type(obj) is not type(src):
        raise AssertionError("merge must be called with matching types")
    if isinstance(obj, SchemaBase):
        return obj.merge(src)
    if isinstance(obj, PRIMITIVE_JSON_TYPES):
        return obj
    if isinstance(obj, Mapping):
        src.update(obj)
        return src
    if isinstance(obj, Sequence):
        return obj
    raise ValueError(f"invalid type in merge: {type(obj).__name__}")


def remove_optional(anno: Any) -> Any:
    """Given a type annotation, which might be TYPE or Optional[TYPE], return TYPE."""
    if type(anno) is not typing._Union:  # type: ignore
        return anno
    args = list(anno.__args__)
    if type(None) not in args:
        raise ValueError("got union which was not Optional")
    args.remove(type(None))
    if len(args) != 1:
        raise ValueError("got union which was not Optional")
    return args[0]


def instance_from_annotation(anno: type, dict_value: Any, prevalidated: bool = False) -> Any:
    typ = remove_optional(anno)
    if issubclass(typ, SchemaBase):
        # For subclasses of SchemaBase we just call either from_dict() or from_none().
        if dict_value is None:
            return typ.from_none()
        return typ.from_dict(dict_value, prevalidated)
    if issubclass(typ, PRIMITIVE_JSON_TYPES):
        # For dict literal types, we just include them directly.
        return dict_value
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


class AutoInit(type):
    """
    AutoInit is a metaclass (it subclasses type) and it is invoked when a type is created, rather
    than an instance of a type is created.  That means, it's invoked at the time a class is DEFINED
    rather than when it is INSTANTIATED.  That lets us do things like apply the @auto_init
    decorator on every single subclass of SchemaBase (whose metaclass=AutoInit) and call it a day.
    """

    def __new__(cls: type, name: str, bases: Tuple, dct: dict) -> Any:
        # Nothing to do on either the SchemaBase class or if there is no __init__ function.
        if name == "SchemaBase" or "__init__" not in dct:
            return super().__new__(cls, name, bases, dct)  # type: ignore

        dct["__init__"] = auto_init(dct["__init__"])

        return super().__new__(cls, name, bases, dct)  # type: ignore


T = TypeVar("T", bound="SchemaBase")


class SchemaBase(metaclass=AutoInit):
    _id: str

    def __init__(self, **kwargs: dict) -> None:
        raise NotImplementedError

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
                raise TypeError("\n".join(errors))

        additional_properties_anno = cls.__annotations__.get("_additional_properties")

        init_args = {}

        # For every key in the dictionary, get the type from the class annotations.  If it is a
        # sublcass of SchemaBase, call from_dict() or from_none() on it based on the value in the
        # input.  Otherwise, make sure it is a primitive type and pass the value to __init__.
        for name, value in d.items():
            anno = cls.__annotations__.get(name, additional_properties_anno)
            if anno is None:
                raise TypeError(
                    f"from_dict() found a key '{name}' input which has no annotation.  This is a "
                    "bug; all SchemaBase subclasses must have annotations which match the json "
                    "schema definitions which they correspond to."
                )
            # Create an instance based on the type annotation.
            init_args[name] = instance_from_annotation(anno, value, prevalidated=True)

        return cls(**init_args)

    def to_dict(self) -> dict:
        return {k: to_dict(v) for k, v in vars(self).items()}

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
            default_json = get_default(self._id, name)

            # Create an instance based on the type annotation.
            default = instance_from_annotation(anno, default_json, prevalidated=False)

            if default is None:
                continue

            setattr(self, name, default)

        # Recurse into all child objects.
        for value in vars(self).values():
            fill_defaults(value)

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
        return type(self)(**{k: copy(v) for k, v in vars(self).items()})

    def merge(self, src: T) -> None:
        if type(src) is not type(self):
            raise AssertionError("merge must be called with matching types")
        src.assert_valid()
        for name, src_value in vars(src).items():
            obj_value = vars(self).get(name)
            merged_value = merge(obj_value, src_value)
            if merged_value is not None:
                setattr(self, name, merged_value)

    # TODO: enable sanity vs completion validation
    def assert_valid(self) -> None:
        errors = expconf.validation_errors(self.to_dict(), self._id)
        if errors:
            raise AssertionError("\n".join(errors))

    # TODO: enable sanity vs completion validation
    def assert_complete(self) -> None:
        errors = expconf.validation_errors(self.to_dict(), self._id)
        if errors:
            raise TypeError("\n".join(errors))


class TestSub(SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v1/test-sub.json"

    val_y: Optional[str] = None

    def __init__(
        self,
        val_y: Optional[str] = None,
    ):
        pass


class TestUnion(SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v1/test-union.json"

    @classmethod
    def from_dict(cls: Type[T], d: dict, prevalidated: bool = False) -> T:
        if cls is not TestUnion:
            # A subclass is calling its inherited from_dict; skip the union behavior.
            return super().from_dict(d, prevalidated)  # type: ignore

        t = d.get("type")
        if t == "a":
            return TestUnionA.from_dict(d, prevalidated)  # type: ignore
        if t == "b":
            return TestUnionB.from_dict(d, prevalidated)  # type: ignore

        raise ValueError("invalid union type")

    # TODO: override .merge() too.


class TestUnionA(TestUnion):
    _id = "http://determined.ai/schemas/expconf/v1/test-union-a.json"

    type: str
    val_a: int

    def __init__(
        self,
        type: str,
        val_a: int,
    ):
        # The AutoInit metaclass sets the initial values automatically.
        pass


class TestUnionB(TestUnion):
    _id = "http://determined.ai/schemas/expconf/v1/test-union-b.json"

    type: str
    val_b: int

    def __init__(
        self,
        type: str,
        val_b: int,
    ):
        # The AutoInit metaclass sets the initial values automatically.
        pass


class TestRoot(SchemaBase):
    _id = "http://determined.ai/schemas/expconf/v1/test-root.json"

    val_x: int
    sub_obj: Optional[TestSub] = None
    sub_union: Optional[TestUnion] = None

    def __init__(
        self,
        val_x: int,
        sub_obj: Optional[TestSub] = None,
        sub_union: Optional[TestUnion] = None,
    ):
        # The AutoInit metaclass sets the initial values automatically.
        pass


if __name__ == "__main__":
    root = TestRoot.from_dict({"val_x": 1, "sub_union": {"type": "a", "val_a": 1}})
    root.assert_complete()

    print(json.dumps(root.to_dict(), indent=4))
    print("---------")
    filled = root.copy()
    filled.fill_defaults()
    print(json.dumps(filled.to_dict(), indent=4))
    print("---------")
    root.sub_obj = TestSub(val_y="pre-merged y")
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
