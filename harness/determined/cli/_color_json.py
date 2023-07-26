"""
colorize JSON output.
test string:
   echo '{"a": "b", "c": ["1", 2.0, 3.1, 5, "4", {"d": true, "e": null, "f": [1, 2, 3]}]}'
"""

import json
from typing import Any, TextIO

import termcolor

KEY = "blue"
PRIMITIVES = None  # avoid coloring to common fg or bg colors.
SEPARATORS = "yellow"
STRING = "green"


def render_json(obj: Any, out: TextIO, indent: str = "  ", sort_keys: bool = False) -> None:
    """
    Render JSON object to output stream with color.
    """

    def do_render(obj: Any, depth: int = 0) -> None:
        if obj is None:
            out.write(termcolor.colored("null", PRIMITIVES))
            return

        if isinstance(obj, bool):
            out.write(termcolor.colored(str(obj).lower(), PRIMITIVES))
            return

        if isinstance(obj, str):
            out.write(termcolor.colored(json.dumps(obj), STRING))
            return

        if isinstance(obj, (int, float)):
            out.write(termcolor.colored(str(obj), PRIMITIVES))
            return

        if isinstance(obj, (list, tuple)):
            if len(obj) == 0:
                out.write(termcolor.colored("[]", SEPARATORS))
                return

            out.write(termcolor.colored("[", SEPARATORS))
            first = True
            for item in obj:
                if not first:
                    out.write(termcolor.colored(",", SEPARATORS))
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                do_render(item, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write(termcolor.colored("]", SEPARATORS))
            return

        if isinstance(obj, dict):
            if len(obj) == 0:
                out.write(termcolor.colored("{}", SEPARATORS))
                return

            out.write(termcolor.colored("{", SEPARATORS))
            first = True
            keys = sorted(obj.keys()) if sort_keys else obj.keys()
            for key in keys:
                value = obj[key]
                if not first:
                    out.write(termcolor.colored(",", SEPARATORS))
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                out.write(termcolor.colored(json.dumps(key), KEY))
                out.write(termcolor.colored(": ", SEPARATORS))
                do_render(value, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write(termcolor.colored("}", SEPARATORS))
            return

        raise ValueError(f"unsupported type: {type(obj).__name__}")

    do_render(obj, depth=0)
    out.write("\n")
