#!/usr/bin/env python3

import argparse
import json
import os
import re
import sys
from typing import Callable, List, Optional, Tuple, TypeVar, Union

Schema = Union[dict, bool]
Errors = List[Tuple[str, str]]


linters = []  # type: List[Callable]


F = TypeVar("F", bound=Callable)


def register_linter(fn: F) -> F:
    linters.append(fn)
    return fn


# Only compound types supported are the nullable supported types.
COMPOUND_TYPES = {
    frozenset(("number", "null")): "number",
    frozenset(("integer", "null")): "integer",
    frozenset(("number", "null")): "number",
    frozenset(("object", "null")): "object",
    frozenset(("array", "null")): "array",
    frozenset(("string", "null")): "string",
    frozenset(("boolean", "null")): "boolean",
    # This only occurs for implicitly nested hyperparameters.
    frozenset(("object", "array")): "null",
}

SUPPORTED_KEYWORDS_BY_TYPE = {
    "number": {
        "minimum",
        "exclusiveMinimum",
        "maximum",
        "exclusiveMaximum",
        "default",
        "unionKey",
        "checks",
    },
    "integer": {
        "minimum",
        "exclusiveMinimum",
        "maximum",
        "exclusiveMaximum",
        "default",
        "unionKey",
        "checks",
    },
    "object": {
        "additionalProperties",
        "required",
        "properties",
        "$ref",
        "default",
        "unionKey",
        "disallowProperties",
        "eventuallyRequired",
        "checks",
        "compareProperties",
        "allOf",
        "optionalRef",
        "$comment",
        "conditional",
    },
    "array": {"items", "default", "unionKey", "minLength", "checks", "$comment"},
    "string": {"pattern", "default", "unionKey", "checks", "$comment"},
    "boolean": {"default", "unionKey", "checks", "$comment"},
    "null": {"default", "unionKey", "checks", "$comment"},
}

TOPLEVEL_KEYWORDS = {"$schema", "$id", "title"}


class LintContext:
    def __init__(
        self, schema: Schema, path: str, toplevel: bool, in_checks: bool, filepath: str
    ) -> None:
        self._schema = schema
        self._path = path
        self.toplevel = toplevel
        self.in_checks = in_checks
        self.filepath = filepath


@register_linter
def check_schema(schema: dict, path: str, ctx: LintContext) -> Errors:
    if not ctx.toplevel:
        return []

    if "$schema" not in schema:
        return [(path, "$schema is missing")]

    exp_schema = "http://json-schema.org/draft-07/schema#"

    if schema["$schema"] != exp_schema:
        return [(path, f'$schema is not "{exp_schema}"')]

    return []


@register_linter
def check_id(schema: dict, path: str, ctx: LintContext) -> Errors:
    if not ctx.toplevel:
        return []

    if "$id" not in schema:
        return [(path, "$id is missing")]

    subpath = path + ".$id"

    exp = "http://determined.ai/schemas/" + ctx.filepath
    if schema["$id"] != exp:
        return [(subpath, f"$id ({schema['$id']}) is not correct for filename")]

    return []


def is_required(object_schema: dict, key: str) -> bool:
    return key in object_schema.get("required", [])


def is_nullable(object_schema: dict, key: str) -> bool:
    sub = object_schema["properties"][key]
    if isinstance(sub, bool):
        return sub
    assert isinstance(sub, dict), f"expected dict but got {sub}"
    if "const" in sub:
        return False
    if "enum" in sub:
        return None in sub["enum"]
    if "type" in sub:
        if isinstance(sub["type"], list):
            return "null" in sub["type"]
        return bool(sub["type"] == "null")
    return False


@register_linter
def check_default_typing(schema: dict, path: str, ctx: LintContext) -> Errors:
    if ctx.in_checks:
        return []
    if not isinstance(schema, dict) or "properties" not in schema:
        return []

    errors = []

    required = schema.get("required", [])
    for key, sub in schema["properties"].items():
        subpath = path + f".{key}"
        if key in required and isinstance(sub, dict) and "default" in sub:
            errors.append((subpath, "default provided for required value"))
        if key not in required and isinstance(sub, dict) and "default" not in sub:
            errors.append((subpath, "default not provided for non-required value"))

    return errors


