import argparse
import functools
from typing import Any, Callable, Dict, List

from determined.common import api, declarative_argparse, util
from determined.common.api import authentication, bindings, certs
from determined.experimental import client

from .errors import FeatureFlagDisabled

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
    """
    Setup a client.Session, which is useful for arbitrary calls against the REST API bindings.
    """
    return api.Session(args.master, args.user, authentication.cli_auth, certs.cli_cert)


def setup_determined(args: argparse.Namespace) -> client.Determined:
    """
    Setup a client.Determined, which is useful for when a cli is really just calling functionality
    that already exists in the python sdk.

    Maybe in the future the sdk and the cli are 1:1 and setup_session() is never used anymore.
    """
    # TODO: this is going to duplicate the cli_auth object created by @authentication.required.
    # TODO: this is going to duplicate the cli_cert object that the cli already creates.
    return client.Determined(args.master, args.user)


def require_feature_flag(feature_flag: str, error_message: str) -> Callable[..., Any]:
    def decorator(function: Callable[..., Any]) -> Callable[..., Any]:
        def wrapper(args: argparse.Namespace) -> None:
            resp = bindings.get_GetMaster(setup_session(args))
            if not resp.to_json().get("rbacEnabled"):
                raise FeatureFlagDisabled(error_message)
            function(args)

        return wrapper

    return decorator
