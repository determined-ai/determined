import cgi
import json
import numbers
import pathlib
import sys
import time
from argparse import FileType, Namespace
from pathlib import Path
from pprint import pformat
from typing import Any, Dict, List, Optional, Set, Tuple

import tabulate
from ruamel import yaml
from termcolor import colored

import determined_common.api.authentication as auth
from determined_cli import checkpoint, render
from determined_cli.declarative_argparse import Arg, Cmd, Group
from determined_cli.trial import logs
from determined_cli.user import authentication_required
from determined_common import api, constants, context
from determined_common.api import gql

# Avoid reporting BrokenPipeError when piping `tabulate` output through
# a filter like `head`.
FLUSH = False


def patch_experiment(args: Namespace, verb: str, patch_doc: Dict[str, Any]) -> None:
    api.patch_experiment(args.master, args.experiment_id, patch_doc)


@authentication_required
def activate(args: Namespace) -> None:
    api.activate_experiment(args.master, args.experiment_id)
    print("Activated experiment {}".format(args.experiment_id))


@authentication_required
def archive(args: Namespace) -> None:
    patch_experiment(args, "archive", {"archived": True})
    print("Archived experiment {}".format(args.experiment_id))


@authentication_required
def cancel(args: Namespace) -> None:
    patch_experiment(args, "cancel", {"state": "STOPPING_CANCELED"})
    print("Canceled experiment {}".format(args.experiment_id))


def follow_experiment_logs(master_url: str, exp_id: int) -> None:
    # Get the ID of this experiment's first trial (i.e., the one with the lowest ID).
    q = api.GraphQLQuery(master_url)
    trials = q.op.trials(
        where=gql.trials_bool_exp(experiment_id=gql.Int_comparison_exp(_eq=exp_id)),
        order_by=[gql.trials_order_by(id=gql.order_by.asc)],
        limit=1,
    )
    trials.id()

    print("Waiting for first trial to begin...")
    while True:
        resp = q.send()
        if resp.trials:
            break
        else:
            time.sleep(0.1)

    first_trial_id = resp.trials[0].id
    print("Following first trial with ID {}".format(first_trial_id))

    # Call `logs --follow` on the new trial.
    logs_args = Namespace(trial_id=first_trial_id, follow=True, master=master_url, tail=None)
    logs(logs_args)


def follow_test_experiment_logs(master_url: str, exp_id: int) -> None:
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
            print(colored(stage + (25 - len(stage)) * ".", color), end="")
            print(colored(" [" + checkbox + "]", color), end="")

            if idx == len(stages) - 1:
                print("\n" if ended else "\r", end="")
            else:
                print(", ", end="")

    q = api.GraphQLQuery(master_url)
    exp = q.op.experiments_by_pk(id=exp_id)
    exp.state()
    steps = exp.trials.steps(order_by=[gql.steps_order_by(id=gql.order_by.asc)])
    steps.checkpoint().id()
    steps.validation().id()

    while True:
        exp = q.send().experiments_by_pk

        # Wait for experiment to start and initialize a trial and step.
        step = None
        if exp.trials and exp.trials[0].steps:
            step = exp.trials[0].steps[0]

        # Update the active stage by examining the status of the experiment. The way the GraphQL
        # library works is that the checkpoint and validation attributes of a step are always
        # present and non-None, but they don't have any attributes of their own when the
        # corresponding database object doesn't exist.
        if exp.state == constants.COMPLETED:
            active_stage = 4
        elif step and hasattr(step.checkpoint, "id"):
            active_stage = 3
        elif step and hasattr(step.validation, "id"):
            active_stage = 2
        elif step:
            active_stage = 1
        else:
            active_stage = 0

        # If the experiment is in a terminal state, output the appropriate
        # message and exit. Otherwise, sleep and repeat.
        if exp.state == "COMPLETED":
            print_progress(active_stage, ended=True)
            print(colored("Model definition test succeeded! 🎉", "green"))
            return
        elif exp.state == constants.CANCELED:
            print_progress(active_stage, ended=True)
            print(
                colored(
                    "Model definition test (ID: {}) canceled before "
                    "model test could complete. Please re-run the "
                    "command.".format(exp_id),
                    "yellow",
                )
            )
            sys.exit(1)
        elif exp.state == constants.ERROR:
            print_progress(active_stage, ended=True)
            trial_id = exp.trials[0].id
            logs_args = Namespace(trial_id=trial_id, master=master_url, tail=None, follow=False)
            logs(logs_args)
            sys.exit(1)
        else:
            print_progress(active_stage, ended=False)
            time.sleep(0.2)


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


