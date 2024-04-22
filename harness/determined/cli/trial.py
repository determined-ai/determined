import argparse
import datetime
import json
import os
import tarfile
import tempfile
from typing import Any, List, Optional, Sequence, Tuple, Union

import termcolor

from determined import cli
from determined.cli import checkpoint, errors, master, render
from determined.common import api
from determined.common.api import bindings
from determined.experimental import client


def none_or_int(string: str) -> Optional[int]:
    if string.lower().strip() in ("null", "none"):
        return None
    return int(string)


def _workload_container_unpack(
    container: bindings.v1WorkloadContainer,
) -> Union[bindings.v1MetricsWorkload, bindings.v1CheckpointWorkload]:
    result = container.training or container.validation or container.checkpoint
    assert result is not None
    return result


def _format_validation(validation: Optional[bindings.v1MetricsWorkload]) -> Optional[str]:
    if not validation:
        return None

    return json.dumps(validation.metrics.to_json(), indent=4)


def _format_checkpoint(ckpt: Optional[bindings.v1CheckpointWorkload]) -> List[Any]:
    if not ckpt:
        return [None, None, None]

    if ckpt.state in (
        bindings.checkpointv1State.COMPLETED,
        bindings.checkpointv1State.DELETED,
    ):
        return [
            ckpt.state,
            ckpt.uuid,
            json.dumps(ckpt.metadata, indent=4),
        ]
    elif ckpt.state in (bindings.checkpointv1State.ACTIVE, bindings.checkpointv1State.ERROR):
        return [ckpt.state, None, json.dumps(ckpt.metadata, indent=4)]
    else:
        raise AssertionError("Invalid checkpoint state: {}".format(ckpt.state))


def _workloads_tabulate(
    workloads: Sequence[bindings.v1WorkloadContainer], metrics: bool
) -> Tuple[List[str], List[List[Any]]]:
    # Print information about individual steps.
    headers = [
        "# of Batches",
        "Report Time",
        "Checkpoint",
        "Checkpoint UUID",
        "Checkpoint Metadata",
        "Validation Metrics",
    ]

    if metrics:
        headers.append("Workload Metrics")

    values = []

    for w in workloads:
        w_unpacked = _workload_container_unpack(w)

        row_metrics = []
        if metrics and w.training:
            row_metrics = [json.dumps(w.training.metrics.to_json(), indent=4)]

        values.append(
            [
                w_unpacked.totalBatches,
                render.format_time(w_unpacked.endTime),
                *_format_checkpoint(w.checkpoint),
                _format_validation(w.validation),
                *row_metrics,
            ]
        )

    return headers, values


def describe_trial(args: argparse.Namespace) -> None:
    session = cli.setup_session(args)

    trial_response = bindings.get_GetTrial(session, trialId=args.trial_id)

    def get_with_offset(offset: int) -> bindings.v1GetTrialWorkloadsResponse:
        return bindings.get_GetTrialWorkloads(
            session,
            offset=offset,
            limit=args.limit,
            trialId=args.trial_id,
            includeBatchMetrics=args.metrics,
        )

    resps = api.read_paginated(get_with_offset, offset=args.offset, pages=args.pages)
    workloads = [w for r in resps for w in r.workloads]

    if args.json:
        data = trial_response.to_json()
        data["workloads"] = [w.to_json() for w in workloads]
        render.print_json(data)
        return

    # Print information about the trial itself.
    headers = [
        "Experiment ID",
        "State",
        "H-Params",
        "Summary Metrics",
        "Started",
        "Ended",
    ]
    trial = trial_response.trial
    values = [
        [
            trial.experimentId,
            trial.state,
            json.dumps(trial.hparams, indent=4),
            json.dumps(trial.summaryMetrics, indent=4),
            render.format_time(trial.startTime),
            render.format_time(trial.endTime),
        ]
    ]
    render.tabulate_or_csv(headers, values, args.csv)

    headers, values = _workloads_tabulate(workloads, metrics=args.metrics)

    print()
    print("Workloads:")
    render.tabulate_or_csv(headers, values, args.csv)

    if args.metrics_summary:
        render.print_json(trial_response.trial.summaryMetrics)


