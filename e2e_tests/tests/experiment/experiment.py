import logging
import os
import re
import subprocess
import sys
import tempfile
import time
from typing import Any, Dict, List, Optional, Sequence

import pytest

from determined.common import api, util
from determined.common.api import bindings
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests.cluster import utils as cluster_utils


def maybe_create_experiment(
    sess: api.Session,
    config_file: str,
    model_def_file: Optional[str] = None,
    create_args: Optional[List[str]] = None,
) -> subprocess.CompletedProcess:
    command = [
        "det",
        "experiment",
        "create",
        config_file,
    ]

    if model_def_file is not None:
        command.append(model_def_file)

    if create_args is not None:
        command += create_args

    env = os.environ.copy()
    env["DET_DEBUG"] = "true"

    return detproc.run(
        sess,
        command,
        universal_newlines=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
    )


def create_experiment(
    sess: api.Session,
    config_file: str,
    model_def_file: Optional[str] = None,
    create_args: Optional[List[str]] = None,
) -> int:
    p = maybe_create_experiment(sess, config_file, model_def_file, create_args)
    assert p.returncode == 0, f"\nstdout:\n{p.stdout} \nstderr:\n{p.stderr}"
    m = re.search(r"Created experiment (\d+)\n", str(p.stdout))
    assert m is not None
    return int(m.group(1))


def maybe_run_autotuning_experiment(
    sess: api.Session,
    config_file: str,
    model_def_file: str,
    create_args: Optional[List[str]] = None,
    search_method_name: str = "_test",
    max_trials: int = 4,
) -> subprocess.CompletedProcess:
    command = [
        "python3",
        "-m",
        "determined.pytorch.dsat",
        search_method_name,
        config_file,
        model_def_file,
        "--max-trials",
        str(max_trials),
    ]

    if create_args is not None:
        command += create_args

    env = os.environ.copy()
    env["DET_DEBUG"] = "true"
    env["DET_MASTER"] = conf.make_master_url()

    return detproc.run(
        sess,
        command,
        universal_newlines=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
    )


def run_autotuning_experiment(
    sess: api.Session,
    config_file: str,
    model_def_file: str,
    create_args: Optional[List[str]] = None,
    search_method_name: str = "_test",
    max_trials: int = 4,
) -> int:
    p = maybe_run_autotuning_experiment(
        sess, config_file, model_def_file, create_args, search_method_name, max_trials
    )
    assert p.returncode == 0, f"\nstdout:\n{p.stdout}\nstderr:\n{p.stderr}"
    m = re.search(r"Created experiment (\d+)\n", str(p.stdout))
    assert m is not None
    return int(m.group(1))


def archive_experiments(
    sess: api.Session, experiment_ids: List[int], name: Optional[str] = None
) -> None:
    body = bindings.v1ArchiveExperimentsRequest(experimentIds=experiment_ids)
    if name is not None:
        filters = bindings.v1BulkExperimentFilters(name=name)
        body = bindings.v1ArchiveExperimentsRequest(experimentIds=[], filters=filters)
    bindings.post_ArchiveExperiments(sess, body=body)


def pause_experiment(sess: api.Session, experiment_id: int) -> None:
    command = ["det", "experiment", "pause", str(experiment_id)]
    detproc.check_call(sess, command)


def pause_experiments(
    sess: api.Session,
    experiment_ids: List[int],
    name: Optional[str] = None,
) -> None:
    body = bindings.v1PauseExperimentsRequest(experimentIds=experiment_ids)
    if name is not None:
        filters = bindings.v1BulkExperimentFilters(name=name)
        body = bindings.v1PauseExperimentsRequest(experimentIds=[], filters=filters)
    bindings.post_PauseExperiments(sess, body=body)


def activate_experiment(sess: api.Session, experiment_id: int) -> None:
    command = ["det", "experiment", "activate", str(experiment_id)]
    detproc.check_call(sess, command)


def activate_experiments(
    sess: api.Session, experiment_ids: List[int], name: Optional[str] = None
) -> None:
    if name is None:
        body = bindings.v1ActivateExperimentsRequest(experimentIds=experiment_ids)
    else:
        filters = bindings.v1BulkExperimentFilters(name=name)
        body = bindings.v1ActivateExperimentsRequest(experimentIds=[], filters=filters)
    bindings.post_ActivateExperiments(sess, body=body)


