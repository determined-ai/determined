from dataclasses import dataclass, field
import os
import re
from shutil import copy, rmtree
import swagger_parser
import typing
from typing_extensions import assert_never, Literal

DIRNAME = os.path.dirname(__file__)
SWAGGER = "proto/build/swagger/determined/api/v1/api.swagger.json"
SWAGGER = os.path.join(DIRNAME, "..", SWAGGER)
STATIC_FOLDER = os.path.join(DIRNAME, "static_ts_files")

Code = str
SwaggerType = typing.Union[
    swagger_parser.TypeAnno, swagger_parser.TypeDef, swagger_parser.Parameter
]


def head(item: typing.Tuple[str, typing.Any]) -> str:
    return item[0].lower()


def upper_first(string: str) -> str:
    return string[0].upper() + string[1:]


# simplified, unsafe-ish camel case function
def camel_case(string: str) -> str:
    replace_upper = lambda x: x.group(1).upper()
    return string[0].lower() + re.sub(r"[^a-zA-Z0-9]([a-zA-Z0-9])", replace_upper, string[1:])


# small class to make adding lines of code slightly easier to read
@dataclass
class IndentedLines:
    lines: typing.List[str] = field(default_factory=list, init=False)
    commenting: bool = field(default=False, init=False)
    indent_level: int = field(default=0)
    tab_char: str = field(default="    ")

    def __post_init__(self):
        assert self.indent_level >= 0, "indent level cannot be lower than 0"

    def indent(self):
        self.indent_level += 1

    def dedent(self):
        self.indent_level = max(self.indent_level - 1, 0)

    def add_line(self, line: str):
        comment_char = " * " if self.commenting else ""
        self.lines += [(self.tab_char * self.indent_level) + comment_char + line]

    def add_lines(self, lines: typing.List[str]):
        for line in lines:
            self.add_line(line)

    def start_comment(self):
        self.add_line("/**")
        self.commenting = True

    def end_comment(self):
        self.commenting = False
        self.add_line(" */")

    def __iadd__(self, line_or_lines: typing.Union[str, typing.List[str]]) -> "IndentedLines":
        if isinstance(line_or_lines, list):
            self.add_lines(line_or_lines)
        else:
            self.add_line(line_or_lines)
        return self

    def __str__(self) -> str:
        return "\n".join(self.lines)


def annotation(anno: swagger_parser.TypeAnno) -> Code:
    if isinstance(anno, swagger_parser.Any):
        return "any"
    if isinstance(anno, swagger_parser.String):
        return "string"
    if isinstance(anno, (swagger_parser.Float, swagger_parser.Int)):
        return "number"
    if isinstance(anno, swagger_parser.Bool):
        return "boolean"
    if isinstance(anno, swagger_parser.DateTime):
        return "Date"
    if isinstance(anno, swagger_parser.Ref):
        if not anno.defn:
            return "void"
        return anno.name[0].upper() + anno.name[1:]
    if isinstance(anno, swagger_parser.Dict):
        if isinstance(anno.values, swagger_parser.Any):
            return "any"
        return f"{{ [key: string]: {annotation(anno.values)}; }}"
    if isinstance(anno, swagger_parser.Sequence):
        return f"Array<{annotation(anno.items)}>"
    assert_never(anno)


def longest_common_prefix(members: typing.List[str]) -> str:
    prefix = min(members, key=len)
    while not all(member.startswith(prefix) for member in members):
        prefix = prefix[:-1]
    return prefix