@authentication_required
def create(args: Namespace) -> None:
    experiment_config = yaml.safe_load(args.config_file.read())
    if not experiment_config or not isinstance(experiment_config, dict):
        print("Error: invalid experiment config file {}".format(args.config_file.name))
        sys.exit(1)
    args.config_file.close()

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
        # Check if the experiment configuration passes master validation by sending a
        # create request with the "validate_only" flag enabled.
        print(colored("Validating experiment configuration...", "yellow"), end="\r")
        api.experiment.create_experiment(
            master_url=args.master,
            config=experiment_config,
            model_context=model_context,
            template=args.template if args.template else None,
            validate_only=True,
            additional_body_fields=additional_body_fields,
        )
        print(colored("Experiment configuration validation succeeded! 🎉", "green"))

        # Create a test experiment.
        exp_id = api.experiment.create_test_experiment(
            master_url=args.master,
            config=experiment_config,
            model_context=model_context,
            template=args.template if args.template else None,
            additional_body_fields=additional_body_fields,
        )
        print(colored("Test experiment ID: {}".format(exp_id), "green"))
        follow_test_experiment_logs(args.master, exp_id)
    else:
        exp_id = api.experiment.create_experiment(
            master_url=args.master,
            config=experiment_config,
            model_context=model_context,
            template=args.template if args.template else None,
            validate_only=True if args.test_mode else False,
            activate=True if not args.paused else False,
            additional_body_fields=additional_body_fields,
        )
        print("Created experiment {}".format(exp_id))
        # Activate the new experiment unless "--paused" is given.
        if not args.paused and args.follow_first_trial:
            follow_experiment_logs(args.master, exp_id)


@authentication_required
def delete_experiment(args: Namespace) -> None:
    if args.yes or render.yes_or_no(
        "Deleting an experiment will result in the unrecoverable \n"
        "deletion of all associated logs, checkpoints, and other \n"
        "metadata associated with the experiment. For a recoverable \n"
        "alternative, see the 'det archive' command. Do you still \n"
        "wish to proceed?"
    ):
        api.delete(args.master, "experiments/{}".format(args.experiment_id))
        print("Successfully deleted experiment {}".format(args.experiment_id))
    else:
        print("Aborting experiment deletion.")