def cancel_experiment(sess: api.Session, experiment_id: int) -> None:
    bindings.post_CancelExperiment(sess, id=experiment_id)
    wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.CANCELED)


def kill_experiment(sess: api.Session, experiment_id: int) -> None:
    bindings.post_KillExperiment(sess, id=experiment_id)
    wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.CANCELED)


def cancel_experiments(
    sess: api.Session, experiment_ids: List[int], name: Optional[str] = None
) -> None:
    if name is None:
        body = bindings.v1CancelExperimentsRequest(experimentIds=experiment_ids)
    else:
        filters = bindings.v1BulkExperimentFilters(name=name)
        body = bindings.v1CancelExperimentsRequest(experimentIds=[], filters=filters)
    bindings.post_CancelExperiments(sess, body=body)


def kill_experiments(
    sess: api.Session, experiment_ids: List[int], name: Optional[str] = None
) -> None:
    if name is None:
        body = bindings.v1KillExperimentsRequest(experimentIds=experiment_ids)
    else:
        filters = bindings.v1BulkExperimentFilters(name=name)
        body = bindings.v1KillExperimentsRequest(experimentIds=[], filters=filters)
    bindings.post_KillExperiments(sess, body=body)


def kill_trial(sess: api.Session, trial_id: int) -> None:
    bindings.post_KillTrial(sess, id=trial_id)
    wait_for_trial_state(sess, trial_id, bindings.trialv1State.CANCELED)


def wait_for_experiment_by_name_is_active(
    sess: api.Session,
    experiment_name: str,
    min_trials: int = 1,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
    log_every: int = 60,
) -> int:
    for seconds_waited in range(max_wait_secs):
        try:
            response = bindings.get_GetExperiments(sess, name=experiment_name).experiments
            if len(response) == 0:
                time.sleep(1)
                continue
            if len(response) > 1:
                pytest.fail(
                    f"Multiple experiments with name={experiment_name}, "
                    f"expected only 1 experiment."
                )
            experiment = response[0]
            experiment_id = experiment.id
        except api.errors.NotFoundException:
            logging.warning(
                f"Experiment not yet available to check state: experiment {experiment_name}"
            )
            time.sleep(0.25)
            continue

        if _is_experiment_active(experiment.state):
            if experiment.numTrials > min_trials:
                return experiment_id
            time.sleep(0.25)
            continue

        if is_terminal_experiment_state(experiment.state):
            report_failed_experiment(sess, experiment_id)

            pytest.fail(
                f"Experiment {experiment_id} terminated in {experiment.state.value} state, "
                f"expected {bindings.experimentv1State.ACTIVE}"
            )

        if seconds_waited > 0 and seconds_waited % log_every == 0:
            print(
                f"Waited {seconds_waited} seconds for experiment {experiment_name} "
                f"(currently {experiment.state.value}) to reach "
                f"{bindings.experimentv1State.ACTIVE}"
            )

        time.sleep(1)

    else:
        pytest.fail(f"Experiment {experiment_name} did not start any trial {max_wait_secs} seconds")


def _is_experiment_active(exp_state: bindings.experimentv1State) -> bool:
    return exp_state in (
        bindings.experimentv1State.ACTIVE,
        bindings.experimentv1State.RUNNING,
        bindings.experimentv1State.QUEUED,
        bindings.experimentv1State.PULLING,
        bindings.experimentv1State.STARTING,
    )


def wait_for_experiment_state(
    sess: api.Session,
    experiment_id: int,
    target_state: bindings.experimentv1State,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
    log_every: int = 60,
) -> None:
    for seconds_waited in range(max_wait_secs):
        try:
            state = experiment_state(sess, experiment_id)
        except api.errors.NotFoundException:
            logging.warning(
                "Experiment not yet available to check state: "
                "experiment {}".format(experiment_id)
            )
            time.sleep(0.25)
            continue

        if state == target_state:
            return

        if is_terminal_experiment_state(state):
            if state != target_state:
                report_failed_experiment(sess, experiment_id)

            pytest.fail(
                f"Experiment {experiment_id} terminated in {state.value} state, "
                f"expected {target_state.value}"
            )

        if seconds_waited > 0 and seconds_waited % log_every == 0:
            print(
                f"Waited {seconds_waited} seconds for experiment {experiment_id} "
                f"(currently {state.value}) to reach {target_state.value}"
            )

        time.sleep(1)

    else:
        if target_state == bindings.experimentv1State.COMPLETED:
            kill_experiment(sess, experiment_id)
        report_failed_experiment(sess, experiment_id)
        pytest.fail(
            "Experiment did not reach target state {} after {} seconds".format(
                target_state.value, max_wait_secs
            )
        )