@register_linter
def check_default_locations(schema: dict, path: str, ctx: LintContext) -> Errors:
    """
    This is a bit artificial, but it's much easier to write the defaulting logic if all default
    values are placed in a consistent location.

    They should only ever be found at:  <root>.properties.<key>.default
    """
    if not isinstance(schema, dict) or "properties" not in schema:
        return []

    errors = []

    for key, sub in schema["properties"].items():
        subpath = path + f".{key}"
        if isinstance(sub, dict) and "default" in sub:
            if sub["default"] == "null":
                errors.append(
                    (
                        subpath + ".default",
                        "default is the literal 'null' string, probable typo",
                    )
                )
            elif (
                not re.match("^<[^>]*>\\.[^.]*$", subpath)
                and sub["default"] is not None
            ):
                # This is pretty valid in json-schema normally, but it makes reading defaults
                # out of json-schema (which we need in multiple languages) much harder.
                errors.append(
                    (subpath + ".default", "non-null default is defined on a subobject")
                )

    return errors


@register_linter
def check_nullable(schema: dict, path: str, ctx: LintContext) -> Errors:
    """Non-Required fields must be nullable; required fields must be non-Nullable."""
    if ctx.in_checks:
        return []
    if not isinstance(schema, dict) or "properties" not in schema:
        return []

    errors = []

    for key, sub in schema["properties"].items():
        if sub is True:
            # Don't complain about the universal match (true).
            continue
        subpath = path + f".{key}"
        # Make sure that nullability matches the requiredness.
        if is_required(schema, key) and is_nullable(schema, key):
            errors.append((subpath, "required property is nullable"))
        if not is_required(schema, key) and not is_nullable(schema, key):
            errors.append((subpath, "non-required property is not nullable"))
        # Make sure that $refs are optional on nullable objects.
        if is_nullable(schema, key) and "$ref" in sub:
            errors.append((subpath, "nullable $ref should be an optionalRef"))
        if not is_nullable(schema, key) and "optionalRef" in sub:
            errors.append((subpath, "non-nullable optionalRef should be a plain $ref"))

    return errors


@register_linter
def check_types_and_keywords(schema: dict, path: str, ctx: LintContext) -> Errors:
    if "type" not in schema:
        return []

    types = schema["type"]
    if not isinstance(types, list):
        types = [types]

    for typ in types:
        if typ not in SUPPORTED_KEYWORDS_BY_TYPE:
            return [(path, f"unsupported type: {typ}")]

    keys = set(schema.keys()).difference(TOPLEVEL_KEYWORDS)
    keys.remove("type")

    for typ in types:
        keys = keys.difference(SUPPORTED_KEYWORDS_BY_TYPE[typ])

    errors = []

    for kw in keys:
        errors.append((path, f"{kw} not allowed in schema of type {typ}"))

    return errors


@register_linter
def check_union(schema: dict, path: str, ctx: LintContext) -> Errors:
    if "union" not in schema:
        return []

    errors = []

    for idx, sub in enumerate(schema["union"]["items"]):
        subpath = path + f"union.items.[{idx}]"
        if not isinstance(sub, dict):
            errors.append((subpath, "is not a json object"))
            continue
        if "unionKey" not in sub:
            errors.append((subpath, "has no unionKey"))
            continue
        if not isinstance(sub["unionKey"], str):
            errors.append((subpath, "unionKey is not a string"))
            continue

    return errors


@register_linter
def check_conditional(schema: dict, path: str, ctx: LintContext) -> Errors:
    if "conditional" not in schema:
        return []

    conditional = schema["conditional"]
    subpath = path + ".conditional"

    errors = []

    if "when" not in conditional and "unless" not in conditional:
        errors.append((subpath, "has no when clause or until clause"))
    if "when" in conditional and "unless" in conditional:
        errors.append((subpath, "has both a when clause and an until clause"))
    if "enforce" not in conditional:
        errors.append((subpath, "has no enforce clause"))

    return errors


@register_linter
def check_compareProperties(schema: dict, path: str, ctx: LintContext) -> Errors:
    if "compareProperties" not in schema:
        return []

    compare = schema["compareProperties"]
    subpath = path + ".compareProperties"

    errors = []  # type: Errors

    if "type" not in compare:
        errors.append((subpath, "has no type"))
    if "a" not in compare:
        errors.append((subpath, "has no a"))
    if "b" not in compare:
        errors.append((subpath, "has no b"))
    if compare["type"] not in {
        "a<b",
        "a_is_subdir_of_b",
    }:
        errors.append((subpath, f'invalid type: {compare["type"]}'))

    return errors


def iter_subdict(schema: dict, path: str, key: str, ctx: LintContext) -> Errors:
    """Helper function to iter_schema()."""

    if key not in schema:
        return []

    child = schema[key]
    path += f".{key}"

    if not isinstance(child, dict):
        return [(path, "expected a dict but got a {type(child).__name__}")]

    errors = []

    for key, sub in child.items():
        errors += iter_schema(sub, path + f".{key}", ctx)

    return errors


