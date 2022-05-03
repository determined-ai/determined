import json
from argparse import Namespace
from typing import Any, Dict, List, Optional

from determined.cli.session import setup_session
from determined.common import constants, experimental
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.experimental import Determined

from . import render


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
        return [None, None, None]

    if checkpoint["state"] in (constants.COMPLETED, constants.DELETED):
        return [
            checkpoint["state"],
            checkpoint["uuid"],
            json.dumps(checkpoint["metadata"], indent=4),
        ]
    elif checkpoint["state"] in (constants.ACTIVE, constants.ERROR):
        return [checkpoint["state"], None, json.dumps(checkpoint["metadata"], indent=4)]
    else:
        raise AssertionError("Invalid checkpoint state: {}".format(checkpoint["state"]))


def render_checkpoint(checkpoint: experimental.Checkpoint, path: Optional[str] = None) -> None:
    if path:
        print("Local checkpoint path:")
        print(path, "\n")

    # Print information about the downloaded step/checkpoint.
    table = [
        ["Experiment ID", checkpoint.training.experiment_id if checkpoint.training else None],
        ["Trial ID", checkpoint.training.trial_id if checkpoint.training else None],
        ["Steps Completed", checkpoint.metadata.get("steps_completed")],
        ["Report Time", render.format_time(checkpoint.report_time)],
        ["Checkpoint UUID", checkpoint.uuid],
        [
            "Validation Metrics",
            (
                json.dumps(checkpoint.training.validation_metrics, indent=4)
                if checkpoint.training
                else None
            ),
        ],
        ["Metadata", json.dumps(checkpoint.metadata or {}, indent=4)],
    ]

    headers, values = zip(*table)  # type: ignore

    render.tabulate_or_csv(headers, [values], False)


@authentication.required
def list_checkpoints(args: Namespace) -> None:
    if args.best:
        sorter = bindings.v1GetExperimentCheckpointsRequestSortBy.SORT_BY_SEARCHER_METRIC
    else:
        sorter = bindings.v1GetExperimentCheckpointsRequestSortBy.SORT_BY_END_TIME
    r = bindings.get_GetExperimentCheckpoints(
        setup_session(args),
        id=args.experiment_id,
        limit=args.best,
        sortBy=sorter,
    )
    checkpoints = r.checkpoints
    searcher_metric = ""
    if len(checkpoints) > 0:
        config = checkpoints[0].training.experimentConfig or {}
        if "searcher" in config and "metric" in config["searcher"]:
            searcher_metric = str(config["searcher"]["metric"])

    def get_validation_metric(c: bindings.v1Checkpoint, metric: str) -> str:
        if (
            c.training.validationMetrics
            and c.training.validationMetrics.avgMetrics
            and metric in c.training.validationMetrics.avgMetrics
        ):
            return str(c.training.validationMetrics.avgMetrics[metric])
        return ""

    headers = [
        "Trial ID",
        "# of Batches",
        "State",
        "Validation Metric",
        "UUID",
        "Resources",
        "Size",
    ]
    values = [
        [
            c.training.trialId,
            c.metadata.get("steps_completed", None),
            c.state.value.replace("STATE_", "") if c.state is not None else "UNSPECIFIED",
            get_validation_metric(c, searcher_metric),
            c.uuid,
            render.format_resources(c.resources),
            render.format_resource_sizes(c.resources),
        ]
        for c in checkpoints
    ]

    render.tabulate_or_csv(headers, values, args.csv)


def download(args: Namespace) -> None:
    checkpoint = Determined(args.master, None).get_checkpoint(args.uuid)

    path = checkpoint.download(path=args.output_dir)

    if args.quiet:
        print(path)
    else:
        render_checkpoint(checkpoint, path)


def describe(args: Namespace) -> None:
    checkpoint = Determined(args.master, None).get_checkpoint(args.uuid)
    render_checkpoint(checkpoint)


main_cmd = Cmd(
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
        ),
        Cmd(
            "describe",
            describe,
            "describe checkpoint",
            [Arg("uuid", type=str, help="checkpoint uuid to describe")],
        ),
    ],
)
args_description = [main_cmd]  # type: List[Any]