def wait_for_trial_state(
    sess: api.Session,
    trial_id: int,
    target_state: bindings.trialv1State,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
    log_every: int = 60,
) -> None:
    for seconds_waited in range(max_wait_secs):
        try:
            state = trial_state(sess, trial_id)
        except api.errors.NotFoundException:
            logging.warning("Trial not yet available to check state: " "trial {}".format(trial_id))
            time.sleep(0.25)
            continue

        if state == target_state:
            return

        if is_terminal_state(state):
            if state != target_state:
                report_failed_trial(sess, trial_id, target_state=target_state, state=state)

            pytest.fail(
                f"Trial {trial_id} terminated in {state.value} state, "
                f"expected {target_state.value}"
            )

        if seconds_waited > 0 and seconds_waited % log_every == 0:
            print(
                f"Waited {seconds_waited} seconds for trial {trial_id} "
                f"(currently {state.value}) to reach {target_state.value}"
            )

        time.sleep(1)

    else:
        state = trial_state(sess, trial_id)
        if target_state == bindings.trialv1State.COMPLETED:
            kill_trial(sess, trial_id)
        report_failed_trial(sess, trial_id, target_state=target_state, state=state)
        pytest.fail(
            "Trial did not reach target state {} after {} seconds".format(
                target_state.value, max_wait_secs
            )
        )


def experiment_has_active_workload(sess: api.Session, experiment_id: int) -> bool:
    r = sess.get("tasks").json()
    for task in r.values():
        if f"Experiment {experiment_id}" in task["name"] and len(task["resources"]) > 0:
            return True

    return False


def wait_for_experiment_active_workload(
    sess: api.Session, experiment_id: int, max_ticks: int = conf.MAX_TASK_SCHEDULED_SECS
) -> None:
    for _ in range(conf.MAX_TASK_SCHEDULED_SECS):
        if experiment_has_active_workload(sess, experiment_id):
            return

        time.sleep(1)

    pytest.fail(
        f"The only trial cannot be scheduled within {max_ticks} seconds.",
    )


def wait_for_at_least_n_trials(
    sess: api.Session,
    experiment_id: int,
    n: int,
    timeout: int = 30,
) -> List["TrialPlusWorkload"]:
    """Wait for enough trials to start, then return the trials found."""
    deadline = time.time() + timeout
    while True:
        trials = experiment_trials(sess, experiment_id)
        if len(trials) >= n:
            return trials
        if time.time() > deadline:
            raise TimeoutError(f"did not see {n} trials running in {timeout}s; trials={trials}")


def wait_for_experiment_workload_progress(
    sess: api.Session, experiment_id: int, max_ticks: int = conf.MAX_TRIAL_BUILD_SECS
) -> None:
    for _ in range(conf.MAX_TRIAL_BUILD_SECS):
        trials = experiment_trials(sess, experiment_id)
        if len(trials) > 0:
            only_trial = trials[0]
            if len(only_trial.workloads) > 1:
                return
        time.sleep(1)

    pytest.fail(
        f"Trial cannot finish first workload within {max_ticks} seconds.",
    )


def experiment_has_completed_workload(sess: api.Session, experiment_id: int) -> bool:
    trials = experiment_trials(sess, experiment_id)

    if not any(trials):
        return False

    for t in trials:
        for s in t.workloads:
            if s.training is not None or s.validation is not None:
                return True
    return False


def experiment_first_trial(sess: api.Session, exp_id: int) -> int:
    trials = bindings.get_GetExperimentTrials(sess, experimentId=exp_id).trials

    assert len(trials) > 0
    trial = trials[0]
    trial_id = trial.id
    return trial_id


def experiment_config_json(sess: api.Session, experiment_id: int) -> Dict[str, Any]:
    r = bindings.get_GetExperiment(api_utils.user_session(), experimentId=experiment_id)
    assert r.experiment and r.experiment.config
    return r.experiment.config


def experiment_state(sess: api.Session, experiment_id: int) -> bindings.experimentv1State:
    r = bindings.get_GetExperiment(sess, experimentId=experiment_id)
    return r.experiment.state


