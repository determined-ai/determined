import base64
import csv
import inspect
import json
import pathlib
import sys
from datetime import timezone
from typing import Any, Dict, Iterable, List, Optional, Sequence, Union

import dateutil.parser
import tabulate
import termcolor

from determined import util as det_util
from determined.common import util, yaml

# Avoid reporting BrokenPipeError when piping `tabulate` output through
# a filter like `head`.
_FORMAT = "presto"
_DEFAULT_VALUE = "N/A"
OMITTED_VALUE = "***"


def select_values(values: List[Dict[str, Any]], headers: Dict[str, str]) -> List[Dict[str, Any]]:
    return [{k: item.get(k, _DEFAULT_VALUE) for k in headers.keys()} for item in values]


def render_table(
    values: List[Dict[str, Any]], headers: Dict[str, str], table_fmt: str = _FORMAT
) -> None:
    # Only display selected columns
    values = select_values(values, headers)

    print(tabulate.tabulate(values, headers, tablefmt=table_fmt), flush=False)


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


def format_base64_as_yaml(source: str) -> str:
    s = yaml.safe_dump(yaml.safe_load(base64.b64decode(source)), default_flow_style=False)

    if not isinstance(s, str):
        raise AssertionError("cannot format base64 string to yaml")
    return s


def format_object_as_yaml(source: Dict[str, Any]) -> str:
    s = yaml.safe_dump(source, default_flow_style=False)
    if not isinstance(s, str):
        raise AssertionError("cannot format object to yaml")
    return s


def format_time(datetime_str: Optional[str]) -> Optional[str]:
    if datetime_str is None:
        return None
    dt = dateutil.parser.parse(datetime_str)
    return dt.astimezone(timezone.utc).strftime("%Y-%m-%d %H:%M:%S%z")


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


def print_json(data: Union[str, Any]) -> None:
    """
    Print JSON data in a human-readable format.
    """
    try:
        if isinstance(data, str):
            data = json.loads(data)
        formatted_json = det_util.json_encode(data, sort_keys=True, indent="  ")
        print(formatted_json)
    except json.decoder.JSONDecodeError:
        print(data)


def report_job_launched(_type: str, _id: str) -> None:
    msg = f"Launched {_type} (id: {_id})."
    print(termcolor.colored(msg, "green"))
