import argparse
import collections
import inspect
import json
import re
import shlex
import shutil
import subprocess
import sys
from argparse import Namespace
from collections.abc import Sequence
from typing import Any, Dict, List, OrderedDict, Tuple
from urllib import parse

from termcolor import colored

import determined.cli.render
from determined import cli
from determined.cli import errors
from determined.common.api import authentication, bindings
from determined.common.api import errors as api_errors
from determined.common.api import request
from determined.common.declarative_argparse import Arg, Cmd


@authentication.required
def token(_: Namespace) -> None:
    token = authentication.must_cli_auth().get_session_token()
    print(token)


@authentication.required
def curl(args: Namespace) -> None:
    assert authentication.cli_auth is not None
    if shutil.which("curl") is None:
        print(colored("curl is not installed on this machine", "red"))
        sys.exit(1)

    parsed = parse.urlparse(args.path)
    if parsed.scheme:
        raise errors.CliError(
            "path argument does not support absolute URLs."
            + " Set the host path through `det` command"
        )

    cmd: List[str] = [
        "curl",
        request.make_url_new(args.master, args.path),
        "-H",
        f"Authorization: Bearer {authentication.cli_auth.get_session_token()}",
        "-s",
    ]
    if args.curl_args:
        cmd += args.curl_args

    if args.x:
        if hasattr(shlex, "join"):  # added in py 3.8
            print(shlex.join(cmd))  # type: ignore
        else:
            print(" ".join(shlex.quote(arg) for arg in cmd))

    if not sys.stdout.isatty():
        output = subprocess.run(cmd)
        sys.exit(output.returncode)

    output = subprocess.run(cmd, stdout=subprocess.PIPE)
    try:
        out = output.stdout.decode("utf8")
        determined.cli.render.print_json(out)
    except UnicodeDecodeError:
        print(
            "Failed to decode response as utf8. Redirect output to capture it.",
            file=sys.stderr,
        )

    sys.exit(output.returncode)


def bindings_sig(fn: Any) -> Tuple[str, List[inspect.Parameter]]:
    """
    return name and sorted API parameters of a bindings function with
    required parameters first.
    """
    sig = inspect.signature(fn)
    params = list(sig.parameters.values())[1:]
    params.sort(key=lambda x: x.name)
    params.sort(key=lambda x: x.default is inspect.Parameter.empty, reverse=True)
    return fn.__name__, params


def bindings_sig_str(name: str, params: List[inspect.Parameter]) -> str:
    def serialize_param(p: inspect.Parameter) -> str:
        return str(p).replace("typing.", "")

    params_str = ", ".join(serialize_param(p) for p in params)
    return f"{name} <= {params_str}" if params else name


def unwrap_optional(annotation: Any) -> Any:
    """
    evaluates and unwraps a typing.Optional annotation to its inner type.
    """
    import typing

    try:
        from typing import get_args, get_origin  # type: ignore
    except ImportError:
        raise errors.CliError("python >= 3.8 is required to use this feature")

    local_context = {
        "typing": typing,
        "Optional": typing.Optional,
        "Union": typing.Union,
        "List": typing.List,
        "Sequence": typing.Sequence,
        "Dict": typing.Dict,
        "NoneType": type(None),
    }
    if isinstance(annotation, str):
        try:
            annotation = eval(annotation.strip(), {}, local_context)
        except NameError:
            pass
    origin = get_origin(annotation)
    args = get_args(annotation)

    if origin is typing.Union:
        if len(args) == 2 and type(None) in args:
            return args[0]
    elif origin is typing.Optional:
        return args[0]
    return annotation


def is_supported_annotation(annot: Any) -> bool:
    """
    determines if a our CLI deserializer supports a given type annotation
    and subsequently a binding's parameter.
    """
    try:
        from typing import get_args, get_origin  # type: ignore
    except ImportError:
        raise errors.CliError("python >= 3.8 is required to use this feature")

    annot = unwrap_optional(annot)
    supported_types = [str, int, float, type(None), bool]
    if annot in supported_types:
        return True

    origin = get_origin(annot)
    args = get_args(annot)

    if origin in [list, Sequence]:
        if args is not None:
            return all(is_supported_annotation(arg) for arg in args)

    return False


def can_be_called_via_cli(params: List[inspect.Parameter]) -> bool:
    """
    determines if a fn with `params` list of parameters can be called with
    our current CLI argument deserialization support.
    """
    required_params = [p for p in params if p.default is inspect.Parameter.empty]
    return all(is_supported_annotation(p.annotation) for p in required_params)


def get_available_bindings(
    show_unusable: bool = False,
) -> OrderedDict[str, List[inspect.Parameter]]:
    """
    return a dictionary of available bindings and their parameters.
    """
    rv: List[Tuple[str, List[inspect.Parameter]]] = []
    for name, obj in inspect.getmembers(bindings):
        if not inspect.isfunction(obj):
            continue
        name, params = bindings_sig(obj)
        if not show_unusable:
            if not can_be_called_via_cli(params):
                continue
            params = [p for p in params if is_supported_annotation(p.annotation)]
        rv.append((name, params))

    rv.sort(key=lambda x: x[0])
    odict: OrderedDict[str, List[inspect.Parameter]] = collections.OrderedDict()
    for name, params in rv:
        odict[name] = params
    return odict


