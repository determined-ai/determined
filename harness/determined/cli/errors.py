import argparse
import sys
from typing import Any, Callable, Optional

from requests import RequestException
from termcolor import colored

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

    def __init__(
        self, message: str, e_stack: Optional[Exception] = None, exit_code: int = 1
    ) -> None:
        """
        Args:
        - e_stack: The exception that caused this error.
        - exit_code: The exit code to use when exiting the CLI.
        """
        super().__init__(message)
        self.exit_code = exit_code
        self.e_stack = e_stack
        self.message = message


CliHandler = Callable[[argparse.Namespace], Any]


def report_cli_errors(func: CliHandler) -> CliHandler:
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
            if e.e_stack:
                print(colored(f"Error: {e}", "red"), file=sys.stderr)
            print(colored(f"Error: {e.message}", "red"), file=sys.stderr)
            sys.exit(e.exit_code)
        except (APIHttpError, RequestException, ConnectionError) as e:
            print(colored(f"Error: {e}", "red"), file=sys.stderr)
            sys.exit(1)
        # TODO: collect and report other types of errors
        # send_analytics("cli_exception", e)

    return wrapper
