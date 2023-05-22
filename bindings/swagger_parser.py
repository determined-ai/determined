import json
import sys
import typing
from dataclasses import dataclass, field

import typing_extensions


class Any:
    def __repr__(self):
        return "Any"


class String:
    def __repr__(self):
        return "str"


class DateTime:
    def __repr__(self):
        return "DateTime"


class Float:
    def __repr__(self):
        return "float"


class Int:
    def __repr__(self):
        return "int"


class Bool:
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


TypeAnno = typing.Union[Sequence, Dict, Float, Ref, Any, String, Int, Bool, DateTime]


@dataclass
class Parameter:
    name: str
    type: TypeAnno
    required: bool
    where: typing_extensions.Literal["query", "body", "path", "definitions"]
    serialized_name: typing.Optional[str] = None
    title: typing.Optional[str] = None

    def __post_init__(self):
        # validations
        assert self.where in ("query", "body", "path", "definitions"), (self.name, self.where)
        assert self.where != "path" or self.required, self.name
        if self.where == "path":
            if not isinstance(self.type, (String, Int)):
                raise AssertionError(f"bad type in path parameter {self.name}: {self.type}")
        if self.where == "query":
            underlying_typ = self.type.items if isinstance(self.type, Sequence) else self.type
            if not isinstance(underlying_typ, (String, Int, Bool, DateTime, Float)):
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


TypeDef = typing.Union[Class, Enum]

TypeDefs = typing.Dict[str, typing.Optional[TypeDef]]


@dataclass
class Function:
    """Remote HTTP-based call"""

    name: str
    method: str  # http method.
    path: str
    params: typing.Dict[str, Parameter]
    responses: typing.Dict[str, TypeAnno]
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

    def operation_name_sc(self) -> str:
        """Returns the name of the operation in snake_case"""
        return f"{self.method}_{self.name}"


@dataclass
class ApiInfo:
    title: str
    description: str
    version: str
    contact: str


@dataclass
class ParseResult:
    defs: TypeDefs
    ops: typing.Dict[str, Function]
    info: ApiInfo


def classify_type(enums: dict, path: str, schema: dict) -> TypeAnno:
    # enforce valid jsonschema:
    assert isinstance(schema, dict), (path, schema)
    if "enum" in schema:
        name = enums[json.dumps(schema["enum"])]
        assert name, (name, schema)
        return Ref(name, url_encodable=True)

    if "$ref" in schema:
        ref = schema["$ref"]
        start = "#/definitions/"
        assert ref.startswith(start), ref
        return Ref(ref[len(start) :])

    if "type" not in schema:
        # When "type" is not present, any json element should be valid.
        return Any()

    # only $refs don't have types
    assert "type" in schema, (path, schema)

    if schema["type"] == "string":
        if schema.get("format") == "date-time":
            return DateTime()
        return String()

    if schema["type"] == "integer":
        return Int()

    if schema["type"] == "boolean":
        return Bool()

    if schema["type"] in ("float", "number"):
        return Float()

    if schema["type"] == "object" and "properties" not in schema:
        adlProps = schema.get("additionalProperties")
        if adlProps is None:
            return Dict(Any())
        return Dict(classify_type(enums, path + ".additionalProperties", adlProps))

    if schema["type"] == "array":
        items = schema.get("items")
        if items is None:
            raise ValueError(path, schema)
        return Sequence(classify_type(enums, path + ".items", items))

    raise ValueError(f"unhandled schema: {schema} @ {path}")


def process_enums(swagger_definitions: dict) -> typing.Dict[int, str]:
    """
    Process enums from swagger definitions. In OpenAPI spec v2 generated
    by  protoc-gen-openapi enums are not linked to a definition and are inlined.
    Here we preprocess them so that they can  be linked to a definition.
    """
    enums = {}
    for name, schema in swagger_definitions.items():
        if "enum" in schema:
            members = schema["enum"]
            if enums.get(json.dumps(members)) is not None:
                print(
                    "ambiguous enum parameter:",
                    name,
                    members,
                    file=sys.stderr,
                )
            enums[json.dumps(members)] = name
    return enums


def classify_definition(enums: dict, name: str, schema: dict):
    path = name
    if "enum" in schema:
        if schema["type"] == "string":
            members = schema["enum"]
            return Enum(name, members, schema["description"])
        raise ValueError("unhandled enum type ({schema['type']}): {schema}")

    if schema["type"] == "object":
        # top-level named objects should be classes, not typed dictionaries:
        assert "additionalProperties" not in schema, (name, schema)
        required = set(schema.get("required", []))
        members = {
            k: Parameter(
                name=k,
                type=classify_type(enums, f"{path}.{k}", v),
                required=(k in required),
                where="definitions",
                serialized_name=None,
                title=v.get("title") or v.get("description"),
            )
            for k, v in schema.get("properties", {}).items()
        }
        description = schema.get("description")
        return Class(name, members, description)
    raise ValueError(f"unhandled schema: {schema} @ {path}")


