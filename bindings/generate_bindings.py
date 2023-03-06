import abc
import json
import os
import sys
import typing

SWAGGER = "proto/build/swagger/determined/api/v1/api.swagger.json"
SWAGGER = os.path.join(os.path.dirname(__file__), "..", SWAGGER)

Code = str
TypeDefs = typing.Dict[str, typing.Optional["TypeDef"]]


class TypeAnno:
    def annotation(self, prequoted=False) -> Code:
        raise NotImplementedError(type(self))

    def need_parse(self) -> bool:
        raise NotImplementedError(type(self))

    def load(self, val: Code) -> Code:
        raise NotImplementedError(type(self))

    def dump(self, val: Code, omit_unset: Code) -> Code:
        raise NotImplementedError(type(self))

    def isnone(self) -> bool:
        # Only Refs to empty structs ever return True; we skip generating them.
        return False

    def need_urlparam_dump(self) -> bool:
        """
        Dump a value for url parameters.

        Defaults to need_parse(), since dump_as_urlparam() defaults to dump()
        """
        return self.need_parse()

    def dump_as_urlparam(self, val: Code) -> Code:
        """
        Dump a value for url parameters.

        Defaults to the normal dump(), but can be overridden.
        """
        return self.dump(val, omit_unset="True")


class TypeDef:
    def gen_def(self) -> Code:
        raise NotImplementedError(type(self))


class NoParse:
    """A compositional class for things where json.loads/dumps is sufficient."""

    def need_parse(self) -> bool:
        return False

    def load(self, val: Code) -> Code:
        return val

    def dump(self, val: Code, omit_unset: Code) -> Code:
        return val


class Any(NoParse, TypeAnno):
    def __repr__(self):
        return "Any"

    def annotation(self, prequoted=False) -> Code:
        return "typing.Any"


class String(NoParse, TypeAnno):
    def __init__(self):
        pass

    def __repr__(self):
        return "str"

    def annotation(self, prequoted=False) -> Code:
        return "str"


class Float(TypeAnno):
    def __init__(self):
        pass

    def __repr__(self):
        return "float"

    def annotation(self, prequoted=False) -> Code:
        return "float"

    def need_parse(self) -> bool:
        return True

    def load(self, val: Code) -> Code:
        return f"float({val})"

    def dump(self, val: Code, omit_unset: Code) -> Code:
        return f"dump_float({val})"


class Int(NoParse, TypeAnno):
    def __init__(self):
        pass

    def __repr__(self):
        return "int"

    def annotation(self, prequoted=False) -> Code:
        return "int"


class Bool(NoParse, TypeAnno):
    def __init__(self):
        pass

    def __repr__(self):
        return "bool"

    def annotation(self, prequoted=False) -> Code:
        return "bool"

    def need_urlparam_dump(self) -> bool:
        return True

    def dump_as_urlparam(self, val: Code) -> Code:
        """
        Covert True to "true" and False to "false", but only to embed in a url parameter.

        By default, requests encodes True as `val=True`.  GRPC pukes unless you encode `val=true`.
        """
        return f"str({val}).lower()"


class Ref(TypeAnno):
    # Collect refs as we instantiate them, for the linking step.
    all_refs = []

    def __init__(self, name: str, url_encodable=False):
        self.name = name
        self.linked = False
        self.defn = None
        self.url_encodable = url_encodable
        Ref.all_refs.append(self)

    def __repr__(self):
        return self.name

    def annotation(self, prequoted=False) -> Code:
        assert self.linked, "link step must be completed before generating code!"
        if not self.defn:
            return "None"
        out = self.name
        if not prequoted:
            return f'"{out}"'
        return out

    def need_parse(self) -> bool:
        assert self.linked, "link step must be completed before generating code!"
        return True

    def isnone(self) -> bool:
        assert self.linked, "link step must be completed before generating code!"
        return self.defn is None

    def load(self, val: Code) -> Code:
        assert self.linked, "link step must be completed before generating code!"
        assert self.defn, "it doesn't make sense to load an empty class"
        assert isinstance(self.defn, (Enum, Class)), (
            self.name,
            type(self.defn).__name__,
        )
        return self.defn.load(val)

    def dump(self, val: Code, omit_unset: Code) -> Code:
        assert self.linked, "link step must be completed before generating code!"
        assert self.defn, "it doesn't make sense to dump an empty class"
        assert isinstance(self.defn, (Enum, Class)), (
            self.name,
            type(self.defn).__name__,
        )
        return self.defn.dump(val, omit_unset)


