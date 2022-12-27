from typing import Callable, Any
import sys
from requests import RequestException
from termcolor import colored
import argparse

from harness.determined.common.api.bindings import APIHttpError


class FeatureFlagDisabled(Exception):
    """
    Exception indicating that there is a currently disabled feature flag
    that is required to use a feature
    """

    pass


class CliError(Exception):
    """
    Base class for all CLI errors.
    """

    pass


def report_cli_errors(func: Callable[[argparse.Namespace], Any]) -> Callable[..., Any]:
    def wrapper(args: argparse.Namespace) -> Any:
        try:
            return func(args)
        except CliError as e:
            """
            DISCUSS: is this a reasonable pattern in Python? if so we could do this at argparse.Cmd
            level there are some expected exceptions that we rather not individually handle
            and early break in cli commands. and for there we assume that we don't want to
            show the call stack to the end user. maybe the callstack can be controlled w/
            a flag. (dev, prod)
            """
            print(colored(f"Error: {e}", "red"), file=sys.stderr)
            sys.exit(1)
        except (APIHttpError, RequestException, ConnectionError) as e:
            print(colored(f"Error: {e}", "red"), file=sys.stderr)
            sys.exit(1)

    return wrapper
