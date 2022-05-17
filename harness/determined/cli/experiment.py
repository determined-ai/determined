import base64
import distutils.util
import io
import json
import numbers
import pathlib
import sys
import time
from argparse import FileType, Namespace
from pathlib import Path
from pprint import pformat
from typing import Any, Callable, Dict, Iterable, List, Optional, Sequence, Set, Tuple, Union

import tabulate

import determined as det
import determined.experimental
import determined.load
from determined import _local_execution_manager
from determined.cli import checkpoint, render
from determined.cli.command import CONFIG_DESC, parse_config_overrides
from determined.cli.session import setup_session
from determined.common import api, constants, context, set_logger, util, yaml
from determined.common.api import authentication, bindings
from determined.common.declarative_argparse import Arg, Cmd, Group
from determined.common.experimental import Determined, session

from .checkpoint import render_checkpoint

# Avoid reporting BrokenPipeError when piping `tabulate` output through
# a filter like `head`.
FLUSH = False


def patch_experiment(args: Namespace, verb: str, patch_doc: Dict[str, Any]) -> None:
    api.patch_experiment(args.master, args.experiment_id, patch_doc)


@authentication.required
def activate(args: Namespace) -> None:
    bindings.post_ActivateExperiment(setup_session(args), id=args.experiment_id)
    print("Activated experiment {}".format(args.experiment_id))


@authentication.required
def archive(args: Namespace) -> None:
    bindings.post_ArchiveExperiment(setup_session(args), id=args.experiment_id)
    print("Archived experiment {}".format(args.experiment_id))


@authentication.required
def cancel(args: Namespace) -> None:
    bindings.post_CancelExperiment(setup_session(args), id=args.experiment_id)
    print("Canceled experiment {}".format(args.experiment_id))


def read_git_metadata(model_def_path: pathlib.Path) -> Tuple[str, str, str, str]:
    """
    Attempt to read the git metadata from the model definition directory. If
    unsuccessful, print a descriptive error statement and exit.
    """
    try:
        from git import Repo
    except ImportError as e:
        print("Error: Please verify that git is installed correctly: {}".format(e))
        sys.exit(1)

    if model_def_path.is_dir():
        repo_path = model_def_path.resolve()
    else:
        repo_path = model_def_path.parent.resolve()

    if not repo_path.joinpath(".git").is_dir():
        print(
            "Error: No git directory found at {}. Please "
            "initialize a git repository or refrain from "
            "using the --git feature.".format(repo_path)
        )
        sys.exit(1)

    try:
        repo = Repo(str(repo_path))
    except Exception as e:
        print("Failed to initialize git repository at ", "{}: {}".format(repo_path, e))
        sys.exit(1)

    if repo.is_dirty():
        print(
            "Git working directory is dirty. Please commit the "
            "following changes before creating an experiment "
            "with the --git feature:\n"
        )
        print(repo.git.status())
        sys.exit(1)

    commit = repo.commit()
    commit_hash = commit.hexsha
    committer = "{} <{}>".format(commit.committer.name, commit.committer.email)
    commit_date = commit.committed_datetime.isoformat()

    # To get the upstream remote URL:
    #
    # (1) Get the current upstream branch name
    #     (https://stackoverflow.com/a/9753364/2596715)
    # (2) Parse the git remote name from the upstream branch name.
    # (3) Retrieve the URL of the remote from the git configuration.
    try:
        upstream_branch = repo.git.rev_parse("@{u}", abbrev_ref=True, symbolic_full_name=True)
        remote_name = upstream_branch.split("/", 1)[0]
        remote_url = repo.git.config("remote.{}.url".format(remote_name), get=True)
        print("Using remote URL '{}' from upstream branch '{}'".format(remote_url, upstream_branch))
    except Exception as e:
        print("Failed to find the upstream branch: ", e)
        sys.exit(1)

    return (remote_url, commit_hash, committer, commit_date)


def _parse_config_file_or_exit(config_file: io.FileIO, config_overrides: Iterable[str]) -> Dict:
    experiment_config = util.safe_load_yaml_with_exceptions(config_file)

    config_file.close()
    if not experiment_config or not isinstance(experiment_config, dict):
        print("Error: invalid experiment config file {!r}".format(config_file.name))
        sys.exit(1)

    parse_config_overrides(experiment_config, config_overrides)

    return experiment_config