class Dict(TypeAnno):
    def __init__(self, values: TypeAnno):
        self.values = values

    def __repr__(self):
        return f"Dict[str, {self.values}]"

    def annotation(self, prequoted=False) -> Code:
        out = f"typing.Dict[str, {self.values.annotation(True)}]"
        if not prequoted:
            return f'"{out}"'
        return out

    def need_parse(self) -> bool:
        return self.values.need_parse()

    def load(self, val: Code) -> Code:
        if not self.need_parse():
            return val
        return f"{{k: {self.values.load('v')} for k, v in {val}.items()}}"

    def dump(self, val: Code, omit_unset: Code) -> Code:
        if not self.need_parse():
            return val
        each = self.values.dump("v", omit_unset)
        return f"{{k: {each} for k, v in {val}.items()}}"


class Sequence(TypeAnno):
    def __init__(self, items):
        self.items = items

    def __repr__(self):
        return f"Sequence[{self.items}]"

    def annotation(self, prequoted=False) -> Code:
        out = f"typing.Sequence[{self.items.annotation(True)}]"
        if not prequoted:
            return f'"{out}"'
        return out

    def need_parse(self) -> bool:
        return self.items.need_parse()

    def load(self, val: Code) -> Code:
        if not self.need_parse():
            return val
        return f"[{self.items.load('x')} for x in {val}]"

    def dump(self, val: Code, omit_unset: Code) -> Code:
        if not self.need_parse():
            return val
        each = self.items.dump("x", omit_unset)
        return f"[{each} for x in {val}]"


class Parameter:
    def __init__(
        self,
        name: str,
        typ: TypeAnno,
        required: bool,
        where: str,
        serialized_name: str = None,
    ) -> None:
        self.name = name
        self.serialized_name = serialized_name
        self.type = typ
        self.required = required
        self.where = where
        # validations
        assert where in ("query", "body", "path", "definitions"), (name, where)
        assert where != "path" or required, name
        if where == "path":
            if not isinstance(typ, (String, Int, Bool)):
                raise AssertionError(f"bad type in path parameter {name}: {typ}")
        if where == "query":
            underlying_typ = typ.items if isinstance(typ, Sequence) else typ
            if not isinstance(underlying_typ, (String, Int, Bool)):
                if not (isinstance(underlying_typ, Ref) and underlying_typ.url_encodable):
                    raise AssertionError(f"bad type in query parameter {name}: {typ}")

    def gen_function_param(self) -> Code:
        if self.required:
            typestr = self.type.annotation()
            default = ""
        else:
            typestr = f'"typing.Optional[{self.type.annotation(prequoted=True)}]"'
            default = " = None"
        return f"    {self.name}: {typestr}{default},"

    def gen_init_param(self) -> Code:
        if self.required:
            typestr = self.type.annotation()
            default = ""
        else:
            typestr = f'"typing.Union[{self.type.annotation(prequoted=True)}, None, Unset]"'
            default = " = _unset"
        return f"    {self.name}: {typestr}{default},"

    def dump(self, omit_unset: Code) -> Code:
        return self.type.dump(self.name, omit_unset)


