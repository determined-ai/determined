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

    def dump(self, val: Code) -> Code:
        raise NotImplementedError(type(self))

    def isnone(self) -> bool:
        # Only Refs to empty structs ever return True; we skip generating them.
        return False


class TypeDef:
    def gen_def(self) -> Code:
        raise NotImplementedError(type(self))


class NoParse:
    """A compositional class for things where json.loads/dumps is sufficient."""

    def need_parse(self) -> bool:
        return False

    def load(self, val: Code) -> Code:
        return val

    def dump(self, val: Code) -> Code:
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

    def dump(self, val: Code) -> Code:
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


class Ref(TypeAnno):
    # Collect refs as we instantiate them, for the linking step.
    all_refs = []

    def __init__(self, name: str):
        self.name = name
        self.linked = False
        self.defn = None
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
        assert isinstance(self.defn, (Enum, Class)), (self.name, type(self.defn).__name__)
        return self.defn.load(val)

    def dump(self, val: Code) -> Code:
        assert self.linked, "link step must be completed before generating code!"
        assert self.defn, "it doesn't make sense to dump an empty class"
        assert isinstance(self.defn, (Enum, Class)), (self.name, type(self.defn).__name__)
        return self.defn.dump(val)


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

    def dump(self, val: Code) -> Code:
        if not self.need_parse():
            return val
        return f"{{k: {self.values.dump('v')} for k, v in {val}.items()}}"


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

    def dump(self, val: Code) -> Code:
        if not self.need_parse():
            return val
        return f"[{self.items.dump('x')} for x in {val}]"

class Parameter:
    def __init__(self, name: str, typ: TypeAnno, required: bool, where: str,
                 serialized_name: str = None) -> None:
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
            if not isinstance(typ, (String, Int, Bool)):
                if not isinstance(typ, Sequence) or not isinstance(typ.items, (String, Int, Bool)):
                    raise AssertionError(f"bad type in query parameter {name}: {typ}")

    def gen_function_param(self) -> Code:
        if self.required:
            typestr = self.type.annotation()
            default = ""
        else:
            typestr = f'"typing.Optional[{self.type.annotation(prequoted=True)}]"'
            default = " = None"
        default = "" if self.required else " = None"
        return f"    {self.name}: {typestr}{default},"

    def dump(self) -> Code:
        return self.type.dump(self.name)


class Class(TypeDef):
    def __init__(self, name: str, params: typing.Dict[str, Parameter]):
        self.name = name
        # self.members = members
        self.params = params

    def load(self, val: Code) -> Code:
        return f"{self.name}.from_json({val})"

    def dump(self, val: Code) -> Code:
        return f"{val}.to_json()"

    def gen_def(self) -> Code:
        out = [f"class {self.name}:"]
        out += ["    def __init__("]
        out += ["        self,"]
        required = sorted(p for p in self.params if self.params[p].required)
        optional = sorted(p for p in self.params if not self.params[p].required)
        for name in required + optional:
            out += ["        " + self.params[name].gen_function_param()]
        # out += [f"        {k}: {v.annotation()}," for k, v in self.members.items()]
        out += ["    ):"]
        out += [f"        self.{k} = {k}" for k in self.params]
        out += [""]
        out += ["    @classmethod"]
        out += ["    def from_json(cls, obj):"]
        out += ["        return cls("]
        for k, v in self.params.items():
            if v.type.need_parse():
                parsed = v.type.load(f'obj["{k}"]')
            else:
                parsed = f'obj["{k}"]'
            if not v.required:
                parsed = parsed + f' if obj.get("{k}", None) is not None else None'
            out.append(f"""             {k}={parsed},""")
        out += ["        )"]
        out += [""]
        out += ["    def to_json(self):"]
        out += ["        return {"]
        for k, v in self.params.items():
            if v.type.need_parse():
                parsed = v.type.dump(f"self.{k}")
            else:
                parsed = f"self.{k}"
            if not v.required:
                parsed = parsed + f" if self.{k} is not None else None"
            out.append(f'             "{k}": {parsed},')
        out += ["        }"]

        return "\n".join(out)