def gen_def(anno: swagger_parser.TypeDef) -> Code:
    code = IndentedLines()
    if isinstance(anno, swagger_parser.Class):
        clean_description = (anno.description or "").replace("\n", " ")
        proper_name = upper_first(anno.name)

        code.start_comment()
        code += f"{clean_description}"
        code += "@export"
        code += f"@interface {proper_name}"
        code.end_comment()

        code += f"export interface {proper_name} {{"
        code.indent()
        for _, param in anno.params.items():
            clean_title = (param.title or "").replace("\n", " ")
            required_txt = ":" if param.required else "?:"
            param_annotation = annotation(param.type)
            code.start_comment()
            code += f"{clean_title}"
            code += f"@type {{{param_annotation}}}"
            code += f"@memberof {proper_name}"
            code.end_comment()
            code += f"{param.name}{required_txt} {param_annotation};"
        code.dedent()
        code += "}"
        return str(code)
    if isinstance(anno, swagger_parser.Enum):
        clean_description = (anno.description or "").replace("\n", " ")
        proper_name = anno.name[0].upper() + anno.name[1:]

        code.start_comment()
        code += clean_description
        code += "@export"
        code += "@enum {string}"  # parser assumes all enums are strings
        code.end_comment()

        code += f"export const {proper_name} = {{"
        code.indent()
        # can't find where this logic exists in the old codegen -- aargh
        prefix = longest_common_prefix(anno.members) if len(anno.members) > 1 else ""
        for member in anno.members:
            clean_member_name = member[len(prefix) :].replace("_", "")
            code += f"{clean_member_name}: '{member}',"
        code.dedent()
        code += "} as const"

        code += f"export type {proper_name} = ValueOf<typeof {proper_name}>"
        return str(code)
    assert_never(anno)


Phase = Literal["params", "fp", "factory", "api"]