def trial_state(sess: api.Session, trial_id: int) -> bindings.trialv1State:
    r = bindings.get_GetTrial(sess, trialId=trial_id)
    return r.trial.state


class TrialPlusWorkload:
    def __init__(
        self, trial: bindings.trialv1Trial, workloads: Sequence[bindings.v1WorkloadContainer]
    ):
        self.trial = trial
        self.workloads = workloads


def experiment_trials(sess: api.Session, experiment_id: int) -> List[TrialPlusWorkload]:
    r1 = bindings.get_GetExperimentTrials(sess, experimentId=experiment_id)
    src_trials = r1.trials
    trials = []
    for trial in src_trials:
        r2 = bindings.get_GetTrial(sess, trialId=trial.id)
        r3 = bindings.get_GetTrialWorkloads(sess, trialId=trial.id, limit=1000)
        trials.append(TrialPlusWorkload(r2.trial, r3.workloads))
    return trials


def cancel_single(sess: api.Session, experiment_id: int, should_have_trial: bool = False) -> None:
    cancel_experiment(sess, experiment_id)

    if should_have_trial:
        trials = experiment_trials(sess, experiment_id)
        assert len(trials) == 1, len(trials)

        trial = trials[0].trial
        assert trial.state == bindings.trialv1State.CANCELED


def kill_single(sess: api.Session, experiment_id: int, should_have_trial: bool = False) -> None:
    kill_experiment(sess, experiment_id)

    if should_have_trial:
        trials = experiment_trials(sess, experiment_id)
        assert len(trials) == 1, len(trials)

        trial = trials[0].trial
        assert trial.state == bindings.trialv1State.CANCELED


def is_terminal_experiment_state(state: bindings.experimentv1State) -> bool:
    return state in (
        bindings.experimentv1State.CANCELED,
        bindings.experimentv1State.COMPLETED,
        bindings.experimentv1State.ERROR,
    )


def is_terminal_state(state: bindings.trialv1State) -> bool:
    return state in (
        bindings.trialv1State.CANCELED,
        bindings.trialv1State.COMPLETED,
        bindings.trialv1State.ERROR,
    )


def num_trials(sess: api.Session, experiment_id: int) -> int:
    return len(experiment_trials(sess, experiment_id))


def num_active_trials(sess: api.Session, experiment_id: int) -> int:
    return sum(
        1
        if t.trial.state
        in [
            bindings.trialv1State.RUNNING,
            bindings.trialv1State.STARTING,
            bindings.trialv1State.PULLING,
        ]
        else 0
        for t in experiment_trials(sess, experiment_id)
    )


def num_completed_trials(sess: api.Session, experiment_id: int) -> int:
    return sum(
        1 if t.trial.state == bindings.trialv1State.COMPLETED else 0
        for t in experiment_trials(sess, experiment_id)
    )


def num_error_trials(sess: api.Session, experiment_id: int) -> int:
    return sum(
        1 if t.trial.state == bindings.trialv1State.ERROR else 0
        for t in experiment_trials(sess, experiment_id)
    )


def trial_logs(sess: api.Session, trial_id: int, follow: bool = False) -> List[str]:
    return [tl.message for tl in api.trial_logs(sess, trial_id, follow=follow)]


