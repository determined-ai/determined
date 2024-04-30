import argparse
import base64
import json
import numbers
import pathlib
import pprint
import sys
import time
from typing import Any, Dict, Iterable, List, Optional, Sequence, Set, Union

import tabulate
import termcolor

import determined as det
from determined import cli, common, experimental, load
from determined.cli import checkpoint, ntsc, project, render, trial
from determined.common import api, context, util
from determined.common.api import bindings, logs
from determined.experimental import client

# Avoid reporting BrokenPipeError when piping `tabulate` output through
# a filter like `head`.
FLUSH = False

ZERO_OR_ONE = "?"


def activate(args: argparse.Namespace) -> None:
    bindings.post_ActivateExperiment(cli.setup_session(args), id=args.experiment_id)
    print(f"Activated experiment {args.experiment_id}")


def archive(args: argparse.Namespace) -> None:
    bindings.post_ArchiveExperiment(cli.setup_session(args), id=args.experiment_id)
    print(f"Archived experiment {args.experiment_id}")


def cancel(args: argparse.Namespace) -> None:
    bindings.post_CancelExperiment(cli.setup_session(args), id=args.experiment_id)
    print(f"Canceled experiment {args.experiment_id}")


def _parse_config_text_or_exit(
    config_text: str, path: str, config_overrides: Iterable[str]
) -> Dict:
    experiment_config = util.safe_load_yaml_with_exceptions(config_text)

    if not experiment_config or not isinstance(experiment_config, dict):
        raise argparse.ArgumentError(None, f"Error: invalid experiment config file {path}")

    config = ntsc.parse_config_overrides(experiment_config, config_overrides)

    return config


def _follow_experiment_logs(sess: api.Session, exp_id: int) -> None:
    # Get the ID of this experiment's first trial (i.e., the one with the lowest ID).
    print("Waiting for first trial to begin...")
    while True:
        trials = bindings.get_GetExperimentTrials(sess, experimentId=exp_id).trials
        if len(trials) > 0:
            break
        else:
            time.sleep(0.1)

    first_trial_id = sorted(t_id.id for t_id in trials)[0]
    print(f"Following first trial with ID {first_trial_id}")
    try:
        tlogs = logs.trial_logs(sess, first_trial_id, follow=True)
        logs.pprint_logs(tlogs)
    finally:
        print(
            termcolor.colored(
                "Trial log stream ended. To reopen log stream, run: "
                "det trial logs -f {}".format(first_trial_id),
                "green",
            )
        )


def _follow_test_experiment_logs(sess: api.Session, exp_id: int) -> None:
    def print_progress(active_stage: int, ended: bool) -> None:
        # There are four sequential stages of verification. Track the
        # current stage with an index into this list.
        stages = [
            "Scheduling task",
            "Testing training",
            "Testing validation",
            "Testing checkpointing",
        ]

        for idx, stage in enumerate(stages):
            if active_stage > idx:
                color = "green"
                checkbox = "✔"
            elif active_stage == idx:
                color = "red" if ended else "yellow"
                checkbox = "✗" if ended else " "
            else:
                color = "white"
                checkbox = " "
            print(termcolor.colored(stage + (25 - len(stage)) * ".", color), end="")
            print(termcolor.colored(" [" + checkbox + "]", color), end="")

            if idx == len(stages) - 1:
                print("\n" if ended else "\r", end="")
            else:
                print(", ", end="")

    while True:
        exp = bindings.get_GetExperiment(sess, experimentId=exp_id).experiment
        trials = bindings.get_GetExperimentTrials(sess, experimentId=exp_id).trials

        # Wait for experiment to start and initialize a trial.
        runner_state = trials[0].runnerState if trials else None

        # Update the active_stage by examining the experiment state and trial runner state.
        if exp.state == bindings.experimentv1State.COMPLETED:
            active_stage = 4
        elif runner_state == "checkpointing":
            active_stage = 3
        elif runner_state == "validating":
            active_stage = 2
        elif runner_state in (None, "UNSPECIFIED", "training"):
            active_stage = 1
        else:
            active_stage = 0

        # If the experiment is in a terminal state, output the appropriate
        # message and exit. Otherwise, sleep and repeat.
        if exp.state == bindings.experimentv1State.COMPLETED:
            print_progress(active_stage, ended=True)
            print(termcolor.colored("Model definition test succeeded! 🎉", "green"))
            return
        elif exp.state == bindings.experimentv1State.CANCELED:
            print_progress(active_stage, ended=True)
            print(
                termcolor.colored(
                    f"Model definition test (ID: {exp_id}) canceled before "
                    "model test could complete. Please re-run the command.",
                    "yellow",
                )
            )
            sys.exit(1)
        elif exp.state == bindings.experimentv1State.ERROR:
            print_progress(active_stage, ended=True)
            trial_id = trials[0].id
            try:
                tlogs = logs.trial_logs(sess, trial_id)
                logs.pprint_logs(tlogs)
            finally:
                print(
                    termcolor.colored(
                        "Trial log stream ended. To reopen log stream, run: "
                        "det trial logs -f {}".format(trial_id),
                        "green",
                    )
                )
            sys.exit(1)
        else:
            print_progress(active_stage, ended=False)
            time.sleep(0.2)