def generate_function(api: str, phase: Phase, function: swagger_parser.Function) -> Code:
    indent_level = 1 if phase == "api" else 2
    code = IndentedLines(indent_level)
    function_name = camel_case(function.name)

    params_by_location: typing.Dict[str, typing.List[swagger_parser.Parameter]] = {}
    for param in function.params.values():
        params_by_location[param.where] = params_by_location.get(param.where, [])
        params_by_location[param.where].append(param)

    params_in_order = (
        params_by_location.get("path", [])
        + params_by_location.get("query", [])
        + params_by_location.get("body", [])
    )

    function_args = ", ".join(
        f"{param.name}{'' if param.required else '?'}: {annotation(param.type)}"
        for param in params_in_order
    )
    call_list = ", ".join(param.name for param in params_in_order)
    if function_args:
        function_args += ", "
        call_list += ", "

    jsdoc: typing.List[str] = []
    if function.summary is not None:
        summary = function.summary.replace("\n", " ")
        jsdoc += [f"@summary {summary}"]
    for param in params_in_order:
        param_name = param.name if param.required else f"[{param.name}]"
        jsdoc += [
            f"@param {{{annotation(param.type)}}} {param_name} {param.title or ''}".replace(
                "\n", " "
            ).strip()
        ]
    jsdoc += ["@param {*} [options] Override http request option."]
    jsdoc += ["@throws {RequiredError}"]  # does the code generator throw anything else?

    if phase == "params":
        code.start_comment()
        code += ""  # something might be missing here
        code += jsdoc
        code.end_comment()

        code += f"{function_name}({function_args}options: any = {{}}): FetchArgs {{"
        code.indent()
        for param in params_in_order:
            if param.required:
                code += f"// verify required parameter '{param.name}' is not null or undefined"
                code += f"if ({param.name} === null || {param.name} === undefined) {{"
                code.indent()
                code += f"throw new RequiredError('{param.name}','Required parameter {param.name} was null or undefined when calling {function_name}.');"
                code.dedent()
                code += "}"
        path_params = params_by_location.get("path")
        fixed_path = re.sub(
            r"{(\w*)}", lambda match: f"{{{camel_case(match.group(1))}}}", function.path
        )
        code += f"const localVarPath = `{fixed_path}`{'' if path_params else ';'}"
        if path_params:
            code.indent()
            for n, param in enumerate(path_params, start=1):
                line = (
                    f'.replace(`{{${{"{param.name}"}}}}`, encodeURIComponent(String({param.name})))'
                )
                code += f"{line}{';' if n == len(path_params) else ''}"
            code.dedent()
        code += "const localVarUrlObj = new URL(localVarPath, BASE_PATH);"
        code += (
            f"const localVarRequestOptions = {{ method: '{function.method.upper()}', ...options }};"
        )
        code += "const localVarHeaderParameter = {} as any;"
        code += "const localVarQueryParameter = {} as any;"
        code += ""

        # should only need bearer token auth -- if this breaks, handle parsing auth types
        if function.needs_auth:
            code += "// authentication BearerToken required"
            code += "if (configuration && configuration.apiKey) {"
            code.indent()
            code += "const localVarApiKeyValue = typeof configuration.apiKey === 'function'"
            code.indent()
            code += '? configuration.apiKey("Authorization")'
            code += ": configuration.apiKey;"
            code.dedent()
            code += 'localVarHeaderParameter["Authorization"] = localVarApiKeyValue;'
            code.dedent()
            code += "}"
            code += ""

        for param in params_by_location.get("query", []):
            null_check = (
                ""
                if isinstance(param.type, (swagger_parser.Sequence, swagger_parser.DateTime))
                else " !== undefined"
            )
            code += f"if ({param.name}{null_check}) {{"
            code.indent()
            if isinstance(param.type, swagger_parser.DateTime):
                code += f"localVarQueryParameter['{param.serialized_name}'] = {param.name}.toISOString()"
            else:
                code += f"localVarQueryParameter['{param.serialized_name}'] = {param.name}"
            code.dedent()
            code += "}"
            code += ""

        # should only be one required body parameter that's application/json? if
        # this breaks, handle parsing consumption types
        body_param = next(iter(params_by_location.get("body", [])), None)
        if body_param:
            code += "localVarHeaderParameter['Content-Type'] = 'application/json';"
            code += ""

        # original code for queryparameters used the node querystrings module --
        # it treats objects and arrays differently.
        code += "objToSearchParams(localVarQueryParameter, localVarUrlObj.searchParams);"
        code += "objToSearchParams(options.query || {}, localVarUrlObj.searchParams);"
        code += (
            "localVarRequestOptions.headers = { ...localVarHeaderParameter, ...options.headers };"
        )

        if body_param:
            if not isinstance(body_param.type, swagger_parser.String):
                code += f"localVarRequestOptions.body = JSON.stringify({body_param.name})"
            else:
                code += "const needsSerialization = localVarRequestOptions.headers['Content-Type'] === 'application/json';"
                code += f"localVarRequestOptions.body = needsSerialization ? JSON.stringify({body_param.name}) : {body_param.name}"

        code += ""
        code += "return {"
        code.indent()
        code += "url: `${localVarUrlObj.pathname}${localVarUrlObj.search}`,"
        code += "options: localVarRequestOptions,"
        code.dedent()
        code += "};"
        code.dedent()
        code += "},"

        return str(code)
    if phase == "fp":
        code.start_comment()
        code += ""
        code += jsdoc
        code.end_comment()

        success_response = function.responses.get("200")
        assert success_response, function

        code += f"{function_name}({function_args}options?: any): (fetch?: FetchAPI, basePath?: string) => Promise<{annotation(success_response)}> {{"
        code.indent()
        code += f"const localVarFetchArgs = {api}FetchParamCreator(configuration).{function_name}({call_list}options);"
        code += "return (fetch: FetchAPI = window.fetch, basePath: string = BASE_PATH) => {"
        code.indent()
        code += "return fetch(basePath + localVarFetchArgs.url, localVarFetchArgs.options).then((response) => {"
        code.indent()
        code += "if (response.status >= 200 && response.status < 300) {"
        code.indent()
        code += "return response.json();"
        code.dedent()
        code += "} else {"
        code.indent()
        code += "throw response;"
        code.dedent()
        code += "}"
        code.dedent()
        code += "});"
        code.dedent()
        code += "};"
        code.dedent()
        code += "},"
        return str(code)
    if phase == "factory":
        code.start_comment()
        code += ""
        code += jsdoc
        code.end_comment()

        code += f"{function_name}({function_args}options?: any) {{"
        code.indent()
        code += (
            f"return {api}Fp(configuration).{function_name}({call_list}options)(fetch, basePath);"
        )
        code.dedent()
        code += "},"
        return str(code)
    if phase == "api":
        code.start_comment()
        code += ""
        code += jsdoc
        code += f"@memberof {api}"
        code.end_comment()

        code += f"public {function_name}({function_args}options?: any) {{"
        code.indent()
        code += f"return {api}Fp(this.configuration).{function_name}({call_list}options)(this.fetch, this.basePath)"
        code.dedent()
        code += "}"
        code += ""
        return str(code)

    assert_never(phase)


