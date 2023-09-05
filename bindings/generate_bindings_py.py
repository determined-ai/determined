import os
import typing

import swagger_parser
from typing_extensions import assert_never

SWAGGER = "proto/build/swagger/determined/api/v1/api.swagger.json"
SWAGGER = os.path.join(os.path.dirname(__file__), "..", SWAGGER)
TAB = "    "

Code = str
SwaggerType = typing.Union[swagger_parser.TypeAnno, swagger_parser.TypeDef]
no_parse_types = (
    swagger_parser.Any,
    swagger_parser.String,
    swagger_parser.DateTime,
    swagger_parser.Int,
    swagger_parser.Bool,
)


def annotation(anno: swagger_parser.TypeAnno, prequoted=False) -> Code:
    if isinstance(anno, swagger_parser.Any):
        return "typing.Any"
    if isinstance(anno, (swagger_parser.String, swagger_parser.DateTime)):
        return "str"
    if isinstance(anno, swagger_parser.Float):
        return "float"
    if isinstance(anno, swagger_parser.Int):
        return "int"
    if isinstance(anno, swagger_parser.Bool):
        return "bool"
    if isinstance(anno, swagger_parser.Ref):
        if anno.defn is None or (
            isinstance(anno.defn, swagger_parser.Class) and not anno.defn.params
        ):
            return "None"
        out = anno.name
        if not prequoted:
            return f'"{out}"'
        return out
    if isinstance(anno, swagger_parser.Dict):
        out = f"typing.Dict[str, {annotation(anno.values, True)}]"
        if not prequoted:
            return f'"{out}"'
        return out
    if isinstance(anno, swagger_parser.Sequence):
        out = f"typing.Sequence[{annotation(anno.items, True)}]"
        if not prequoted:
            return f'"{out}"'
        return out
    assert_never(anno)


def need_parse(anno: swagger_parser.TypeAnno) -> bool:
    if isinstance(anno, no_parse_types):
        return False
    if isinstance(anno, (swagger_parser.Float, swagger_parser.Ref)):
        return True
    if isinstance(anno, swagger_parser.Dict):
        return need_parse(anno.values)
    if isinstance(anno, swagger_parser.Sequence):
        return need_parse(anno.items)
    assert_never(anno)


def load(anno: SwaggerType, val: Code) -> Code:
    if isinstance(anno, no_parse_types):
        return val
    if isinstance(anno, swagger_parser.Float):
        return f"float({val})"
    if isinstance(anno, swagger_parser.Ref):
        assert anno.defn
        return load(anno.defn, val)
    if isinstance(anno, swagger_parser.Dict):
        if not need_parse(anno):
            return val
        return f"{{k: {load(anno.values, 'v')} for k, v in {val}.items()}}"
    if isinstance(anno, swagger_parser.Sequence):
        if not need_parse(anno):
            return val
        return f"[{load(anno.items, 'x')} for x in {val}]"
    if isinstance(anno, swagger_parser.Class):
        return f"{anno.name}.from_json({val})"
    if isinstance(anno, swagger_parser.Enum):
        return f"{anno.name}({val})"
    assert_never(anno)


def dump(anno: SwaggerType, val: Code, omit_unset: Code) -> Code:
    if isinstance(anno, no_parse_types):
        return val
    if isinstance(anno, swagger_parser.Float):
        return f"dump_float({val})"
    if isinstance(anno, swagger_parser.Ref):
        assert anno.defn
        return dump(anno.defn, val, omit_unset)
    if isinstance(anno, swagger_parser.Dict):
        if not need_parse(anno):
            return val
        each = dump(anno.values, "v", omit_unset)
        return f"{{k: {each} for k, v in {val}.items()}}"
    if isinstance(anno, swagger_parser.Sequence):
        if not need_parse(anno):
            return val
        each = dump(anno.items, "x", omit_unset)
        return f"[{each} for x in {val}]"
    if isinstance(anno, swagger_parser.Class):
        return f"{val}.to_json({omit_unset})"
    if isinstance(anno, swagger_parser.Enum):
        return f"{val}.value"
    assert_never(anno)


