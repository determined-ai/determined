"""Rendering utilities for the CLI.

The to_json functions in this module are used to convert resource objects into dicts that look
similar to their analagous resources from bindings to_json functions.

This allows the CLI to use the similar code to render resources from both the bindings and the
experimental API.

For example, the following two lines of pseudocode return similar objects:
* bindings.v1Model.to_json()
* _render.model_to_json(model.Model)
"""
import csv
import datetime
import inspect
import json
import os
import pathlib
import sys
from typing import Any, Dict, Iterable, List, Optional, Sequence, TextIO, Union

import tabulate
import termcolor
from dateutil import parser

from determined import experimental
from determined import util as det_util
from determined.common import util

# Avoid reporting BrokenPipeError when piping `tabulate` output through
# a filter like `head`.
_FORMAT = "presto"
_DEFAULT_VALUE = "N/A"
OMITTED_VALUE = "***"


def select_values(values: List[Dict[str, Any]], headers: Dict[str, str]) -> List[List[Any]]:
    return [[item.get(k, _DEFAULT_VALUE) for k in headers.keys()] for item in values]


def unmarshal(
    class_: Any, data: Dict[str, Any], transforms: Optional[Dict[str, Any]] = None
) -> Any:
    if not transforms:
        transforms = {}
    spec = inspect.getfullargspec(class_)
    init_args = {}
    for arg in spec.args[1:]:
        transform = transforms.get(arg, lambda x: x)
        init_args[arg] = transform(data[arg])
    return class_(**init_args)


class Animator:
    """
    Animator is a simple class for rendering a loading animation in the terminal.
    Use to communicate progress to the user when a call may take a while.
    """

    MAX_LINE_LENGTH = 80

    def __init__(self, message: str = "Loading") -> None:
        self.message = message
        self.step = 0

    def next(self) -> None:
        self.render_frame(self.step, self.message)
        self.step += 1

    @staticmethod
    def render_frame(step: int, message: str) -> None:
        animation = "|/-\\"
        sys.stdout.write("\r" + message + " " + animation[step % len(animation)] + " ")
        sys.stdout.flush()

    def reset(self) -> None:
        self.clear()
        self.step = 0

    @staticmethod
    def clear(message: str = "Loading done.") -> None:
        sys.stdout.write("\r" + " " * Animator.MAX_LINE_LENGTH + "\r")
        sys.stdout.write("\r" + message + "\n")
        sys.stdout.flush()


def render_objects(
    generic: Any, values: Iterable[Any], default_value: str = "N/A", table_fmt: str = _FORMAT
) -> None:
    keys = inspect.getfullargspec(generic).args[1:]
    headers = [key.replace("_", " ").title() for key in keys]
    if len(headers) == 0:
        raise ValueError("must have at least one header to display")

    def _coerce(r: Any) -> Iterable[Any]:
        for key in keys:
            value = getattr(r, key)
            if value is None:
                yield default_value
            else:
                yield value

    values = [_coerce(renderable) for renderable in values]
    print(tabulate.tabulate(values, headers, tablefmt=table_fmt), flush=False)


def format_object_as_yaml(source: Dict[str, Any]) -> str:
    s = util.yaml_safe_dump(source, default_flow_style=False)
    if not isinstance(s, str):
        raise AssertionError("cannot format object to yaml")
    return s


def format_time(datetime_str: Optional[str]) -> Optional[str]:
    if datetime_str is None:
        return None
    dt = parser.parse(datetime_str)
    return dt.astimezone(datetime.timezone.utc).strftime("%Y-%m-%d %H:%M:%S%z")


def format_percent(f: Optional[float]) -> Optional[str]:
    if f is None:
        return None

    return "{:.1%}".format(f)


def format_resource_sizes(resources: Optional[Dict[str, str]]) -> str:
    if resources is None:
        return ""
    else:
        sizes = map(float, resources.values())
        return util.sizeof_fmt(sum(sizes))


def format_resources(resources: Optional[Dict[str, str]]) -> str:
    if resources is None:
        return ""
    else:
        return "\n".join(sorted(resources.keys()))


def tabulate_or_csv(
    headers: Union[Dict[str, str], Sequence[str]],
    values: Sequence[Iterable[Any]],
    as_csv: bool,
    outfile: Optional[pathlib.Path] = None,
) -> None:
    out = outfile.open("w") if outfile else sys.stdout
    if as_csv or outfile:
        writer = csv.writer(out)
        writer.writerow(headers)
        writer.writerows(values)
    else:
        # Tabulate needs to accept dict[str, str], but mypy thinks it cannot, so
        # we suppress that error.
        print(
            tabulate.tabulate(values, headers, tablefmt="presto"),
            file=out,
            flush=False,
        )