def generate_api(tag: str, functions: typing.List[swagger_parser.Function]) -> Code:
    code = IndentedLines()
    api_name = f"{tag}Api"
    functions_in_order = sorted(functions, key=lambda f: f.name.lower())

    # fetch param creator
    code.start_comment()
    code += f"{api_name} - fetch parameter creator"
    code += "@export"
    code.end_comment()
    code += (
        f"export const {api_name}FetchParamCreator = function (configuration?: Configuration) {{"
    )
    code.indent()
    code += "return {"
    # resetting indent level here so the function generator can take over
    cur_indent_level = code.indent_level
    code.indent_level = 0
    for function in functions_in_order:
        code += generate_function(api_name, "params", function)
    code.indent_level = cur_indent_level
    code += "}"
    code.dedent()
    code += "};"
    code += ""

    # functional programming interface
    code.start_comment()
    code += f"{api_name} - functional programming interface"
    code += "@export"
    code.end_comment()
    code += f"export const {api_name}Fp = function (configuration?: Configuration) {{"
    code.indent()
    code += "return {"
    cur_indent_level = code.indent_level
    code.indent_level = 0
    for function in functions_in_order:
        code += generate_function(api_name, "fp", function)
    code.indent_level = cur_indent_level
    code += "}"
    code.dedent()
    code += "};"
    code += ""

    # factory interface
    code.start_comment()
    code += f"{api_name} - factory interface"
    code += "@export"
    code.end_comment()
    code += f"export const {api_name}Factory = function (configuration?: Configuration, fetch?: FetchAPI, basePath?: string) {{"
    code.indent()
    code += "return {"
    cur_indent_level = code.indent_level
    code.indent_level = 0
    for function in functions_in_order:
        code += generate_function(api_name, "factory", function)
    code.indent_level = cur_indent_level
    code += "}"
    code.dedent()
    code += "};"
    code += ""

    # class interface
    code.start_comment()
    code += f"{api_name} - object-oriented interface"
    code += "@export"
    code += "@class"
    code += "@extends {BaseAPI}"
    code.end_comment()
    code += f"export class {api_name} extends BaseAPI {{"
    for function in functions_in_order:
        code += generate_function(api_name, "api", function)
    code += "}"
    code += ""
    return str(code)


