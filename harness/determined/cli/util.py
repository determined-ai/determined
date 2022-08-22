from enum import Enum
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


class PageOpts(str, Enum):
    single = "1"
    all = "all"


PAGE_CHOICES = [v.value for v in PageOpts]
DEFAULT_PAGE_CHOICE = PageOpts.all


def make_pagination_args(
    limit: int = 200,
    offset: Optional[int] = None,
    pages: PageOpts = PageOpts.all,
    supports_reverse: bool = False,
) -> List[Arg]:
    if pages not in PAGE_CHOICES:
        raise NotImplementedError

    res = [
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
        Arg(
            "--pages",
            type=PageOpts,
            choices=PAGE_CHOICES,
            default=pages.value,
            help="when set to 'all', fetch all available data; when '1', fetch a single page",
        ),
    ]

    if supports_reverse:
        res += [
            Arg(
                "--reverse",
                default=False,
                action="store_true",
                help="Reverse the requested order of results.",
            ),
        ]

    return res


default_pagination_args = make_pagination_args()


def limit_offset_paginator(
    method: Callable,
    agg_field: str,
    sess: api.Session,
    limit: int = 200,
    offset: Optional[int] = None,
    pages: PageOpts = PageOpts.all,
    **kwargs: Any,
) -> List[Any]:
    all_objects: List[Any] = []
    internal_offset = offset or 0
    while True:
        r = method(sess, limit=limit, offset=internal_offset, **kwargs)
        page_objects = getattr(r, agg_field)
        all_objects += page_objects
        internal_offset += len(page_objects)
        if len(page_objects) < limit or pages == PageOpts.single:
            break
    return all_objects
