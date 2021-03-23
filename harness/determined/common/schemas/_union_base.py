from typing import Any, Callable, Dict, List, Tuple, Type, TypeVar, cast

from determined.common import schemas


class UnionBaseMeta(type):
    """
    UnionBaseMeta raises an error if you forget to set the _union_key on a UnionBase.
    """

    def __new__(cls: type, name: str, bases: Tuple, dct: dict) -> Any:
        # Allow the UnionBase class itself to skip the tests and modifications.
        if name != "UnionBase":
            if "_union_key" not in dct:
                raise TypeError(f"{name}._union_key must be defined")
            if not isinstance(dct.get("_union_key"), str):
                raise TypeError(f"{name}._union_key must be a string")

            # Each subclass class has a unique _members dict.
            dct["_members"] = {}

        return super().__new__(cls, name, bases, dct)  # type: ignore


T = TypeVar("T", bound=Type[schemas.SchemaBase])


class UnionBase(schemas.SchemaBase, metaclass=UnionBaseMeta):
    """
    UnionBase is a base class for handling Determined's union schemas (like hyperparameters).  Each
    subclass of UnionBase should decorate several members and should call .finalize() once with the
    typing.Union of all of the member classes.  All type annotations should use the typing.Union
    rather than the UnionBase, because the union members should never be subclasses of the UnionBase
    class at all.

    The reason for using a distinct typing.Union for annotations rather than the UnionBase subclass
    is that inheritance is the wrong pattern for good type checking.  Imagine that you bump the
    version of the union type and one of its members (but not the others); if you use inheritance
    you have to have duplicates of all of the members which were not altered only for them to
    subclass the new union type.

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

        # Create a typing.Union to be used in annotations.
        MyUnion_Type = Union[MemberA, MemberB]

        # Finalize the union with them matching typing.Union.
        MyUnion.finalize(MyUnion_Type)


        if __name__ == "__main__":
            # returns an instance of MemberA:
            my_union = MyUnion.from_dict({"type": "a", "val_a": 1})

            # returns an instance of MemberB:
            my_union = MyUnion.from_dict({"type": "b", "val_b": 1})

            # prints "{"type": "b", "val_b": 1}"
            print(my_union.to_dict())
    """

    # _union_key must be defined on all subclasses.
    _union_key = ""  # type: str

    # _members is used by .from_dict(); it maps union_id strings to SchemaBase member classes.
    # Note that the UnionBaseMeta ensure that each UnionBase subclass has its own _members dict.
    _members = {}  # type: Dict[str, Type[schemas.SchemaBase]]

    # _union_types maps Union[...] annotations to UnionBase classes on which to call .from_dict().
    # _union_types is used directly by _instance_from_annotation().
    _union_types = {}  # type: Dict[frozenset, Type[UnionBase]]

    def __init__(self) -> None:
        raise NotImplementedError(
            f"union type {type(self).__name__} cannot be instantiated; use .from_dict() or "
            "instantiate a member class directly"
        )

    @classmethod
    def from_dict(cls, d: dict, prevalidated: bool = False) -> schemas.SchemaBase:  # type: ignore
        union_id = d.get(cls._union_key)
        if union_id not in cls._members:
            raise ValueError("invalid union type")

        return cls._members[union_id].from_dict(d, prevalidated)

    @classmethod
    def member(cls, union_id: Any) -> Callable[[T], T]:
        def wrapper(member_cls: T) -> T:
            if cls in UnionBase._union_types.values():
                raise TypeError(
                    f"unable to call {cls.__name__}.member() after {cls.__name__}.finalize() has "
                    "already been called."
                )

            if union_id in cls._members:
                raise TypeError(
                    f"unable to decorate {member_cls.__name__} with "
                    f"@{cls.__name__}.member(union_id='{union_id}') after "
                    f"{cls._members[union_id].__name__} has already been decorated with the same "
                    "union_id"
                )

            # Associate this key/member pair for this union base.
            cls._members[union_id] = member_cls

            # Classes with multiple @member decorators must have matching _union_key and _union_type
            # or to_dict() could not always be correct.
            if hasattr(member_cls, "_union_key"):
                member_union_key = member_cls._union_key  # type: ignore
                member_union_id = member_cls._union_id  # type: ignore
                if member_union_key != cls._union_key:
                    raise TypeError(
                        f"class {member_cls.__name__} cannot be decorated with "
                        f"@{cls.__name__}.member() because {cls.__name__}._union_key "
                        f"('{cls._union_key}') does not match the previously set "
                        f"{member_cls.__name__}._union_key value ('{member_union_key}')"
                    )
                if member_union_id != union_id:
                    raise TypeError(
                        f"class {member_cls.__name__} cannot be decorated with "
                        f"@member({union_id}) because '{union_id}' does not match the union_id "
                        f"from a previous decorator ('{member_union_id}')"
                    )

            # Add union metadata to the member.
            member_cls._union_key = cls._union_key  # type: ignore
            member_cls._union_id = union_id  # type: ignore

            return member_cls

        return wrapper

    @classmethod
    def finalize(cls, union_type: Any) -> None:
        # Even though Union[] types define correct equality tests, prior to python 3.7 I don't know
        # of any programmatic way to create Unions with e.g. the None-type removed (to get the
        # Union[] from within an Optional[Union[]]).  Since we need that exact operation in
        # _instance_from_annotation(), we key off of a frozenset of Union.__args__.
        args = union_type.__args__
        args = cast(List[type], args)
        frozen_args = frozenset(args)

        # Disallow multiple classes from calling .finalize() with matching typing.Unions.
        if frozen_args in UnionBase._union_types:
            raise TypeError(
                f"unable to finalize {cls.__name__} with a union of "
                f"[{', '.join(a.__name__ for a in args)}]"
            )

        # Ensure that the typing.Union provied matches the calls made to @members.
        if frozen_args != frozenset(cls._members.values()):
            raise TypeError(
                f"Unable to finalize {cls.__name__} with a "
                f"Union[{', '.join(a.__name__ for a in args)}], which does not match set of "
                f"@members: [{', '.join(m.__name__ for m in cls._members.values())}]"
            )

        UnionBase._union_types[frozen_args] = cls
