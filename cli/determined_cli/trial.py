import distutils.util
import json
import time
from argparse import Namespace
from typing import Any, List, Optional, Tuple

from termcolor import colored

from determined_cli import render
from determined_common import api, checkpoint, constants
from determined_common.api import gql
from determined_common.check import check_gt

from .declarative_argparse import Arg, Cmd, Group
from .user import authentication_required


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


def format_validation(validation: gql.validations) -> List[Any]:
    if not validation:
        return [None, None]

    if validation.state == constants.COMPLETED:
        return [constants.COMPLETED, json.dumps(validation.metrics, indent=4)]
    elif validation.state in (constants.ACTIVE, constants.ERROR):
        return [validation.state, None]
    else:
        raise AssertionError("Invalid validation state: {}".format(validation.state))


@authentication_required
def describe_trial(args: Namespace) -> None:
    q = api.GraphQLQuery(args.master)
    trial = q.op.trials_by_pk(id=args.trial_id)
    trial.end_time()
    trial.experiment_id()
    trial.hparams()
    trial.start_time()
    trial.state()

    steps = trial.steps()
    steps.metrics()
    steps.id()
    steps.state()
    steps.start_time()
    steps.end_time()

    checkpoint = steps.checkpoint()
    checkpoint.state()
    checkpoint.uuid()

    validation = steps.validation()
    validation.state()
    validation.metrics()

    resp = q.send()

    if args.json:
        print(json.dumps(resp.trials_by_pk.__to_json_value__(), indent=4))
        return

    trial = resp.trials_by_pk

    # Print information about the trial itself.
    headers = ["Experiment ID", "State", "H-Params", "Start Time", "End Time"]
    values = [
        [
            trial.experiment_id,
            trial.state,
            json.dumps(trial.hparams, indent=4),
            render.format_time(trial.start_time),
            render.format_time(trial.end_time),
        ]
    ]
    render.tabulate_or_csv(headers, values, args.csv)

    # Print information about individual steps.
    headers = [
        "Step #",
        "State",
        "Start Time",
        "End Time",
        "Checkpoint",
        "Checkpoint UUID",
        "Validation",
        "Validation Metrics",
    ]
    if args.metrics:
        headers.append("Step Metrics")

    values = [
        [
            s.id,
            s.state,
            render.format_time(s.start_time),
            render.format_time(s.end_time),
            *format_checkpoint(s.checkpoint),
            *format_validation(s.validation),
            *([json.dumps(s.metrics, indent=4)] if args.metrics else []),
        ]
        for s in trial.steps
    ]

    print()
    print("Steps:")
    render.tabulate_or_csv(headers, values, args.csv)


@authentication_required
def logs(args: Namespace) -> None:
    def logs_query(tail: Optional[int] = None, greater_than_id: Optional[int] = None) -> Any:
        q = api.GraphQLQuery(args.master)
        limit = None
        order_by = [gql.trial_logs_order_by(id=gql.order_by.asc)]
        where = gql.trial_logs_bool_exp(trial_id=gql.Int_comparison_exp(_eq=args.trial_id))
        if greater_than_id is not None:
            where.id = gql.Int_comparison_exp(_gt=greater_than_id)
        if tail is not None:
            order_by = [gql.trial_logs_order_by(id=gql.order_by.desc)]
            limit = tail
        logs = q.op.trial_logs(where=where, order_by=order_by, limit=limit)
        logs.id()
        logs.message()
        return q

    def process_response(logs: Any, latest_log_id: int) -> Tuple[int, bool]:
        changes = False
        for log in logs:
            check_gt(log.id, latest_log_id)
            latest_log_id = log.id
            msg = api.decode_bytes(log.message)
            print(msg, end="")
            changes = True

        return latest_log_id, changes

    resp = logs_query(args.tail).send()
    logs = resp.trial_logs
    # Due to limitations of the GraphQL API, which mimics SQL, requesting a tail means we have to
    # get the results in descending ID order and reverse them afterward.
    if args.tail is not None:
        logs = reversed(logs)
    latest_log_id, _ = process_response(logs, -1)

    # "Follow" mode is implemented as a loop in the CLI. We assume that
    # newer log messages have a numerically larger ID than older log
    # messages, so we keep track of the max ID seen so far.
    if args.follow:
        state_query = api.GraphQLQuery(args.master)
        state_query.op.trials_by_pk(id=args.trial_id).state()

        no_change_count = 0
        try:
            while True:
                # Poll for new logs every 100 ms.
                time.sleep(0.1)

                # The `tail` parameter only makes sense the first time we
                # fetch logs.
                resp = logs_query(greater_than_id=latest_log_id).send()
                latest_log_id, changes = process_response(resp.trial_logs, latest_log_id)
                no_change_count = 0 if changes else no_change_count + 1

                # Wait for 10 poll requests before checking that the experiment is in a stopped
                # state.
                if no_change_count >= 10:
                    no_change_count = 0
                    resp = state_query.send()
                    if resp.trials_by_pk.state in constants.TERMINAL_STATES:
                        raise KeyboardInterrupt()
        except KeyboardInterrupt:
            resp = state_query.send()

            print(
                colored(
                    "Trial is in the {} state. To reopen log stream, run: "
                    "det trial logs -f {}".format(resp.trials_by_pk.state, args.trial_id),
                    "green",
                )
            )


