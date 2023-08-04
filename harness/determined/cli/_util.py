import argparse
import functools
import sys
from typing import Any, Callable, Dict, List, Sequence

import termcolor

from determined.cli import errors, render
from determined.common import api, declarative_argparse, util
from determined.common.api import authentication, bindings, certs
from determined.experimental import client

output_format_args: Dict[str, declarative_argparse.Arg] = {
    "json": declarative_argparse.Arg(
        "--json",
        action="store_true",
        help="Output in JSON format",
    ),
    "yaml": declarative_argparse.Arg(
        "--yaml",
        action="store_true",
        help="Output in YAML format",
    ),
    "csv": declarative_argparse.Arg(
        "--csv",
        action="store_true",
        help="Output in CSV format",
    ),
    "table": declarative_argparse.Arg(
        "--table",
        action="store_true",
        help="Output in table format",
    ),
}


PAGE_CHOICES = [v.value for v in api.PageOpts]
DEFAULT_PAGE_CHOICE = api.PageOpts.all


def make_pagination_args(
    limit: int = 200,
    offset: int = 0,
    pages: api.PageOpts = api.PageOpts.all,
    supports_reverse: bool = False,
) -> List[declarative_argparse.Arg]:
    if pages not in PAGE_CHOICES:
        raise NotImplementedError

    res = [
        declarative_argparse.Arg(
            "--limit",
            type=int,
            default=limit,
            help="Maximum items per page of results",
        ),
        declarative_argparse.Arg(
            "--offset",
            type=int,
            default=offset,
            help="Number of items to skip before starting page of results",
        ),
        declarative_argparse.Arg(
            "--pages",
            type=api.PageOpts,
            choices=PAGE_CHOICES,
            default=pages.value,
            help="when set to 'all', fetch all available data; when '1', fetch a single page",
        ),
    ]

    if supports_reverse:
        res += [
            declarative_argparse.Arg(
                "--reverse",
                default=False,
                action="store_true",
                help="Reverse the requested order of results.",
            ),
        ]

    return res


default_pagination_args = make_pagination_args()


def login_sdk_client(func: Callable[[argparse.Namespace], Any]) -> Callable[..., Any]:
    @functools.wraps(func)
    def f(namespace: argparse.Namespace) -> Any:
        client.login(master=namespace.master, user=namespace.user)
        return func(namespace)

    return f


def setup_session(args: argparse.Namespace) -> api.Session:
    master_url = args.master or util.get_default_master_address()
    cert = certs.default_load(master_url)
    retry = api.default_retry()

    return api.Session(master_url, args.user, authentication.cli_auth, cert, retry)


def require_feature_flag(feature_flag: str, error_message: str) -> Callable[..., Any]:
    def decorator(function: Callable[..., Any]) -> Callable[..., Any]:
        def wrapper(args: argparse.Namespace) -> None:
            resp = bindings.get_GetMaster(setup_session(args))
            if not resp.to_json().get("rbacEnabled"):
                raise errors.FeatureFlagDisabled(error_message)
            function(args)

        return wrapper

    return decorator


def print_warnings(warnings: Sequence[bindings.v1LaunchWarning]) -> None:
    for warning in warnings:
        print(termcolor.colored(api.WARNING_MESSAGE_MAP[warning], "yellow"), file=sys.stderr)


def wait_ntsc_ready(session: api.Session, ntsc_type: api.NTSC_Kind, eid: str) -> None:
    """
    Use to wait for a notebook, tensorboard, or shell command to become ready.
    """
    name = ntsc_type.value
    loading_animator = render.Animator(f"Waiting for {name} to become ready")
    err_msg = api.task_is_ready(session, eid, loading_animator.next)
    msg = f"{name} (id: {eid}) is ready." if not err_msg else f"Waiting stopped: {err_msg}"
    loading_animator.clear(msg)
    if err_msg:
        raise errors.CliError(err_msg)


# not_found_errs mirrors NotFoundErrs from the golang api/errors.go. In the cases where
# Python errors override the golang errors, this ensures the error messages stay consistent.
def not_found_errs(
    category: str, name: str, session: api.Session
) -> api.errors.BadRequestException:
    resp = bindings.get_GetMaster(session)
    msg = f"{category} '{name}' not found"
    if not resp.to_json().get("rbacEnabled"):
        return api.errors.NotFoundException(msg)
    return api.errors.NotFoundException(msg + ", please check your permissions.")
