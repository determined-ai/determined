from typing import Any, Callable, Dict, List, Optional

from determined.common import api
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

pagination_args_fetchone = [
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


def make_pagination_args_fetchall(limit: int = 200, offset: Optional[int] = None) -> List[Arg]:
    return [
        Arg(
            "--limit",
            type=int,
            default=limit,
            help="Maximum items per page of results",
        ),
        Arg(
            "--offset",
            type=int,
            default=offset,
            help="Number of items to skip before starting page of results",
        ),
    ]


pagination_args_fetchall = make_pagination_args_fetchall()


def limit_offset_paginator(
    method: Callable,
    agg_field: str,
    sess: api.Session,
    limit: int = 200,
    offset: Optional[int] = None,
    **kwargs: Any,
) -> List[Any]:
    all_objects: List[Any] = []
    internal_offset = offset or 0
    while True:
        r = method(sess, limit=limit, offset=internal_offset, **kwargs)
        page_objects = getattr(r, agg_field)
        all_objects += page_objects
        internal_offset += len(page_objects)
        if offset is not None or len(page_objects) < limit:
            break
    return all_objects
