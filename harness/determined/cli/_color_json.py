# test string:
#    echo '{"a": "b", "c": ["1", 2.0, 3.1, 5, "4", {"d": true, "e": null, "f": [1, 2, 3]}]}'

import json
import sys
from typing import Any, TextIO

grn = "\x1b[32m"
blu = "\x1b[94m"
gry = "\x1b[90m"
res = "\x1b[m"


def render_json(obj: Any, out: TextIO, indent="  "):
    def do_render(obj, depth=0):
        if obj is None:
            out.write(gry)
            out.write("null")
            out.write(res)
            return

        if isinstance(obj, bool):
            out.write(str(obj).lower())
            return

        if isinstance(obj, str):
            out.write(grn)
            out.write(json.dumps(obj))
            out.write(res)
            return

        if isinstance(obj, (int, float)):
            out.write(str(obj))
            return

        if isinstance(obj, (list, tuple)):
            if len(obj) == 0:
                out.write("[]")
                return

            out.write("[")
            first = True
            for item in obj:
                if not first:
                    out.write(",")
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                do_render(item, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write("]")
            return

        if isinstance(obj, dict):
            if len(obj) == 0:
                out.write("{}")
                return

            out.write("{")
            first = True
            for key, value in obj.items():
                if not first:
                    out.write(",")
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                out.write(blu)
                out.write(json.dumps(key))
                out.write(res)
                out.write(": ")
                do_render(value, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write("}")
            return

        raise ValueError(f" unsupported type: {type(obj).__name__}")

    do_render(obj, depth=0)
    out.write("\n")


if __name__ == "__main__":
    obj = json.load(sys.stdin)
    render_json(obj, sys.stdout)
