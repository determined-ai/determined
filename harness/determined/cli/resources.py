import argparse
import sys

import requests

from determined import cli


# Print the body of a response in chunks so we don't have to buffer the whole thing.
def print_response(r: requests.Response) -> None:
    for chunk in r.iter_content(chunk_size=4096):
        sys.stdout.buffer.write(chunk)


def csv(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    params = {"timestamp_after": args.timestamp_after, "timestamp_before": args.timestamp_before}
    path = "resources/allocation/allocations-csv"
    print_response(sess.get(path, params=params))


def raw(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    params = {"timestamp_after": args.timestamp_after, "timestamp_before": args.timestamp_before}
    path = "api/v1/resources/allocation/raw" if args.json else "resources/allocation/raw"
    print_response(sess.get(path, params=params))


def aggregated(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    params = {
        "start_date": args.start_date,
        "end_date": args.end_date,
        "period": "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_MONTHLY"
        if args.monthly
        else "RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY",
    }
    path = (
        "api/v1/resources/allocation/aggregated" if args.json else "resources/allocation/aggregated"
    )
    print_response(sess.get(path, params=params))


args_description: cli.ArgsDescription = [
    cli.Cmd(
        "res|ources",
        None,
        "query historical resource allocation",
        [
            cli.Cmd(
                "alloc|ations",
                csv,
                "get a detailed csv of resource allocation at an allocation level",
                [
                    cli.Arg("timestamp_after"),
                    cli.Arg("timestamp_before"),
                ],
            ),
            cli.Cmd(
                "raw",
                raw,
                "get raw allocation information",
                [
                    cli.Arg("timestamp_after"),
                    cli.Arg("timestamp_before"),
                    cli.Arg("--json", action="store_true", help="output JSON rather than CSV"),
                ],
            ),
            cli.Cmd(
                "agg|regated",
                aggregated,
                "get aggregated allocation information",
                [
                    cli.Arg("start_date", help="first date to include"),
                    cli.Arg("end_date", help="last date to include"),
                    cli.Arg("--json", action="store_true", help="output JSON rather than CSV"),
                    cli.Arg(
                        "--monthly",
                        action="store_true",
                        help="aggregate by month rather than by day",
                    ),
                ],
            ),
        ],
    )
]