def workloads_with_training(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> List[bindings.v1MetricsWorkload]:
    ret: List[bindings.v1MetricsWorkload] = []
    for w in workloads:
        if w.training:
            ret.append(w.training)
    return ret


def workloads_with_validation(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> List[bindings.v1MetricsWorkload]:
    ret: List[bindings.v1MetricsWorkload] = []
    for w in workloads:
        if w.validation:
            ret.append(w.validation)
    return ret


def workloads_with_checkpoint(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> List[bindings.v1CheckpointWorkload]:
    ret: List[bindings.v1CheckpointWorkload] = []
    for w in workloads:
        if w.checkpoint:
            ret.append(w.checkpoint)
    return ret


def check_if_string_present_in_trial_logs(
    sess: api.Session, trial_id: int, target_string: str
) -> bool:
    logs = trial_logs(sess, trial_id, follow=True)
    for log_line in logs:
        if target_string in log_line:
            return True
    print(f"{target_string} not found in trial {trial_id} logs:\n{''.join(logs)}", file=sys.stderr)
    return False


def assert_patterns_in_trial_logs(sess: api.Session, trial_id: int, patterns: List[str]) -> None:
    """Match each regex pattern in the list to the logs, one-at-a-time, in order."""
    assert patterns, "must provide at least one pattern"
    patterns_iter = iter(patterns)
    p = re.compile(next(patterns_iter))
    logs = trial_logs(sess, trial_id, follow=True)
    for log_line in logs:
        if p.search(log_line) is None:
            continue
        # Matched a pattern.
        try:
            p = re.compile(next(patterns_iter))
        except StopIteration:
            # All patterns have been matched.
            return
    # Some patterns were not found.
    text = '"\n  "'.join([p.pattern, *patterns_iter])
    raise ValueError(
        f'the following patterns:\n  "{text}"\nwere not found in the trial logs:\n\n{"".join(logs)}'
    )


def assert_performed_initial_validation(sess: api.Session, exp_id: int) -> None:
    trials = experiment_trials(sess, exp_id)

    assert len(trials) > 0
    workloads = trials[0].workloads

    assert len(workloads) > 0
    zeroth_step = workloads_with_validation(workloads)[0]

    assert zeroth_step.totalBatches == 0


def last_workload_matches_last_checkpoint(
    workloads: Sequence[bindings.v1WorkloadContainer],
) -> None:
    assert len(workloads) > 0

    checkpoint_workloads = workloads_with_checkpoint(workloads)
    assert len(checkpoint_workloads) > 0
    last_checkpoint = checkpoint_workloads[-1]
    assert last_checkpoint.state == bindings.checkpointv1State.COMPLETED

    last_workload = workloads[-1]
    if last_workload.training or last_workload.validation:
        last_workload_detail = last_workload.training or last_workload.validation
        assert last_workload_detail is not None
        assert last_workload_detail.totalBatches == last_checkpoint.totalBatches
    elif last_workload.checkpoint:
        last_checkpoint_detail = last_workload.checkpoint
        assert last_checkpoint_detail is not None
        assert last_checkpoint_detail.totalBatches == last_checkpoint.totalBatches
        assert last_checkpoint_detail.state == bindings.checkpointv1State.COMPLETED


def assert_performed_final_checkpoint(sess: api.Session, exp_id: int) -> None:
    trials = experiment_trials(sess, exp_id)
    assert len(trials) > 0
    last_workload_matches_last_checkpoint(trials[0].workloads)


def run_cmd_and_print_on_error(sess: api.Session, cmd: List[str]) -> None:
    """
    We run some commands to make sure they work, but we don't need their output polluting the logs.
    """
    p = detproc.Popen(sess, cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = p.communicate()
    ret = p.wait()
    if ret != 0:
        print(f"cmd failed: {cmd} exited {ret}", file=sys.stderr)
        print("====== stdout from failed command ======", file=sys.stderr)
        print(out.decode("utf8"), file=sys.stderr)
        print("====== end of stdout ======", file=sys.stderr)
        print("====== stderr from failed command ======", file=sys.stderr)
        print(err.decode("utf8"), file=sys.stderr)
        print("====== end of stderr ======", file=sys.stderr)
        raise ValueError(f"cmd failed: {cmd} exited {ret}")


def run_describe_cli_tests(sess: api.Session, experiment_id: int) -> None:
    """
    Runs `det experiment describe` CLI command on a finished
    experiment. Will raise an exception if `det experiment describe`
    encounters a traceback failure.
    """
    # "det experiment describe" without metrics.
    with tempfile.TemporaryDirectory() as tmpdir:
        run_cmd_and_print_on_error(
            sess,
            [
                "det",
                "experiment",
                "describe",
                str(experiment_id),
                "--outdir",
                tmpdir,
            ],
        )

        assert os.path.exists(os.path.join(tmpdir, "experiments.csv"))
        assert os.path.exists(os.path.join(tmpdir, "workloads.csv"))
        assert os.path.exists(os.path.join(tmpdir, "trials.csv"))

    # "det experiment describe" with metrics.
    with tempfile.TemporaryDirectory() as tmpdir:
        run_cmd_and_print_on_error(
            sess,
            [
                "det",
                "experiment",
                "describe",
                str(experiment_id),
                "--metrics",
                "--outdir",
                tmpdir,
            ],
        )

        assert os.path.exists(os.path.join(tmpdir, "experiments.csv"))
        assert os.path.exists(os.path.join(tmpdir, "workloads.csv"))
        assert os.path.exists(os.path.join(tmpdir, "trials.csv"))


def run_list_cli_tests(sess: api.Session, experiment_id: int) -> None:
    """
    Runs list-related CLI commands on a finished experiment. Will raise an
    exception if the CLI command encounters a traceback failure.
    """

    run_cmd_and_print_on_error(sess, ["det", "experiment", "list-trials", str(experiment_id)])
    run_cmd_and_print_on_error(
        sess,
        ["det", "experiment", "list-checkpoints", str(experiment_id)],
    )
    run_cmd_and_print_on_error(
        sess,
        [
            "det",
            "experiment",
            "list-checkpoints",
            "--best",
            str(1),
            str(experiment_id),
        ],
    )


def report_failed_experiment(sess: api.Session, experiment_id: int) -> None:
    trials = experiment_trials(sess, experiment_id)
    active = sum(1 for t in trials if t.trial.state == bindings.trialv1State.RUNNING)
    paused = sum(1 for t in trials if t.trial.state == bindings.trialv1State.PAUSED)
    stopping_completed = sum(
        1 for t in trials if t.trial.state == bindings.trialv1State.STOPPING_COMPLETED
    )
    stopping_canceled = sum(
        1 for t in trials if t.trial.state == bindings.trialv1State.STOPPING_CANCELED
    )
    stopping_error = sum(1 for t in trials if t.trial.state == bindings.trialv1State.STOPPING_ERROR)
    completed = sum(1 for t in trials if t.trial.state == bindings.trialv1State.COMPLETED)
    canceled = sum(1 for t in trials if t.trial.state == bindings.trialv1State.CANCELED)
    errored = sum(1 for t in trials if t.trial.state == bindings.trialv1State.ERROR)
    stopping_killed = sum(
        1 for t in trials if t.trial.state == bindings.trialv1State.STOPPING_KILLED
    )

    print(
        f"Experiment {experiment_id}: {len(trials)} trials, {completed} completed, "
        f"{active} active, {paused} paused, {stopping_completed} stopping-completed, "
        f"{stopping_canceled} stopping-canceled, {stopping_error} stopping-error, "
        f"{stopping_killed} stopping-killed, {canceled} canceled, {errored} errored",
        file=sys.stderr,
    )

    for trial in trials:
        print_trial_logs(sess, trial.trial.id)


def report_failed_trial(
    sess: api.Session,
    trial_id: int,
    target_state: bindings.trialv1State,
    state: bindings.trialv1State,
) -> None:
    print(f"Trial {trial_id} was not {target_state.value} but {state.value}", file=sys.stderr)
    print_trial_logs(sess, trial_id)


def print_trial_logs(sess: api.Session, trial_id: int) -> None:
    print(f"******** Start of logs for trial {trial_id} ********", file=sys.stderr)
    print("".join(trial_logs(sess, trial_id)), file=sys.stderr)
    print(f"******** End of logs for trial {trial_id} ********", file=sys.stderr)


def run_basic_test(
    sess: api.Session,
    config_file: str,
    model_def_file: str,
    expected_trials: Optional[int],
    create_args: Optional[List[str]] = None,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
    expect_workloads: bool = True,
    expect_checkpoints: bool = True,
    priority: int = -1,
) -> int:
    assert os.path.isdir(model_def_file)
    experiment_id = create_experiment(sess, config_file, model_def_file, create_args)
    if priority != -1:
        set_priority(sess, experiment_id=experiment_id, priority=priority)

    wait_for_experiment_state(
        sess,
        experiment_id,
        bindings.experimentv1State.COMPLETED,
        max_wait_secs=max_wait_secs,
    )
    assert num_active_trials(sess, experiment_id) == 0
    verify_completed_experiment_metadata(
        sess, experiment_id, expected_trials, expect_workloads, expect_checkpoints
    )
    return experiment_id


def run_basic_autotuning_test(
    sess: api.Session,
    config_file: str,
    model_def_file: str,
    expected_trials: Optional[int],
    create_args: Optional[List[str]] = None,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
    expect_workloads: bool = True,
    expect_checkpoints: bool = True,
    priority: int = -1,
    expect_client_failed: bool = False,
    search_method_name: str = "_test",
    max_trials: int = 4,
) -> int:
    assert os.path.isdir(model_def_file)
    orchestrator_exp_id = run_autotuning_experiment(
        sess, config_file, model_def_file, create_args, search_method_name, max_trials
    )
    if priority != -1:
        set_priority(sess, experiment_id=orchestrator_exp_id, priority=priority)

    # Wait for the Autotuning Single Searcher ("Orchestrator") to finish
    wait_for_experiment_state(
        sess,
        orchestrator_exp_id,
        bindings.experimentv1State.COMPLETED,
        max_wait_secs=max_wait_secs,
    )
    assert num_active_trials(sess, orchestrator_exp_id) == 0
    verify_completed_experiment_metadata(
        sess, orchestrator_exp_id, expected_trials, expect_workloads, expect_checkpoints
    )
    client_exp_id = fetch_autotuning_client_experiment(sess, orchestrator_exp_id)

    # Wait for the Autotuning Custom Searcher Experiment ("Client Experiment") to finish
    wait_for_experiment_state(
        sess,
        client_exp_id,
        bindings.experimentv1State.COMPLETED
        if not expect_client_failed
        else bindings.experimentv1State.ERROR,
        max_wait_secs=max_wait_secs,
    )
    assert num_active_trials(sess, orchestrator_exp_id) == 0
    verify_completed_experiment_metadata(
        sess, orchestrator_exp_id, expected_trials, expect_workloads, expect_checkpoints
    )
    return client_exp_id


def fetch_autotuning_client_experiment(sess: api.Session, exp_id: int) -> int:
    command = ["det", "experiment", "logs", str(exp_id)]
    env = os.environ.copy()
    env["DET_DEBUG"] = "true"
    p = detproc.run(
        sess,
        command,
        universal_newlines=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env,
    )
    assert p.returncode == 0, f"\nstdout:\n{p.stdout} \nstderr:\n{p.stderr}"
    m = re.search(r"Created experiment (\d+)\n", str(p.stdout))
    assert m is not None
    return int(m.group(1))


def set_priority(sess: api.Session, experiment_id: int, priority: int) -> None:
    command = [
        "det",
        "experiment",
        "set",
        "priority",
        str(experiment_id),
        str(priority),
    ]

    p = detproc.run(
        sess, command, universal_newlines=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    assert p.returncode == 0, f"\nstdout:\n{p.stdout} \nstderr:\n{p.stderr}"


def verify_completed_experiment_metadata(
    sess: api.Session,
    experiment_id: int,
    num_expected_trials: Optional[int],
    expect_workloads: bool = True,
    expect_checkpoints: bool = True,
) -> None:
    # If `expected_trials` is None, the expected number of trials is
    # non-deterministic.
    if num_expected_trials is not None:
        assert num_trials(sess, experiment_id) == num_expected_trials
        assert num_completed_trials(sess, experiment_id) == num_expected_trials

    # Check that every trial and step is COMPLETED.
    trials = experiment_trials(sess, experiment_id)
    assert len(trials) > 0

    for t in trials:
        trial = t.trial
        if trial.state != bindings.trialv1State.COMPLETED:
            report_failed_trial(
                sess,
                trial.id,
                target_state=bindings.trialv1State.COMPLETED,
                state=trial.state,
            )
            pytest.fail(f"Trial {trial.id} was not STATE_COMPLETED but {trial.state.value}")

        if not expect_workloads:
            continue

        if len(t.workloads) == 0:
            print_trial_logs(sess, trial.id)
            raise AssertionError(
                f"trial {trial.id} is in {trial.state.value} state but has 0 steps/workloads"
            )

        # Check that batches appear in increasing order.
        batch_ids = []
        for s in t.workloads:
            if s.training:
                batch_ids.append(s.training.totalBatches)
            if s.validation:
                batch_ids.append(s.validation.totalBatches)
            if s.checkpoint:
                batch_ids.append(s.checkpoint.totalBatches)
                assert s.checkpoint.state in {
                    bindings.checkpointv1State.COMPLETED,
                    bindings.checkpointv1State.DELETED,
                }
        assert all(x <= y for x, y in zip(batch_ids, batch_ids[1:]))

    # The last step of every trial should be the same batch number as the last checkpoint.
    if expect_checkpoints:
        for t in trials:
            last_workload_matches_last_checkpoint(t.workloads)

    shared = os.environ.get("SHARED_CLUSTER", False)
    if not shared:
        # When the experiment completes, all slots should now be free. This requires terminating the
        # experiment's last container, which might take some time (especially on Slurm where our
        # polling is longer).
        max_secs_to_free_slots = 300 if api_utils.is_hpc() else 30
        for _ in range(max_secs_to_free_slots):
            if cluster_utils.num_free_slots(sess) == cluster_utils.num_slots(sess):
                break
            time.sleep(1)
        else:
            raise AssertionError(f"Slots failed to free after experiment {experiment_id}")

    # Run a series of CLI tests on the finished experiment, to sanity check
    # that basic CLI commands don't raise errors.
    run_describe_cli_tests(sess, experiment_id)
    run_list_cli_tests(sess, experiment_id)


# Use Determined to run an experiment that we expect to fail.
def run_failure_test(
    sess: api.Session, config_file: str, model_def_file: str, error_str: Optional[str] = None
) -> int:
    experiment_id = create_experiment(sess, config_file, model_def_file)

    wait_for_experiment_state(sess, experiment_id, bindings.experimentv1State.ERROR)

    # The searcher is configured with a `max_trials` of 8. Since the
    # first step of each trial results in an error, there should be no
    # completed trials.
    #
    # Most of the trials should result in ERROR, but depending on that
    # seems fragile: if we support task preemption in the future, we
    # might start a trial but cancel it before we hit the error in the
    # model definition.

    assert num_active_trials(sess, experiment_id) == 0
    assert num_completed_trials(sess, experiment_id) == 0
    assert num_error_trials(sess, experiment_id) >= 1

    # For each failed trial, check for the expected error in the logs.
    trials = experiment_trials(sess, experiment_id)
    for t in trials:
        trial = t.trial
        if trial.state != bindings.trialv1State.ERROR:
            continue

        logs = trial_logs(sess, trial.id)
        if error_str is not None:
            try:
                assert any(error_str in line for line in logs)
            except AssertionError:
                # Display error log for triage of this failure
                print(f"Trial {trial.id} log did not contain expected message:  {error_str}")
                print_trial_logs(sess, trial.id)
                raise

    return experiment_id


def run_basic_test_with_temp_config(
    sess: api.Session,
    config: Dict[Any, Any],
    model_def_path: str,
    expected_trials: Optional[int],
    create_args: Optional[List[str]] = None,
    max_wait_secs: int = conf.DEFAULT_MAX_WAIT_SECS,
) -> int:
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            util.yaml_safe_dump(config, f)
        experiment_id = run_basic_test(
            sess,
            tf.name,
            model_def_path,
            expected_trials,
            create_args,
            max_wait_secs=max_wait_secs,
        )
    return experiment_id


def run_failure_test_with_temp_config(
    sess: api.Session,
    config: Dict[Any, Any],
    model_def_path: str,
    error_str: Optional[str] = None,
) -> int:
    with tempfile.NamedTemporaryFile() as tf:
        with open(tf.name, "w") as f:
            util.yaml_safe_dump(config, f)
        return run_failure_test(sess, tf.name, model_def_path, error_str=error_str)


def shared_fs_checkpoint_config() -> Dict[str, str]:
    return {
        "type": "shared_fs",
        "host_path": "/tmp",
        "storage_path": "determined-integration-checkpoints",
    }


def s3_checkpoint_config(secrets: Dict[str, str], prefix: Optional[str] = None) -> Dict[str, str]:
    config_dict = {
        "type": "s3",
        "access_key": secrets["INTEGRATIONS_S3_ACCESS_KEY"],
        "secret_key": secrets["INTEGRATIONS_S3_SECRET_KEY"],
        "bucket": secrets["INTEGRATIONS_S3_BUCKET"],
    }
    if prefix is not None:
        config_dict["prefix"] = prefix

    return config_dict


def s3_checkpoint_config_no_creds() -> Dict[str, str]:
    return {"type": "s3", "bucket": "determined-ai-examples"}


def root_user_home_bind_mount() -> Dict[str, str]:
    return {"host_path": "/tmp", "container_path": "/root"}


def has_at_least_one_checkpoint(sess: api.Session, experiment_id: int) -> bool:
    for trial in experiment_trials(sess, experiment_id):
        if len(workloads_with_checkpoint(trial.workloads)) > 0:
            return True
    return False


def wait_for_at_least_one_checkpoint(
    sess: api.Session, experiment_id: int, timeout: int = 120
) -> None:
    for _ in range(timeout):
        if has_at_least_one_checkpoint(sess, experiment_id):
            return
        else:
            time.sleep(1)
    pytest.fail(f"Experiment did not reach at least one checkpoint after {timeout} seconds")