def download(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    det = client.Determined._from_session(sess)

    if [args.latest, args.best, args.uuid].count(True) != 1:
        raise ValueError("exactly one of --latest, --best, or --uuid must be set")
    if args.sort_by is not None and not args.best:
        raise ValueError("--sort-by and --smaller-is-better flags can only be used with --best")
    if (args.sort_by is None) != (args.smaller_is_better is None):
        raise ValueError("--sort-by and --smaller-is-better must be set together")

    if args.uuid:
        ckpt = det.get_checkpoint(args.uuid)
    else:  # Downloaded checkpoint is selected from a trial
        if args.latest:
            sort_by = client.CheckpointSortBy.BATCH_NUMBER
            order_by = client.OrderBy.DESC
        else:  # mode is "best"
            sort_by = args.sort_by
            if sort_by is None:
                order_by = None
            elif args.smaller_is_better:
                order_by = client.OrderBy.ASC
            else:
                order_by = client.OrderBy.DESC

        ckpts = det.get_trial(args.trial_id).list_checkpoints(
            sort_by=sort_by,
            order_by=order_by,
        )
        if not ckpts:
            raise ValueError(f"No checkpoints found for trial {args.trial_id}")

        downloadable_states = [
            client.CheckpointState.COMPLETED,
            client.CheckpointState.PARTIALLY_DELETED,
        ]
        while len(ckpts) > 0:
            ckpt = ckpts.pop()
            if ckpt.state in downloadable_states:
                break
        if len(ckpts) == 0:
            raise errors.CliError("Download failed:  No downloadable checkpoint found")

    path = ckpt.download(path=args.output_dir)

    if args.quiet:
        print(path)
    else:
        checkpoint.render_checkpoint(ckpt, path)


def kill_trial(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    sess.post(f"/api/v1/trials/{args.trial_id}/kill")
    print("Killed trial", args.trial_id)


def trial_logs(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        logs = api.trial_logs(
            sess,
            args.trial_id,
            head=args.head,
            tail=args.tail,
            follow=args.follow,
            agent_ids=args.agent_ids,
            container_ids=args.container_ids,
            rank_ids=args.rank_ids,
            sources=args.sources,
            stdtypes=args.stdtypes,
            min_level=None if args.level is None else bindings.v1LogLevel[args.level],
            timestamp_before=args.timestamp_before,
            timestamp_after=args.timestamp_after,
        )
        if args.json:
            for log in logs:
                render.print_json(log.to_json())
        else:
            api.pprint_logs(logs)
    finally:
        print(
            termcolor.colored(
                "Trial log stream ended. To reopen log stream, run: "
                "det trial logs -f {}".format(args.trial_id),
                "green",
            )
        )


def set_log_retention(args: argparse.Namespace) -> None:
    if not args.forever and not isinstance(args.days, int):
        raise cli.CliError(
            "Please provide an argument to set log retention. --days sets the number of days to"
            " retain logs from the end time of the task, eg. `det t set log-retention 1 --days 50`."
            " --forever retains logs indefinitely, eg.`det t set log-retention 1 --forever`."
        )
    elif isinstance(args.days, int) and (args.days < -1 or args.days > 32767):
        raise cli.CliError(
            "Please provide a valid value for --days. The allowed range is between -1 and "
            "32767 days."
        )
    bindings.put_PutTrialRetainLogs(
        cli.setup_session(args),
        body=bindings.v1PutTrialRetainLogsRequest(
            trialId=args.trial_id, numDays=-1 if args.forever else args.days
        ),
        trialId=args.trial_id,
    )


def generate_support_bundle(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    try:
        output_dir = args.output_dir
        if output_dir is None:
            output_dir = os.getcwd()

        dt = datetime.datetime.now().strftime("%Y%m%dT%H%M%S")
        bundle_name = f"det-bundle-trial-{args.trial_id}-{dt}"
        fullpath = os.path.join(output_dir, f"{bundle_name}.tar.gz")

        with tempfile.TemporaryDirectory() as temp_dir, tarfile.open(fullpath, "w:gz") as bundle:
            trial_logs_filepath = write_trial_logs(sess, args, temp_dir)
            master_logs_filepath = write_master_logs(sess, args, temp_dir)
            api_experiment_filepath, api_trail_filepath = write_api_call(sess, args, temp_dir)

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


def write_trial_logs(sess: api.Session, args: argparse.Namespace, temp_dir: str) -> str:
    trial_logs = api.trial_logs(sess, args.trial_id)
    file_path = os.path.join(temp_dir, "trial_logs.txt")
    with open(file_path, "w") as f:
        for log in trial_logs:
            f.write(log.message)

    return file_path


def write_master_logs(sess: api.Session, args: argparse.Namespace, temp_dir: str) -> str:
    responses = bindings.get_MasterLogs(sess)
    file_path = os.path.join(temp_dir, "master_logs.txt")
    with open(file_path, "w") as f:
        for response in responses:
            f.write(master.format_log_entry(response.logEntry) + "\n")
    return file_path


def write_api_call(sess: api.Session, args: argparse.Namespace, temp_dir: str) -> Tuple[str, str]:
    api_experiment_filepath = os.path.join(temp_dir, "api_experiment_call.json")
    api_trial_filepath = os.path.join(temp_dir, "api_trial_call.json")

    trial_obj = bindings.get_GetTrial(sess, trialId=args.trial_id).trial
    experiment_id = trial_obj.experimentId
    exp_obj = bindings.get_GetExperiment(sess, experimentId=experiment_id)

    create_json_file_in_dir(exp_obj.to_json(), api_experiment_filepath)
    create_json_file_in_dir(trial_obj.to_json(), api_trial_filepath)
    return api_experiment_filepath, api_trial_filepath


def create_json_file_in_dir(content: Any, file_path: str) -> None:
    with open(file_path, "w") as f:
        json.dump(content, f)


logs_args_description: cli.ArgsDescription = [
    cli.Arg(
        "-f",
        "--follow",
        action="store_true",
        help="follow the logs of a running trial, similar to tail -f",
    ),
    cli.Group(
        cli.Arg(
            "--head",
            type=int,
            help="number of lines to show, counting from the beginning "
            "of the log (default is all)",
        ),
        cli.Arg(
            "--tail",
            type=int,
            help="number of lines to show, counting from the end of the log (default is all)",
        ),
    ),
    cli.Arg(
        "--agent-id",
        dest="agent_ids",
        action="append",
        help="agents to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--container-id",
        dest="container_ids",
        action="append",
        help="containers to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--rank-id",
        dest="rank_ids",
        type=int,
        action="append",
        help="containers to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--timestamp-before",
        help="show logs only from before (RFC 3339 format), e.g. '2021-10-26T23:17:12Z'",
    ),
    cli.Arg(
        "--timestamp-after",
        help="show logs only from after (RFC 3339 format), e.g. '2021-10-26T23:17:12Z'",
    ),
    cli.Arg(
        "--level",
        dest="level",
        help=(
            "show logs with this level or higher "
            f"({', '.join([lvl.name for lvl in bindings.v1LogLevel])})"
        ),
        choices=[lvl.name for lvl in bindings.v1LogLevel],
    ),
    cli.Arg(
        "--source",
        dest="sources",
        action="append",
        help="sources to show logs from (repeat for multiple values)",
    ),
    cli.Arg(
        "--stdtype",
        dest="stdtypes",
        action="append",
        help="output stream to show logs from (repeat for multiple values)",
    ),
]

args_description: cli.ArgsDescription = [
    cli.Cmd(
        "t|rial",
        None,
        "manage trials",
        [
            cli.Cmd(
                "describe",
                describe_trial,
                "describe trial",
                [
                    cli.Arg("trial_id", type=int, help="trial ID"),
                    cli.Arg(
                        "--metrics-summary",
                        action="store_true",
                        help="display summary of metrics",
                    ),
                    cli.Arg(
                        "--metrics",
                        action="store_true",
                        help="display full metrics, such as batch metrics",
                    ),
                    cli.Group(
                        cli.output_format_args["csv"],
                        cli.output_format_args["json"],
                    ),
                    *cli.make_pagination_args(limit=1000),
                ],
            ),
            cli.Cmd(
                "download",
                download,
                "download checkpoint for trial",
                [
                    cli.Arg("trial_id", type=int, help="trial ID"),
                    cli.Group(
                        cli.Arg(
                            "--best",
                            action="store_true",
                            help="download the checkpoint with the best validation metric",
                        ),
                        cli.Arg(
                            "--latest",
                            action="store_true",
                            help="download the most recent checkpoint",
                        ),
                        cli.Arg(
                            "--uuid",
                            type=str,
                            help="download a checkpoint by specifying its UUID",
                        ),
                        required=True,
                    ),
                    cli.Arg(
                        "-o",
                        "--output-dir",
                        type=str,
                        default=None,
                        help="Desired output directory for the checkpoint",
                    ),
                    cli.Arg(
                        "--sort-by",
                        type=str,
                        default=None,
                        help="The name of the validation metric to sort on. This argument is only "
                        "used with --best. If --best is passed without --sort-by, the "
                        "experiment's searcher metric is assumed. If this argument is specified, "
                        "--smaller-is-better must also be specified.",
                    ),
                    cli.Arg(
                        "--smaller-is-better",
                        type=cli.string_to_bool,
                        metavar="(true|false)",
                        default=None,
                        help="The sort order for metrics when using --best with --sort-by. For "
                        "example, 'accuracy' would require passing '--smaller-is-better false'. If "
                        "--sort-by is specified, this argument must be specified.",
                    ),
                    cli.Arg(
                        "-q",
                        "--quiet",
                        action="store_true",
                        help="only print the path to the checkpoint",
                    ),
                ],
            ),
            cli.Cmd(
                "support-bundle",
                generate_support_bundle,
                "support bundle",
                [
                    cli.Arg("trial_id", type=int, help="trial ID"),
                    cli.Arg(
                        "-o",
                        "--output-dir",
                        type=str,
                        default=None,
                        help="Desired output directory for the logs",
                    ),
                ],
            ),
            cli.Cmd(
                "logs",
                trial_logs,
                "fetch trial logs",
                [
                    cli.Arg("trial_id", type=int, help="trial ID"),
                    cli.output_format_args["json"],
                    *logs_args_description,
                ],
            ),
            cli.Cmd(
                "kill",
                kill_trial,
                "forcibly terminate a trial",
                [cli.Arg("trial_id", help="trial ID")],
            ),
            cli.Cmd(
                "set",
                None,
                "set trial attributes",
                [
                    cli.Cmd(
                        "log-retention",
                        set_log_retention,
                        "set `log-retention-days` for a trial",
                        [
                            cli.Arg("trial_id", type=int, help="trial ID"),
                            cli.Group(
                                cli.Arg(
                                    "--days",
                                    type=none_or_int,
                                    help="number of days to retain the logs for, from the "
                                    "end time of the task, . allowed range: -1 to 32767.",
                                ),
                                cli.Arg(
                                    "--forever",
                                    action="store_true",
                                    help="retain logs forever",
                                    required=False,
                                ),
                            ),
                        ],
                    ),
                ],
            ),
        ],
    ),
]