class Class(TypeDef):
    def __init__(self, name: str, params: typing.Dict[str, Parameter]):
        self.name = name
        self.params = params

    def load(self, val: Code) -> Code:
        return f"{self.name}.from_json({val})"

    def dump(self, val: Code, omit_unset: Code) -> Code:
        return f"{val}.to_json({omit_unset})"

    def gen_def(self) -> Code:
        required = sorted(p for p in self.params if self.params[p].required)
        optional = sorted(p for p in self.params if not self.params[p].required)
        out = [f"class {self.name}:"]
        for k in optional:
            v = self.params[k]
            out += [f'    {k}: "typing.Optional[{v.type.annotation(prequoted=True)}]" = None']
        out += [""]
        out += ["    def __init__("]
        out += ["        self,"]
        out += ["        *,"]
        for k in required + optional:
            v = self.params[k]
            out += ["    " + v.gen_init_param()]
        out += ["    ):"]
        for k in required:
            out += [f"        self.{k} = {k}"]
        for k in optional:
            out += [f"        if not isinstance({k}, Unset):"]
            out += [f"            self.{k} = {k}"]
        out += [""]
        out += ["    @classmethod"]
        out += [f'    def from_json(cls, obj: Json) -> "{self.name}":']
        out += ['        kwargs: "typing.Dict[str, typing.Any]" = {']
        for k in required:
            v = self.params[k]
            if v.type.need_parse():
                parsed = v.type.load(f'obj["{k}"]')
            else:
                parsed = f'obj["{k}"]'
            out += [f'            "{k}": {parsed},']
        out += ["        }"]
        for k in optional:
            v = self.params[k]
            if v.type.need_parse():
                parsed = v.type.load(f'obj["{k}"]')
                parsed = parsed + f' if obj["{k}"] is not None else None'
            else:
                parsed = f'obj["{k}"]'
            out += [f'        if "{k}" in obj:']
            out += [f'            kwargs["{k}"] = {parsed}']
        out += ["        return cls(**kwargs)"]
        out += [""]
        out += ["    def to_json(self, omit_unset: bool = False) -> typing.Dict[str, typing.Any]:"]
        out += ['        out: "typing.Dict[str, typing.Any]" = {']
        for k in required:
            v = self.params[k]
            if v.type.need_parse():
                parsed = v.type.dump(f"self.{k}", "omit_unset")
            else:
                parsed = f"self.{k}"
            out.append(f'            "{k}": {parsed},')
        out += ["        }"]
        for k in optional:
            v = self.params[k]
            if v.type.need_parse():
                parsed = v.type.dump(f"self.{k}", "omit_unset")
                parsed = f"None if self.{k} is None else {parsed}"
            else:
                parsed = f"self.{k}"
            out += [f'        if not omit_unset or "{k}" in vars(self):']
            out += [f'            out["{k}"] = {parsed}']
        out += ["        return out"]

        return "\n".join(out)


class Enum(TypeDef):
    def __init__(self, name, members):
        self.name = name
        self.members = members

    def load(self, val: Code) -> Code:
        return f"{self.name}({val})"

    def dump(self, val: Code, omit_unset: Code) -> Code:
        return f"{val}.value"

    def gen_def(self) -> Code:
        out = [f"class {self.name}(enum.Enum):"]
        out += [f'    {v} = "{v}"' for v in self.members]
        return "\n".join(out)


class Function:
    def __init__(
        self,
        name: str,
        method: str,
        path: str,
        params: typing.Dict[str, Parameter],
        responses: typing.Dict[str, dict],
        streaming: bool,
    ):
        self.name = name
        self.method = method
        self.path = path
        self.params = params
        self.responses = responses
        self.streaming = streaming

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

    def gen_def(self) -> Code:
        # Function name.
        out = [f"def {self.method}_{self.name}("]

        # Function parameters.
        out += ['    session: "api.Session",']
        if self.params:
            out += ["    *,"]

        required = sorted(p for p in self.params if self.params[p].required)
        optional = sorted(p for p in self.params if not self.params[p].required)
        for name in required + optional:
            out += [self.params[name].gen_function_param()]

        # Function return type.
        # We wrap the return type annotation for streaming or union responses.
        need_quotes = self.streaming or len(self.responses) > 1
        returntypes = set(r.annotation(prequoted=need_quotes) for r in self.responses.values())
        returntypestr = ",".join(sorted(returntypes))
        if len(returntypes) > 1:
            returntypestr = f"typing.Union[{returntypestr}]"
        if self.streaming:
            returntypestr = f"typing.Iterable[{returntypestr}]"
        if need_quotes:
            returntypestr = f'"{returntypestr}"'

        out += [f") -> {returntypestr}:"]

        # Function body.
        path_params = sorted(p for p in self.params if self.params[p].where == "path")
        body_params = sorted(p for p in self.params if self.params[p].where == "body")
        query_params = sorted(p for p in self.params if self.params[p].where == "query")

        pathstr = f'"{self.path}"'
        if path_params:
            # Happily, we can just generate an f-string based on the path swagger gives us.
            pathstr = "f" + pathstr

        if query_params:
            out += ["    _params = {"]
            for p in query_params:
                param = self.params[p]
                if param.type.need_urlparam_dump():
                    value = f"{param.type.dump_as_urlparam(param.name)}"
                    if not param.required:
                        value += f" if {param.name} is not None else None"
                else:
                    value = param.name
                out += [f'        "{self.params[p].serialized_name}": {value},']
            out += ["    }"]
        else:
            out += ["    _params = None"]

        if "body" in self.params:
            # It is important that request bodies omit unset values so that PATCH request bodies
            # do not include extraneous None values.
            bodystr = self.params["body"].dump("True")
        else:
            bodystr = "None"
        out += ["    _resp = session._do_request("]
        out += [f'        method="{self.method.upper()}",']
        out += [f"        path={pathstr},"]
        out += ["        params=_params,"]
        out += [f"        json={bodystr},"]
        out += ["        data=None,"]
        out += ["        headers=None,"]
        out += ["        timeout=None,"]
        out += [f"        stream={self.streaming},"]
        out += ["    )"]
        for expect, returntype in self.responses.items():
            out += [f"    if _resp.status_code == {expect}:"]
            if not self.streaming:
                if returntype.isnone():
                    out += ["        return"]
                else:
                    out += [f'        return {returntype.load("_resp.json()")}']
            else:
                assert not returntype.isnone(), "unable to stream empty result class: {self}"
                # Too many quotes to do bit inline:
                yieldable = returntype.load('_j["result"]')
                out += [
                    f"        for _line in _resp.iter_lines():",
                    f"            _j = json.loads(_line)",
                    f'            if "error" in _j:',
                    f"                raise APIHttpStreamError(",
                    f'                    "{self.method}_{self.name}",',
                    f'                    runtimeStreamError.from_json(_j["error"])',
                    f"            )",
                    f"            yield {yieldable}",
                    f"        return",
                ]
        out += [f'    raise APIHttpError("{self.method}_{self.name}", _resp)']

        return "\n".join(out)


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