def yes_or_no(prompt: str) -> bool:
    """Get a yes or no answer from the CLI user."""
    yes = ("y", "yes")
    no = ("n", "no")
    try:
        while True:
            choice = input(prompt + " [{}/{}]: ".format(yes[0], no[0])).strip().lower()
            if choice in yes:
                return True
            if choice in no:
                return False
            print(
                "Please respond with {} or {}".format(
                    "/".join("'{}'".format(y) for y in yes), "/".join("'{}'".format(n) for n in no)
                )
            )
    except KeyboardInterrupt:
        # Add a newline to mimic a return when sending normal inputs.
        print()
        return False


COLORS = {
    "KEY": "blue",
    "PRIMITIVES": None,  # avoid coloring to common fg or bg colors.
    "SEPARATORS": "yellow",
    "STRING": "green",
}


def render_colorized_json(
    obj: Any, out: TextIO, indent: str = "  ", sort_keys: bool = False
) -> None:
    """
    Render JSON object to output stream with color.
    """

    def do_render(obj: Any, depth: int = 0) -> None:
        if obj is None:
            out.write(termcolor.colored("null", COLORS["PRIMITIVES"]))
            return

        if isinstance(obj, bool):
            out.write(termcolor.colored(str(obj).lower(), COLORS["PRIMITIVES"]))
            return

        if isinstance(obj, str):
            out.write(termcolor.colored(json.dumps(obj), COLORS["STRING"]))
            return

        if isinstance(obj, (int, float)):
            out.write(termcolor.colored(str(obj), COLORS["PRIMITIVES"]))
            return

        if isinstance(obj, (list, tuple)):
            if len(obj) == 0:
                out.write(termcolor.colored("[]", COLORS["SEPARATORS"]))
                return

            out.write(termcolor.colored("[", COLORS["SEPARATORS"]))
            first = True
            for item in obj:
                if not first:
                    out.write(termcolor.colored(",", COLORS["SEPARATORS"]))
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                do_render(item, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write(termcolor.colored("]", COLORS["SEPARATORS"]))
            return

        if isinstance(obj, dict):
            if len(obj) == 0:
                out.write(termcolor.colored("{}", COLORS["SEPARATORS"]))
                return

            out.write(termcolor.colored("{", COLORS["SEPARATORS"]))
            first = True
            keys = sorted(obj.keys()) if sort_keys else obj.keys()
            for key in keys:
                value = obj[key]
                if not first:
                    out.write(termcolor.colored(",", COLORS["SEPARATORS"]))
                first = False
                out.write("\n")
                out.write(indent * (depth + 1))
                out.write(termcolor.colored(json.dumps(key), COLORS["KEY"]))
                out.write(termcolor.colored(": ", COLORS["SEPARATORS"]))
                do_render(value, depth + 1)
            out.write("\n")
            out.write(indent * depth)
            out.write(termcolor.colored("}", COLORS["SEPARATORS"]))
            return

        raise ValueError(f"unsupported type: {type(obj).__name__}")

    do_render(obj, depth=0)
    out.write("\n")


def _coloring_enabled() -> bool:
    return sys.stdout.isatty() and os.environ.get("DET_CLI_COLORIZE", "").lower() in ("1", "true")


def print_json(data: Union[str, Any]) -> None:
    """
    Print JSON data in a human-readable format.
    """
    DEFAULT_INDENT = "  "
    try:
        if isinstance(data, str):
            data = json.loads(data)
        if _coloring_enabled():
            render_colorized_json(data, sys.stdout, indent=DEFAULT_INDENT, sort_keys=True)
            return
        formatted_json = det_util.json_encode(data, sort_keys=True, indent=DEFAULT_INDENT)
        print(formatted_json)
    except json.decoder.JSONDecodeError:
        print(data)


def report_job_launched(_type: str, _id: str, name: str) -> None:
    msg = f"Launched {_type} (id: {_id}, name: {name})."
    print(termcolor.colored(msg, "green"))


def model_to_json(model: experimental.Model) -> Dict[str, Any]:
    """Convert a experimental.Model to a bindings-style to_json dict."""
    return {
        "name": model.name,
        "id": model.model_id,
        "description": model.description,
        "creation_time": model.creation_time,
        "last_updated_time": model.last_updated_time,
        "metadata": model.metadata,
        "archived": model.archived,
    }


def project_to_json(project: experimental.Project) -> Dict[str, Any]:
    """Convert a experimental.Project to a bindings-style to_json dict."""
    return {
        "archived": project.archived,
        "description": project.description,
        "id": project.id,
        "name": project.name,
        "notes": project.notes,
        "numActiveExperiments": project.n_active_experiments,
        "numExperiments": project.n_experiments,
        "workspaceId": project.workspace_id,
        "username": project.username,
    }
