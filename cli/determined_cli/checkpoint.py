import json
from argparse import Namespace
from typing import Any, List

from determined_common import api, constants, experimental
from determined_common.api import gql
from determined_common.experimental import Determined

from . import render, user
from .declarative_argparse import Arg, Cmd


def format_validation(validation: gql.validations) -> List[Any]:
    if not validation:
        return [None, None]

    if validation.state == constants.COMPLETED:
        return [constants.COMPLETED, json.dumps(validation.metrics, indent=4)]
    elif validation.state in (constants.ACTIVE, constants.ERROR):
        return [validation.state, None]
    else:
        raise AssertionError("Invalid validation state: {}".format(validation.state))


# TODO(neilc): Report more info about checkpoints and validations.
def format_checkpoint(checkpoint: gql.checkpoints) -> List[Any]:
    if not checkpoint:
        return [None, None]

    if checkpoint.state in (constants.COMPLETED, constants.DELETED):
        return [checkpoint.state, checkpoint.uuid]
    elif checkpoint.state in (constants.ACTIVE, constants.ERROR):
        return [checkpoint.state, None]
    else:
        raise AssertionError("Invalid checkpoint state: {}".format(checkpoint.state))


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
    q = api.GraphQLQuery(args.master)
    q.op.experiments_by_pk(id=args.experiment_id).config(path="checkpoint_storage")

    order_by = [
        gql.checkpoints_order_by(
            validation=gql.validations_order_by(
                metric_values=gql.validation_metrics_order_by(signed=gql.order_by.asc)
            )
        )
    ]

    limit = None
    if args.best is not None:
        if args.best < 0:
            raise AssertionError("--best must be a non-negative integer")
        limit = args.best

    checkpoints = q.op.checkpoints(
        where=gql.checkpoints_bool_exp(
            step=gql.steps_bool_exp(
                trial=gql.trials_bool_exp(
                    experiment_id=gql.Int_comparison_exp(_eq=args.experiment_id)
                )
            )
        ),
        order_by=order_by,
        limit=limit,
    )
    checkpoints.end_time()
    checkpoints.labels()
    checkpoints.resources()
    checkpoints.start_time()
    checkpoints.state()
    checkpoints.step_id()
    checkpoints.trial_id()
    checkpoints.uuid()

    checkpoints.step.validation.metric_values.raw()

    resp = q.send()

    headers = ["Trial ID", "Step ID", "State", "Validation Metric", "UUID", "Resources", "Size"]
    values = [
        [
            c.trial_id,
            c.step_id,
            c.state,
            c.step.validation.metric_values.raw
            if c.step.validation and c.step.validation.metric_values
            else None,
            c.uuid,
            render.format_resources(c.resources),
            render.format_resource_sizes(c.resources),
        ]
        for c in resp.checkpoints
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
