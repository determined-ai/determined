"""
Clean generated swagger files for end-user consumption.  Input files are a file
to be modified and a patch file to merge into the real file.

usage: swagger.py GENERATED_JSON PATCH_JSON
"""

import json
import os
import sys
from typing import Dict


def merge_dict(d1: Dict, d2: Dict) -> None:
    """
    Modifies d1 in-place to contain values from d2.  If any value
    in d1 is a dictionary (or dict-like), *and* the corresponding
    value in d2 is also a dictionary, then merge them in-place.
    """

    for k, v2 in d2.items():
        v1 = d1.get(k)
        if isinstance(v1, dict) and isinstance(v2, dict):
            merge_dict(v1, v2)
        else:
            d1[k] = v2


def capitalize(s: str) -> str:
    if len(s) <= 1:
        return s.title()
    return s[0].upper() + s[1:]

def to_lower_camel_case(snake_str):
    components = snake_str.split('_')
    return components[0] + ''.join(x.title() for x in components[1:])


def clean(path: str, patch: str) -> None:
    with open(path, "r") as f:
        spec = json.load(f)

    for key, value in spec["definitions"].items():
        # Remove definitions that should be hidden from the user.
        if key == "protobufAny":
            value["title"] = "Object"
        elif key == "protobufNullValue":
            value["title"] = "NullValue"

        # Clean up titles.
        if "title" not in value:
            value["title"] = "".join(capitalize(k) for k in key.split(sep="v1"))

        if "required" in value:
            value["required"] = [to_lower_camel_case(attr) for attr in value["required"]]

    unpublished_ops = [
        ('/api/v1/experiments', 'post'),
    ]
    for route, method in unpublished_ops:
        del spec['paths'][route][method]

    with open(patch, "r") as f:
        merge_dict(spec, json.load(f))

    with open(path, "w") as f:
        json.dump(spec, f)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Incorrect number of arguments.  Usage:", file=sys.stderr)
        print(__doc__, file=sys.stderr)
        sys.exit(1)
    path, patch = sys.argv[1:3]
    clean(path, patch)
