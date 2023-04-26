"""
colorize JSON output.
test string:
   echo '{"a": "b", "c": ["1", 2.0, 3.1, 5, "4", {"d": true, "e": null, "f": [1, 2, 3]}]}'
"""

import json
import sys
from typing import Any, TextIO

from termcolor import colored


def render_json(obj: Any, out: TextIO, indent: str = "  ") -> None:
    """
    Render JSON object to output stream with color.
    """

    def do_render(obj: Any, depth: int = 0) -> None:
        if obj is None:
            out.write(colored("null", "white"))
            return

        if isinstance(obj, bool):
            out.write(colored(str(obj).lower(), "white"))
            return

        if isinstance(obj, str):
            out.write(colored(json.dumps(obj), "green"))
            return

        if isinstance(obj, (int, float)):
            out.write(colored(str(obj), "white"))
            return

        if isinstance(obj, (list, tuple)):
            if len(obj) == 0:
                out.write(colored("[]", "cyan"))
                return

            out.write(colored("[", "cyan"))
            first = True
            for item in obj:
                if not first:
                    out.write(colored(",", "cyan"))
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                do_render(item, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write(colored("]", "cyan"))
            return

        if isinstance(obj, dict):
            if len(obj) == 0:
                out.write(colored("{}", "yellow"))
                return

            out.write(colored("{", "yellow"))
            first = True
            for key, value in obj.items():
                if not first:
                    out.write(colored(",", "yellow"))
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                out.write(colored(json.dumps(key), "blue"))
                out.write(colored(": ", "yellow"))
                do_render(value, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write(colored("}", "yellow"))
            return

        raise ValueError(f"unsupported type: {type(obj).__name__}")

    do_render(obj, depth=0)
    out.write("\n")


if __name__ == "__main__":
    obj = json.load(sys.stdin)
    render_json(obj, sys.stdout)
