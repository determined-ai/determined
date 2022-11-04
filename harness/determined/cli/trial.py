import distutils.util
import json
import os
import tarfile
import tempfile
from argparse import Namespace
from datetime import datetime
from typing import Any, List, Optional, Sequence, Tuple, Union

from determined import cli
from determined.cli import render
from determined.common import api, constants
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd, Group
from determined.common.experimental import Determined

from .checkpoint import render_checkpoint


def _workload_container_unpack(
    container: bindings.v1WorkloadContainer,
) -> Union[bindings.v1MetricsWorkload, bindings.v1CheckpointWorkload]:
    result = container.training or container.validation or container.checkpoint
    assert result is not None
    return result


def _format_state(
    state: Union[bindings.determinedcheckpointv1State, bindings.determinedexperimentv1State]
) -> str:
    return str(state.value).replace("STATE_", "")


def _format_validation(validation: Optional[bindings.v1MetricsWorkload]) -> List[Any]:
    if not validation:
        return [None, None]

    # TODO(ilia): Get rid of `constants` in favor of the ones from bindings.
    state = _format_state(validation.state)
    if state == constants.COMPLETED:
        return [constants.COMPLETED, json.dumps(validation.metrics.to_json(), indent=4)]
    elif state in (constants.ACTIVE, constants.ERROR):
        return [state, None]
    else:
        raise AssertionError("Invalid state: {}".format(validation.state))


def _format_checkpoint(checkpoint: Optional[bindings.v1CheckpointWorkload]) -> List[Any]:
    if not checkpoint:
        return [None, None, None]

    state = _format_state(checkpoint.state)
    if state in (constants.COMPLETED, constants.DELETED):
        return [
            state,
            checkpoint.uuid,
            json.dumps(checkpoint.metadata, indent=4),
        ]
    elif state in (constants.ACTIVE, constants.ERROR):
        return [checkpoint.state, None, json.dumps(checkpoint.metadata, indent=4)]
    else:
        raise AssertionError("Invalid checkpoint state: {}".format(checkpoint.state))


def _workloads_tabulate(
    workloads: Sequence[bindings.v1WorkloadContainer], metrics: bool
) -> Tuple[List[str], List[List[Any]]]:
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

    if metrics:
        headers.append("Workload Metrics")

    values = []

    for w in workloads:
        w_unpacked = _workload_container_unpack(w)

        row_metrics = []
        if metrics:
            metrics_workload = w.training or w.validation
            if metrics_workload:
                row_metrics = [json.dumps(metrics_workload.metrics.to_json(), indent=4)]

        values.append(
            [
                w_unpacked.totalBatches,
                _format_state(w_unpacked.state),
                render.format_time(w_unpacked.endTime),
                *_format_checkpoint(w.checkpoint),
                *_format_validation(w.validation),
                *row_metrics,
            ]
        )

    return headers, values


@authentication.required
def describe_trial(args: Namespace) -> None:
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
        print(json.dumps(data, indent=4))
        return

    # Print information about the trial itself.
    headers = [
        "Experiment ID",
        "State",
        "H-Params",
        "Start Time",
        "End Time",
    ]
    trial = trial_response.trial
    values = [
        [
            trial.experimentId,
            trial.state,
            json.dumps(trial.hparams, indent=4),
            render.format_time(trial.startTime),
            render.format_time(trial.endTime),
        ]
    ]
    render.tabulate_or_csv(headers, values, args.csv)

    headers, values = _workloads_tabulate(workloads, metrics=args.metrics)

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
    logs = api.trial_logs(
        cli.setup_session(args),
        args.trial_id,
        head=args.head,
        tail=args.tail,
        follow=args.follow,
        agent_ids=args.agent_ids,
        container_ids=args.container_ids,
        rank_ids=args.rank_ids,
        sources=args.sources,
        stdtypes=args.stdtypes,
        min_level=args.level,
        timestamp_before=args.timestamp_before,
        timestamp_after=args.timestamp_after,
    )
    api.pprint_trial_logs(args.trial_id, logs)


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
    session = cli.setup_session(args)
    trial_logs = api.trial_logs(session, args.trial_id)
    file_path = os.path.join(temp_dir, "trial_logs.txt")
    with open(file_path, "w") as f:
        for log in trial_logs:
            f.write(log.message)

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

    trial_obj = bindings.get_GetTrial(cli.setup_session(args), trialId=args.trial_id).trial
    experiment_id = trial_obj.experimentId
    exp_obj = bindings.get_GetExperiment(cli.setup_session(args), experimentId=experiment_id)

    create_json_file_in_dir(exp_obj.to_json(), api_experiment_filepath)
    create_json_file_in_dir(trial_obj.to_json(), api_trial_filepath)
    return api_experiment_filepath, api_trial_filepath


def create_json_file_in_dir(content: Any, file_path: str) -> None:
    with open(file_path, "w") as f:
        json.dump(content, f)


logs_args_description = [
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
            help="number of lines to show, counting from the end of the log (default is all)",
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
        help="show logs only from before (RFC 3339 format), e.g. '2021-10-26T23:17:12Z'",
    ),
    Arg(
        "--timestamp-after",
        help="show logs only from after (RFC 3339 format), e.g. '2021-10-26T23:17:12Z'",
    ),
    Arg(
        "--level",
        dest="level",
        help="show logs with this level or higher "
        + "(TRACE, DEBUG, INFO, WARNING, ERROR, CRITICAL)",
        choices=["TRACE", "DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"],
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
]  # type: List[Any]

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
                    Arg(
                        "--metrics",
                        action="store_true",
                        help="display full metrics, such as batch metrics",
                    ),
                    Group(
                        Arg("--csv", action="store_true", help="print as CSV"),
                        Arg("--json", action="store_true", help="print JSON"),
                    ),
                    *cli.make_pagination_args(limit=1000),
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
                ]
                + logs_args_description,
            ),
            Cmd(
                "kill", kill_trial, "forcibly terminate a trial", [Arg("trial_id", help="trial ID")]
            ),
        ],
    ),
]  # type: List[Any]
