import distutils.util
import json
import time
from argparse import Namespace
from typing import Any, List, Optional, Tuple

from termcolor import colored

from determined_cli import render
from determined_common import api, constants
from determined_common.api import gql
from determined_common.experimental import Determined

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

    checkpoint_gql = steps.checkpoint()
    checkpoint_gql.state()
    checkpoint_gql.uuid()

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
    offset, state = 0, None

    def print_logs(limit: Optional[int] = None) -> None:
        nonlocal offset, state
        path = "trials/{}/logsv2?offset={}".format(args.trial_id, offset)
        if limit:
            path = "{}&limit=?".format(limit)
        for log in api.get(args.master, path).json():
            print(log["message"], end="")
            offset, state = log["id"], log["state"]

    print_logs(args.tail)
    if args.follow:
        return

    try:
        while True:
            print_logs()
            if state in constants.TERMINAL_STATES:
                break
            time.sleep(0.1)
    except KeyboardInterrupt:
        pass
    finally:
        print(
            colored(
                "Trial is in the {} state. To reopen log stream, run: "
                "det trial logs -f {}".format(state, args.trial_id),
                "green",
            )
        )


def download(args: Namespace) -> None:
    checkpoint = (
        Determined(args.master, None)
        .get_trial(args.trial_id)
        .select_checkpoint(
            latest=args.latest,
            best=args.best,
            uuid=args.uuid,
            sort_by=args.sort_by,
            smaller_is_better=args.smaller_is_better,
        )
    )

    path = checkpoint.download(path=args.output_dir)

    if args.quiet:
        print(path)
        return

    print("Local checkpoint path:")
    print(path, "\n")

    # Print information about the downloaded step/checkpoint.
    table = [
        ["Batch #", checkpoint.batch_number],
        ["Start Time", render.format_time(checkpoint.start_time)],
        ["End Time", render.format_time(checkpoint.end_time)],
        ["Checkpoint UUID", checkpoint.uuid],
        ["Validation Metrics", format_validation(checkpoint.validation)[1]],
    ]

    headers, values = zip(*table)  # type: ignore

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
                        "--sort-by",
                        type=str,
                        default=None,
                        help="The name of the validation metric to sort on. This argument is only "
                        "used with --best. If --best is passed without --sort-by, the "
                        "experiment's searcher metric is assumed. If this argument is specified, "
                        "--smaller-is-better must also be specified.",
                    ),
                    Arg(
                        "--smaller-is-better",
                        type=lambda s: bool(distutils.util.strtobool(s)),
                        default=None,
                        help="The sort order for metrics when using --best with --sort-by. For "
                        "example, 'accuracy' would require passing '--smaller-is-better false'. If "
                        "--sort-by is specified, this argument must be specified.",
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