@authentication_required
def describe(args: Namespace) -> None:
    ids = [int(x) for x in args.experiment_ids.split(",")]

    q = api.GraphQLQuery(args.master)
    exps = q.op.experiments(where=gql.experiments_bool_exp(id=gql.Int_comparison_exp(_in=ids)))
    exps.archived()
    exps.config()
    exps.end_time()
    exps.id()
    exps.progress()
    exps.start_time()
    exps.state()

    trials = exps.trials(order_by=[gql.trials_order_by(id=gql.order_by.asc)])
    trials.end_time()
    trials.hparams()
    trials.id()
    trials.start_time()
    trials.state()

    steps = trials.steps(order_by=[gql.steps_order_by(id=gql.order_by.asc)])
    steps.end_time()
    steps.id()
    steps.start_time()
    steps.state()
    steps.trial_id()

    steps.checkpoint.end_time()
    steps.checkpoint.start_time()
    steps.checkpoint.state()

    steps.validation.end_time()
    steps.validation.start_time()
    steps.validation.state()

    if args.metrics:
        steps.metrics(path="avg_metrics")
        steps.validation.metrics()

    resp = q.send()

    # Re-sort the experiment objects to match the original order.
    exps_by_id = {e.id: e for e in resp.experiments}
    experiments = [exps_by_id[id] for id in ids]

    if args.json:
        print(json.dumps(resp.__to_json_value__()["experiments"], indent=4))
        return

    # Display overall experiment information.
    headers = [
        "Experiment ID",
        "State",
        "Progress",
        "Start Time",
        "End Time",
        "Description",
        "Archived",
        "Labels",
    ]
    values = [
        [
            e.id,
            e.state,
            render.format_percent(e.progress),
            render.format_time(e.start_time),
            render.format_time(e.end_time),
            e.config.get("description"),
            e.archived,
            ", ".join(sorted(e.config.get("labels", []))),
        ]
        for e in experiments
    ]
    if not args.outdir:
        outfile = None
        print("Experiment:")
    else:
        outfile = args.outdir.joinpath("experiments.csv")
    render.tabulate_or_csv(headers, values, args.csv, outfile)

    # Display trial-related information.
    headers = ["Trial ID", "Experiment ID", "State", "Start Time", "End Time", "H-Params"]
    values = [
        [
            t.id,
            e.id,
            t.state,
            render.format_time(t.start_time),
            render.format_time(t.end_time),
            json.dumps(t.hparams, indent=4),
        ]
        for e in experiments
        for t in e.trials
    ]
    if not args.outdir:
        outfile = None
        print("\nTrials:")
    else:
        outfile = args.outdir.joinpath("trials.csv")
    render.tabulate_or_csv(headers, values, args.csv, outfile)

    # Display step-related information.
    if args.metrics:
        # Accumulate the scalar training and validation metric names from all provided experiments.
        t_metrics_names = sorted({n for e in experiments for n in scalar_training_metrics_names(e)})
        t_metrics_headers = ["Training Metric: {}".format(name) for name in t_metrics_names]

        v_metrics_names = sorted(
            {n for e in experiments for n in scalar_validation_metrics_names(e)}
        )
        v_metrics_headers = ["Validation Metric: {}".format(name) for name in v_metrics_names]
    else:
        t_metrics_headers = []
        v_metrics_headers = []

    headers = (
        ["Trial ID", "Step ID", "State", "Start Time", "End Time"]
        + t_metrics_headers
        + [
            "Checkpoint State",
            "Checkpoint Start Time",
            "Checkpoint End Time",
            "Validation State",
            "Validation Start Time",
            "Validation End Time",
        ]
        + v_metrics_headers
    )

    values = []
    for e in experiments:
        for t in e.trials:
            for step in t.steps:
                t_metrics_fields = []
                if hasattr(step, "metrics"):
                    avg_metrics = step.metrics
                    for name in t_metrics_names:
                        if name in avg_metrics:
                            t_metrics_fields.append(avg_metrics[name])
                        else:
                            t_metrics_fields.append(None)

                checkpoint = step.checkpoint
                if checkpoint:
                    checkpoint_state = checkpoint.state
                    checkpoint_start_time = checkpoint.start_time
                    checkpoint_end_time = checkpoint.end_time
                else:
                    checkpoint_state = None
                    checkpoint_start_time = None
                    checkpoint_end_time = None

                validation = step.validation
                if validation:
                    validation_state = validation.state
                    validation_start_time = validation.start_time
                    validation_end_time = validation.end_time

                else:
                    validation_state = None
                    validation_start_time = None
                    validation_end_time = None

                if args.metrics:
                    v_metrics_fields = [
                        api.metric.get_validation_metric(name, validation)
                        for name in v_metrics_names
                    ]
                else:
                    v_metrics_fields = []

                row = (
                    [
                        step.trial_id,
                        step.id,
                        step.state,
                        render.format_time(step.start_time),
                        render.format_time(step.end_time),
                    ]
                    + t_metrics_fields
                    + [
                        checkpoint_state,
                        render.format_time(checkpoint_start_time),
                        render.format_time(checkpoint_end_time),
                        validation_state,
                        render.format_time(validation_start_time),
                        render.format_time(validation_end_time),
                    ]
                    + v_metrics_fields
                )
                values.append(row)

    if not args.outdir:
        outfile = None
        print("\nSteps:")
    else:
        outfile = args.outdir.joinpath("steps.csv")
    render.tabulate_or_csv(headers, values, args.csv, outfile)


