from typing import Dict

from determined.common.declarative_argparse import Arg

format_args: Dict[str, Arg] = {
    "json": Arg(
        "--json",
        action="store_true",
        help="Output in JSON format",
    ),
    "yaml": Arg(
        "--yaml",
        action="store_true",
        help="Output in YAML format",
    ),
    "csv": Arg(
        "--csv",
        action="store_true",
        help="Output in CSV format",
    ),
    "table": Arg(
        "--table",
        action="store_true",
        help="Output in table format",
    ),
}

pagination_args = [
    Arg(
        "--offset",
        type=int,
        default=0,
        help="Offset the returned set.",
    ),
    Arg(
        "--limit",
        type=int,
        default=50,
        help="Limit the returned set.",
    ),
    Arg(
        "--reverse",
        default=False,
        action="store_true",
        help="Reverse the requested order of results.",
    ),
]
