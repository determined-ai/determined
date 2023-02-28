#!/usr/bin/env python3
from dataclasses import dataclass, field
import typing

TypeDefs = typing.Dict[str, typing.Optional["TypeDef"]]


class TypeAnno:
    pass


class TypeDef:
    pass


class NoParse:
    pass


class Any(NoParse, TypeAnno):
    def __repr__(self):
        return "Any"


class String(NoParse, TypeAnno):
    def __init__(self):
        pass

    def __repr__(self):
        return "str"


class Float(TypeAnno):
    def __init__(self):
        pass

    def __repr__(self):
        return "float"


class Int(NoParse, TypeAnno):
    def __init__(self):
        pass

    def __repr__(self):
        return "int"


class Bool(NoParse, TypeAnno):
    def __init__(self):
        pass

    def __repr__(self):
        return "bool"


@dataclass
class Dict(TypeAnno):
    values: TypeAnno

    def __repr__(self):
        return f"Dict[str, {self.values}]"


@dataclass
class Sequence(TypeAnno):
    items: TypeAnno

    def __repr__(self):
        return f"Sequence[{self.items}]"


@dataclass
class Parameter:
    name: str
    type: TypeAnno
    required: bool
    where: str
    serialized_name: typing.Optional[str] = None

    def __post_init__(self):
        # validations
        assert self.where in ("query", "body", "path", "definitions"), (self.name, self.where)
        assert self.where != "path" or self.required, self.name
        if self.where == "path":
            if not isinstance(self.type, (String, Int, Bool)):
                raise AssertionError(f"bad type in path parameter {self.name}: {self.type}")
        if self.where == "query":
            underlying_typ = self.type.items if isinstance(self.type, Sequence) else self.type
            if not isinstance(underlying_typ, (String, Int, Bool)):
                if not (isinstance(underlying_typ, Ref) and underlying_typ.url_encodable):
                    raise AssertionError(f"bad type in query parameter {self.name}: {self.type}")


@dataclass
class Class(TypeDef):
    name: str
    params: typing.Dict[str, Parameter]


@dataclass
class Enum(TypeDef):
    name: str
    members: typing.List[str]  # ?


@dataclass
class Ref(TypeAnno):
    # Collect refs as we instantiate them, for the linking step.
    all_refs: typing.ClassVar[typing.List["Ref"]] = []

    name: str
    url_encodable: bool = False
    linked: bool = field(default=False, init=False)
    defn: typing.Optional[TypeDef] = field(default=None, init=False)

    def __post_init__(self):
        Ref.all_refs.append(self)

    def __repr__(self):
        return self.name


@dataclass
class Function:
    name: str
    method: str
    path: str
    params: typing.Dict[str, Parameter]
    responses: typing.Dict[str, TypeAnno]
    streaming: bool

    def __repr__(self) -> str:
        out = (
            f"Function({self.name}):\n"
            f"    self.method = {self.method.upper()}\n"
            f"    self.params = {self.params}\n"
            f"    responses = {{"
        )
        for code, resp in self.responses.items():
            out += f"\n       {code} = {resp}"
        out += "\n    }"
        return out
