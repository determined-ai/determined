import json
import sys
import typing
from dataclasses import dataclass
from .types import (
    TypeDefs,
    Function,
    TypeAnno,
    Ref,
    Any,
    String,
    Int,
    Bool,
    Float,
    Dict,
    Sequence,
    Enum,
    Parameter,
    Class,
)


@dataclass
class ParseResult:
    defs: TypeDefs
    ops: typing.Dict[str, Function]


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


def process_definitions(swagger_definitions: dict, enums: dict) -> TypeDefs:
    defs = {}  # type: TypeDefs
    for name, schema in swagger_definitions.items():
        path = name
        if "enum" in schema:
            if schema["type"] == "string":
                members = schema["enum"]
                defs[name] = Enum(name, members)
                continue
            raise ValueError("unhandled enum type ({schema['type']}): {schema}")

        if schema["type"] == "object":
            # top-level named objects should be classes, not typed dictionaries:
            assert "additionalProperties" not in schema, (name, schema)
            if "properties" in schema:
                required = set(schema.get("required", []))
                members = {
                    k: Parameter(
                        k, classify_type(enums, f"{path}.{k}", v), (k in required), "definitions"
                    )
                    for k, v in schema["properties"].items()
                }
                defs[name] = Class(name, members)
                continue
            else:
                # empty responses or empty requests... we don't care.
                defs[name] = None
                continue
        raise ValueError(f"unhandled schema: {schema} @ {path}")
    return defs


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


def process_paths(swagger_paths: dict, enums: dict) -> typing.Dict[str, Function]:
    ops = {}
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

                if rspec.get("schema", {}).get("title", "").startswith("Stream result"):
                    # We expect a specific structure to streaming endpoints.
                    assert rschema["type"] == "object", rschema
                    assert "additionalProperties" not in rschema, rschema
                    rprops = rschema["properties"]
                    assert set(rprops.keys()) == set(("result", "error")), rschema
                    result_type = classify_type(
                        enums, f"{name}.responses.{code}.properties.result", rprops["result"]
                    )
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
                params[pname] = Parameter(pname, ptype, required, where, serialized_name)

            assert is_expected_path(path), (path, name)
            path = path.replace(".", "_")
            op = Function(name, method, path, params, responses, streaming)
            ops[name] = op
    return ops


def link_all_refs(defs: TypeDefs) -> None:
    for ref in Ref.all_refs:
        ref.linked = True
        ref.defn = defs[ref.name]


def load(path: str) -> ParseResult:
    with open(path) as f:
        swagger_json = json.load(f)
    enums = process_enums(swagger_json["definitions"])
    defs = process_definitions(swagger_json["definitions"], enums)
    ops = process_paths(swagger_json["paths"], enums)
    link_all_refs(defs)
    return ParseResult(ops=ops, defs=defs)
