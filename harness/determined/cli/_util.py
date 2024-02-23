import argparse
import sys
from typing import Any, Callable, Dict, List, Sequence

import termcolor

from determined import cli
from determined.cli import errors, render
from determined.common import api, declarative_argparse
from determined.common.api import authentication, bindings

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


def unauth_session(args: argparse.Namespace) -> api.UnauthSession:
    master_url = args.master
    return api.UnauthSession(master=master_url, cert=cli.cert, max_retries=0)


def setup_session(args: argparse.Namespace) -> api.Session:
    master_url = args.master
    utp = authentication.login_with_cache(
        master_address=master_url,
        requested_user=args.user,
        password=None,
        cert=cli.cert,
    )
    return api.Session(master_url, utp, cli.cert)


def require_feature_flag(feature_flag: str, error_message: str) -> Callable[..., Any]:
    def decorator(function: Callable[..., Any]) -> Callable[..., Any]:
        def wrapper(args: argparse.Namespace) -> None:
            resp = bindings.get_GetMaster(unauth_session(args))
            if not resp.rbacEnabled:
                raise errors.FeatureFlagDisabled(error_message)
            function(args)

        return wrapper

    return decorator


def print_launch_warnings(warnings: Sequence[bindings.v1LaunchWarning]) -> None:
    for warning in warnings:
        print(termcolor.colored(api.WARNING_MESSAGE_MAP[warning], "yellow"), file=sys.stderr)


def wait_ntsc_ready(session: api.Session, ntsc_type: api.NTSC_Kind, eid: str) -> None:
    """
    Use to wait for a notebook, tensorboard, or shell command to become ready.
    """
    name = ntsc_type.value
    loading_animator = render.Animator(f"Waiting for {name} to become ready")
    err_msg = api.wait_for_task_ready(
        session=session,
        task_id=eid,
        progress_report=loading_animator.next,
        timeout=60 * 30,  # seconds
    )
    msg = f"{name} (id: {eid}) is ready." if not err_msg else f"Waiting stopped: {err_msg}"
    loading_animator.clear(msg)
    if err_msg:
        raise errors.CliError(err_msg)


def warn(message: str) -> None:
    print(termcolor.colored(message, "yellow"), file=sys.stderr)