def process_definitions(swagger_definitions: dict, enums: dict) -> TypeDefs:
    return {
        name: classify_definition(enums, name, schema)
        for name, schema in swagger_definitions.items()
    }


def is_expected_path(text: str) -> bool:
    """
    Check if any dots appear outside of curly braces, if any.
    This is assuming there are no nested curly braces.
    """
    in_braces = False
    for c in text:
        if c == "{":
            in_braces = True
        elif c == "}":
            in_braces = False
        elif c == "." and not in_braces:
            return False
    return True


def process_paths(
    swagger_paths: dict, enums: dict
) -> typing.Tuple[typing.Dict[str, Function], typing.Dict[str, Class]]:
    ops = {}
    extra_classes = {}
    for path, methods in swagger_paths.items():
        for method, spec in methods.items():
            name = spec["operationId"]
            # Figure out response types.
            responses = {}
            streaming = None
            bad_op = False
            for code, rspec in spec["responses"].items():
                rschema = rspec["schema"]
                if code == "default":
                    # We expect all "default" responses to be runtimeErrors, and we ignore them.
                    default_type = classify_type(enums, f"{name}.responses.default", rschema)
                    assert isinstance(default_type, Ref), rschema
                    assert default_type.name == "runtimeError", rschema
                    # Safe to ignore this return type.
                    continue

                if rschema.get("type") == "":
                    # not a valid response schema, skipping
                    bad_op = True
                    break

                if rschema.get("title", "").startswith("Stream result"):
                    # We expect a specific structure to streaming endpoints.
                    assert rschema["type"] == "object", rschema
                    assert "additionalProperties" not in rschema, rschema
                    rprops = rschema["properties"]
                    assert set(rprops.keys()) == set(("result", "error")), rschema
                    error_type = classify_type(
                        enums, f"{name}.responses.{code}.properties.error", rprops["error"]
                    )
                    # We expect all "error" results to be runtimeStreamError.  They are parsed in
                    # code generated by Function.gen_def().
                    assert isinstance(error_type, Ref), rschema
                    assert error_type.name == "runtimeStreamError", rschema
                    if streaming is False:
                        raise ValueError(
                            f"a method must be either all-streaming or all-nonstreaming: {rspec}"
                        )
                    streaming = True

                    # handle inlined objects similar to how swagger-codegen does
                    result_type_class = classify_definition(
                        enums, f"{name}.response.{code}", rschema
                    )
                    assert isinstance(result_type_class, Class)
                    result_type_class_ref = result_type_class.params["result"].type
                    assert isinstance(result_type_class_ref, Ref)
                    pascal_name = result_type_class_ref.name
                    pascal_name = pascal_name[0].upper() + pascal_name[1:]
                    result_type_class.name = f"StreamResultOf{pascal_name}"
                    result_type = Ref(result_type_class.name)
                    extra_classes[result_type_class.name] = result_type_class

                    responses[code] = result_type
                    continue

                responses[code] = classify_type(enums, f"{name}.responses.{code}", rschema)
                if streaming is True:
                    raise ValueError(
                        f"a method must be either all-streaming or all-nonstreaming: {rspec}"
                    )
                streaming = False

            if bad_op:
                continue

            assert streaming is not None

            # Figure out parameters.
            params = {}
            for pspec in spec.get("parameters", []):
                where = pspec["in"]
                serialized_name = None
                if where == "query":  # preserve query parameter names
                    serialized_name = pname = pspec["name"]
                pname = pspec["name"].replace(".", "_")
                required = pspec.get("required", False)
                if "schema" in pspec:
                    pschema = pspec["schema"]
                else:
                    # swagger has some weird inlining going on here...
                    inlined = ("type", "format", "items", "properties", "enum")
                    pschema = {k: pspec[k] for k in inlined if k in pspec}
                ptype = classify_type(enums, f"{name}.{pname}", pschema)
                title = pspec.get("title") or pspec.get("description")
                params[pname] = Parameter(pname, ptype, required, where, serialized_name, title)

            assert is_expected_path(path), (path, name)
            path = path.replace(".", "_")
            tags = set(spec["tags"])
            summary = spec["summary"]
            needs_auth = "security" not in spec
            op = Function(
                name, method, path, params, responses, streaming, tags, summary, needs_auth
            )
            ops[name] = op
    return ops, extra_classes


def link_all_refs(defs: TypeDefs) -> None:
    for ref in Ref.all_refs:
        ref.linked = True
        ref.defn = defs[ref.name]


def parse(path: str) -> ParseResult:
    with open(path) as f:
        swagger_json = json.load(f)
    enums = process_enums(swagger_json["definitions"])
    defs = process_definitions(swagger_json["definitions"], enums)
    ops, streaming_refs = process_paths(swagger_json["paths"], enums)
    defs.update(streaming_refs)
    link_all_refs(defs)

    info_keys = ("title", "description", "version")
    info_json = swagger_json["info"]
    info = ApiInfo(
        **{
            **{k: info_json[k] for k in info_keys if k in info_json},
            **{"contact": info_json["contact"]["email"]},
        }
    )

    return ParseResult(ops=ops, defs=defs, info=info)
