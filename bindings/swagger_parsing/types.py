#!/usr/bin/env python3
from dataclasses import dataclass, field
import typing
import typing_extensions


class NoParse:
    pass


class Any(NoParse):
    def __repr__(self):
        return "Any"


class String(NoParse):
    def __repr__(self):
        return "str"


class DateTime(NoParse):
    def __repr__(self):
        return "DateTime"


class Float:
    def __repr__(self):
        return "float"


class Int(NoParse):
    def __repr__(self):
        return "int"


class Bool(NoParse):
    def __repr__(self):
        return "bool"


@dataclass
class Dict:
    values: "TypeAnno"

    def __repr__(self):
        return f"Dict[str, {self.values}]"


@dataclass
class Sequence:
    items: "TypeAnno"

    def __repr__(self):
        return f"Sequence[{self.items}]"


@dataclass
class Parameter:
    name: str
    type: "TypeAnno"
    required: bool
    where: typing_extensions.Literal["query", "body", "path", "definitions"]
    serialized_name: typing.Optional[str] = None
    title: typing.Optional[str] = None

    def __post_init__(self):
        # validations
        assert self.where in ("query", "body", "path", "definitions"), (self.name, self.where)
        assert self.where != "path" or self.required, self.name
        if self.where == "path":
            if not isinstance(self.type, (String, Int, Bool)):
                raise AssertionError(f"bad type in path parameter {self.name}: {self.type}")
        if self.where == "query":
            underlying_typ = self.type.items if isinstance(self.type, Sequence) else self.type
            if not isinstance(underlying_typ, (String, Int, Bool, DateTime)):
                if not (isinstance(underlying_typ, Ref) and underlying_typ.url_encodable):
                    raise AssertionError(f"bad type in query parameter {self.name}: {self.type}")


@dataclass
class Class:
    name: str
    params: typing.Dict[str, Parameter]
    description: typing.Optional[str]


@dataclass
class Enum:
    name: str
    members: typing.List[str]
    description: typing.Optional[str]


@dataclass
class Ref:
    # Collect refs as we instantiate them, for the linking step.
    all_refs: typing.ClassVar[typing.List["Ref"]] = []

    name: str
    url_encodable: bool = False
    linked: bool = field(default=False, init=False)
    defn: typing.Optional["TypeDef"] = field(default=None, init=False)

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
    responses: typing.Dict[str, "TypeAnno"]
    streaming: bool
    tags: typing.Set[str]
    summary: str
    needs_auth: bool

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


TypeAnno = typing.Union[Sequence, Dict, Float, Ref, Any, String, Int, Bool, DateTime]
TypeDef = typing.Union[Class, Enum]

TypeDefs = typing.Dict[str, typing.Optional[TypeDef]]
