import argparse
import collections
import inspect
import re
import shlex
import shutil
import subprocess
import sys
from argparse import Namespace
from typing import Any, List, OrderedDict, Tuple
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
    params.sort(key=lambda x: x.name)
    return fn.__name__, params


def _bindings_sig_str(name: str, params: List[inspect.Parameter]) -> str:
    def serialize_param(p: inspect.Parameter) -> str:
        return str(p)

    params_str = ", ".join(serialize_param(p) for p in params)
    return f"{name} <= {params_str}" if params else name


def _is_primitive_parameter(p: inspect.Parameter) -> bool:
    if p.annotation in [str, int, float, bool]:
        return True
    # is in a list of above
    if hasattr(p.annotation, "__origin__") and p.annotation.__origin__ in [list, List]:
        return p.annotation.__args__[0] in [str, int, float, bool]
    return False


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


def _get_available_bindings(show_unusable: bool = False):
    rv: List[Tuple[str, List[inspect.Parameter]]] = []
    for name, obj in inspect.getmembers(bindings):
        if not inspect.isfunction(obj):
            continue
        name, params = _bindings_sig(obj)
        if not show_unusable and not _can_be_called_via_cli(params):
            continue
        if not show_unusable:
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


def _test():
    _, params = _bindings_sig(bindings.get_GetExperiment)
    assert _can_be_called_via_cli(params) is True, params

    _, params = _bindings_sig(bindings.post_UpdateJobQueue)
    assert _can_be_called_via_cli(params) is False, params


_test()


@authentication.required
def call_bindings(args: Namespace) -> None:
    sess = cli.setup_session(args)
    fn_name: str = args.name
    fns = _get_available_bindings(show_unusable=False)
    try:
        fn = getattr(bindings, fn_name)
    except AttributeError:
        # try to fuzzy match case insensitive as well
        matches = [n for n in fns.keys() if re.match(f".*{fn_name}.*", n, re.IGNORECASE)]
        if not matches:
            raise errors.CliError(f"no such binding: {fn_name}")
        # if len(matches) > 1:
        #     raise errors.CliError(f"multiple bindings match for {fn_name}: {matches}")
        input(f"did you mean {matches[0]}? (press enter to continue)")
        fn_name = matches[0]
        fn = getattr(bindings, fn_name)

    params = fns[fn_name]
    try:
        assert len(args.args) <= len(params), "too many arguments"
        # turn the positional args into kwargs
        kwargs = {p.name: args.args[i] for i, p in enumerate(params)}
        rv = fn(sess, **kwargs)
    except TypeError as e:
        raise errors.CliError(
            "expected arguments: "
            + _bindings_sig_str(
                fn_name,
                params,
            ),
            e,
        )

    if rv is None:
        return

    # see it it has a to_json method
    if hasattr(rv, "to_json"):
        cli.render.print_json(rv.to_json())
    else:
        print(rv.experiment)


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
                                help="arguments to pass to the function",
                            ),
                        ],
                    ),
                ],
            ),
        ],
    ),
]  # type: List[Any]