def gen_paginated(defs: TypeDefs) -> typing.List[str]:
    paginated = []
    for k, defn in defs.items():
        defn = defs[k]
        if defn is None or not isinstance(defn, Class):
            continue
        # Note that our goal is to mimic duck typing, so we only care if the "pagination" attribute
        # exists with a v1Pagination type.
        if any(n == "pagination" and p.type.name == "v1Pagination" for n, p in defn.params.items()):
            paginated.append(defn.name)

    if not paginated:
        return []

    out = []
    out += ["# Paginated is a union type of objects whose .pagination"]
    out += ["# attribute is a v1Pagination-type object."]
    out += ["Paginated = typing.Union["]
    out += [f"    {name}," for name in sorted(paginated)]
    out += ["]"]
    return out


def pybindings(swagger: dict) -> str:
    prefix = """
# Code generated by generate_bindings.py. DO NOT EDIT.
import enum
import json
import math
import typing

import requests

if typing.TYPE_CHECKING:
    from determined.common import api

# flake8: noqa
Json = typing.Any


# Unset is a type to distinguish between things not set and things set to None.
class Unset:
    pass


_unset = Unset()


def dump_float(val: typing.Any) -> typing.Any:
    if math.isnan(val):
        return "Nan"
    if math.isinf(val):
        return "Infinity" if val > 0 else "-Infinity"
    return val


class APIHttpError(Exception):
    # APIHttpError is used if an HTTP(s) API request fails.
    def __init__(self, operation_name: str, response: requests.Response) -> None:
        self.response = response
        self.operation_name = operation_name
        self.message = (
            f"API Error: {operation_name} failed: {response.reason}."
        )

    def __str__(self) -> str:
        return self.message


class APIHttpStreamError(APIHttpError):
    # APIHttpStreamError is used if an streaming API request fails mid-stream.
    def __init__(self, operation_name: str, error: "runtimeStreamError") -> None:
        self.operation_name = operation_name
        self.error = error
        self.message = (
            f"Stream Error during {operation_name}: {error.message}"
        )

    def __str__(self) -> str:
        return self.message

""".lstrip()

    out = [prefix]

    enums = process_enums(swagger["definitions"])
    defs = process_definitions(swagger["definitions"], enums)
    ops = process_paths(swagger["paths"], enums)
    link_all_refs(defs)

    for k in sorted(defs):
        defn = defs[k]
        if defn is None:
            continue
        out += [defn.gen_def()]
        out += [""]

    for k in sorted(ops):
        out += [ops[k].gen_def()]
        out += [""]

    # Also generate a list of Paginated response types.
    out += gen_paginated(defs)

    return "\n".join(out).strip()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--output", "-o", action="store", required=True, help="output file")
    args = parser.parse_args()

    with open(SWAGGER) as f:
        swagger = json.load(f)
    bindings = pybindings(swagger)
    with open(args.output, "w") as f:
        print(bindings, file=f)
