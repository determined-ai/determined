"""
colorize JSON output.
test string:
   echo '{"a": "b", "c": ["1", 2.0, 3.1, 5, "4", {"d": true, "e": null, "f": [1, 2, 3]}]}'
"""

import json
from typing import Any, TextIO

from termcolor import colored

KEY = "blue"
PRIMITIVES = "white"
SEPARATORS = "yellow"
STRING = "green"


def render_json(obj: Any, out: TextIO, indent: str = "  ", sort_keys: bool = False) -> None:
    """
    Render JSON object to output stream with color.
    """

    def do_render(obj: Any, depth: int = 0) -> None:
        if obj is None:
            out.write(colored("null", PRIMITIVES))
            return

        if isinstance(obj, bool):
            out.write(colored(str(obj).lower(), PRIMITIVES))
            return

        if isinstance(obj, str):
            out.write(colored(json.dumps(obj), STRING))
            return

        if isinstance(obj, (int, float)):
            out.write(colored(str(obj), PRIMITIVES))
            return

        if isinstance(obj, (list, tuple)):
            if len(obj) == 0:
                out.write(colored("[]", SEPARATORS))
                return

            out.write(colored("[", SEPARATORS))
            first = True
            for item in obj:
                if not first:
                    out.write(colored(",", SEPARATORS))
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                do_render(item, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write(colored("]", SEPARATORS))
            return

        if isinstance(obj, dict):
            if len(obj) == 0:
                out.write(colored("{}", SEPARATORS))
                return

            out.write(colored("{", SEPARATORS))
            first = True
            keys = sorted(obj.keys()) if sort_keys else obj.keys()
            for key in keys:
                value = obj[key]
                if not first:
                    out.write(colored(",", SEPARATORS))
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                out.write(colored(json.dumps(key), KEY))
                out.write(colored(": ", SEPARATORS))
                do_render(value, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write(colored("}", SEPARATORS))
            return

        raise ValueError(f"unsupported type: {type(obj).__name__}")

    do_render(obj, depth=0)
    out.write("\n")