def gen_init_param(param: swagger_parser.Parameter) -> Code:
    if param.required:
        typestr = annotation(param.type)
        default = ""
    else:
        typestr = f'"typing.Union[{annotation(param.type, prequoted=True)}, None, Unset]"'
        default = " = _unset"
    return f"    {param.name}: {typestr}{default},"


def gen_function_param(param: swagger_parser.Parameter):
    if param.required:
        typestr = annotation(param.type)
        default = ""
    else:
        typestr = f'"typing.Optional[{annotation(param.type, prequoted=True)}]"'
        default = " = None"
    return f"    {param.name}: {typestr}{default},"


def is_streaming_response(defn: typing.Optional[swagger_parser.TypeDef]):
    if isinstance(defn, swagger_parser.Class) and set(defn.params.keys()) == set(
        ["result", "error"]
    ):
        error_ref = defn.params["error"]
        return (
            isinstance(error_ref.type, swagger_parser.Ref)
            and error_ref.type.name == "runtimeStreamError"
        )
    return False


def unwrap_streaming_response(anno: swagger_parser.TypeAnno):
    if not isinstance(anno, swagger_parser.Ref) or not is_streaming_response(anno.defn):
        return anno
    assert isinstance(anno.defn, swagger_parser.Class)
    return anno.defn.params["result"].type


def breakup_comment(comment: typing.Optional[str]) -> typing.List[str]:
    if not comment:
        return []
    lines = list(filter(lambda x: bool(x), [line.strip() for line in comment.split("\n")]))
    return lines


def gen_function_docstring(func: swagger_parser.Function) -> typing.List[str]:
    """generate docstring for generated functions with summarized parameter support"""
    out = []
    required = sorted((k, v) for k, v in func.params.items() if v.required)
    optional = sorted((k, v) for k, v in func.params.items() if not v.required)
    params_in_order = [
        param for _, param in required + optional if param.title is not None and param.title.strip()
    ]

    if not params_in_order and not func.summary:
        return out
    out += ['"""']
    summary_lines = breakup_comment(func.summary)
    if summary_lines:
        out[-1] += summary_lines[0]
        out += summary_lines[1:]
    if params_in_order:
        out += [""]
        for param in params_in_order:
            out += [f"- {param.name}: {param.title}"]

    if len(out) == 1:
        out[-1] += '"""'
    else:
        out += ['"""']
    return out


def description_to_docstring(description: typing.Optional[str]) -> typing.List[str]:
    out = []
    lines = breakup_comment(description)
    if not lines:
        return []
    elif len(lines) == 1:
        out += [f'"""{lines[0]}"""']
    else:
        out += ['"""' + lines[0]]
        out += lines[1:]
        out += ['"""']
    return out


