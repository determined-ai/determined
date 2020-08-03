import distutils.util
import json
from argparse import Namespace
from typing import Any, List

from determined_cli import render
from determined_common import api
from determined_common.api.authentication import authentication_required
from determined_common.experimental import Determined

from .checkpoint import format_checkpoint, format_validation, render_checkpoint
from .declarative_argparse import Arg, Cmd, Group


@authentication_required
def describe_trial(args: Namespace) -> None:
    if args.metrics:
        r = api.get(args.master, "trials/{}/metrics".format(args.trial_id))
    else:
        r = api.get(args.master, "trials/{}".format(args.trial_id))

    trial = r.json()

    if args.json:
        print(json.dumps(trial, indent=4))
        return

    # Print information about the trial itself.
    headers = [
        "Experiment ID",
        "State",
        "H-Params",
        "Start Time",
        "End Time",
    ]
    values = [
        [
            trial["experiment_id"],
            trial["state"],
            json.dumps(trial["hparams"], indent=4),
            render.format_time(trial["start_time"]),
            render.format_time(trial["end_time"]),
        ]
    ]
    render.tabulate_or_csv(headers, values, args.csv)

    # Print information about individual steps.
    headers = [
        "# of Batches",
        "State",
        "Start Time",
        "End Time",
        "Checkpoint",
        "Checkpoint UUID",
        "Checkpoint Metadata",
        "Validation",
        "Validation Metrics",
    ]
    if args.metrics:
        headers.append("Workload Metrics")

    values = [
        [
            s["prior_batches_processed"] + s["num_batches"],
            s["state"],
            render.format_time(s["start_time"]),
            render.format_time(s["end_time"]),
            *format_checkpoint(s["checkpoint"]),
            *format_validation(s["validation"]),
            *([json.dumps(s["metrics"], indent=4)] if args.metrics else []),
        ]
        for s in trial["steps"]
    ]

    print()
    print("Workloads:")
    render.tabulate_or_csv(headers, values, args.csv)


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
    else:
        render_checkpoint(checkpoint, path)


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
                api.experiment.logs,
                "fetch trial logs",
                [
                    Arg("trial_id", type=int, help="trial ID"),
                    Arg(
                        "-f",
                        "--follow",
                        action="store_true",
                        help="follow the logs of a running trial, similar to tail -f",
                    ),
                    Group(
                        Arg(
                            "--head",
                            type=int,
                            help="number of lines to show, counting from the beginning "
                            "of the log (default is all)",
                        ),
                        Arg(
                            "--tail",
                            type=int,
                            help="number of lines to show, counting from the end "
                            "of the log (default is all)",
                        ),
                    ),
                ],
            ),
            Cmd(
                "kill", kill_trial, "forcibly terminate a trial", [Arg("trial_id", help="trial ID")]
            ),
        ],
    ),
]  # type: List[Any]
