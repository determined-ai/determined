import json
from argparse import Namespace
from typing import Any, List, Optional

from determined import cli
from determined.common import experimental
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd
from determined.common.experimental import Determined
from determined.experimental.client import DownloadMode

from . import render


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
        cli.setup_session(args),
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

    path = checkpoint.download(path=args.output_dir, mode=args.mode)

    if args.quiet:
        print(path)
    else:
        render_checkpoint(checkpoint, path)


def describe(args: Namespace) -> None:
    checkpoint = Determined(args.master, None).get_checkpoint(args.uuid)
    render_checkpoint(checkpoint)


@authentication.required
def delete_checkpoints(args: Namespace) -> None:
    if args.yes or render.yes_or_no(
        "Deleting checkpoints will result in deletion of all data associated\n"
        "with each checkpoint in the checkpoint storage. Do you still want to proceed?"
    ):
        c_uuids = args.checkpoints_uuids.split(",")
        delete_body = bindings.v1DeleteCheckpointsRequest(checkpointUuids=c_uuids)
        bindings.delete_DeleteCheckpoints(cli.setup_session(args), body=delete_body)
        print("Deletion of checkpoints {} is in progress".format(args.checkpoints_uuids))
    else:
        print("Aborting deletion of checkpoints.")


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
                Arg(
                    "--mode",
                    choices=list(DownloadMode),
                    default=DownloadMode.AUTO,
                    type=DownloadMode,
                    help=(
                        "Select different download modes: "
                        f"'{DownloadMode.DIRECT}' to directly download from checkpoint storage; "
                        f"'{DownloadMode.MASTER}' to download via the master; "
                        f"'{DownloadMode.AUTO}' to first attempt a direct download and fall "
                        f"back to '{DownloadMode.MASTER}'."
                    ),
                ),
            ],
        ),
        Cmd(
            "describe",
            describe,
            "describe checkpoint",
            [Arg("uuid", type=str, help="checkpoint uuid to describe")],
        ),
        Cmd(
            "delete",
            delete_checkpoints,
            "delete checkpoints",
            [
                Arg("checkpoints_uuids", help="comma-separated list of checkpoints to delete"),
                Arg(
                    "--yes",
                    action="store_true",
                    default=False,
                    help="automatically answer yes to prompts",
                ),
            ],
        ),
    ],
)
args_description = [main_cmd]  # type: List[Any]