@authentication_required
def config(args: Namespace) -> None:
    q = api.GraphQLQuery(args.master)
    q.op.experiments_by_pk(id=args.experiment_id).config()
    resp = q.send()
    yaml.safe_dump(resp.experiments_by_pk.config, stream=sys.stdout, default_flow_style=False)


@authentication_required
def download_model_def(args: Namespace) -> None:
    resp = api.get(args.master, "experiments/{}/model_def".format(args.experiment_id))
    value, params = cgi.parse_header(resp.headers["Content-Disposition"])
    if value == "attachment" and "filename" in params:
        with args.output_dir.joinpath(params["filename"]).open("wb") as f:
            f.write(resp.content)
    else:
        raise api.errors.BadResponseException(
            "Unexpected Content-Disposition header format. {}: {}".format(value, params)
        )


@authentication_required
def kill_experiment(args: Namespace) -> None:
    api.post(args.master, "experiments/{}/kill".format(args.experiment_id))
    print("Killed experiment {}".format(args.experiment_id))


@authentication_required
def wait(args: Namespace) -> None:
    while True:
        q = api.GraphQLQuery(args.master)
        q.op.experiments_by_pk(id=args.experiment_id).state()
        resp = q.send()
        state = resp.experiments_by_pk.state

        if state in constants.TERMINAL_STATES:
            print("Experiment {} terminated with state {}".format(args.experiment_id, state))
            if state == constants.COMPLETED:
                sys.exit(0)
            else:
                sys.exit(1)

        time.sleep(args.polling_interval)


@authentication_required
def list_experiments(args: Namespace) -> None:
    where = None
    if not args.all:
        user = api.Authentication.instance().get_session_user()
        where = gql.experiments_bool_exp(
            archived=gql.Boolean_comparison_exp(_eq=False),
            owner=gql.users_bool_exp(username=gql.String_comparison_exp(_eq=user)),
        )

    q = api.GraphQLQuery(args.master)
    exps = q.op.experiments(order_by=[gql.experiments_order_by(id=gql.order_by.desc)], where=where)
    exps.archived()
    exps.config()
    exps.end_time()
    exps.id()
    exps.owner.username()
    exps.progress()
    exps.start_time()
    exps.state()

    resp = q.send()

    def format_experiment(e: Any) -> List[Any]:
        result = [
            e.id,
            e.owner.username,
            e.config["description"],
            e.state,
            render.format_percent(e.progress),
            render.format_time(e.start_time),
            render.format_time(e.end_time),
        ]
        if args.all:
            result.append(e.archived)
        return result

    headers = ["ID", "Owner", "Description", "State", "Progress", "Start Time", "End Time"]
    if args.all:
        headers.append("Archived")

    values = [format_experiment(e) for e in resp.experiments]
    render.tabulate_or_csv(headers, values, args.csv)


def is_number(value: Any) -> bool:
    return isinstance(value, numbers.Number)


def scalar_training_metrics_names(exp: Any) -> Set[str]:
    """
    Given an experiment history, return the names of training metrics
    that are associated with scalar, numeric values.

    This function assumes that all batches in an experiment return
    consistent training metric names and types. Therefore, the first
    non-null batch metrics dictionary is used to extract names.
    """
    for trial in exp.trials:
        for step in trial.steps:
            metrics = step.metrics
            if not metrics:
                continue
            return set(metrics.keys())

    return set()


def scalar_validation_metrics_names(exp: Any) -> Set[str]:
    for trial in exp.trials:
        for step in trial.steps:
            try:
                v_metrics = step.validation.metrics["validation_metrics"]
                return {metric for metric, value in v_metrics.items() if is_number(value)}
            except Exception:
                pass

    return set()


@authentication_required
def list_trials(args: Namespace) -> None:
    q = api.GraphQLQuery(args.master)
    trials = q.op.trials(
        order_by=[gql.trials_order_by(id=gql.order_by.asc)],
        where=gql.trials_bool_exp(experiment_id=gql.Int_comparison_exp(_eq=args.experiment_id)),
    )
    trials.id()
    trials.state()
    trials.hparams()
    trials.start_time()
    trials.end_time()
    trials.steps_aggregate().aggregate.count()

    resp = q.send()

    headers = ["Trial ID", "State", "H-Params", "Start Time", "End Time", "# of Steps"]
    values = [
        [
            t.id,
            t.state,
            json.dumps(t.hparams, indent=4),
            render.format_time(t.start_time),
            render.format_time(t.end_time),
            t.steps_aggregate.aggregate.count,
        ]
        for t in resp.trials
    ]

    render.tabulate_or_csv(headers, values, args.csv)


