import distutils.util
import json
import os
import tarfile
import tempfile
import typing
from argparse import Namespace
from datetime import datetime
from typing import Any, List, Tuple

from determined.cli import render
from determined.cli.session import setup_session
from determined.common import api
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd, Group
from determined.common.experimental import Determined

from .checkpoint import format_checkpoint, format_validation, render_checkpoint


@authentication.required
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
        "Report Time",
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
            s["total_batches"],
            s["state"],
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


@authentication.required
def kill_trial(args: Namespace) -> None:
    api.post(args.master, "/api/v1/trials/{}/kill".format(args.trial_id))
    print("Killed trial {}".format(args.trial_id))


@authentication.required
def trial_logs(args: Namespace) -> None:
    api.pprint_trial_logs(
        args.master,
        args.trial_id,
        head=args.head,
        tail=args.tail,
        follow=args.follow,
        agent_ids=args.agent_ids,
        container_ids=args.container_ids,
        rank_ids=args.rank_ids,
        sources=args.sources,
        stdtypes=args.stdtypes,
        level_above=args.level,
        timestamp_before=args.timestamp_before,
        timestamp_after=args.timestamp_after,
    )


@authentication.required
def generate_support_bundle(args: Namespace) -> None:
    try:
        output_dir = args.output_dir
        if output_dir is None:
            output_dir = os.getcwd()

        dt = datetime.now().strftime("%Y%m%dT%H%M%S")
        bundle_name = f"det-bundle-trial-{args.trial_id}-{dt}"
        fullpath = os.path.join(output_dir, f"{bundle_name}.tar.gz")

        with tempfile.TemporaryDirectory() as temp_dir, tarfile.open(fullpath, "w:gz") as bundle:
            trial_logs_filepath = write_trial_logs(args, temp_dir)
            master_logs_filepath = write_master_logs(args, temp_dir)
            api_experiment_filepath, api_trail_filepath = write_api_call(args, temp_dir)

            bundle.add(
                trial_logs_filepath,
                arcname=os.path.join(bundle_name, os.path.basename(trial_logs_filepath)),
            )
            bundle.add(
                master_logs_filepath,
                arcname=os.path.join(bundle_name, os.path.basename(master_logs_filepath)),
            )
            bundle.add(
                api_trail_filepath,
                arcname=os.path.join(bundle_name, os.path.basename(api_trail_filepath)),
            )
            bundle.add(
                api_experiment_filepath,
                arcname=os.path.join(bundle_name, os.path.basename(api_experiment_filepath)),
            )

            print(f"bundle path: {fullpath}")

    except FileNotFoundError:
        print("Could not create the bundle because the output_dir provived was not found.")


def write_trial_logs(args: Namespace, temp_dir: str) -> str:
    trial_logs = api.trial_logs(args.master, args.trial_id)
    file_path = os.path.join(temp_dir, "trial_logs.txt")
    with open(file_path, "w") as f:
        for log in trial_logs:
            f.write(log["message"])

    return file_path


def write_master_logs(args: Namespace, temp_dir: str) -> str:
    response = api.get(args.master, "logs")
    file_path = os.path.join(temp_dir, "master_logs.txt")
    with open(file_path, "w") as f:
        for log in response.json():
            f.write("{} [{}]: {}\n".format(log["time"], log["level"], log["message"]))
    return file_path


def write_api_call(args: Namespace, temp_dir: str) -> Tuple[str, str]:
    api_experiment_filepath = os.path.join(temp_dir, "api_experiment_call.json")
    api_trial_filepath = os.path.join(temp_dir, "api_trial_call.json")

    trial_obj = bindings.get_GetTrial(setup_session(args), trialId=args.trial_id).trial
    experiment_id = trial_obj.experimentId
    exp_obj = bindings.get_GetExperiment(setup_session(args), experimentId=experiment_id)

    create_json_file_in_dir(exp_obj.to_json(), api_experiment_filepath)
    create_json_file_in_dir(trial_obj.to_json(), api_trial_filepath)
    return api_experiment_filepath, api_trial_filepath


def create_json_file_in_dir(content: typing.Any, file_path: str) -> None:
    with open(file_path, "w") as f:
        json.dump(content, f)


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
                            "--uuid",
                            type=str,
                            help="download a checkpoint by specifying its UUID",
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
                "support-bundle",
                generate_support_bundle,
                "support bundle",
                [
                    Arg("trial_id", type=int, help="trial ID"),
                    Arg(
                        "-o",
                        "--output-dir",
                        type=str,
                        default=None,
                        help="Desired output directory for the logs",
                    ),
                ],
            ),
            Cmd(
                "logs",
                trial_logs,
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
                    Arg(
                        "--agent-id",
                        dest="agent_ids",
                        action="append",
                        help="agents to show logs from (repeat for multiple values)",
                    ),
                    Arg(
                        "--container-id",
                        dest="container_ids",
                        action="append",
                        help="containers to show logs from (repeat for multiple values)",
                    ),
                    Arg(
                        "--rank-id",
                        dest="rank_ids",
                        type=int,
                        action="append",
                        help="containers to show logs from (repeat for multiple values)",
                    ),
                    Arg(
                        "--timestamp-before",
                        help="show logs only from before (RFC 3339 format),"
                        " e.g. '2021-10-26T23:17:12Z'",
                    ),
                    Arg(
                        "--timestamp-after",
                        help="show logs only from after (RFC 3339 format),"
                        " e.g. '2021-10-26T23:17:12Z'",
                    ),
                    Arg(
                        "--level",
                        dest="level",
                        help="show logs with this level or higher "
                        + "(TRACE, DEBUG, INFO, WARNING, ERROR, CRITICAL)",
                    ),
                    Arg(
                        "--source",
                        dest="sources",
                        action="append",
                        help="sources to show logs from (repeat for multiple values)",
                    ),
                    Arg(
                        "--stdtype",
                        dest="stdtypes",
                        action="append",
                        help="output stream to show logs from (repeat for multiple values)",
                    ),
                ],
            ),
            Cmd(
                "kill", kill_trial, "forcibly terminate a trial", [Arg("trial_id", help="trial ID")]
            ),
        ],
    ),
]  # type: List[Any]