def iter_sublist(schema: dict, path: str, key: str, ctx: LintContext) -> Errors:
    """Helper function to iter_schema()."""

    if key not in schema:
        return []

    child = schema[key]
    path += f".{key}"

    if not isinstance(child, list):
        return [(path, f"expected a list but got a {type(child).__name__}")]

    errors = []

    for idx, sub in enumerate(child):
        errors += iter_schema(sub, path + f"[{idx}]", ctx)

    return errors


def iter_subschema(schema: dict, path: str, key: str, ctx: LintContext) -> Errors:
    """Helper function to iter_schema()."""

    if key not in schema:
        return []

    child = schema[key]
    path += f".{key}"

    return iter_schema(child, path, ctx)


def iter_union(schema: dict, path: str, ctx: LintContext) -> Errors:
    """Helper function to iter_schema()."""

    if "union" not in schema:
        return []

    child = schema["union"]
    path += ".union"

    if not isinstance(child, dict):
        return [(path, f"expected a dict but got a {type(child).__name__}")]

    return iter_sublist(child, path, "items", ctx)


def iter_schema(
    schema: dict,
    path: str,
    ctx: Optional[LintContext] = None,
    in_checks: bool = False,
    filepath: Optional[str] = None,
) -> Errors:
    """
    Iterate through structural elements of a schema.  In the following example:

        {
            "type": "string",
            "required": ["meh"],
            "additionalProperties": false,
            "properties": {
                "meh": { "const": "some_val" }
            }
        }

    ... the root object, the `false`, and the `{ "const": "some_val" }` are each structural.

    Everthing else is content-related and non-structural.
    """

    if not isinstance(schema, (dict, bool)):
        return [(path, "schema should be a dictionary or a bool")]

    # True or False are special.
    if isinstance(schema, bool):
        return []

    errors = []

    # Apply linters to this structural element.
    if ctx is None:
        assert filepath, "filepath must be provided when ctx is None"
        ctx = LintContext(
            schema, path, toplevel=True, in_checks=in_checks, filepath=filepath
        )
    else:
        ctx = LintContext(schema, path, False, ctx.in_checks, ctx.filepath)
    for linter in linters:
        try:
            errors += linter(schema, path, ctx)
        except Exception as e:
            raise ValueError(
                f"error processing schema:\n{json.dumps(schema, indent=4)}"
            ) from e

    # Descend into child dicts of structural elements.
    for kw in ["properties"]:
        errors += iter_subdict(schema, path, kw, ctx)

    for kw in ["checks"]:
        ctx.in_checks = True
        errors += iter_subdict(schema, path, kw, ctx)

    # Descend into child lists of structural elements.
    for kw in ["oneOf", "anyOf", "allOf"]:
        errors += iter_sublist(schema, path, kw, ctx)

    # Descend directly into child structural elements.
    for kw in ["items", "additionalProperties", "not"]:
        errors += iter_subschema(schema, path, kw, ctx)

    # Descend into custom structural elements.
    errors += iter_union(schema, path, ctx)

    return errors


def fmt(files: List[str], reformat: bool) -> Errors:
    errors = []

    for file in files:
        with open(file) as f:
            text = f.read()
        jobj = json.loads(text)
        # Apply the same linting that `python -mjson.tool` would apply.
        fixed = json.dumps(jobj, sort_keys=False, indent=4) + "\n"
        if fixed != text:
            if reformat:
                with open(file, "w") as f:
                    f.write(fixed)
            else:
                doc = os.path.relpath(file, os.path.dirname(__file__))
                errors.append((f"<{doc}>", "would reformat"))

    return errors


def lint(files: List[str], reformat: bool) -> List[str]:
    assert files, "no files provided"

    invalids = []
    errors = []

    for file in files:
        with open(file) as f:
            try:
                schema = json.loads(f.read())
            except json.decoder.JSONDecodeError as e:
                invalids.append(f"{file} not valid: {e}")
                continue

        doc = os.path.relpath(file, os.path.dirname(__file__))

        try:
            errors += iter_schema(
                schema, f"<{doc}>", in_checks=doc.startswith("check-"), filepath=doc
            )
        except Exception as e:
            raise ValueError(f"failure processing {file}") from e

    # Exit now if there are invalid errors.
    if invalids:
        return invalids

    errors += fmt(files, reformat)

    if errors:
        pathwidth = max(len(path) for path, error in errors)
        msgs = ["%-*s  %s" % (pathwidth, path, error) for path, error in errors]
        return sorted(msgs)

    return []


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="static analysis for json schema")
    parser.add_argument("files", nargs="*", help="files to check")
    parser.add_argument("--check", action="store_true", help="no auto-reformatting")
    args = parser.parse_args()
    errors = lint(args.files, reformat=not args.check)
    if errors:
        print("\n".join(errors), file=sys.stderr)
        sys.exit(1)