@authentication_required
def pause(args: Namespace) -> None:
    patch_experiment(args, "pause", {"state": "PAUSED"})
    print("Paused experiment {}".format(args.experiment_id))


@authentication_required
def set_description(args: Namespace) -> None:
    patch_experiment(args, "change description of", {"description": args.description})
    print("Set description of experiment {} to '{}'".format(args.experiment_id, args.description))


@authentication_required
def add_label(args: Namespace) -> None:
    patch_experiment(args, "add label to", {"labels": {args.label: True}})
    print("Added label '{}' to experiment {}".format(args.label, args.experiment_id))


@authentication_required
def remove_label(args: Namespace) -> None:
    patch_experiment(args, "remove label from", {"labels": {args.label: None}})
    print("Removed label '{}' from experiment {}".format(args.label, args.experiment_id))


@authentication_required
def set_max_slots(args: Namespace) -> None:
    patch_experiment(args, "change `max_slots` of", {"resources": {"max_slots": args.max_slots}})
    print("Set `max_slots` of experiment {} to {}".format(args.experiment_id, args.max_slots))


@authentication_required
def set_weight(args: Namespace) -> None:
    patch_experiment(args, "change `weight` of", {"resources": {"weight": args.weight}})
    print("Set `weight` of experiment {} to {}".format(args.experiment_id, args.weight))


@authentication_required
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
            "Step ID",
            "State",
            "Validation Metric\n({})".format(metric_name),
            "UUID",
            "Resources",
        ]
        values = [
            [
                c["trial_id"],
                c["step_id"],
                c["state"],
                api.metric.get_validation_metric(metric_name, c["step"]["validation"]),
                c["uuid"],
                render.format_resources(c["resources"]),
            ]
            for c in sorted(checkpoints, key=lambda c: (c["trial_id"], c["step_id"]))
            if "step" in c and c["step"].get("validation") is not None
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


@authentication_required
def unarchive(args: Namespace) -> None:
    patch_experiment(args, "archive", {"archived": False})
    print("Unarchived experiment {}".format(args.experiment_id))


def none_or_int(string: str) -> Optional[int]:
    if string.lower().strip() in ("null", "none"):
        return None
    return int(string)


def experiment_id_completer(prefix: str, parsed_args: Namespace, **kwargs: Any) -> List[str]:
    auth.initialize_session(parsed_args.master, parsed_args.user, try_reauth=True)
    q = api.GraphQLQuery(parsed_args.master)
    q.op.experiments().id()
    resp = q.send()
    return [str(e["id"]) for e in resp.experiments]


def experiment_id_arg(help: str) -> Arg:
    return Arg("experiment_id", type=int, help=help, completer=experiment_id_completer)


args_description = Cmd(
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
                Arg("--csv", action="store_true", help="print as CSV"),
            ],
        ),
        Cmd(
            "list-checkpoints lc",
            checkpoint.list,
            "list checkpoints of experiment",
            [
                experiment_id_arg("experiment ID"),
                Arg(
                    "--best",
                    type=int,
                    help="Return the best N checkpoints for this experiment. "
                    "If this flag is used, only checkpoints with an associated "
                    "validation metric will be considered.",
                ),
                Arg(
                    "-d",
                    "--download-dir",
                    type=Path,
                    help="download the listed checkpoints to this directory. "
                    "The resources of each checkpoint will be saved in a "
                    "subdirectory labeled with the experiment ID, trial ID, "
                    "and step ID. This flag is only supported for experiments "
                    "configured to use S3 or GCS checkpoint storage.",
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
                    "--template",
                    type=str,
                    help="name of template to apply to the experiment configuration",
                ),
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
                        "step and validation step run successfully and that "
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
            ],
        ),
    ],
)