def submit_experiment(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    config_text = args.config_file.read()
    args.config_file.close()
    experiment_config = _parse_config_text_or_exit(config_text, args.config_file.name, args.config)
    model_context = context.read_v1_context(args.model_def, args.include)

    if args.config:
        # The user provided tweaks as cli args, so we have to reserialize the submitted experiment
        # config.  This will unfortunately remove comments they had in the yaml, so we only do it
        # when we have to.
        yaml_dump = util.yaml_safe_dump(experiment_config)
        assert yaml_dump is not None
        config_text = yaml_dump

    req = bindings.v1CreateExperimentRequest(
        activate=not args.paused,
        config=config_text,
        modelDefinition=model_context,
        parentId=None,
        projectId=args.project_id,
        template=args.template,
        validateOnly=bool(args.test_mode),
    )

    if args.test_mode:
        print(termcolor.colored("Validating experiment configuration...", "yellow"), end="\r")
        bindings.post_CreateExperiment(sess, body=req)
        print(termcolor.colored("Experiment configuration validation succeeded! 🎉", "green"))

        print(termcolor.colored("Creating test experiment...", "yellow"), end="\r")
        req.validateOnly = False
        test_config = det._make_test_experiment_config(experiment_config)
        req.config = util.yaml_safe_dump(test_config)
        resp = bindings.post_CreateExperiment(sess, body=req)
        print(termcolor.colored(f"Created test experiment {resp.experiment.id}", "green"))

        _follow_test_experiment_logs(sess, resp.experiment.id)

    else:
        resp = bindings.post_CreateExperiment(sess, body=req)

        print(f"Created experiment {resp.experiment.id}")

        if resp.warnings:
            cli.print_launch_warnings(resp.warnings)

        if not args.paused and args.follow_first_trial:
            if args.publish:
                from determined.cli import proxy

                port_map = proxy.parse_port_map_flag(args.publish)
                with proxy.tunnel_experiment(sess, resp.experiment.id, port_map):
                    _follow_experiment_logs(sess, resp.experiment.id)
            else:
                _follow_experiment_logs(sess, resp.experiment.id)


def continue_experiment(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    if args.config_file:
        config_text = args.config_file.read()
        args.config_file.close()
        experiment_config = _parse_config_text_or_exit(
            config_text, args.config_file.name, args.config
        )
    else:
        experiment_config = ntsc.parse_config_overrides({}, args.config)

    config_text = util.yaml_safe_dump(experiment_config)

    req = bindings.v1ContinueExperimentRequest(
        id=args.experiment_id,
        overrideConfig=config_text,
    )
    bindings.post_ContinueExperiment(sess, body=req)
    print(f"Continued experiment {args.experiment_id}")

    if args.follow_first_trial:
        _follow_experiment_logs(sess, args.experiment_id)


def local_experiment(args: argparse.Namespace) -> None:
    if not args.test_mode:
        raise NotImplementedError(
            "Local training mode (--local mode without --test mode) is not yet supported. Please "
            "try local test mode by adding the --test flag or cluster training mode by removing "
            "the --local flag."
        )

    config_text = args.config_file.read()
    args.config_file.close()
    experiment_config = _parse_config_text_or_exit(config_text, args.config_file.name, args.config)
    entrypoint = experiment_config["entrypoint"]

    # --local --test mode only makes sense for the legacy trial entrypoints.  Otherwise the user
    # would just run their training script directly.
    if not det.util.match_legacy_trial_class(entrypoint):
        raise NotImplementedError(
            "Local test mode (--local --test) is only supported for Trial-like entrypoints. "
            "Script-like entrypoints are not supported, but maybe you can just invoke your script "
            "directly?"
        )

    common.set_logger(bool(experiment_config.get("debug", False)))

    with det._local_execution_manager(args.model_def.resolve()):
        trial_class = load.trial_class_from_entrypoint(entrypoint)
        experimental.test_one_batch(trial_class=trial_class, config=experiment_config)


def create(args: argparse.Namespace) -> None:
    if args.local:
        local_experiment(args)
    else:
        submit_experiment(args)


def delete_experiment(args: argparse.Namespace) -> None:
    if args.yes or render.yes_or_no(
        "Deleting an experiment will result in the unrecoverable \n"
        "deletion of all associated logs, checkpoints, and other \n"
        "metadata associated with the experiment. For a recoverable \n"
        "alternative, see the 'det archive' command. Do you still \n"
        "wish to proceed?"
    ):
        bindings.delete_DeleteExperiment(cli.setup_session(args), experimentId=args.experiment_id)
        print(f"Deletion of experiment {args.experiment_id} is in progress")
    else:
        print("Aborting experiment deletion.")


def describe(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    responses: List[bindings.v1GetExperimentResponse] = []
    for experiment_id in args.experiment_ids.split(","):
        r = bindings.get_GetExperiment(sess, experimentId=experiment_id)
        responses.append(r)

    if args.json:
        render.print_json([resp.to_json() for resp in responses])
        return
    exps = [resp.experiment for resp in responses]

    # Display overall experiment information.
    headers = [
        "Experiment ID",
        "State",
        "Progress",
        "Started",
        "Ended",
        "Name",
        "Description",
        "Archived",
        "Resource Pool",
        "Labels",
    ]
    values: List[List] = [
        [
            exp.id,
            exp.state,
            render.format_percent(exp.progress),
            render.format_time(exp.startTime),
            render.format_time(exp.endTime),
            exp.name,
            exp.description,
            exp.archived,
            exp.resourcePool,
            ", ".join(sorted(exp.labels or [])),
        ]
        for exp in exps
    ]
    if not args.outdir:
        outfile = None
        print("Experiment:")
    else:
        outfile = args.outdir.joinpath("experiments.csv")
    render.tabulate_or_csv(headers, values, args.csv, outfile)

    # Display trial-related information.

    def get_all_trials(exp_id: int) -> List[bindings.trialv1Trial]:
        def get_with_offset(offset: int) -> bindings.v1GetExperimentTrialsResponse:
            return bindings.get_GetExperimentTrials(
                sess,
                offset=offset,
                experimentId=exp_id,
            )

        resps = api.read_paginated(get_with_offset)
        return [t for r in resps for t in r.trials]

    trials_for_experiment = {exp.id: get_all_trials(exp.id) for exp in exps}

    headers = ["Trial ID", "Experiment ID", "State", "Started", "Ended", "H-Params"]
    values = [
        [
            trial_obj.id,
            exp.id,
            trial_obj.state.value.replace("STATE_", ""),
            render.format_time(trial_obj.startTime),
            render.format_time(trial_obj.endTime),
            json.dumps(trial_obj.hparams, indent=4),
        ]
        for exp in exps
        for trial_obj in trials_for_experiment[exp.id]
    ]
    if not args.outdir:
        outfile = None
        print("\nTrials:")
    else:
        outfile = args.outdir.joinpath("trials.csv")
    render.tabulate_or_csv(headers, values, args.csv, outfile)

    # Display workload-related information.

    def get_all_workloads(trial_id: int) -> List[bindings.v1WorkloadContainer]:
        def get_with_offset(offset: int) -> bindings.v1GetTrialWorkloadsResponse:
            return bindings.get_GetTrialWorkloads(
                sess,
                offset=offset,
                trialId=trial_id,
                limit=500,
            )

        resps = api.read_paginated(get_with_offset)
        return [w for r in resps for w in r.workloads]

    all_workloads = {
        exp.id: {t.id: get_all_workloads(t.id) for t in trials_for_experiment[exp.id]}
        for exp in exps
    }

    t_metrics_headers: List[str] = []
    t_metrics_names: List[str] = []
    v_metrics_headers: List[str] = []
    v_metrics_names: List[str] = []
    if args.metrics:
        # Accumulate the scalar training and validation metric names from all provided experiments.
        for exp in exps:
            sample_trial = trials_for_experiment[exp.id][0]
            sample_workloads = all_workloads[exp.id][sample_trial.id]
            t_metrics_names += scalar_training_metrics_names(sample_workloads)
            v_metrics_names += scalar_validation_metrics_names(sample_workloads)
        t_metrics_names = sorted(set(t_metrics_names))
        t_metrics_headers = [f"Training Metric: {name}" for name in t_metrics_names]
        v_metrics_names = sorted(set(v_metrics_names))
        v_metrics_headers = [f"Validation Metric: {name}" for name in v_metrics_names]

    headers = (
        ["Trial ID", "# of Batches", "State", "Report Time"]
        + t_metrics_headers
        + [
            "Checkpoint State",
            "Checkpoint Report Time",
            "Validation State",
            "Validation Report Time",
        ]
        + v_metrics_headers
    )

    values = []
    for exp in exps:
        for trial_obj in trials_for_experiment[exp.id]:
            workloads = all_workloads[exp.id][trial_obj.id]
            wl_output: Dict[int, List[Any]] = {}
            for workload in workloads:
                t_metrics_fields = []
                wl_detail: Optional[
                    Union[bindings.v1MetricsWorkload, bindings.v1CheckpointWorkload]
                ] = None
                if workload.training:
                    wl_detail = workload.training
                    for name in t_metrics_names:
                        if (
                            wl_detail.metrics
                            and wl_detail.metrics.avgMetrics
                            and (name in wl_detail.metrics.avgMetrics)
                        ):
                            t_metrics_fields.append(wl_detail.metrics.avgMetrics[name])
                        else:
                            t_metrics_fields.append(None)
                else:
                    t_metrics_fields = [None for name in t_metrics_names]

                if workload.checkpoint:
                    wl_detail = workload.checkpoint

                if workload.checkpoint and wl_detail:
                    assert isinstance(wl_detail, bindings.v1CheckpointWorkload)
                    checkpoint_state = wl_detail.state.value
                    checkpoint_end_time = wl_detail.endTime
                else:
                    checkpoint_state = ""
                    checkpoint_end_time = None

                v_metrics_fields = []
                if workload.validation:
                    wl_detail = workload.validation
                    validation_state = "STATE_COMPLETED"
                    validation_end_time = wl_detail.endTime
                    for name in v_metrics_names:
                        if (
                            wl_detail.metrics
                            and wl_detail.metrics.avgMetrics
                            and (name in wl_detail.metrics.avgMetrics)
                        ):
                            v_metrics_fields.append(wl_detail.metrics.avgMetrics[name])
                        else:
                            v_metrics_fields.append(None)
                else:
                    validation_state = ""
                    validation_end_time = None
                    v_metrics_fields = [None for name in v_metrics_names]

                if wl_detail:
                    if wl_detail.totalBatches in wl_output:
                        # condense training, checkpoints, validation workloads into one step-like
                        # row for compatibility with previous versions of describe
                        merge_row = wl_output[wl_detail.totalBatches]
                        merge_row[3] = max(merge_row[3], render.format_time(wl_detail.endTime))
                        for idx, tfield in enumerate(t_metrics_fields):
                            if tfield and merge_row[4 + idx] is None:
                                merge_row[4 + idx] = tfield
                        start_checkpoint = 4 + len(t_metrics_fields)
                        if checkpoint_state:
                            merge_row[start_checkpoint] = checkpoint_state.replace("STATE_", "")
                            merge_row[start_checkpoint + 1] = render.format_time(
                                checkpoint_end_time
                            )
                        if validation_end_time:
                            merge_row[start_checkpoint + 3] = render.format_time(
                                validation_end_time
                            )
                        if validation_state:
                            merge_row[start_checkpoint + 2] = validation_state.replace("STATE_", "")
                        for idx, vfield in enumerate(v_metrics_fields):
                            if vfield and merge_row[start_checkpoint + idx + 4] is None:
                                merge_row[start_checkpoint + idx + 4] = vfield
                    else:
                        row = (
                            [
                                trial_obj.id,
                                wl_detail.totalBatches,
                                (checkpoint_state or validation_state).replace("STATE_", ""),
                                render.format_time(wl_detail.endTime),
                            ]
                            + t_metrics_fields
                            + [
                                checkpoint_state.replace("STATE_", ""),
                                render.format_time(checkpoint_end_time),
                                validation_state.replace("STATE_", ""),
                                render.format_time(validation_end_time),
                            ]
                            + v_metrics_fields
                        )
                        wl_output[wl_detail.totalBatches] = row

            # Done processing one trial's workloads, add to output values.
            values += sorted(wl_output.values(), key=lambda a: int(a[1]))

    if not args.outdir:
        outfile = None
        print("\nWorkloads:")
    else:
        outfile = args.outdir.joinpath("workloads.csv")
    render.tabulate_or_csv(headers, values, args.csv, outfile)


def experiment_logs(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    trials = bindings.get_GetExperimentTrials(sess, experimentId=args.experiment_id).trials
    if len(trials) == 0:
        raise api.not_found_errs("experiment", args.experiment_id, sess)
    first_trial_id = sorted(t_id.id for t_id in trials)[0]
    try:
        logs = api.trial_logs(
            sess,
            first_trial_id,
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
                "det trial logs -f {}".format(first_trial_id),
                "green",
            )
        )


def set_log_retention(args: argparse.Namespace) -> None:
    if not args.forever and not isinstance(args.days, int):
        raise cli.CliError(
            "Please provide an argument to set log retention. --days sets the number of days to"
            " retain logs from the end time of the task, eg. `det e set log-retention 1 --days 50`."
            " --forever retains logs indefinitely, eg.`det e set log-retention 1 --forever`."
        )
    elif isinstance(args.days, int) and (args.days < -1 or args.days > 32767):
        raise cli.CliError(
            "Please provide a valid value for --days. The allowed range is between -1 and "
            "32767 days."
        )
    bindings.put_PutExperimentRetainLogs(
        cli.setup_session(args),
        body=bindings.v1PutExperimentRetainLogsRequest(
            experimentId=args.experiment_id, numDays=-1 if args.forever else args.days
        ),
        experimentId=args.experiment_id,
    )


def config(args: argparse.Namespace) -> None:
    result = bindings.get_GetExperiment(
        cli.setup_session(args), experimentId=args.experiment_id
    ).experiment.config
    util.yaml_safe_dump(result, stream=sys.stdout, default_flow_style=False)


def download_model_def(args: argparse.Namespace) -> None:
    resp = bindings.get_GetModelDef(cli.setup_session(args), experimentId=args.experiment_id)
    dst = f"experiment_{args.experiment_id}_model_def.tgz"
    with args.output_dir.joinpath(dst).open("wb") as f:
        f.write(base64.b64decode(resp.b64Tgz))


def download(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp = client.Experiment(args.experiment_id, sess)

    ckpt_order_by = None

    if args.smaller_is_better is True:
        ckpt_order_by = client.OrderBy.ASC
    if args.smaller_is_better is False:
        ckpt_order_by = client.OrderBy.DESC

    checkpoints = exp.list_checkpoints(
        sort_by=args.sort_by,
        order_by=ckpt_order_by,
        max_results=args.top_n,
    )

    top_level = pathlib.Path(args.output_dir)
    top_level.mkdir(parents=True, exist_ok=True)
    for ckpt in checkpoints:
        path = ckpt.download(str(top_level.joinpath(ckpt.uuid)))
        if args.quiet:
            print(path)
        else:
            checkpoint.render_checkpoint(ckpt, path)
            print()


def kill_experiment(args: argparse.Namespace) -> None:
    bindings.post_KillExperiment(cli.setup_session(args), id=args.experiment_id)
    print(f"Killed experiment {args.experiment_id}")


def wait(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp = client.Experiment(args.experiment_id, sess)
    state = exp.wait(interval=args.polling_interval)
    if state != client.ExperimentState.COMPLETED:
        sys.exit(1)


@cli.session
def list_experiments(args: argparse.Namespace, sess: api.Session) -> None:
    def get_with_offset(offset: int) -> bindings.v1GetExperimentsResponse:
        return bindings.get_GetExperiments(
            sess,
            offset=offset,
            archived=None if args.all else False,
            limit=args.limit,
            users=None if args.all else [sess.username],
        )

    resps = api.read_paginated(get_with_offset, offset=args.offset, pages=args.pages)
    all_experiments = [e for r in resps for e in r.experiments]

    def format_experiment(e: bindings.v1Experiment) -> List[Any]:
        result = [
            e.id,
            e.username,
            e.name,
            e.forkedFrom,
            e.state.value.replace("STATE_", ""),
            render.format_percent(e.progress),
            render.format_time(e.startTime),
            render.format_time(e.endTime),
            e.resourcePool,
        ]  # type: List[Any]
        if args.show_project:
            result = [e.workspaceName, e.projectName] + result
        if args.all:
            result.append(e.archived)
        return result

    headers = [
        "ID",
        "Owner",
        "Name",
        "Parent ID",
        "State",
        "Progress",
        "Started",
        "Ended",
        "Resource Pool",
    ]
    if args.show_project:
        headers = ["Workspace", "Project"] + headers
    if args.all:
        headers.append("Archived")

    values = [format_experiment(e) for e in all_experiments]
    render.tabulate_or_csv(headers, values, args.csv)


def is_number(value: Any) -> bool:
    return isinstance(value, numbers.Number)


def scalar_training_metrics_names(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> Set[str]:
    """
    Given an experiment history, return the names of training metrics
    that are associated with scalar, numeric values.

    This function assumes that all batches in an experiment return
    consistent training metric names and types. Therefore, the first
    non-null batch metrics dictionary is used to extract names.
    """
    for workload in workloads:
        if workload.training:
            metrics = workload.training.metrics
            if not metrics:
                continue
            return set(metrics.avgMetrics.keys())

    return set()


def scalar_validation_metrics_names(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> Set[str]:
    for workload in workloads:
        if workload.validation:
            metrics = workload.validation.metrics
            if not metrics:
                continue
            return set(metrics.avgMetrics.keys())

    return set()


def list_trials(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)

    def get_with_offset(offset: int) -> bindings.v1GetExperimentTrialsResponse:
        return bindings.get_GetExperimentTrials(
            sess,
            offset=offset,
            experimentId=args.experiment_id,
            limit=args.limit,
        )

    resps = api.read_paginated(get_with_offset, offset=args.offset, pages=args.pages)
    all_trials = [t for r in resps for t in r.trials]

    headers = ["Trial ID", "State", "H-Params", "Started", "Ended", "# of Batches"]
    values = [
        [
            t.id,
            t.state.value.replace("STATE_", ""),
            json.dumps(t.hparams, indent=4),
            render.format_time(t.startTime),
            render.format_time(t.endTime),
            t.totalBatchesProcessed,
        ]
        for t in all_trials
    ]

    render.tabulate_or_csv(headers, values, args.csv)


def pause(args: argparse.Namespace) -> None:
    bindings.post_PauseExperiment(cli.setup_session(args), id=args.experiment_id)
    print(f"Paused experiment {args.experiment_id}")


def set_description(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp = bindings.get_GetExperiment(sess, experimentId=args.experiment_id).experiment
    exp_patch = bindings.v1PatchExperiment.from_json(exp.to_json())
    exp_patch.description = args.description
    bindings.patch_PatchExperiment(sess, body=exp_patch, experiment_id=args.experiment_id)
    print(f"Set description of experiment {args.experiment_id} to '{args.description}'")


def set_name(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp = bindings.get_GetExperiment(sess, experimentId=args.experiment_id).experiment
    exp_patch = bindings.v1PatchExperiment.from_json(exp.to_json())
    exp_patch.name = args.name
    bindings.patch_PatchExperiment(sess, body=exp_patch, experiment_id=args.experiment_id)
    print(f"Set name of experiment {args.experiment_id} to '{args.name}'")


def add_label(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp = bindings.get_GetExperiment(sess, experimentId=args.experiment_id).experiment
    exp_patch = bindings.v1PatchExperiment.from_json(exp.to_json())
    if exp_patch.labels is None:
        exp_patch.labels = []
    if args.label not in exp_patch.labels:
        exp_patch.labels = list(exp_patch.labels) + [args.label]
        bindings.patch_PatchExperiment(sess, body=exp_patch, experiment_id=args.experiment_id)
    print(f"Added label '{args.label}' to experiment {args.experiment_id}")


def remove_label(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp = bindings.get_GetExperiment(sess, experimentId=args.experiment_id).experiment
    exp_patch = bindings.v1PatchExperiment.from_json(exp.to_json())
    if (exp_patch.labels) and (args.label in exp_patch.labels):
        exp_patch.labels = [label for label in exp_patch.labels if label != args.label]
        bindings.patch_PatchExperiment(sess, body=exp_patch, experiment_id=args.experiment_id)
    print(f"Removed label '{args.label}' from experiment {args.experiment_id}")


def set_max_slots(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp_patch = bindings.v1PatchExperiment(
        id=args.experiment_id,
        resources=bindings.PatchExperimentPatchResources(maxSlots=args.max_slots),
    )
    bindings.patch_PatchExperiment(sess, body=exp_patch, experiment_id=args.experiment_id)
    print(f"Set `max_slots` of experiment {args.experiment_id} to {args.max_slots}")


def set_weight(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp_patch = bindings.v1PatchExperiment(
        id=args.experiment_id, resources=bindings.PatchExperimentPatchResources(weight=args.weight)
    )
    bindings.patch_PatchExperiment(sess, body=exp_patch, experiment_id=args.experiment_id)
    print(f"Set `weight` of experiment {args.experiment_id} to {args.weight}")


def set_priority(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    exp_patch = bindings.v1PatchExperiment(
        id=args.experiment_id,
        resources=bindings.PatchExperimentPatchResources(priority=args.priority),
    )
    bindings.patch_PatchExperiment(sess, body=exp_patch, experiment_id=args.experiment_id)
    print(f"Set `priority` of experiment {args.experiment_id} to {args.priority}")


def set_gc_policy(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    policy = {
        "save_experiment_best": args.save_experiment_best,
        "save_trial_best": args.save_trial_best,
        "save_trial_latest": args.save_trial_latest,
    }
    if not args.yes:
        r = sess.get(f"experiments/{args.experiment_id}/preview_gc", params=policy)
        response = r.json()
        checkpoints = response["checkpoints"]
        metric_name = response["metric_name"]

        headers = [
            "Trial ID",
            "# of Batches",
            "State",
            f"Validation Metric\n({metric_name})",
            "UUID",
            "Resources",
        ]
        values = [
            [
                c["TrialID"],
                c["StepsCompleted"],
                c["State"],
                api.metric.get_validation_metric(
                    metric_name, {"metrics": {"validation_metrics": c["ValidationMetrics"]}}
                ),
                c["UUID"],
                render.format_resources(c["Resources"]),
            ]
            for c in sorted(checkpoints, key=lambda c: (c["TrialID"], c["ReportTime"]))
            if "step" in c and c["step"].get("validation")
        ]

        if len(values) != 0:
            print(
                "The following checkpoints with validation will be deleted "
                "by applying this GC Policy:"
            )
            print(tabulate.tabulate(values, headers, tablefmt="presto"), flush=FLUSH)
        print(
            f"This policy will delete {len(values)} checkpoints with "
            f"validations and {len(checkpoints) - len(values)} checkpoints without validations."
        )

    if args.yes or render.yes_or_no(
        "Changing the checkpoint garbage collection policy of an "
        "experiment may result\n"
        "in the unrecoverable deletion of checkpoints.  Do you wish to "
        "proceed?"
    ):
        exp_patch = bindings.v1PatchExperiment(
            id=args.experiment_id,
            checkpointStorage=bindings.PatchExperimentPatchCheckpointStorage(
                saveExperimentBest=args.save_experiment_best,
                saveTrialBest=args.save_trial_best,
                saveTrialLatest=args.save_trial_latest,
            ),
        )
        bindings.patch_PatchExperiment(sess, body=exp_patch, experiment_id=args.experiment_id)
        print(f"Set GC policy of experiment {args.experiment_id} to\n{pprint.pformat(policy)}")
    else:
        print("Aborting operations.")


def unarchive(args: argparse.Namespace) -> None:
    bindings.post_UnarchiveExperiment(cli.setup_session(args), id=args.experiment_id)
    print(f"Unarchived experiment {args.experiment_id}")


def move_experiment(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    (w, p) = project.project_by_name(sess, args.workspace_name, args.project_name)
    req = bindings.v1MoveExperimentRequest(
        destinationProjectId=p.id,
        experimentId=args.experiment_id,
    )
    bindings.post_MoveExperiment(sess, body=req, experimentId=args.experiment_id)
    print(f'Moved experiment {args.experiment_id} to project "{args.project_name}"')


def delete_tensorboard_files(args: argparse.Namespace) -> None:
    sess = cli.setup_session(args)
    d = client.Determined._from_session(sess)
    exp = d.get_experiment(args.experiment_id)
    exp.delete_tensorboard_files()


def none_or_int(string: str) -> Optional[int]:
    if string.lower().strip() in ("null", "none"):
        return None
    return int(string)


def experiment_id_arg(help: str) -> cli.Arg:  # noqa: A002
    return cli.Arg("experiment_id", type=int, help=help)


main_cmd = cli.Cmd(
    "e|xperiment",
    None,
    "manage experiments",
    [
        # Inspection commands.
        cli.Cmd(
            "list ls",
            list_experiments,
            "list experiments",
            [
                cli.Arg(
                    "--all",
                    "-a",
                    action="store_true",
                    help="show all experiments (including archived and other users')",
                ),
                cli.Arg(
                    "--show_project",
                    action="store_true",
                    help="include columns for workspace name and project name",
                ),
                *cli.default_pagination_args,
                cli.Arg("--csv", action="store_true", help="print as CSV"),
            ],
            is_default=True,
        ),
        cli.Cmd(
            "config", config, "display experiment config", [experiment_id_arg("experiment ID")]
        ),
        cli.Cmd(
            "describe",
            describe,
            "describe experiment",
            [
                cli.Arg(
                    "experiment_ids", help="comma-separated list of experiment IDs to describe"
                ),
                cli.Arg("--metrics", action="store_true", help="display full metrics"),
                cli.Group(
                    cli.output_format_args["csv"],
                    cli.output_format_args["json"],
                    cli.Arg("--outdir", type=pathlib.Path, help="directory to save output"),
                ),
            ],
        ),
        cli.Cmd(
            "logs",
            experiment_logs,
            "fetch logs of the first trial of an experiment",
            [
                experiment_id_arg("experiment ID"),
                cli.output_format_args["json"],
                *trial.logs_args_description,
            ],
        ),
        cli.Cmd(
            "download-model-def",
            download_model_def,
            "download model definition",
            [
                experiment_id_arg("experiment ID"),
                cli.Arg("--output-dir", type=pathlib.Path, help="output directory", default="."),
            ],
        ),
        cli.Cmd(
            "list-trials lt",
            list_trials,
            "list trials of experiment",
            [
                experiment_id_arg("experiment ID"),
                *cli.default_pagination_args,
                cli.Arg("--csv", action="store_true", help="print as CSV"),
            ],
        ),
        cli.Cmd(
            "list-checkpoints lc",
            checkpoint.list_checkpoints,
            "list checkpoints of experiment",
            [
                experiment_id_arg("experiment ID"),
                cli.Arg(
                    "--best",
                    type=int,
                    help="Return the best N checkpoints for this experiment. "
                    "If this flag is used, only checkpoints with an associated "
                    "validation metric will be considered.",
                    metavar="N",
                ),
                cli.Arg("--csv", action="store_true", help="print as CSV"),
            ],
        ),
        # Create command.
        cli.Cmd(
            "create",
            create,
            "create experiment",
            [
                cli.Arg(
                    "config_file",
                    type=argparse.FileType("r"),
                    help="experiment config file (.yaml)",
                ),
                cli.Arg(
                    "model_def",
                    nargs=ZERO_OR_ONE,
                    default=None,
                    type=pathlib.Path,
                    help="file or directory containing model definition",
                ),
                cli.Arg(
                    "-i",
                    "--include",
                    action="append",
                    default=[],
                    type=pathlib.Path,
                    help="additional files to copy into the task container",
                ),
                cli.Arg(
                    "--local",
                    action="store_true",
                    help="Create the experiment in local mode instead of submitting it to the "
                    "cluster. Requires --test. For more information, visit How to Debug Models "
                    "(https://docs.determined.ai/latest/model-dev-guide/debug-models.html).",
                ),
                cli.Arg(
                    "--template",
                    type=str,
                    help="name of template to apply to the experiment configuration",
                ),
                cli.Arg("--project_id", type=int, help="place this experiment inside this project"),
                cli.Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                cli.Group(
                    cli.Arg(
                        "-f",
                        "--follow-first-trial",
                        action="store_true",
                        help="follow the logs of the first trial that is created",
                    ),
                    cli.Arg("--paused", action="store_true", help="do not activate the experiment"),
                    cli.Arg(
                        "-t",
                        "--test-mode",
                        action="store_true",
                        help="Test the experiment configuration and model "
                        "definition by creating and scheduling a very small "
                        "experiment. This command will verify that a training "
                        "workload and validation workload run successfully and that "
                        "checkpoints can be saved. The test experiment will "
                        "be archived on creation.",
                    ),
                ),
                cli.Arg(
                    "-p",
                    "--publish",
                    action="append",
                    default=[],
                    type=str,
                    help="publish task ports to the host",
                ),
            ],
        ),
        # Continue experiment command.
        cli.Cmd(
            "continue",
            continue_experiment,
            "resume or recover training for a single-searcher experiment",
            [
                experiment_id_arg("experiment ID to continue"),
                cli.Arg(
                    "--config-file",
                    type=argparse.FileType("r"),
                    help="experiment config file (.yaml)",
                ),
                cli.Arg("--config", action="append", default=[], help=ntsc.CONFIG_DESC),
                cli.Arg(
                    "-f",
                    "--follow-first-trial",
                    action="store_true",
                    help="follow logs of the trial that is being continued",
                ),
            ],
        ),
        # Lifecycle management commands.
        cli.Cmd(
            "activate unpause",
            activate,
            "activate experiment, i.e., unpause a paused experiment",
            [experiment_id_arg("experiment ID to activate")],
        ),
        cli.Cmd(
            "cancel", cancel, "cancel experiment", [experiment_id_arg("experiment ID to cancel")]
        ),
        cli.Cmd("pause", pause, "pause experiment", [experiment_id_arg("experiment ID to pause")]),
        cli.Cmd(
            "archive",
            archive,
            "archive experiment",
            [experiment_id_arg("experiment ID to archive")],
        ),
        cli.Cmd(
            "unarchive",
            unarchive,
            "unarchive experiment",
            [experiment_id_arg("experiment ID to unarchive")],
        ),
        cli.Cmd(
            "delete",
            delete_experiment,
            "delete experiment",
            [
                cli.Arg("experiment_id", help="delete experiment"),
                cli.Arg(
                    "--yes",
                    action="store_true",
                    default=False,
                    help="automatically answer yes to prompts",
                ),
            ],
        ),
        cli.Cmd(
            "download",
            download,
            "download checkpoints for an experiment",
            [
                experiment_id_arg("experiment ID to download"),
                cli.Arg(
                    "-o",
                    "--output-dir",
                    type=str,
                    default="checkpoints",
                    help="Desired top level directory for the checkpoints. "
                    "Checkpoints will be downloaded to "
                    "<output_dir>/<checkpoint_uuid>/<checkpoint_files>.",
                ),
                cli.Arg(
                    "--top-n",
                    type=int,
                    default=1,
                    help="The number of checkpoints to download for the "
                    "experiment. The checkpoints are sorted by validation "
                    "metric as defined by --sort-by and --smaller-is-better."
                    "This command will select the best N checkpoints from the "
                    "top performing N trials of the experiment.",
                ),
                cli.Arg(
                    "--sort-by",
                    type=str,
                    default=None,
                    help="The name of the validation metric to sort on. Without --sort-by, the "
                    "experiment's searcher metric is assumed. If this argument is specified, "
                    "--smaller-is-better must also be specified.",
                ),
                cli.Arg(
                    "--smaller-is-better",
                    type=lambda s: util.strtobool(s),
                    default=None,
                    help="The sort order for metrics when using --sort-by. For "
                    "example, 'accuracy' would require passing '--smaller-is-better false'. If "
                    "--sort-by is specified, this argument must be specified.",
                ),
                cli.Arg(
                    "-q",
                    "--quiet",
                    action="store_true",
                    help="Only print the paths to the checkpoints.",
                ),
            ],
        ),
        cli.Cmd(
            "kill",
            kill_experiment,
            "kill experiment",
            [cli.Arg("experiment_id", help="experiment ID")],
        ),
        cli.Cmd(
            "wait",
            wait,
            "wait for experiment to reach terminal state",
            [
                experiment_id_arg("experiment ID"),
                cli.Arg(
                    "--polling-interval",
                    type=int,
                    default=5,
                    help="the interval (in seconds) to poll for updated state",
                ),
            ],
        ),
        # Attribute setting commands.
        cli.Cmd(
            "label",
            None,
            "manage experiment labels",
            [
                cli.Cmd(
                    "add",
                    add_label,
                    "add label",
                    [experiment_id_arg("experiment ID"), cli.Arg("label", help="label")],
                ),
                cli.Cmd(
                    "remove",
                    remove_label,
                    "remove label",
                    [experiment_id_arg("experiment ID"), cli.Arg("label", help="label")],
                ),
            ],
        ),
        cli.Cmd(
            "move",
            move_experiment,
            "move experiment to another project",
            [
                experiment_id_arg("experiment ID to move"),
                cli.Arg("workspace_name", help="Name of destination workspace"),
                cli.Arg("project_name", help="Name of destination project"),
            ],
        ),
        cli.Cmd(
            "set",
            None,
            "set experiment attributes",
            [
                cli.Cmd(
                    "description",
                    set_description,
                    "set experiment description",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        cli.Arg("description", help="experiment description"),
                    ],
                ),
                cli.Cmd(
                    "name",
                    set_name,
                    "set experiment name",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        cli.Arg("name", help="experiment name"),
                    ],
                ),
                cli.Cmd(
                    "gc-policy",
                    set_gc_policy,
                    "set experiment GC policy and run GC",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        cli.Arg(
                            "--save-experiment-best",
                            type=int,
                            required=True,
                            help="number of best checkpoints per experiment " "to save",
                        ),
                        cli.Arg(
                            "--save-trial-best",
                            type=int,
                            required=True,
                            help="number of best checkpoints per trial to save",
                        ),
                        cli.Arg(
                            "--save-trial-latest",
                            type=int,
                            required=True,
                            help="number of latest checkpoints per trial to save",
                        ),
                        cli.Arg(
                            "--yes",
                            action="store_true",
                            default=False,
                            help="automatically answer yes to prompts",
                        ),
                    ],
                ),
                cli.Cmd(
                    "max-slots",
                    set_max_slots,
                    "set `max_slots` of experiment",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        cli.Arg("max_slots", type=none_or_int, help="max slots"),
                    ],
                ),
                cli.Cmd(
                    "log-retention",
                    set_log_retention,
                    "set `log-retention-days` for an experiment",
                    [
                        experiment_id_arg("experiment ID"),
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
                cli.Cmd(
                    "weight",
                    set_weight,
                    "set `weight` of experiment",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        cli.Arg("weight", type=float, help="weight"),
                    ],
                ),
                cli.Cmd(
                    "priority",
                    set_priority,
                    "set `priority` of experiment",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        cli.Arg("priority", type=int, help="priority"),
                    ],
                ),
            ],
        ),
        cli.Cmd(
            "delete-tb-files",
            delete_tensorboard_files,
            "delete TensorBoard files associated with the provided experiment ID",
            [
                cli.Arg("experiment_id", type=int, help="Experiment ID"),
            ],
        ),
    ],
)

args_description = [main_cmd]  # type: List[Any]