def parse_param_value(param: inspect.Parameter, value: str) -> Any:
    annot = unwrap_optional(param.annotation)

    if annot is str:
        return value
    elif annot is float:
        return float(value)
    elif annot is int:
        return int(value)
    elif annot is bool:
        assert value.lower() in ["true", "false"]
        return value.lower() == "true"
    try:
        value = json.loads(value)
    except json.JSONDecodeError:
        raise ValueError(f"Invalid JSON for {param.name}")

    return value


def parse_args_to_kwargs(args: List[str], params: List[inspect.Parameter]) -> Dict[str, Any]:
    """
    deserialize a list of CLI arguments destined for a bindings function into
    a dictionary of keyword arguments.
    various formats are supported:
    - positional arguments
    - keyword arguments
    - json values
    - positional values coming afetr keyword arguments
    """
    kwargs: Dict[str, Any] = {}
    params_d: Dict[str, inspect.Parameter] = {p.name: p for p in params}

    for idx, arg in enumerate(args):
        key: str = ""
        value: Any = None
        if "=" in arg:
            key, value = arg.split("=", 1)
        else:
            key = params[idx].name
            value = arg
        param = params_d.get(key)
        if not param:
            raise ValueError(f"Unknown argument {key}")
        value = parse_param_value(param, value)
        if key in kwargs:
            raise ValueError(f"Argument {key} specified twice")
        kwargs[key] = value

    assert len(kwargs) <= len(params), "too many arguments"
    return kwargs


def print_response(data: Any) -> None:
    if data is None:
        return
    if hasattr(data, "to_json"):
        cli.render.print_json(data.to_json())
    elif inspect.isgenerator(data):
        for v in data:
            print_response(v)
    else:
        print(data)


def list_bindings(args: Namespace) -> None:
    for name, params in get_available_bindings(show_unusable=args.show_unusable).items():
        print(bindings_sig_str(name, params))


def auto_complete_binding(available_calls: List[str], fn_name: str) -> str:
    """
    utility to allow partial matching of binding names.
    """
    if fn_name in available_calls:
        return fn_name
    simplified_name = re.sub(r"[^a-zA-Z]", "", fn_name)
    matches = [
        n
        for n in available_calls
        if re.match(f".*({fn_name}|{simplified_name}).*", n, re.IGNORECASE)
    ]
    if not matches:
        raise errors.CliError(f"no such binding found: {fn_name}")
    if not sys.stdout.isatty():
        raise errors.CliError(
            f"no exact matches for '{fn_name}'. Did you mean:" + "\n{}".format("\n".join(matches))
        )
    indexed_matches = [f"{idx}: {match}" for idx, match in list(enumerate(matches))]
    selected_idx = (
        input(
            f"{len(matches)} call(s) matched '{fn_name}'."
            + " Pick one by index.\n"
            + "\n".join(indexed_matches)
            + "\n"
        )
        or "0"
    )
    fn_name = matches[int(selected_idx.strip())]
    return fn_name


@authentication.required
def call_bindings(args: Namespace) -> None:
    """
    support calling some bindings with primitive arguments via the cli
    """
    sess = cli.setup_session(args)
    fn_name: str = args.name
    fns = get_available_bindings(show_unusable=False)
    fn_name = auto_complete_binding(list(fns.keys()), fn_name)
    fn = getattr(bindings, fn_name)
    params = fns[fn_name]
    try:
        kwargs = parse_args_to_kwargs(args.args, params)
        output = fn(sess, **kwargs)
        print_response(output)
    except TypeError as e:
        raise errors.CliError(
            "Usage: "
            + bindings_sig_str(
                fn_name,
                params,
            )
            + f"\n\n{str(e)}",
        )
    except (api_errors.BadRequestException, api_errors.APIException) as e:
        raise errors.CliError(
            "Received an API error:" + f"\n\n{str(e)}",
        )


args_description = [
    Cmd(
        "dev",
        None,
        argparse.SUPPRESS,
        [
            Cmd("auth-token", token, "print the active user's auth token", []),
            Cmd(
                "c|url",
                curl,
                "invoke curl",
                [
                    Arg(
                        "-x", help="display the curl command that will be run", action="store_true"
                    ),
                    Arg("path", help="relative path to curl (e.g. /api/v1/experiments?x=z)"),
                    Arg("curl_args", nargs=argparse.REMAINDER, help="curl arguments"),
                ],
            ),
            Cmd(
                "b|indings",
                None,
                "print the active user's auth token",
                [
                    Cmd(
                        "list ls",
                        list_bindings,
                        "list available api bindings to call",
                        [
                            Arg(
                                "--show-unusable",
                                action="store_true",
                                help="shows all bindings, even those that"
                                + "cannot be called via the cli",
                            ),
                        ],
                        is_default=True,
                    ),
                    Cmd(
                        "c|all",
                        call_bindings,
                        "call a function from bindings",
                        [
                            Arg("name", help="name of the function to call"),
                            Arg(
                                "args",
                                nargs=argparse.REMAINDER,
                                help="arguments to pass to the function, positional or kw=value",
                            ),
                        ],
                    ),
                ],
            ),
        ],
    ),
]  # type: List[Any]
