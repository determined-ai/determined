#!/usr/bin/env python3

import argparse
import json
import os
import sys
from typing import List, Optional


def camel_to_snake(name: str) -> str:
    """Convert CamelCase to snake_case, handling acronyms properly."""
    out = name[0].lower()
    for c0, c1, c2 in zip(name[:-2], name[1:-1], name[2:]):
        # Catch lower->upper transitions.
        if c0.islower() and c1.isupper():
            out += "_"
        # Catch acronym endings.
        if c0.isupper() and c1.isupper() and c2.islower():
            out += "_"
        out += c1.lower()
    out += name[-1].lower()
    return out


class Schema:
    def __init__(self, url: str, text: str) -> None:
        self.url = url
        self.text = text
        try:
            self.schema = json.loads(text)
        except Exception as e:
            raise ValueError(f"{url} is not a valid json file") from e
        self.golang_title = self.schema["title"] + self.version().upper()
        self.python_title = camel_to_snake(self.golang_title)

    def version(self) -> str:
        return os.path.basename(os.path.dirname(self.url))


def read_schemas(files: List[str]) -> List[Schema]:
    schemas = []
    urlbase = "http://determined.ai/schemas"
    for file in files:
        urlend = os.path.relpath(file, os.path.dirname(__file__))
        url = os.path.join(urlbase, urlend)
        with open(file) as f:
            schema = Schema(url, f.read())
            schemas.append(schema)
    # Sort schemas so that the output is deterministic.
    schemas.sort(key=lambda s: s.url)
    return schemas


def gen_go_schemas_package(schemas: List[Schema]) -> List[str]:
    """
    Generate a file at the level of pkg/schemas/ that has all of the schemas embedded into it for
    all config types.

    This is necesary to have a single place that can create validators with all of the schema
    urls, so that schemas of one type are free to reference schemas of another type.
    """
    lines = []
    lines.append("// This is a generated file.  Editing it will make you sad.")
    lines.append("")
    lines.append("package schemas")
    lines.append("")
    lines.append("import (")
    lines.append('\t"encoding/json"')
    lines.append('\t"github.com/santhosh-tekuri/jsonschema/v2"')
    lines.append(")")
    lines.append("")

    # Global variables (lazily loaded but otherwise constants).
    lines.append("var (")
    # Schema texts.
    lines.extend(
        [f"\ttext{schema.golang_title} = []byte(`{schema.text}`)" for schema in schemas]
    )
    # Cached schema values, initially nil.
    lines.extend([f"\tschema{schema.golang_title} interface{{}}" for schema in schemas])
    # Cached map of urls to schema values, initially nil.
    lines.append("\tcachedSchemaMap map[string]interface{}")
    lines.append("\tcachedSchemaBytesMap map[string][]byte")
    lines.append(")")
    lines.append("")

    # Schema getters.  These are exported so that they can be used in the individual packages.
    for schema in schemas:
        lines.extend(
            [
                f"func Parsed{schema.golang_title}() interface{{}} {{",
                f"\tif schema{schema.golang_title} != nil {{",
                f"\t\treturn schema{schema.golang_title}",
                "\t}",
                f"\terr := json.Unmarshal(text{schema.golang_title}, &schema{schema.golang_title})",
                "\tif err != nil {",
                f'\t\tpanic("invalid embedded json for {schema.golang_title}")',
                "\t}",
                f"\treturn schema{schema.golang_title}",
                "}",
            ]
        )
        lines.append("")

    # SchemaBytesMap, used internally by NewCompiler, which has to have a list of all schemas.
    lines.append("func schemaBytesMap() map[string][]byte {")
    lines.append("\tif cachedSchemaBytesMap != nil {")
    lines.append("\t\treturn cachedSchemaBytesMap")
    lines.append("\t}")
    lines.append("\tvar url string")
    lines.append("\tcachedSchemaBytesMap = map[string][]byte{}")
    for schema in schemas:
        lines.append(f'\turl = "{schema.url}"')
        lines.append(f"\tcachedSchemaBytesMap[url] = text{schema.golang_title}")
    lines.append("\treturn cachedSchemaBytesMap")
    lines.append("}")

    return lines


def gen_go_package(schemas: List[Schema], package: str) -> List[str]:
    """
    Generate a file at the level of e.g. pkg/schemas/expconf that defines the schemas.Schema
    interface and schemas.Defaultable interfcae (if applicable) for all the objects in this package.
    """
    lines = []
    lines.append("// This is a generated file.  Editing it will make you sad.")
    lines.append("")
    lines.append(f"package {package}")
    lines.append("")
    lines.append("import (")
    lines.append('\t"encoding/json"')
    lines.append('\t"github.com/santhosh-tekuri/jsonschema/v2"')
    lines.append('\t"github.com/determined-ai/determined/master/pkg/schemas"')
    lines.append(")")
    lines.append("")

    # Implement the Schema interface for all objects.
    for schema in schemas:
        if not schema.python_title.startswith("check_"):
            x = schema.golang_title[0].lower()
            lines.append("")
            lines.append(
                f"func ({x} {schema.golang_title}) ParsedSchema() interface{{}} {{"
            )
            lines.append(f"\treturn schemas.Parsed{schema.golang_title}()")
            lines.append("}")
            lines.append("")
            lines.append(
                f"func ({x} {schema.golang_title}) SanityValidator() *jsonschema.Schema {{"
            )
            lines.append(f'\treturn schemas.GetSanityValidator("{schema.url}")')
            lines.append("}")
            lines.append("")
            lines.append(
                f"func ({x} {schema.golang_title}) CompletenessValidator() *jsonschema.Schema {{"
            )
            lines.append(f'\treturn schemas.GetCompletenessValidator("{schema.url}")')
            lines.append("}")

    return lines


def gen_python(schemas: List[Schema]) -> List[str]:
    lines = []
    lines.append("# This is a generated file.  Editing it will make you sad.")
    lines.append("")
    lines.append("import json")
    lines.append("")
    lines.append("schemas = {")
    for schema in schemas:
        lines.append(f'    "{schema.url}": json.loads(')
        lines.append(f'        r"""\n{schema.text}\n"""')
        lines.append("    ),")
    lines.append("}")

    return lines


def main(
    language: str, package: Optional[str], files: List[str], output: Optional[str]
) -> None:
    assert language in ["go", "python"], "language must be 'go' or 'python'"
    if language == "go":
        assert package is not None, "--package must be provided for the go generator"
    else:
        assert package is None, "--package must not be provided to the python generator"
    assert files, "no input files"
    assert output is not None, "missing output file"

    schemas = read_schemas(files)

    if language == "go":
        if package == "schemas":
            lines = gen_go_schemas_package(schemas)
        else:
            lines = gen_go_package(schemas, package)
    else:
        lines = gen_python(schemas)

    text = "\n".join([*lines, "\n"])

    # Write the output file.
    with open(output, "w") as f:
        f.write(text)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="generate code with embedded schemas")
    parser.add_argument("language", help="go or python")
    parser.add_argument("files", nargs="*", help="input files")
    parser.add_argument("--output")
    parser.add_argument("--package")
    args = parser.parse_args()

    try:
        main(args.language, args.package, args.files, args.output)
    except AssertionError as e:
        print(e, file=sys.stderr)
        sys.exit(1)
