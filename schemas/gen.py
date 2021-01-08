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
        self.golang_title = self.schema["title"] + "V1"
        self.python_title = camel_to_snake(self.golang_title)


def read_schemas(files: List[str]) -> List[Schema]:
    schemas = []
    urlbase = "http://determined.ai/schemas/expconf/v1"
    for file in files:
        url = os.path.join(urlbase, os.path.basename(file))
        with open(file) as f:
            schema = Schema(url, f.read())
            schemas.append(schema)
    # Sort schemas so that the output is deterministic.
    schemas.sort(key=lambda s: s.url)
    return schemas


def gen_go(schemas: List[Schema]) -> List[str]:
    lines = []
    lines.append("// This is a generated file.  Editing it will make you sad.")
    lines.append("")
    lines.append("package expconf")
    lines.append("")
    lines.append('import "encoding/json"')
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

    # Schema getters.
    for schema in schemas:
        lines.extend(
            [
                f"func parsed{schema.golang_title}() interface{{}} {{",
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

    # SchemaBytesMap.
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

    # SchemaMap.
    lines.append("func schemaMap() map[string]interface{} {")
    lines.append("\tif cachedSchemaMap != nil {")
    lines.append("\t\treturn cachedSchemaMap")
    lines.append("\t}")
    lines.append("\tvar url string")
    lines.append("\tcachedSchemaMap = map[string]interface{}{}")
    for schema in schemas:
        lines.append(f'\turl = "{schema.url}"')
        lines.append(f"\tcachedSchemaMap[url] = parsed{schema.golang_title}()")
    lines.append("\treturn cachedSchemaMap")
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


def main(language: str, files: List[str], output: Optional[str]) -> None:
    assert language in ["go", "python"], "language must be 'go' or 'python'"
    assert files, "no input files"
    assert output is not None, "missing output file"

    schemas = read_schemas(files)

    if language == "go":
        lines = gen_go(schemas)
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
    args = parser.parse_args()

    try:
        main(args.language, args.files, args.output)
    except AssertionError as e:
        print(e, file=sys.stderr)
        sys.exit(1)
