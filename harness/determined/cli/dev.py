import argparse
import collections
import inspect
import re
import shlex
import shutil
import subprocess
import sys
import typing
from argparse import Namespace
from collections.abc import Sequence as abc_Sequence
from typing import Any, Dict, List, OrderedDict, Tuple
from urllib import parse

from termcolor import colored

import determined.cli.render
from determined import cli
from determined.cli import errors
from determined.common.api import authentication, bindings, request
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
    output = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    if output.stderr:
        print(output.stderr.decode("utf8"), file=sys.stderr)

    out = output.stdout.decode("utf8")
    determined.cli.render.print_json(out)

    sys.exit(output.returncode)


def _bindings_sig(fn: Any) -> Tuple[str, List[inspect.Parameter]]:
    sig = inspect.signature(fn)
    # throw out session
    params = list(sig.parameters.values())[1:]
    # stable order of arguments
    params.sort(key=lambda x: x.name)
    # put non optionals first
    params.sort(key=lambda x: x.default is inspect.Parameter.empty, reverse=True)
    return fn.__name__, params


def _bindings_sig_str(name: str, params: List[inspect.Parameter]) -> str:
    def serialize_param(p: inspect.Parameter) -> str:
        return str(p).replace("typing.", "")

    params_str = ", ".join(serialize_param(p) for p in params)
    return f"{name} <= {params_str}" if params else name


def _is_primitive_annotation(a: Any) -> bool:
    try:
        from typing import get_args, get_origin  # type: ignore
    except ImportError:
        raise errors.CliError("python >= 3.8 is required to use this feature")
    if isinstance(a, str):
        try:
            a = eval(a.strip())
        except NameError:
            pass
    if a in [str, int, float, type(None), bool]:
        return True

    origin = get_origin(a)
    args = get_args(a)

    if origin is typing.Union:
        # handle Optional[X] as a special case.
        if len(args) == 2 and type(None) in args:
            return _is_primitive_annotation(args[0])
    elif origin in [list, tuple, set, frozenset, abc_Sequence]:
        return False  # TODO: we don't support the cli interface for these yet.
        if args is not None:
            return all(_is_primitive_annotation(arg) for arg in args)

    return False


def _is_primitive_parameter(p: inspect.Parameter) -> bool:
    return _is_primitive_annotation(p.annotation)


def _can_be_called_via_cli(params: List[inspect.Parameter]) -> bool:
    """
    if all the non-optionals are primitives, then we can call it via the cli
    """
    for p in params:
        if p.default is not inspect.Parameter.empty:
            continue
        if not _is_primitive_parameter(p):
            return False
    return True


def _get_available_bindings(
    show_unusable: bool = False,
) -> OrderedDict[str, List[inspect.Parameter]]:
    rv: List[Tuple[str, List[inspect.Parameter]]] = []
    for name, obj in inspect.getmembers(bindings):
        if not inspect.isfunction(obj):
            continue
        name, params = _bindings_sig(obj)
        if not show_unusable:
            if not _can_be_called_via_cli(params):
                continue
            params = [p for p in params if _is_primitive_parameter(p)]
        rv.append((name, params))

    rv.sort(key=lambda x: x[0])
    odict: OrderedDict[str, List[inspect.Parameter]] = collections.OrderedDict()
    for name, params in rv:
        odict[name] = params
    return odict


def list_bindings(args: Namespace) -> None:
    for name, params in _get_available_bindings(show_unusable=args.show_unusable).items():
        print(_bindings_sig_str(name, params))


def _parse_args_to_kwargs(args: Namespace, params: List[inspect.Parameter]) -> Dict[str, Any]:
    kwargs: Dict[str, Any] = {}
    params_d: Dict[str, inspect.Parameter] = {p.name: p for p in params}

    for idx, arg in enumerate(args.args):
        key, value = "", ""
        if "=" in arg:
            key, value = arg.split("=", 1)
        else:
            key = params[idx].name
            value = arg
        param = params_d.get(key)
        if not param:
            raise ValueError(f"Unknown argument {key}")
        if param.annotation is float:
            value = float(value)
        elif param.annotation is int:
            value = int(value)
        elif param.annotation is bool:
            value = bool(value)
        if key in kwargs:
            raise ValueError(f"Argument {key} specified twice")
        kwargs[key] = value

    assert len(kwargs) <= len(params), "too many arguments"
    return kwargs


def _print_resposne(d: Any) -> None:
    if d is None:
        return
    if hasattr(d, "to_json"):
        cli.render.print_json(d.to_json())
    elif inspect.isgenerator(d):
        for v in d:
            _print_resposne(v)
    else:
        print(d)


@authentication.required
def call_bindings(args: Namespace) -> None:
    """
    support calling some bindings with primitive arguments via the cli
    """
    sess = cli.setup_session(args)
    fn_name: str = args.name
    fns = _get_available_bindings(show_unusable=False)
    try:
        fn = getattr(bindings, fn_name)
    except AttributeError:
        matches = [n for n in fns.keys() if re.match(f".*{fn_name}.*", n, re.IGNORECASE)]
        if not matches:
            raise errors.CliError(f"no such binding: {fn_name}")
        print(f"found {len(matches)} matches for '{fn_name}': {list(enumerate(matches))}")
        user_idx = (
            input(
                "Which one do you want? enter the index to continue"
                + f"(default: 0 {matches[0]}): "
            )
            or "0"
        )
        user_idx = int(user_idx.strip())
        fn_name = matches[user_idx]
        fn = getattr(bindings, fn_name)

    params = fns[fn_name]
    try:
        kwargs = _parse_args_to_kwargs(args, params)
        rv = fn(sess, **kwargs)
    except Exception as e:  # we could explicitly check TypeError but let's provide more hints
        raise errors.CliError(
            "Usage: "
            + _bindings_sig_str(
                fn_name,
                params,
            )
            + f"\n{str(e)}",
            e,
        )

    _print_resposne(rv)


args_description = [
    Cmd(
        "dev",
        None,
        argparse.SUPPRESS,
        [
            Cmd("auth-token", token, "print the active user's auth token", []),
            Cmd(
                "curl",
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
                        "list",
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
                        "call",
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