@authentication_required
def download(args: Namespace) -> None:
    path, ckpt = checkpoint.download(
        args.trial_id,
        latest=args.latest,
        best=args.best,
        output_dir=args.output_dir,
        uuid=args.uuid,
        master=args.master,
        metric_name=args.metric_name,
        smaller_is_better=args.smaller_is_better,
    )

    if args.quiet:
        print(path)
        return

    print("Local checkpoint path:")
    print(path, "\n")

    # Print information about the downloaded step/checkpoint.
    table = [
        ["Step #", ckpt.step.id],
        ["Start Time", render.format_time(ckpt.step.start_time)],
        ["End Time", render.format_time(ckpt.step.end_time)],
        ["Checkpoint UUID", format_checkpoint(ckpt)[1]],
        ["Validation Metrics", format_validation(ckpt.validation)[1]],
    ]

    headers, values = zip(*table)

    render.tabulate_or_csv(headers, [values], False)


@authentication_required
def kill_trial(args: Namespace) -> None:
    api.post(args.master, "trials/{}/kill".format(args.trial_id))
    print("Killed trial {}".format(args.trial_id))


args_description = [
    Cmd(
        "t|rial",
        None,
        "manage trials",
        [
            Cmd(
                "describe",
                describe_trial,
                "describe trial",
                [
                    Arg("trial_id", type=int, help="trial ID"),
                    Arg("--metrics", action="store_true", help="display full metrics"),
                    Group(
                        Arg("--csv", action="store_true", help="print as CSV"),
                        Arg("--json", action="store_true", help="print JSON"),
                    ),
                ],
            ),
            Cmd(
                "download",
                download,
                "download checkpoint for trial",
                [
                    Arg("trial_id", type=int, help="trial ID"),
                    Group(
                        Arg(
                            "--best",
                            action="store_true",
                            help="download the checkpoint with the best validation metric",
                        ),
                        Arg(
                            "--latest",
                            action="store_true",
                            help="download the most recent checkpoint",
                        ),
                        Arg(
                            "--uuid", type=str, help="download a checkpoint by specifying its UUID",
                        ),
                        required=True,
                    ),
                    Arg(
                        "-o",
                        "--output-dir",
                        type=str,
                        default=None,
                        help="Desired output directory for the checkpoint",
                    ),
                    Arg(
                        "--metric-name",
                        type=str,
                        default=None,
                        help="The name of the validation metric to sort on. This argument is only "
                        "used with --best. If --best is passed without --metric-name, the "
                        "experiment's searcher metric is assumed. If this argument is specified, "
                        "--smaller-is-better must also be specified.",
                    ),
                    Arg(
                        "--smaller-is-better",
                        type=lambda s: bool(distutils.util.strtobool(s)),
                        default=None,
                        help="The sort order for metrics when using --best with --metric-name. For "
                        "example, 'accuracy' would require passing '--smaller-is-better false'. If "
                        "--metric-name is specified, this argument must be specified.",
                    ),
                    Arg(
                        "-q",
                        "--quiet",
                        action="store_true",
                        help="only print the path to the checkpoint",
                    ),
                ],
            ),
            Cmd(
                "logs",
                logs,
                "fetch trial logs",
                [
                    Arg("trial_id", type=int, help="trial ID"),
                    Arg(
                        "-f",
                        "--follow",
                        action="store_true",
                        help="follow the logs of a running trial, similar to tail -f",
                    ),
                    Arg(
                        "--tail",
                        type=int,
                        help="number of lines to show, counting from the end "
                        "of the log (default is all)",
                    ),
                ],
            ),
            Cmd(
                "kill", kill_trial, "forcibly terminate a trial", [Arg("trial_id", help="trial ID")]
            ),
        ],
    ),
]  # type: List[Any]