@authentication.required
def submit_experiment(args: Namespace) -> None:
    experiment_config = _parse_config_file_or_exit(args.config_file, args.config)
    model_context = context.Context.from_local(args.model_def, constants.MAX_CONTEXT_SIZE)

    additional_body_fields = {}
    if args.git:
        (
            additional_body_fields["git_remote"],
            additional_body_fields["git_commit"],
            additional_body_fields["git_committer"],
            additional_body_fields["git_commit_date"],
        ) = read_git_metadata(args.model_def)

    if args.test_mode:
        api.experiment.create_test_experiment_and_follow_logs(
            args.master,
            experiment_config,
            model_context,
            template=args.template if args.template else None,
            additional_body_fields=additional_body_fields,
        )
    else:
        api.experiment.create_experiment_and_follow_logs(
            master_url=args.master,
            config=experiment_config,
            model_context=model_context,
            template=args.template if args.template else None,
            additional_body_fields=additional_body_fields,
            activate=not args.paused,
            follow_first_trial_logs=args.follow_first_trial,
        )


def local_experiment(args: Namespace) -> None:
    if not args.test_mode:
        raise NotImplementedError(
            "Local training mode (--local mode without --test mode) is not yet supported. Please "
            "try local test mode by adding the --test flag or cluster training mode by removing "
            "the --local flag."
        )

    experiment_config = _parse_config_file_or_exit(args.config_file, args.config)
    entrypoint = experiment_config["entrypoint"]

    # --local --test mode only makes sense for the legacy trial entrypoints.  Otherwise the user
    # would just run their training script directly.
    if not det.util.match_legacy_trial_class(entrypoint):
        raise NotImplementedError(
            "Local test mode (--local --test) is only supported for Trial-like entrypoints. "
            "Script-like entrypoints are not supported, but maybe you can just invoke your script "
            "directly?"
        )

    set_logger(bool(experiment_config.get("debug", False)))

    with _local_execution_manager(args.model_def.resolve()):
        trial_class = determined.load.trial_class_from_entrypoint(entrypoint)
        determined.experimental.test_one_batch(trial_class=trial_class, config=experiment_config)


def create(args: Namespace) -> None:
    if args.local:
        local_experiment(args)
    else:
        submit_experiment(args)


def limit_offset_paginator(
    method: Callable,
    agg_field: str,
    sess: session.Session,
    limit: int = 200,
    offset: Optional[int] = None,
    **kwargs: Any
) -> List[Any]:
    all_objects: List[Any] = []
    internal_offset = offset or 0
    while True:
        r = method(sess, limit=limit, offset=internal_offset, **kwargs)
        page_objects = getattr(r, agg_field)
        all_objects += page_objects
        internal_offset += len(page_objects)
        if offset is not None or len(page_objects) < limit:
            break
    return all_objects


@authentication.required
def delete_experiment(args: Namespace) -> None:
    if args.yes or render.yes_or_no(
        "Deleting an experiment will result in the unrecoverable \n"
        "deletion of all associated logs, checkpoints, and other \n"
        "metadata associated with the experiment. For a recoverable \n"
        "alternative, see the 'det archive' command. Do you still \n"
        "wish to proceed?"
    ):
        bindings.delete_DeleteExperiment(setup_session(args), experimentId=args.experiment_id)
        print("Delete of experiment {} is in progress".format(args.experiment_id))
    else:
        print("Aborting experiment deletion.")


