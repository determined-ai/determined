import json
from argparse import Namespace
from typing import Any, Dict, List

from determined_common import api, constants, experimental
from determined_common.experimental import Determined

from . import render, user
from .declarative_argparse import Arg, Cmd


def format_validation(validation: Dict[str, Any]) -> List[Any]:
    if not validation:
        return [None, None]

    if validation["state"] == constants.COMPLETED:
        return [constants.COMPLETED, json.dumps(validation["metrics"], indent=4)]
    elif validation["state"] in (constants.ACTIVE, constants.ERROR):
        return [validation["state"], None]
    else:
        raise AssertionError("Invalid validation state: {}".format(validation["state"]))


# TODO(neilc): Report more info about checkpoints and validations.
def format_checkpoint(checkpoint: Dict[str, Any]) -> List[Any]:
    if not checkpoint:
        return [None, None]

    if checkpoint["state"] in (constants.COMPLETED, constants.DELETED):
        return [checkpoint["state"], checkpoint["uuid"]]
    elif checkpoint["state"] in (constants.ACTIVE, constants.ERROR):
        return [checkpoint["state"], None]
    else:
        raise AssertionError("Invalid checkpoint state: {}".format(checkpoint["state"]))


def render_checkpoint(checkpoint: experimental.Checkpoint, path: str) -> None:
    print("Local checkpoint path:")
    print(path, "\n")

    # Print information about the downloaded step/checkpoint.
    table = [
        ["Batch #", checkpoint.batch_number],
        ["Start Time", render.format_time(checkpoint.start_time)],
        ["End Time", render.format_time(checkpoint.end_time)],
        ["Checkpoint UUID", checkpoint.uuid],
        ["Validation Metrics", json.dumps(checkpoint.validation.metrics, indent=4)],
    ]

    headers, values = zip(*table)  # type: ignore

    render.tabulate_or_csv(headers, [values], False)


@user.authentication_required
def list(args: Namespace) -> None:
    params = {}
    if args.best is not None:
        if args.best < 0:
            raise AssertionError("--best must be a non-negative integer")
        params["best"] = args.best

    r = api.get(
        args.master, "experiments/{}/checkpoints".format(args.experiment_id), params=params
    ).json()
    searcher_metric = r["metric_name"]

    headers = ["Trial ID", "Step ID", "State", "Validation Metric", "UUID", "Resources", "Size"]
    values = [
        [
            c["trial_id"],
            c["step_id"],
            c["state"],
            api.metric.get_validation_metric(searcher_metric, c["step"]["validation"]),
            c["uuid"],
            render.format_resources(c["resources"]),
            render.format_resource_sizes(c["resources"]),
        ]
        for c in r["checkpoints"]
    ]

    render.tabulate_or_csv(headers, values, args.csv)


def download(args: Namespace) -> None:
    checkpoint = Determined(args.master, None).get_checkpoint(args.uuid)

    path = checkpoint.download(path=args.output_dir)

    if args.quiet:
        print(path)
    else:
        render_checkpoint(checkpoint, path)


args_description = Cmd(
    "c|heckpoint",
    None,
    "manage checkpoints",
    [
        Cmd(
            "download",
            download,
            "download checkpoint from persistent storage",
            [
                Arg("uuid", type=str, help="Download a checkpoint by specifying its UUID."),
                Arg(
                    "-o",
                    "--output-dir",
                    type=str,
                    help="Desired output directory for the checkpoint.",
                ),
                Arg(
                    "-q",
                    "--quiet",
                    action="store_true",
                    help="Only print the path to the checkpoint.",
                ),
            ],
        )
    ],
)
