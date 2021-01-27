import enum
import dataclasses
from typing import Optional, Mapping, Sequence, Union, Any, Tuple, TypeVar, Type
import inspect
import numbers
import typing
import json
from pprint import pprint

from determined_common.schemas import expconf

DICT_LITERAL_TYPES = (numbers.Number, str, bool, type(None))

def get_default(url, prop):
    from determined_common.schemas.expconf import _v1_gen
    return _v1_gen.schemas[url].get("properties", {}).get(prop, {}).get("default")

def to_dict(val: Any) -> Any:
    """Recurse through an object, calling .to_dict() on all subclasses of Base."""
    if isinstance(val, Base):
        return val.to_dict()
    if isinstance(val, DICT_LITERAL_TYPES):
        return val
    if isinstance(val, Mapping):
        return {k: to_dict(v) for k, v in val.items()}
    if isinstance(val, Sequence):
        return [to_dict(i) for i in val]
    raise ValueError(f"invalid type in to_dict: {type(val).__name__}")

def fill_defaults(val: Any) -> None:
    """Recurse through an object, calling .fill_defaults() on all subclasses of Base."""
    if isinstance(val, Base):
        val.fill_defaults()
        return
    if isinstance(val, DICT_LITERAL_TYPES):
        return
    if isinstance(val, Mapping):
        return {k: fill_defaults(v) for k, v in val.items()}
    if isinstance(val, Sequence):
        return [fill_defaults(i) for i in val]
    raise ValueError(f"invalid type in fill_defaults: {type(val).__name__}")


def copy(val: Any) -> Any:
    """Recurse through an object, calling .copy() on all subclasses of Base."""
    if isinstance(val, Base):
        return val.copy()
    if isinstance(val, DICT_LITERAL_TYPES):
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
    if isinstance(obj, Base):
        return obj.merge(src)
    if isinstance(obj, DICT_LITERAL_TYPES):
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


def instance_from_annotation(anno, dict_value, prevalidated=False) -> Any:
    typ = remove_optional(anno)
    if issubclass(typ, Base):
        # For subclasses of Base we just call either from_dict() or from_none().
        if dict_value is None:
            return typ.from_none()
        return typ.from_dict(dict_value, prevalidated)
    if issubclass(typ, DICT_LITERAL_TYPES):
        # For dict literal types, we just include them directly.
        return dict_value
    raise TypeError(f"invalid type annotation on Base object: {anno}")


def auto_init(old_init):
    old_sig = inspect.signature(old_init)

    def set_all_attrs(self, *args, **kwargs) -> None:
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
    def __new__(cls: type, name: str, bases: Tuple, dct: dict) -> Any:
        # print(f"cls:{cls}, name:{name}, bases:{bases}, dct:{dct}")
        # Nothing to do if there is no __init__ function.
        if "__init__" not in dct:
            return super().__new__(cls, name, bases, dct)  # type: ignore

        # Basic validation.
        if "_id" not in dct:
            raise AssertionError("missing _id class attribute")

        dct["__init__"] = auto_init(dct["__init__"])

        return super().__new__(cls, name, bases, dct)  # type: ignore


T = TypeVar('T', bound='Base')

class Base(metaclass=AutoInit):
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
        # sublcass of Base, call from_dict() or from_none() on it based on the value in the input.
        # Otherwise, make sure it is in DICT_LITERAL_TYPES and pass the value to __init__ directly.
        for name, value in d.items():
            anno = cls.__annotations__.get(name, additional_properties_anno)
            if anno is None:
                raise TypeError(
                    f"from_dict() found a key '{name}' input which has no annotation.  This is a "
                    "bug; all Base subclasses must have annotations which match the json schema "
                    "definitions which they correspond to."
                )
            # Create an instance based on the type annotation.
            init_args[name] = instance_from_annotation(anno, value, prevalidated=True)

        return cls(**init_args)

    def to_dict(self) -> dict:
        return {k: to_dict(v) for k, v in vars(self).items()}

    def fill_defaults(self) -> T:
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
        for name, value in vars(self).items():
            fill_defaults(value)

        # Finally, set any runtime defaults.
        self.runtime_defaults()

        return self

    def runtime_defaults(self) -> None:
        """
        runtime_defaults is called at the end of Base.fill_defaults(), where values which are
        filled out at runtime can be populated dynamically.

        Only a few classes define this, like the ExperimentConfig (the description) and
        ReproducibilityConfig (the experiment seed).
        """
        pass

    def copy(self) -> T:
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


class TestSub(Base):
    _id = "http://determined.ai/schemas/expconf/v1/test-sub.json"

    val_y: str
    val_z: Optional[str] = None

    def __init__(
        self,
        val_y: str,
        val_z: Optional[str] = None,
    ):
        pass


class TestUnion(Base):
    _id = "http://determined.ai/schemas/expconf/v1/test-union.json"

    def __init__(self):
        raise NotImplementedError

    @classmethod
    def from_dict(cls: Type[T], d: dict, prevalidated: bool = False) -> T:
        if cls is not TestUnion:
            # A subclass is calling its inherited from_dict; skip the union behavior.
            return super().from_dict(d, prevalidated)

        t = d.get("type")
        if t == "a":
            return TestUnionA.from_dict(d, prevalidated)
        if t == "b":
            return TestUnionB.from_dict(d, prevalidated)

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

class TestRoot(Base):
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
    root = TestRoot.from_dict({
        "val_x": 1,
        "sub_union": {"type": "a", "val_a": 1}
    })
    # root = TestRoot(val_x=1)
    root.assert_complete()

    print(json.dumps(root.to_dict(), indent=4))
    print("---------")
    filled = root.copy().fill_defaults()
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
    #             "val_y": "asdf",
    #             "val_z": "default_z"
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
    #             "val_y": "pre-merged y",
    #             "val_z": "default_z"
    #         }
    #     }