@authentication.required
def describe(args: Namespace) -> None:
    session = setup_session(args)
    exps = []
    for experiment_id in args.experiment_ids.split(","):
        r = bindings.get_GetExperiment(session, experimentId=experiment_id)
        if args.json:
            exps.append(r.to_json())
        else:
            exps.append(r.experiment)

    if args.json:
        print(json.dumps(exps, indent=4))
        return

    # Display overall experiment information.
    headers = [
        "Experiment ID",
        "State",
        "Progress",
        "Start Time",
        "End Time",
        "Name",
        "Description",
        "Archived",
        "Resource Pool",
        "Labels",
    ]
    values = [
        [
            exp.id,
            exp.state.value.replace("STATE_", ""),
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
    trials_for_experiment: Dict[str, Sequence[bindings.trialv1Trial]] = {}
    for exp in exps:
        trials_for_experiment[exp.id] = bindings.get_GetExperimentTrials(
            session, experimentId=exp.id
        ).trials

    headers = ["Trial ID", "Experiment ID", "State", "Start Time", "End Time", "H-Params"]
    values = [
        [
            trial.id,
            exp.id,
            trial.state.value.replace("STATE_", ""),
            render.format_time(trial.startTime),
            render.format_time(trial.endTime),
            json.dumps(trial.hparams, indent=4),
        ]
        for exp in exps
        for trial in trials_for_experiment[exp.id]
    ]
    if not args.outdir:
        outfile = None
        print("\nTrials:")
    else:
        outfile = args.outdir.joinpath("trials.csv")
    render.tabulate_or_csv(headers, values, args.csv, outfile)

    # Display step-related information.
    t_metrics_headers: List[str] = []
    t_metrics_names: List[str] = []
    v_metrics_headers: List[str] = []
    v_metrics_names: List[str] = []
    if args.metrics:
        # Accumulate the scalar training and validation metric names from all provided experiments.
        for exp in exps:
            sample_trial = trials_for_experiment[exp.id][0]
            sample_workloads = bindings.get_GetTrial(session, trialId=sample_trial.id).workloads
            t_metrics_names += scalar_training_metrics_names(sample_workloads)
            v_metrics_names += scalar_validation_metrics_names(sample_workloads)
        t_metrics_names = sorted(set(t_metrics_names))
        t_metrics_headers = ["Training Metric: {}".format(name) for name in t_metrics_names]
        v_metrics_names = sorted(set(v_metrics_names))
        v_metrics_headers = ["Validation Metric: {}".format(name) for name in v_metrics_names]

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

    wl_output: Dict[int, List[Any]] = {}
    for exp in exps:
        for trial in trials_for_experiment[exp.id]:
            workloads = bindings.get_GetTrial(session, trialId=trial.id).workloads
            for workload in workloads:
                t_metrics_fields = []
                wl_detail: Optional[
                    Union[bindings.v1MetricsWorkload, bindings.v1CheckpointWorkload]
                ] = None
                if workload.training:
                    wl_detail = workload.training
                    for name in t_metrics_names:
                        if wl_detail.metrics and (name in wl_detail.metrics):
                            t_metrics_fields.append(wl_detail.metrics[name])
                        else:
                            t_metrics_fields.append(None)
                else:
                    t_metrics_fields = [None for name in t_metrics_names]

                if workload.checkpoint:
                    wl_detail = workload.checkpoint

                if workload.checkpoint and wl_detail:
                    checkpoint_state = wl_detail.state.value
                    checkpoint_end_time = wl_detail.endTime
                else:
                    checkpoint_state = ""
                    checkpoint_end_time = None

                v_metrics_fields = []
                if workload.validation:
                    wl_detail = workload.validation
                    validation_state = wl_detail.state.value
                    validation_end_time = wl_detail.endTime
                    for name in v_metrics_names:
                        if wl_detail.metrics and (name in wl_detail.metrics):
                            v_metrics_fields.append(wl_detail.metrics[name])
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
                                trial.id,
                                wl_detail.totalBatches,
                                wl_detail.state.value.replace("STATE_", ""),
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

    if not args.outdir:
        outfile = None
        print("\nWorkloads:")
    else:
        outfile = args.outdir.joinpath("workloads.csv")
    values = sorted(wl_output.values(), key=lambda a: int(a[1]))
    render.tabulate_or_csv(headers, values, args.csv, outfile)


@authentication.required
def config(args: Namespace) -> None:
    result = bindings.get_GetExperiment(setup_session(args), experimentId=args.experiment_id).config
    yaml.safe_dump(result, stream=sys.stdout, default_flow_style=False)


@authentication.required
def download_model_def(args: Namespace) -> None:
    resp = bindings.get_GetModelDef(setup_session(args), experimentId=args.experiment_id)
    with args.output_dir.joinpath(str(args.experiment_id)).open("wb") as f:
        f.write(base64.b64decode(resp.b64Tgz))


def download(args: Namespace) -> None:
    exp = Determined(args.master, args.user).get_experiment(args.experiment_id)
    checkpoints = exp.top_n_checkpoints(
        args.top_n, sort_by=args.sort_by, smaller_is_better=args.smaller_is_better
    )

    top_level = pathlib.Path(args.output_dir)
    top_level.mkdir(parents=True, exist_ok=True)
    for ckpt in checkpoints:
        path = ckpt.download(str(top_level.joinpath(ckpt.uuid)))
        if args.quiet:
            print(path)
        else:
            render_checkpoint(ckpt, path)
            print()


@authentication.required
def kill_experiment(args: Namespace) -> None:
    bindings.post_KillExperiment(setup_session(args), id=args.experiment_id)
    print("Killed experiment {}".format(args.experiment_id))


@authentication.required
def wait(args: Namespace) -> None:
    while True:
        r = bindings.get_GetExperiment(
            setup_session(args), experimentId=args.experiment_id
        ).experiment

        if r.state.value.replace("STATE_", "") in constants.TERMINAL_STATES:
            print(
                "Experiment {} terminated with state {}".format(
                    args.experiment_id, r.state.value.replace("STATE_", "")
                )
            )
            if r.state.value.replace("STATE_", "") == constants.COMPLETED:
                sys.exit(0)
            else:
                sys.exit(1)

        time.sleep(args.polling_interval)


@authentication.required
def list_experiments(args: Namespace) -> None:
    kwargs = {
        "limit": args.limit,
        "offset": args.offset,
    }
    if not args.all:
        kwargs["archived"] = False
        kwargs["users"] = [authentication.must_cli_auth().get_session_user()]
    all_experiments: List[bindings.v1Experiment] = limit_offset_paginator(
        bindings.get_GetExperiments, "experiments", setup_session(args), **kwargs
    )

    def format_experiment(e: Any) -> List[Any]:
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
        ]
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
        "Start Time",
        "End Time",
        "Resource Pool",
    ]
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
            return set(metrics.keys())

    return set()


