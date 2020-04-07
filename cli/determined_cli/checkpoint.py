import sys
from argparse import Namespace

from determined_common import api, storage
from determined_common.api import gql

from . import render, user
from .declarative_argparse import Arg, Cmd


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
    config = resp.experiments_by_pk.config

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

    if args.download_dir is not None:
        manager = storage.build(config)
        if not (
            isinstance(manager, storage.S3StorageManager)
            or isinstance(manager, storage.GCSStorageManager)
        ):
            print(
                "Downloading from S3 or GCS requires the experiment to be configured with "
                "S3 or GCS checkpointing, {} found instead".format(config["type"])
            )
            sys.exit(1)

        for checkpoint in resp.checkpoints:
            metadata = storage.StorageMetadata.from_json(checkpoint.__to_json_value__())
            ckpt_dir = args.download_dir.joinpath(
                "exp-{}-trial-{}-step-{}".format(
                    args.experiment_id, checkpoint.trial_id, checkpoint.step_id
                )
            )
            print("Downloading checkpoint {} to {}".format(checkpoint.uuid, ckpt_dir))
            manager.download(metadata, ckpt_dir)


@user.authentication_required
def download_cmd(args: Namespace) -> None:
    download(args.master, args.trial_id, args.step_id, args.output_dir)


def download(master: str, trial_id: int, step_id: int, output_dir: str) -> None:
    q = api.GraphQLQuery(master)

    step = q.op.steps_by_pk(trial_id=trial_id, id=step_id)
    step.checkpoint.labels()
    step.checkpoint.resources()
    step.checkpoint.uuid()
    step.trial.experiment.config(path="checkpoint_storage")
    step.trial.experiment_id()

    resp = q.send()

    step = resp.steps_by_pk
    if not step:
        raise ValueError("Trial {} step {} not found".format(trial_id, step_id))

    if not step.checkpoint:
        raise ValueError("Trial {} step {} has no checkpoint".format(trial_id, step_id))

    storage_config = step.trial.experiment.config
    manager = storage.build(storage_config)
    if not (
        isinstance(manager, storage.S3StorageManager)
        or isinstance(manager, storage.GCSStorageManager)
    ):
        raise AssertionError(
            "Downloading from S3 or GCS requires the experiment to be configured with "
            "S3 or GCS checkpointing, {} found instead".format(storage_config["type"])
        )
    metadata = storage.StorageMetadata.from_json(step.checkpoint.__to_json_value__())
    manager.download(metadata, output_dir)


args_description = Cmd(
    "c|heckpoint",
    None,
    "manage checkpoints",
    [
        Cmd(
            "download",
            download_cmd,
            "download checkpoint from S3 or GCS",
            [
                Arg("trial_id", help="trial ID", type=int),
                Arg("step_id", help="step ID", type=int),
                Arg("output_dir", help="output directory", default="."),
            ],
        )
    ],
)