def gen_function(func: swagger_parser.Function) -> Code:
    # Function name.
    out = [f"def {func.operation_name_sc()}("]

    # Function parameters.
    out += ['    session: "api.Session",']
    if func.params:
        out += ["    *,"]

    required = sorted((k, v) for k, v in func.params.items() if v.required)
    optional = sorted((k, v) for k, v in func.params.items() if not v.required)

    for _, param in required + optional:
        out += [gen_function_param(param)]

    # Function return type.
    # We wrap the return type annotation for streaming or union responses.
    need_quotes = func.streaming or len(func.responses) > 1
    if func.streaming:
        func.responses = {code: unwrap_streaming_response(r) for code, r in func.responses.items()}
    returntypes = set(annotation(r, prequoted=need_quotes) for r in func.responses.values())
    returntypestr = ",".join(sorted(returntypes))
    if len(returntypes) > 1:
        returntypestr = f"typing.Union[{returntypestr}]"
    if func.streaming:
        returntypestr = f"typing.Iterable[{returntypestr}]"
    if need_quotes:
        returntypestr = f'"{returntypestr}"'

    out += [f") -> {returntypestr}:"]
    out += [TAB + line if line else "" for line in gen_function_docstring(func)]

    # Function body.
    has_path_params = any(p for p in func.params.values() if p.where == "path")
    # body_params = sorted(p for p in self.params if func.params[p].where == "body") # not in use
    query_params = sorted((k, v) for k, v in func.params.items() if v.where == "query")

    pathstr = f'"{func.path}"'
    if has_path_params:
        # Happily, we can just generate an f-string based on the path swagger gives us.
        pathstr = "f" + pathstr

    if query_params:
        out += ["    _params = {"]
        for _, param in query_params:
            if isinstance(param.type, swagger_parser.Bool):
                value = f"str({param.name}).lower()"
                if not param.required:
                    value += f" if {param.name} is not None else None"
            elif need_parse(param.type):
                value = f"{dump(param.type, param.name, omit_unset='True')}"
                if not param.required:
                    value += f" if {param.name} is not None else None"
            else:
                value = param.name
            out += [f'        "{param.serialized_name}": {value},']
        out += ["    }"]
    else:
        out += ["    _params = None"]

    if "body" in func.params:
        # It is important that request bodies omit unset values so that PATCH request bodies
        # do not include extraneous None values.
        body_param = func.params["body"]
        bodystr = dump(body_param.type, body_param.name, "True")
    else:
        bodystr = "None"
    out += ["    _resp = session._do_request("]
    out += [f'        method="{func.method.upper()}",']
    out += [f"        path={pathstr},"]
    out += ["        params=_params,"]
    out += [f"        json={bodystr},"]
    out += ["        data=None,"]
    out += ["        headers=None,"]
    out += ["        timeout=None,"]
    out += [f"        stream={func.streaming},"]
    out += ["    )"]
    for expect, returntype in func.responses.items():
        out += [f"    if _resp.status_code == {expect}:"]
        is_none = isinstance(returntype, swagger_parser.Ref) and (
            returntype.defn is None
            or isinstance(returntype.defn, swagger_parser.Class)
            and not returntype.defn.params
        )
        if not func.streaming:
            if is_none:
                out += ["        return"]
            else:
                out += [f'        return {load(returntype, "_resp.json()")}']
        else:
            assert not is_none, "unable to stream empty result class: {func}"
            # Too many quotes to do it inline:
            yieldable = load(returntype, '_j["result"]')
            out += [
                f"        try:",
                f"            for _line in _resp.iter_lines(chunk_size=1024 * 1024):",
                f"                _j = json.loads(_line)",
                f'                if "error" in _j:',
                f"                    raise APIHttpStreamError(",
                f'                        "{func.operation_name_sc()}",',
                f'                        runtimeStreamError.from_json(_j["error"])',
                f"                )",
                f"                yield {yieldable}",
                f"        except requests.exceptions.ChunkedEncodingError:",
                f'            raise APIHttpStreamError("{func.operation_name_sc()}", runtimeStreamError(message="ChunkedEncodingError"))',
                f"        return",
            ]
    out += [f'    raise APIHttpError("{func.operation_name_sc()}", _resp)']

    return "\n".join(out)


def gen_class(klass: swagger_parser.Class) -> Code:
    required = sorted((k, v) for k, v in klass.params.items() if v.required)
    optional = sorted((k, v) for k, v in klass.params.items() if not v.required)

    out = [f"class {klass.name}(Printable):"]
    if klass.description:
        out += [TAB + line if line else "" for line in description_to_docstring(klass.description)]
    for k, v in optional:
        out += [f'    {k}: "typing.Optional[{annotation(v.type, prequoted=True)}]" = None']
    out += [""]
    out += ["    def __init__("]
    out += ["        self,"]
    out += ["        *,"]
    for k, v in required + optional:
        out += ["    " + gen_init_param(v)]
    out += ["    ):"]
    for k, _ in required:
        out += [f"        self.{k} = {k}"]
    for k, _ in optional:
        out += [f"        if not isinstance({k}, Unset):"]
        out += [f"            self.{k} = {k}"]
    out += [""]
    out += ["    @classmethod"]
    out += [f'    def from_json(cls, obj: Json) -> "{klass.name}":']
    out += ['        kwargs: "typing.Dict[str, typing.Any]" = {']
    for k, v in required:
        if need_parse(v.type):
            parsed = load(v.type, f'obj["{k}"]')
        else:
            parsed = f'obj["{k}"]'
        out += [f'            "{k}": {parsed},']
    out += ["        }"]
    for k, v in optional:
        if need_parse(v.type):
            parsed = load(v.type, f'obj["{k}"]')
            parsed = parsed + f' if obj["{k}"] is not None else None'
        else:
            parsed = f'obj["{k}"]'
        out += [f'        if "{k}" in obj:']
        out += [f'            kwargs["{k}"] = {parsed}']
    out += ["        return cls(**kwargs)"]
    out += [""]
    out += ["    def to_json(self, omit_unset: bool = False) -> typing.Dict[str, typing.Any]:"]
    out += ['        out: "typing.Dict[str, typing.Any]" = {']
    for k, v in required:
        if need_parse(v.type):
            parsed = dump(v.type, f"self.{k}", "omit_unset")
        else:
            parsed = f"self.{k}"
        out.append(f'            "{k}": {parsed},')
    out += ["        }"]
    for k, v in optional:
        if need_parse(v.type):
            parsed = dump(v.type, f"self.{k}", "omit_unset")
            parsed = f"None if self.{k} is None else {parsed}"
        else:
            parsed = f"self.{k}"
        out += [f'        if not omit_unset or "{k}" in vars(self):']
        out += [f'            out["{k}"] = {parsed}']
    out += ["        return out"]

    return "\n".join(out)