def tsbindings(swagger: swagger_parser.ParseResult) -> str:
    clean_description = swagger.info.description.replace("\n", " ")
    prelude = f"""
/**
 * {swagger.info.title}
 * {clean_description}
 *
 * OpenAPI spec version: {swagger.info.version}
 * Contact: {swagger.info.contact}
 *
 * NOTE: Do not edit the class manually.
 */


import {{ Configuration }} from "./configuration";

type ValueOf<T> = T[keyof T];
const BASE_PATH = "http://localhost".replace(/\/+$/, "");

const convert = (v: unknown): string => {{
    switch (typeof v) {{
        case 'string':
        case 'boolean': {{
            return encodeURIComponent(v)
        }}
        case 'bigint': {{
            return '' + v
        }}
        case 'number': {{
            if (Number.isFinite(v))  {{
                return encodeURIComponent(v);
            }}
            return '';
        }}
        default: {{
            return '';
        }}
    }}
}}

const objToSearchParams = (obj: {{}}, searchParams: URLSearchParams) => {{
    Object.entries(obj).forEach(([key, value]) => {{
        if (Array.isArray(value) && value.length > 0) {{
            searchParams.set(key, convert(value[0]))
            value.slice(1).forEach((subValue) => searchParams.append(key, convert(subValue)))
        }} else {{
            searchParams.set(key, convert(value))
        }}
    }});
}};

/**
 *
 * @export
 */
export const COLLECTION_FORMATS = {{
    csv: ",",
    ssv: " ",
    tsv: "\\t",
    pipes: "|",
}};

/**
 *
 * @export
 * @interface FetchAPI
 */
export interface FetchAPI {{
    (url: string, init?: any): Promise<Response>;
}}

/**
 *
 * @export
 * @interface FetchArgs
 */
export interface FetchArgs {{
    url: string;
    options: any;
}}

/**
 *
 * @export
 * @class BaseAPI
 */
export class BaseAPI {{
    protected configuration: Configuration;

    constructor(configuration?: Configuration, protected basePath: string = BASE_PATH, protected fetch: FetchAPI = window.fetch) {{
        if (configuration) {{
            this.configuration = configuration;
            this.basePath = configuration.basePath || this.basePath;
        }}
    }}
}};

/**
 *
 * @export
 * @class RequiredError
 * @extends {{Error}}
 */
export class RequiredError extends Error {{
    name: "RequiredError"
    constructor(public field: string, msg?: string) {{
        super(msg);
    }}
}}

"""
    out = [prelude]

    # workaround for streaming function behavior
    runtime_stream_error = swagger.defs["runtimeStreamError"]
    assert runtime_stream_error
    runtime_stream_error_ref = swagger_parser.Ref(name="RuntimeStreamError")
    swagger_parser.Ref.all_refs.append(runtime_stream_error_ref)
    runtime_stream_error_ref.defn = runtime_stream_error
    runtime_stream_error_ref.linked = True

    ops_by_tag = {}
    for defn in swagger.ops.values():
        # fix naming conventions to match swagger-codegen here
        defn.name = camel_case(defn.name)
        for param in defn.params.values():
            param.name = camel_case(param.name)
        # group ops by tag name
        for tag in defn.tags:
            fixed_tag = upper_first(camel_case(tag))
            ops_by_tag[fixed_tag] = ops_by_tag.get(fixed_tag, [])
            ops_by_tag[fixed_tag].append(defn)

    for _, defn in sorted(swagger.defs.items(), key=head):
        if defn is None:
            continue
        out += [gen_def(defn)]

    for tag, functions in sorted(ops_by_tag.items(), key=head):
        out += [generate_api(tag, functions)]

    return "\n".join(out).strip()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--output", "-o", action="store", required=True, help="output folder")
    args = parser.parse_args()

    swagger = swagger_parser.parse(SWAGGER)
    bindings = tsbindings(swagger)

    if os.path.isdir(args.output):
        rmtree(args.output, ignore_errors=True)
    os.makedirs(args.output)

    api_path = os.path.join(args.output, "api.ts")
    with open(api_path, "w") as f:
        print(bindings, file=f)

    clean_description = swagger.info.description.replace("\n", " ")
    for path in os.listdir(STATIC_FOLDER):
        path = os.path.join(STATIC_FOLDER, path)
        if os.path.isfile(path):
            if path.endswith(".template"):
                with open(path, "r") as f:
                    contents = f.read()
                    contents = contents.format(swagger=swagger, clean_description=clean_description)
                destination = os.path.join(
                    args.output, os.path.basename(path).replace(".template", "")
                )
                print(destination)
                with open(destination, "w") as f:
                    print(contents, file=f)
            else:
                copy(path, args.output)