def scalar_validation_metrics_names(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> Set[str]:
    for workload in workloads:
        if workload.validation:
            metrics = workload.validation.metrics
            if not metrics:
                continue
            return set(metrics.keys())

    return set()


@authentication.required
def list_trials(args: Namespace) -> None:
    all_trials: List[bindings.trialv1Trial] = limit_offset_paginator(
        bindings.get_GetExperimentTrials,
        "trials",
        setup_session(args),
        experimentId=args.experiment_id,
        limit=args.limit,
        offset=args.offset,
    )

    headers = ["Trial ID", "State", "H-Params", "Start Time", "End Time", "# of Batches"]
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


@authentication.required
def pause(args: Namespace) -> None:
    bindings.post_PauseExperiment(setup_session(args), id=args.experiment_id)
    print("Paused experiment {}".format(args.experiment_id))


@authentication.required
def set_description(args: Namespace) -> None:
    session = setup_session(args)
    exp = bindings.get_GetExperiment(session, experimentId=args.experiment_id).experiment
    exp_patch = bindings.v1PatchExperiment.from_json(exp.to_json())
    exp_patch.description = args.description
    bindings.patch_PatchExperiment(session, body=exp_patch, experiment_id=args.experiment_id)
    print("Set description of experiment {} to '{}'".format(args.experiment_id, args.description))


@authentication.required
def set_name(args: Namespace) -> None:
    session = setup_session(args)
    exp = bindings.get_GetExperiment(session, experimentId=args.experiment_id).experiment
    exp_patch = bindings.v1PatchExperiment.from_json(exp.to_json())
    exp_patch.name = args.name
    bindings.patch_PatchExperiment(session, body=exp_patch, experiment_id=args.experiment_id)
    print("Set name of experiment {} to '{}'".format(args.experiment_id, args.name))


@authentication.required
def add_label(args: Namespace) -> None:
    session = setup_session(args)
    exp = bindings.get_GetExperiment(session, experimentId=args.experiment_id).experiment
    exp_patch = bindings.v1PatchExperiment.from_json(exp.to_json())
    if exp_patch.labels is None:
        exp_patch.labels = []
    if args.label not in exp_patch.labels:
        exp_patch.labels = list(exp_patch.labels) + [args.label]
        bindings.patch_PatchExperiment(session, body=exp_patch, experiment_id=args.experiment_id)
    print("Added label '{}' to experiment {}".format(args.label, args.experiment_id))


@authentication.required
def remove_label(args: Namespace) -> None:
    session = setup_session(args)
    exp = bindings.get_GetExperiment(session, experimentId=args.experiment_id).experiment
    exp_patch = bindings.v1PatchExperiment.from_json(exp.to_json())
    if (exp_patch.labels) and (args.label in exp_patch.labels):
        exp_patch.labels = [label for label in exp_patch.labels if label != args.label]
        bindings.patch_PatchExperiment(session, body=exp_patch, experiment_id=args.experiment_id)
    print("Removed label '{}' from experiment {}".format(args.label, args.experiment_id))


@authentication.required
def set_max_slots(args: Namespace) -> None:
    patch_experiment(args, "change `max_slots` of", {"resources": {"max_slots": args.max_slots}})
    print("Set `max_slots` of experiment {} to {}".format(args.experiment_id, args.max_slots))


@authentication.required
def set_weight(args: Namespace) -> None:
    patch_experiment(args, "change `weight` of", {"resources": {"weight": args.weight}})
    print("Set `weight` of experiment {} to {}".format(args.experiment_id, args.weight))


@authentication.required
def set_priority(args: Namespace) -> None:
    patch_experiment(args, "change `priority` of", {"resources": {"priority": args.priority}})
    print("Set `priority` of experiment {} to {}".format(args.experiment_id, args.priority))


@authentication.required
def set_gc_policy(args: Namespace) -> None:
    policy = {
        "save_experiment_best": args.save_experiment_best,
        "save_trial_best": args.save_trial_best,
        "save_trial_latest": args.save_trial_latest,
    }

    if not args.yes:
        r = api.get(
            args.master, "experiments/{}/preview_gc".format(args.experiment_id), params=policy
        )
        response = r.json()
        checkpoints = response["checkpoints"]
        metric_name = response["metric_name"]

        headers = [
            "Trial ID",
            "# of Batches",
            "State",
            "Validation Metric\n({})".format(metric_name),
            "UUID",
            "Resources",
        ]
        values = [
            [
                c["trial_id"],
                c["step"]["total_batches"],
                c["state"],
                api.metric.get_validation_metric(metric_name, c["step"]["validation"]),
                c["uuid"],
                render.format_resources(c["resources"]),
            ]
            for c in sorted(checkpoints, key=lambda c: (c["trial_id"], c["end_time"]))
            if "step" in c and c["step"].get("validation")
        ]

        if len(values) != 0:
            print(
                "The following checkpoints with validation will be deleted "
                "by applying this GC Policy:"
            )
            print(tabulate.tabulate(values, headers, tablefmt="presto"), flush=FLUSH)
        print(
            "This policy will delete {} checkpoints with "
            "validations and {} checkpoints without validations.".format(
                len(values), len(checkpoints) - len(values)
            )
        )

    if args.yes or render.yes_or_no(
        "Changing the checkpoint garbage collection policy of an "
        "experiment may result\n"
        "in the unrecoverable deletion of checkpoints.  Do you wish to "
        "proceed?"
    ):
        patch_experiment(args, "change gc policy of", {"checkpoint_storage": policy})
        print("Set GC policy of experiment {} to\n{}".format(args.experiment_id, pformat(policy)))
    else:
        print("Aborting operations.")


@authentication.required
def unarchive(args: Namespace) -> None:
    bindings.post_UnarchiveExperiment(setup_session(args), id=args.experiment_id)
    print("Unarchived experiment {}".format(args.experiment_id))


def none_or_int(string: str) -> Optional[int]:
    if string.lower().strip() in ("null", "none"):
        return None
    return int(string)


def experiment_id_arg(help: str) -> Arg:  # noqa: A002
    return Arg("experiment_id", type=int, help=help)


# do not use util.py's pagination_args because default behavior here is
# to hide pagination and unify all experiment pages into one output
pagination_args = [
    Arg(
        "--limit",
        type=int,
        default=200,
        help="Maximum items per page of results",
    ),
    Arg(
        "--offset",
        type=int,
        default=None,
        help="Number of items to skip before starting page of results",
    ),
]
main_cmd = Cmd(
    "e|xperiment",
    None,
    "manage experiments",
    [
        # Inspection commands.
        Cmd(
            "list",
            list_experiments,
            "list experiments",
            [
                Arg(
                    "--all",
                    "-a",
                    action="store_true",
                    help="show all experiments (including archived and other users')",
                ),
                *pagination_args,
                Arg("--csv", action="store_true", help="print as CSV"),
            ],
            is_default=True,
        ),
        Cmd("config", config, "display experiment config", [experiment_id_arg("experiment ID")]),
        Cmd(
            "describe",
            describe,
            "describe experiment",
            [
                Arg("experiment_ids", help="comma-separated list of experiment IDs to describe"),
                Arg("--metrics", action="store_true", help="display full metrics"),
                Group(
                    Arg("--csv", action="store_true", help="print as CSV"),
                    Arg("--json", action="store_true", help="print as JSON"),
                    Arg("--outdir", type=Path, help="directory to save output"),
                ),
            ],
        ),
        Cmd(
            "download-model-def",
            download_model_def,
            "download model definition",
            [
                experiment_id_arg("experiment ID"),
                Arg("--output-dir", type=Path, help="output directory", default="."),
            ],
        ),
        Cmd(
            "list-trials lt",
            list_trials,
            "list trials of experiment",
            [
                experiment_id_arg("experiment ID"),
                *pagination_args,
                Arg("--csv", action="store_true", help="print as CSV"),
            ],
        ),
        Cmd(
            "list-checkpoints lc",
            checkpoint.list_checkpoints,
            "list checkpoints of experiment",
            [
                experiment_id_arg("experiment ID"),
                Arg(
                    "--best",
                    type=int,
                    help="Return the best N checkpoints for this experiment. "
                    "If this flag is used, only checkpoints with an associated "
                    "validation metric will be considered.",
                    metavar="N",
                ),
                Arg("--csv", action="store_true", help="print as CSV"),
            ],
        ),
        # Create command.
        Cmd(
            "create",
            create,
            "create experiment",
            [
                Arg("config_file", type=FileType("r"), help="experiment config file (.yaml)"),
                Arg("model_def", type=Path, help="file or directory containing model definition"),
                Arg(
                    "-g",
                    "--git",
                    action="store_true",
                    help="Associate git metadata with this experiment. This "
                    "flag assumes that git is installed, a .git repository "
                    "exists in the model definition directory, and that the "
                    "git working tree of that repository is empty.",
                ),
                Arg(
                    "--local",
                    action="store_true",
                    help="Create the experiment in local mode instead of submitting it to the "
                    "cluster. For more information, see documentation on det.experimental.create()",
                ),
                Arg(
                    "--template",
                    type=str,
                    help="name of template to apply to the experiment configuration",
                ),
                Arg("--config", action="append", default=[], help=CONFIG_DESC),
                Group(
                    Arg(
                        "-f",
                        "--follow-first-trial",
                        action="store_true",
                        help="follow the logs of the first trial that is created",
                    ),
                    Arg("--paused", action="store_true", help="do not activate the experiment"),
                    Arg(
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
            ],
        ),
        # Lifecycle management commands.
        Cmd(
            "activate",
            activate,
            "activate experiment",
            [experiment_id_arg("experiment ID to activate")],
        ),
        Cmd("cancel", cancel, "cancel experiment", [experiment_id_arg("experiment ID to cancel")]),
        Cmd("pause", pause, "pause experiment", [experiment_id_arg("experiment ID to pause")]),
        Cmd(
            "archive",
            archive,
            "archive experiment",
            [experiment_id_arg("experiment ID to archive")],
        ),
        Cmd(
            "unarchive",
            unarchive,
            "unarchive experiment",
            [experiment_id_arg("experiment ID to unarchive")],
        ),
        Cmd(
            "delete",
            delete_experiment,
            "delete experiment",
            [
                Arg("experiment_id", help="delete experiment"),
                Arg(
                    "--yes",
                    action="store_true",
                    default=False,
                    help="automatically answer yes to prompts",
                ),
            ],
        ),
        Cmd(
            "download",
            download,
            "download checkpoints for an experiment",
            [
                experiment_id_arg("experiment ID to download"),
                Arg(
                    "-o",
                    "--output-dir",
                    type=str,
                    default="checkpoints",
                    help="Desired top level directory for the checkpoints. "
                    "Checkpoints will be downloaded to "
                    "<output_dir>/<checkpoint_uuid>/<checkpoint_files>.",
                ),
                Arg(
                    "--top-n",
                    type=int,
                    default=1,
                    help="The number of checkpoints to download for the "
                    "experiment. The checkpoints are sorted by validation "
                    "metric as defined by --sort-by and --smaller-is-better."
                    "This command will select the best N checkpoints from the "
                    "top performing N trials of the experiment.",
                ),
                Arg(
                    "--sort-by",
                    type=str,
                    default=None,
                    help="The name of the validation metric to sort on. Without --sort-by, the "
                    "experiment's searcher metric is assumed. If this argument is specified, "
                    "--smaller-is-better must also be specified.",
                ),
                Arg(
                    "--smaller-is-better",
                    type=lambda s: bool(distutils.util.strtobool(s)),
                    default=None,
                    help="The sort order for metrics when using --sort-by. For "
                    "example, 'accuracy' would require passing '--smaller-is-better false'. If "
                    "--sort-by is specified, this argument must be specified.",
                ),
                Arg(
                    "-q",
                    "--quiet",
                    action="store_true",
                    help="Only print the paths to the checkpoints.",
                ),
            ],
        ),
        Cmd(
            "kill", kill_experiment, "kill experiment", [Arg("experiment_id", help="experiment ID")]
        ),
        Cmd(
            "wait",
            wait,
            "wait for experiment to reach terminal state",
            [
                experiment_id_arg("experiment ID"),
                Arg(
                    "--polling-interval",
                    type=int,
                    default=5,
                    help="the interval (in seconds) to poll for updated state",
                ),
            ],
        ),
        # Attribute setting commands.
        Cmd(
            "label",
            None,
            "manage experiment labels",
            [
                Cmd(
                    "add",
                    add_label,
                    "add label",
                    [experiment_id_arg("experiment ID"), Arg("label", help="label")],
                ),
                Cmd(
                    "remove",
                    remove_label,
                    "remove label",
                    [experiment_id_arg("experiment ID"), Arg("label", help="label")],
                ),
            ],
        ),
        Cmd(
            "set",
            None,
            "set experiment attributes",
            [
                Cmd(
                    "description",
                    set_description,
                    "set experiment description",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        Arg("description", help="experiment description"),
                    ],
                ),
                Cmd(
                    "name",
                    set_name,
                    "set experiment name",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        Arg("name", help="experiment name"),
                    ],
                ),
                Cmd(
                    "gc-policy",
                    set_gc_policy,
                    "set experiment GC policy and run GC",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        Arg(
                            "--save-experiment-best",
                            type=int,
                            required=True,
                            help="number of best checkpoints per experiment " "to save",
                        ),
                        Arg(
                            "--save-trial-best",
                            type=int,
                            required=True,
                            help="number of best checkpoints per trial to save",
                        ),
                        Arg(
                            "--save-trial-latest",
                            type=int,
                            required=True,
                            help="number of latest checkpoints per trial to save",
                        ),
                        Arg(
                            "--yes",
                            action="store_true",
                            default=False,
                            help="automatically answer yes to prompts",
                        ),
                    ],
                ),
                Cmd(
                    "max-slots",
                    set_max_slots,
                    "set `max_slots` of experiment",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        Arg("max_slots", type=none_or_int, help="max slots"),
                    ],
                ),
                Cmd(
                    "weight",
                    set_weight,
                    "set `weight` of experiment",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        Arg("weight", type=float, help="weight"),
                    ],
                ),
                Cmd(
                    "priority",
                    set_priority,
                    "set `priority` of experiment",
                    [
                        experiment_id_arg("experiment ID to modify"),
                        Arg("priority", type=int, help="priority"),
                    ],
                ),
            ],
        ),
    ],
)

args_description = [main_cmd]  # type: List[Any]
