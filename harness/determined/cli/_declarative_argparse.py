import argparse
import functools
import itertools
from typing import Any, Callable, List, NamedTuple, Optional, Tuple, Union, cast

import termcolor

from determined.common import util


def make_prefixes(desc: str) -> List[str]:
    parts = desc.split("|")
    ret = [parts[0]]
    for part in parts[1:]:
        ret.append(ret[-1] + part)
    return ret


def generate_aliases(spec: str) -> Tuple[str, List[str]]:
    """
    Take the given string and split it by spaces. For each word, split it by
    pipe characters and compute the result of joining each prefix of that
    list. Return a big list containing all the results, except that the result
    of joining the whole first word is pulled out.

    "c|heck|out co"
    => ["c|heck|out", "co"]
    => [["c", "heck", "out"], ["co"]]
    => [["c", "check", "checkout"], ["co"]]
    => "checkout", ["c", "check", "co"]
    """
    prefixes = [make_prefixes(s) for s in spec.split()]
    main = prefixes[0].pop()
    return main, list(itertools.chain.from_iterable(prefixes))


# ArgsDescription is a description of the subcommands and arguments for CLI arg parsing.
ArgsDescription = List[Union["Arg", "Cmd", "Group", "ArgGroup", "BoolOptArg"]]


def deprecation_warning(message: str, color: bool = True) -> str:
    msg = f"DEPRECATED: {message}"
    return termcolor.colored(msg, "yellow") if color else msg


# Classes used to represent the structure of an argument parser setup; these
# are turned into actual `argparse` objects by `add_args`.
class Cmd:
    """Describes a subcommand."""

    def __init__(
        self,
        name: str,
        func: Optional[Callable],
        help_str: str,
        subs: ArgsDescription,
        is_default: bool = False,
        deprecation_message: Optional[str] = None,
    ) -> None:
        """
        `subs` is a list containing `Cmd`, `Arg`, and `Group` that describes
        the arguments, subcommands, and mutually exclusive argument groups
        for this command.
        """
        self.name = name
        self.help_str = help_str
        self.deprecation_message = deprecation_message
        self.func = func
        # wrap the fn in deprecation warning.
        if self.deprecation_message and self.func:
            self.func = util.deprecated(deprecation_warning(self.deprecation_message))(self.func)

        if self.func:
            # Force the help string onto the actual function for later. This
            # can be used to print the help string
            self.func.__name__ = help_str
        self.subs = subs
        self.is_default = is_default


class Arg:
    """
    Describes an argument. Arguments to the constructor are passed to
    `add_argument`.
    """

    def __init__(self, *args: Any, completer: Optional[Callable] = None, **kwargs: Any) -> None:
        self.args = args
        self.kwargs = kwargs
        self.completer = completer


class Group:
    """Describes a mutually exclusive group of options."""

    def __init__(self, *options: Arg, **kwargs: Any) -> None:
        self.options = options
        self.kwargs = kwargs


class ArgGroup:
    """
    Describes a named conceptual group of options. Arguments are passed to
    `add_argument_group`.
    """

    def __init__(
        self,
        title: Optional[str] = None,
        description: Optional[str] = None,
        child_args: Optional[List[Arg]] = None,
    ) -> None:
        self.title = title
        self.description = description
        self.child_args = child_args or []


class BoolOptArg(NamedTuple):
    """Describes a boolean --foo / --no-foo flag pair."""

    true_name: str
    false_name: str
    dest: str
    default: Optional[bool] = None
    true_help: Optional[str] = None
    false_help: Optional[str] = None


def wrap_func(parser: argparse.ArgumentParser, func: Callable) -> Callable:
    @functools.wraps(func)
    def wrapper(args: argparse.Namespace) -> Any:
        args.func = func
        return func(parser.parse_args([], args))

    return wrapper


def help_func(parser: argparse.ArgumentParser) -> Callable:
    """
    Return a function that prints help for the given parser. Using this doesn't
    exit during the call to to `parse_args` itself, which would be ideal, but
    only when the function from the `parse_args` result is called. It looks
    about the same as long as you do the second right after the first, at
    least.
    """

    def inner_func(args: argparse.Namespace) -> Any:
        parser.print_help()

    return inner_func


def add_args(parser: argparse.ArgumentParser, description: ArgsDescription, depth: int = 0) -> None:
    """
    Populate the given parser with arguments, as specified by the
    description. The description is a list of Arg, Cmd, and Group objects.
    """
    subparsers = None
    help_parser = None

    def description_sort_key(desc: Any) -> str:
        if isinstance(desc, Cmd):
            return desc.name

        # `sorted` is stable, so we shouldn't change the relative
        # positioning of non-Cmd arg descriptions.
        return ""

    # Sort descriptions alphabetically by name before passing them to
    # argparse. This ensures that `help` output is sorted
    # alphabetically.
    description = sorted(description, key=description_sort_key)

    for thing in description:
        if isinstance(thing, Cmd):
            if subparsers is None:
                metavar = "sub" * depth + "command"
                subparsers = parser.add_subparsers(metavar=metavar)

                # If there are any subcommands at all, also add a `help`
                # subcommand.
                help_parser = subparsers.add_parser("help", help="show help for this command")
                help_parser.set_defaults(func=help_func(parser))

            main_name, aliases = generate_aliases(thing.name)

            subparser_kwargs = {
                "aliases": aliases,
                "formatter_class": argparse.ArgumentDefaultsHelpFormatter,
            }
            if thing.help_str != argparse.SUPPRESS:
                if thing.deprecation_message:
                    thing.help_str += " " + deprecation_warning(
                        thing.deprecation_message, color=False
                    )
                subparser_kwargs["help"] = thing.help_str
            subparser = subparsers.add_parser(main_name, **subparser_kwargs)

            subparser.set_defaults(func=thing.func)
            subparser.set_defaults(**{("_" + "sub" * depth + "command"): thing.name})

            # If this is the default subcommand, make calling the parent with
            # no subcommand behave the same as calling this subcommand with no
            # arguments.
            if thing.is_default:
                thing.func = cast(Callable, thing.func)
                parser.set_defaults(func=wrap_func(subparser, thing.func))

            add_args(subparser, thing.subs, depth + 1)

        elif isinstance(thing, Arg):
            arg = parser.add_argument(*thing.args, **thing.kwargs)
            arg.completer = thing.completer  # type: ignore

        elif isinstance(thing, Group):
            group = parser.add_mutually_exclusive_group(**thing.kwargs)
            for option in thing.options:
                group.add_argument(*option.args, **option.kwargs)

        elif isinstance(thing, ArgGroup):
            arg_group = parser.add_argument_group(thing.title, thing.description)
            for child_arg in thing.child_args:
                arg_group.add_argument(*child_arg.args, **child_arg.kwargs)

        elif isinstance(thing, BoolOptArg):
            parser.add_argument(
                thing.true_name,
                dest=thing.dest,
                action="store_true",
                help=thing.true_help,
                default=thing.default,
            )
            parser.add_argument(
                thing.false_name,
                dest=thing.dest,
                action="store_false",
                help=thing.false_help,
                default=argparse.SUPPRESS,
            )

    # If there are any subcommands but none claimed the default action, make
    # the default print help.
    if subparsers is not None and parser.get_default("func") is None:
        parser.set_defaults(func=help_func(parser))


def string_to_bool(s: str) -> bool:
    """Converts string values to boolean for flag arguments (e.g. --active=true)"""
    return util.strtobool(s)