class Enum(TypeDef):
    def __init__(self, name, members):
        self.name = name
        self.members = members

    def load(self, val: Code) -> Code:
        return f"{val}"

    def dump(self, val: Code) -> Code:
        return f"{val}"

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
    ):
        self.name = name
        self.method = method
        self.path = path
        self.params = params
        self.responses = responses

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
        out += ['    session: "client.Session",']
        if self.params:
            out += ["    *,"]

        required = sorted(p for p in self.params if self.params[p].required)
        optional = sorted(p for p in self.params if not self.params[p].required)
        for name in required + optional:
            out += [self.params[name].gen_function_param()]

        # Function return type.
        # (simplifying assumptions; if broken we need more logic)
        responses = {**self.responses}
        default = responses.pop("default")
        assert isinstance(default, Ref) and default.name == "runtimeError", (self.name, default)

        if len(responses) == 1:
            returntype = next(iter(responses.values()))
            returntypestr = returntype.annotation()
        else:
            returntypes = set(r.annotation(prequoted=True) for r in responses.values())
            returntypestr = '"Union[' + ", ".join(sorted(returntypes)) + ']"'
        assert len(responses) == 1, (self.name, responses)

        out += [f') -> {returntypestr}:']

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
                out += [f'        "{self.params[p].serialized_name}": {self.params[p].name},']
            out += ["    }"]
        else:
            out += ["    _params = None"]

        if "body" in self.params:
            bodystr = self.params["body"].dump()
        else:
            bodystr = "None"

        out += ["    _req = session._do_request("]
        out += [f"        method=\"{self.method.upper()}\","]
        out += [f"        path={pathstr},"]
        out += ["        params=_params,"]
        out += [f"        json={bodystr},"]
        out += ["        data=None,"]
        out += ["        headers=None,"]
        out += ["        timeout=None,"]
        out += ["    )"]
        for expect, returntype in responses.items():
            out += [f"    if _req.status_code == {expect}:"]
            if returntype.isnone():
                out += ['        return']
            else:
                out += [f'        return {returntype.load("_req.json()")}']
        # XXX: raise something else here
        out += ["    raise ValueError(_req.status_code)"]

        return "\n".join(out)


def classify_type(path: str, schema: dict) -> TypeAnno:
    # enforce valid jsonschema:
    assert isinstance(schema, dict), (path, schema)
    # XXX: how to handle inlined enums?
    # assert "enum" not in schema, (path, schema)

    if "$ref" in schema:
        ref = schema["$ref"]
        start = "#/definitions/"
        assert ref.startswith(start), ref
        return Ref(ref[len(start) :])

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
        return Dict(classify_type(path + ".additionalProperties", adlProps))

    if schema["type"] == "array":
        items = schema.get("items")
        if items is None:
            raise ValueError(path, schema)
            return Sequence(Any())
        return Sequence(classify_type(path + ".items", items))

    raise ValueError(f"unhandled schema: {schema} @ {path}")


def process_definitions(swagger_definitions: dict) -> TypeDefs:
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
                    k: Parameter(k, classify_type(path, v), (k in required), "definitions") for k, v in schema["properties"].items()
                }
                # TODO create parameters
                defs[name] = Class(name, members)
                continue
            else:
                # empty responses or empty requests... we don't care.
                defs[name] = None
                continue
        raise ValueError(f"unhandled schema: {schema} @ {path}")
    return defs


def process_paths(swagger_paths: dict, defs: TypeDefs) -> typing.Dict[str, Function]:
    ops = {}
    for path, methods in swagger_paths.items():
        for method, spec in methods.items():
            name = spec["operationId"]
            # Figure out response types.
            responses = {}
            bad_op = False
            for code, rspec in spec["responses"].items():
                if rspec.get("schema", {}).get("title", "").startswith("Stream result"):
                    # TODO: support streaming endpoints.
                    bad_op = True
                    break
                if rspec["schema"].get("type") == "":
                    # not a valid response schema, skipping
                    bad_op = True
                    break
                responses[code] = classify_type(f"{name}.responses.{code}", rspec["schema"])
            if bad_op:
                continue

            # Figure out parameters.
            params = {}
            for pspec in spec.get("parameters", []):
                where = pspec["in"]
                serialized_name = None
                if where == "query": # preserve query parameter names
                    serialized_name = pname = pspec["name"]
                pname = pspec["name"].replace(".", "_")
                required = pspec.get("required", False)
                if "schema" in pspec:
                    pschema = pspec["schema"]
                else:
                    # swagger has some weird inlining going on here...
                    inlined = ("type", "format", "items", "properties", "enum")
                    pschema = {k: pspec[k] for k in inlined if k in pspec}
                ptype = classify_type(f"{name}.{pname}", pschema)
                params[pname] = Parameter(pname, ptype, required, where, serialized_name)

            # TODO: Validate before altering the whole path.
            path = path.replace(".", "_")
            op = Function(name, method, path, params, responses)
            ops[name] = op
    return ops


def link_all_refs(defs: TypeDefs) -> None:
    for ref in Ref.all_refs:
        ref.linked = True
        ref.defn = defs[ref.name]


def pybindings(swagger: dict) -> str:
    out = ["import math"]
    out += ["import enum"]
    out += ["import typing"]
    out += [""]
    out += ["if typing.TYPE_CHECKING:"]
    out += ["    from determined.experimental import client"]
    out += [""]
    out += ["def dump_float(val):"]
    out += ["    if math.isnan(val):"]
    out += ['        return "Nan"']
    out += ["    if math.isinf(val):"]
    out += ['        return "Infinity" if val > 0 else "-Infinity"']
    out += ["    return val"]
    out += [""]

    defs = process_definitions(swagger["definitions"])
    ops = process_paths(swagger["paths"], defs)
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

    return "\n".join(out).strip()


if __name__ == "__main__":
    with open(SWAGGER) as f:
        swagger = json.load(f)

    print(pybindings(swagger))