def gen_enum(enum: swagger_parser.Enum) -> Code:
    out = [f"class {enum.name}(DetEnum):"]
    if enum.description:
        out += [TAB + line if line else "" for line in description_to_docstring(enum.description)]
    prefix = os.path.commonprefix(enum.members)
    skip = len(prefix) if prefix.endswith("_") else 0
    out += [f'    {v[skip:]} = "{v}"' for v in enum.members]
    return "\n".join(out)


def gen_def(anno: swagger_parser.TypeDef) -> Code:
    if isinstance(anno, swagger_parser.Class):
        return gen_class(anno)
    if isinstance(anno, swagger_parser.Enum):
        return gen_enum(anno)
    assert_never(anno)


def gen_paginated(defs: swagger_parser.TypeDefs) -> typing.List[str]:
    paginated = []
    for k, defn in defs.items():
        defn = defs[k]
        if defn is None or not isinstance(defn, swagger_parser.Class):
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


def skip_defn(defn: swagger_parser.TypeDef):
    return (isinstance(defn, swagger_parser.Class) and not defn.params) or is_streaming_response(
        defn
    )


def pybindings(swagger: swagger_parser.ParseResult) -> str:
    prefix = """
# Code generated by generate_bindings.py. DO NOT EDIT.
import enum
import json
import math
import os
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


class DetEnum(enum.Enum):
    def __str__(self) -> str:
        skip = len(self.prefix())
        return f"{self.value[skip:]}"
    @classmethod
    def prefix(cls) -> str:
        prefix: str = os.path.commonprefix([e.value for e in cls])
        return prefix if prefix.endswith("_") else ""


class Printable:
    # A mixin to provide a __str__ method for classes with attributes.
    def __str__(self) -> str:
        allowed_types = (str, int, float, bool, DetEnum)
        attrs = []
        for k, v in self.__dict__.items():
            if v is None: continue
            if isinstance(v, list):
                vals = [str(x) if isinstance(x, allowed_types) else "..." for x in v]
                attrs.append(f'{k}=[{", ".join(vals)}]')
            elif isinstance(v, allowed_types):
                attrs.append(f'{k}={v}')
            else:
                attrs.append(f'{k}=...')
        attrs_str = ', '.join(attrs)
        return f'{self.__class__.__name__}({attrs_str})'


""".lstrip()

    out = [prefix]

    for _, defn in sorted(swagger.defs.items()):
        if defn is None or skip_defn(defn):
            continue
        out += [gen_def(defn)]
        out += [""]

    for _, op in sorted(swagger.ops.items()):
        out += [gen_function(op)]
        out += [""]

    # Also generate a list of Paginated response types.
    out += gen_paginated(swagger.defs)

    return "\n".join(out).strip()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--output", "-o", action="store", required=True, help="output file")
    args = parser.parse_args()

    swagger = swagger_parser.parse(SWAGGER)
    bindings = pybindings(swagger)
    with open(args.output, "w") as f:
        print(bindings, file=f)
